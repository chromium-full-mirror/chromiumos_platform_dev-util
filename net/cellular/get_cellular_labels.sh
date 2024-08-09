#!/bin/bash
# Copyright 2023 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# This script is to be sideloaded onto the DUT and executed while deploying the DUT.
# It reads the SIM and modem attributes from the DUT and populates the shivas commands.
# It supports formulating starfish slot mapping label only for the major US carriers.
# For the rest of the world, the slot mapping label will need to be manually populted.
# In any case, manual review of the command outputs is strongly recommended.
# Guide here: go/cros_cellular_labels_guide

MM1=org.freedesktop.ModemManager1
MM1_IF=org.freedesktop.ModemManager1
DB_PROP=org.freedesktop.DBus.Properties.Get
MM1_IF_MODEM=${MM1_IF}.Modem
MM1_IF_3GPP=${MM1_IF}.Modem.Modem3gpp
MM1_IF_SIM=${MM1_IF}.Sim
SF="Starfish:~$ "
PIN="\033[0;31m\e[1m\e[3m<PIN>\e[0m\e[0m\033[0m"
PUK="\033[0;31m\e[1m\e[3m<PUK>\e[0m\e[0m\033[0m"
OWN="\033[0;31m\e[1m\e[3m<OWNNUMBER>\e[0m\e[0m\033[0m"
# todo: add others
# supports only the following carrier types
CARRIERLIST=( "ATT" "TMOBILE" "VERIZON" "SPRINT" "FI" \
              "DOCOMO" "SOFTBANK" "KDDI" "RAKUTEN" \
              "VODAFONE" "EE" "ROGER" "BELL" "TELUS" \
              "AMARISOFT" "STARFISH" "STARFISHPLUS" )

mm_obj() {
  mmcli -L 2>/dev/null | awk '/\/org\/freedesktop\/ModemManager1\/Modem\// { print $1 }'
}

ord() {
  LC_CTYPE=C printf '%d ' "'$1"
}

print_ascii() {
  foo="$1"
  for (( k=0; k<${#foo}; k++ )); do
    ord "${foo:${k}:1}"
  done
}

sf_find() {
  out=$(find /dev/serial/by-id/ | grep Starfish)
  dev=$(readlink -f "${out}")
  echo "${dev}"
}

sf_cmd() {
  dev="$1"
  cmd="$2"
  stty -F "${dev}" 115200 raw -echo
  exec 3<>"${dev}"
    cat <&3 &
    PID=$!
      echo -ne "\r${cmd}\r" >&3
      sleep 2s
    kill "${PID}"
    wait "${PID}" 2>/dev/null
  exec 3<&-
}

sf_slots() {
  sims=()
  raw=$(sf_cmd "$1" "sim status")
  raw="${raw//$'\r\n'/';'}"
  raw="${raw//${SF}/''}"
  raw="${raw//$'\x1b\x5b\x31\x32\x44\x1b\x5b\x4a'/''}"
  IFS=";" read -r -a outp <<< "${raw}"
  for i in "${outp[@]}"
  do
    # echo -e "$i"
    # print_ascii "$i"
    IFS=" " read -r -a outparr <<< "${i}"
    if [[ ${#outparr[@]} == 7 ]]; then
      if [[ "${outparr[6]}" == "Found" ]]; then
        sims+=("${outparr[4]}")
      fi
    fi
  done
  echo "${sims[@]}"
}

sf_connect() {
  raw=$(sf_cmd "$1" "sim connect -n $2")
  raw="${raw//$'\r\n'/';'}"
  raw="${raw//${SF}/''}"
  raw="${raw//$'\x1b\x5b\x31\x32\x44\x1b\x5b\x4a'/''}"
  IFS=";" read -r -a outp <<< "${raw}"
  nlines=${#outp[@]}
  if [[ "${outp[${nlines}-1]}" =~ " <inf> sim: SIM Mux set to $2"$ ]]; then
    echo "$2"
  else
    echo "-1"
  fi
}

# Gets a DBus property of an interface of a DBus object.
dbus_property() {
  ifc="$1"
  obj="$2"
  prop="$3"
  dbus-send --system --print-reply --fixed --dest="${MM1}" "${obj}" "${DB_PROP}" "string:${ifc}" "string:${prop}"
}

if [ "$#" -ne 2 ]; then
    echo "Illegal number of parameters. Usage:"
    echo -e "\t\e[1m\e[3m$0 <<hostname>> <<carriername>>\e[0m\e[0m"
    echo "Example usage:"
    echo -e "\t\e[1m\e[3m$0 chromeos8-row5-rack20-host24 starfishplus\e[0m\e[0m"
    exit 1
fi
name="${1}"
inputcarrier="${2^^}"
carrierfound=false
# check if carrier input is supported
for xcarrier in "${CARRIERLIST[@]}"; do
  if [ "${xcarrier}" = "${inputcarrier}" ]; then
    carrierfound=true
    break
  fi
done
if [ "${carrierfound}" = false ]; then
  echo "Unsupported carrier: ${inputcarrier}."
  echo "Only the following carriers are supported:"
  echo "${CARRIERLIST[@]}"
  echo "(case insensitive)"
  exit 1
fi
stop modemfwd >/dev/null
issf=false
sims=( "0" )
if [[ "${inputcarrier}" == STARFISH* ]]; then
  issf=true
  dev=$(sf_find)
  raw=$(sf_slots "${dev}")
  IFS=" " read -r -a sims <<< "${raw}"
fi
nsims=${#sims[@]}
sf_label=""
index=1
sim_label=""
for slot in "${sims[@]}"
do
  if [ "${issf}" = true ]; then
    outp1=$(sf_connect "${dev}" "${slot}")
    if [[ "${outp1}" != "${slot}" ]]; then
      echo "error inserting sim in slot ${slot}! skipping..."
      continue
    fi
    if [ "${inputcarrier}" = "STARFISH" ]; then
      modem reset >/dev/null
      sleep 10s
    else
      sleep 20s
    fi
  fi
  mmobj=$(mm_obj)
  simobj=$(dbus_property "${MM1_IF_MODEM}" "${mmobj}" "Sim")
  if [[ ${index} == 1 ]]; then
    imei=$(dbus_property "${MM1_IF_3GPP}" "${mmobj}" "Imei")
    mfr=$(dbus_property "${MM1_IF_MODEM}" "${mmobj}" "Manufacturer")
    model=$(dbus_property "${MM1_IF_MODEM}" "${mmobj}" "Model")
    modem="MODEM_TYPE_UNSPECIFIED"
    case "${mfr}" in
      "Fibocom Wireless Inc.")
        case "${model}" in
          "L850-GL")
            modem="MODEM_TYPE_FIBOCOMM_L850GL"
            ;;
          "Fibocom NL668 Modem")
            modem="MODEM_TYPE_NL668"
            ;;
          "Fibocom FM101-GL Module")
            modem="MODEM_TYPE_FM101"
            ;;
        esac
        ;;
      "QUALCOMM INCORPORATED")
        if [ "${model}" = "Qualcomm(R) Snapdragon(TM) 7c Compute Platform" ]; then
          modem="MODEM_TYPE_QUALCOMM_SC7180"
        fi
        ;;
      "Intel")
        if [ "${model}" = "MBIM [14C3:4D75]" ]; then
          modem="MODEM_TYPE_FM350"
        fi
        ;;
      "Quectel")
        if [ "${model}" = "Quectel EM060K-GL" ]; then
          modem="MODEM_TYPE_EM060"
        fi
        ;;
      *)
        ;;
    esac
    modeminfoUpdateCmd="shivas update dut -name ${name} -modeminfo ${modem},${imei},,${nsims}"
  fi
  eid=$(dbus_property "${MM1_IF_SIM}" "${simobj}" "Eid")
  iccid=$(dbus_property "${MM1_IF_SIM}" "${simobj}" "SimIdentifier")
  net1=$(dbus_property "${MM1_IF_3GPP}" "${mmobj}" "OperatorName")
  if [ -z "${net1}" ]; then
    # Sometimes, the operator name is not in the modem properties when not yet registered.
    net1=$(dbus_property "${MM1_IF_SIM}" "${simobj}" "OperatorName")
  fi

  own1=$(dbus_property "${MM1_IF_MODEM}" "${mmobj}" "OwnNumbers")
  own=$(echo "${own1}" | cut -d ' ' -f 2)
  net="NETWORK_OTHER"
  carrier=""
  pin="${PIN}"
  puk="${PUK}"
  case "${net1}" in
    "Verizon Wireless")
      net="NETWORK_VERIZON"
      carrier="${slot}_verizon"
      ;;
    "T-Mobile")
      net="NETWORK_TMOBILE"
      carrier="${slot}_tmobile"
      ;;
    "AT&T")
      net="NETWORK_ATT"
      carrier="${slot}_att"
      ;;
    "Callbox Operator Nam"|"USIM")
      net="NETWORK_AMARISOFT"
      carrier="${slot}_amarisoft"
      pin=1234
      puk=88888888
      # Callbox DUTs don't have a phone number
      if [ "${own}" = "" ]; then
        own="\"\""
      fi
      ;;
    *)
      net="NETWORK_OTHER"
      net1=$(echo "${net1}" | tr -d '[:space:]')
      if [ "${net1}" = "" ]; then
        net1="unknown"
      fi
      carrier="${slot}_${net1,,}"
      ;;
  esac

  if [ "${own}" = "" ]; then
    own=${OWN}
  fi
  sim_label="${sim_label} -siminfo SIM_PHYSICAL,${index},${eid},FALSE,${iccid},${pin},${puk},${net},${own}"
  sf_label="${sf_label}${carrier}"
  if [[ ${index} != "${nsims}" ]]; then
    sf_label="${sf_label},"
  fi
  (( index++ ))
done

carrierUpdateCmd="shivas update dut -name ${name} -carrier ${inputcarrier}"
echo -e "\n${carrierUpdateCmd}"

if [ "${issf}" = true ]; then
  sfmappingUpdateCmd="shivas update dut -name ${name} -starfishSlotMapping \"${sf_label}\""
  echo -e "\n${sfmappingUpdateCmd}"
fi

echo -e "\n${modeminfoUpdateCmd}"

siminfoUpdateCmd="shivas update dut -name ${name}${sim_label}"
echo -e "\n${siminfoUpdateCmd}"

echo -e "\nplease retrieve ${PIN}, ${PUK} and ${OWN} values from:"
echo "        1. go/cellular_sim_details"
echo "                OR"
echo "        2. the SIM card holding tray"
echo "                OR"
echo "        3. the network service provider"

start modemfwd >/dev/null
