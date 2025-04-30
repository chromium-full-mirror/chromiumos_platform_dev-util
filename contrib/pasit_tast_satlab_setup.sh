#!/bin/bash
#
# This script is used to setup Satlab and tunnels for running tast tests.
# more info: go/pasit-satlab-tast
#
# Usage:
#   pasit_tast_satlab_setup.sh <DUT_HOSTNAME> <SATLAB_IP> [<CROS_CHECKOUT_DIR>]
#
# Example:
#   pasit_tast_satlab_setup.sh satlab-0wgtfqin1846802a-host1 100.115.13.100
#
# The script will setup the following tunnels:
#   - 2200:DUT_HOSTNAME:22 (DUT SSH)
#   - 8300:172.17.0.4:8300 (Passport)
#   - 8080:192.168.231.110:80 (IP Power Switch)
#   - 9999:SERVO_IP:9999 (Servod)
#
#
# roughly this script does the following:
# 1. Determine the DUT/Servo parameters
# 2. start servod container on the Satlab
# 3. start passport container on the Satlab
# 4. setup a tunnel to the DUT, Servod/Passport containers, and the pasit
#    IP power switch
#
# once that is done the user can run tast locally following the instructions
# at go/satlab-local-pasit
#
# Dependencies:
#   - ChromeOS checkout
#   - shivas command installed on PATH go/shivas
#   - crosfleet command installed on PATH go/crosfleet-cli
#   - jq command installed (`sudo apt install jq`)

DUT_HOSTNAME="${1}"
SATLAB_IP="${2}"
# default to the user's chromiumos checkout or let the user specify one
CROS_CHECKOUT_DIR="${3:-$HOME/chromiumos}"
DUT_LEASE_MINUTES=120

DUT_SSH_TUNNEL_PORT=2200
PASSPORT_PORT=8300
IP_POWER_SWITCH_TUNNEL_PORT=8080
SERVOD_PORT=9999

function setup_tunnel() {
  TUNNEL_PGREP="ssh -fNT -L.*satlab-"
  echo "Killing existing tunnels to satlab-*..."
  pgrep -f "${TUNNEL_PGREP}" | xargs kill -9 > /dev/null 2>&1
  # background tunnel to the DUT
  if [[ ! -z "${SERVO_IP}" ]]; then
    SERVO_TUNNEL="-L ${SERVOD_PORT}:${SERVO_IP}:${SERVOD_PORT}"
  fi
  ssh -fNT -L ${DUT_SSH_TUNNEL_PORT}:${DUT_HOSTNAME}:22 -L ${PASSPORT_PORT}:172.17.0.4:${PASSPORT_PORT} -L ${IP_POWER_SWITCH_TUNNEL_PORT}:192.168.231.110:80 ${SERVO_TUNNEL} "${SATLAB_IP}" &
  sleep 5
  # check if the tunnel is up
  if ! pgrep -f "${TUNNEL_PGREP}" > /dev/null 2>&1; then
    echo -e "\n\nFailed to start SSH tunnel to ${SATLAB_IP}"
    exit 1
  fi
  echo "SSH tunnel to ${DUT_HOSTNAME} started"
}

function start_passport() {
  # last step is to start passport container
  echo -e "\n\nStarting passport container..."
  ${CROS_CHECKOUT_DIR}/src/platform/passport/scripts/docker_on_remote.sh "${SATLAB_IP}"
}

function run_command_on_satlab() {
  # runs command on satlab in a bash shell so that we get env vars
  # and path as if we were running command from interactive shell
  # this assumes that no variable expanision is needed on the remote
  # ssh shell
  ssh "${SATLAB_IP}" -t "bash -l -c '$*'"
}

function setup_servo() {
  echo -e "\n\nStarting servod container..."
  local DUT_INFO_FILE="$(mktemp)"
  if ! shivas get dut -json "${DUT_HOSTNAME}" > "${DUT_INFO_FILE}"; then
    echo -e "\n\nFailed to get DUT info from shivas for ${DUT_HOSTNAME}"
    echo "Please check if the DUT is a satlab DUT and that shivas command is installed, on your path and logged in."
    echo "You can try running the following command to validate shivas is working:"
    echo -e "\tshivas get dut ${DUT_HOSTNAME}"
    echo "You can login to shivas by running:"
    echo -e "\tshivas login"
    exit 1
  fi
  local SERVO_SERIAL=$(jq -r '.. | .servoSerial? | select(. != null)' "${DUT_INFO_FILE}")
  if [[ -z "${SERVO_SERIAL}" ]]; then
    echo -e "\n\nFailed to get servo serial for ${DUT_HOSTNAME}"
    echo "Please check if the DUT is a satlab DUT and that shivas is installed and logged in."
    echo "You can try running the following command to validate shivas is working:"
    echo -e "\tshivas get dut ${DUT_HOSTNAME}"
    exit 1
  fi

  local SERVO_CONTAINER_NAME=$(jq -r '.. | .servoHostname? | select(. != null)' "${DUT_INFO_FILE}")
  if [[ -z "${SERVO_CONTAINER_NAME}" ]]; then
    SERVO_CONTAINER_NAME="${DUT_HOSTNAME}-docker_servod"
  fi
  rm "${DUT_INFO_FILE}"

  # need to setup the tunnel before we can get the BOARD and MODEL from the DUT
  setup_tunnel

  local BOARD=$(cros shell localhost:2200 cat /etc/lsb-release | grep "CHROMEOS_RELEASE_BOARD=" | sed 's/CHROMEOS_RELEASE_BOARD=//')
  if [[ -z "${BOARD}" ]]; then
    echo -e "\n\nFailed to get board name for ${DUT_HOSTNAME}"
    exit 1
  fi

  local MODEL=$(cros shell localhost:2200 cros_config / name)
  if [[ -z "${MODEL}" ]]; then
    echo -e "\n\nFailed to get model name for ${DUT_HOSTNAME}"
    exit 1
  fi

  echo "Setting up servod for ${DUT_HOSTNAME}..."
  echo "Servo serial: ${SERVO_SERIAL}"
  echo "SERVO_CONTAINER_NAME: ${SERVO_CONTAINER_NAME}"
  echo "BOARD: ${BOARD}"
  echo "MODEL: ${MODEL}"

  run_command_on_satlab satlab servod stop -host "${DUT_HOSTNAME}"
  if ! run_command_on_satlab satlab servod start -host "${DUT_HOSTNAME}" -board "${BOARD}" -model "${MODEL}" -servo-serial "${SERVO_SERIAL}" -servod-container-name "${SERVO_CONTAINER_NAME}"; then
    echo -e "\n\nFailed to start servod for ${DUT_HOSTNAME}"
    exit 1
  fi

  SERVO_IP="$(ssh "${SATLAB_IP}" /usr/local/bin/docker inspect -f '{{range.NetworkSettings.Networks}}{{.IPAddress}}{{end}}' "${SERVO_CONTAINER_NAME}")"
  echo "SERVO_IP: ${SERVO_IP}"
}

if [[ -z "${DUT_HOSTNAME}" ]]; then
  echo "Usage: pasit_tast_satlab_setup.sh <DUT_HOSTNAME>"
  exit 1
fi

if [[ -z "${SATLAB_IP}" ]]; then
  echo "Usage: pasit_tast_satlab_setup.sh <DUT_HOSTNAME> <SATLAB_IP>"
  echo "You can find the IP of your Satlab in the 'drone_server' entry of the following link:"
  echo "https://chromeos-swarming.appspot.com/bot?id=crossk-${DUT_HOSTNAME}"
  exit 1
fi

if [[ ! -d "${CROS_CHECKOUT_DIR}" ]]; then
  echo "CROS_CHECKOUT_DIR ${CROS_CHECKOUT_DIR} does not exist please specify a valid path."
  exit 1
fi
pushd "${CROS_CHECKOUT_DIR}" > /dev/null

echo -e "\n\nChecking if satlab ${SATLAB_IP} is logged in..."
# check that we can ssh to the satlab and that we are logged in
WHOAMI="$(run_command_on_satlab satlab whoami 2>&1)"
if [[ "${WHOAMI}" =~ "Not logged in." ]]; then
  echo -e "\n\nYou are not logged in to satlab ${SATLAB_IP} please follow steps below to login:\n\n"

  echo "satlab login ${SATLAB_IP}"
  if ! run_command_on_satlab satlab login; then
    echo -e "\n\nFailed to login to satlab ${SATLAB_IP}"
    exit 1
  fi
fi

echo -e "\n\nChecking if DUT ${DUT_HOSTNAME} is leased..."
# check that DUT is leased and if not lease it
if ! crosfleet dut leases | grep -q "${DUT_HOSTNAME}"; then
  echo -e "\n\nDUT ${DUT_HOSTNAME} is not leased, leasing now..."
  crosfleet dut lease -minutes ${DUT_LEASE_MINUTES} -host "${DUT_HOSTNAME}"
fi

start_passport

setup_servo

echo -e "\n\nRestarting tunnel with servo port forwarding to ${SERVO_IP}..."
setup_tunnel

echo -e "\n\nReady to run tests on ${DUT_HOSTNAME}..."
echo "see go/satlab-local-pasit for more details"
