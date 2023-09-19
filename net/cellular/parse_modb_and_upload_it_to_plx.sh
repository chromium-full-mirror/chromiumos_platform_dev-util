#!/bin/bash

# Copyright 2023 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# This is a script to generate capacitor tables from the modb database.
# The script parses the modb using gqui, creates some tables and uploads them
# to colossus.
# The capacitor tables are then imported in the cellular dashboard PLX scripts.
# The script should be run outside the chroot because gqui is not available
# inside it.

src_path="$(pwd | sed  's/chromiumos\/src.*/chromiumos\/src/')"
modb_path="${src_path}"/platform2/shill/mobile_operator_db/serviceproviders.textproto
proto_path="${src_path}"/platform2/shill/mobile_operator_db/mobile_operator_db.proto

if [ ! -f "${modb_path}" ]; then
  echo "Cannot locate modb path. Try running inside the chromium OS src checkout"
  exit 1
fi

cns_path=/cns/ok-d/home/croscellular/shill/mobile_operator_db
gqui from "flatten(textproto:${modb_path}, mno.data.mobile_apn)" \
  proto "${proto_path}":MobileOperatorDB \
  "select mno.data.uuid, mno.data.mobile_apn.apn," \
    "mno.data.mobile_apn.username, mno.data.mobile_apn.password," \
    "mno.data.country, mno.data.localized_name[0].name" \
  --outfile=capacitor:"${cns_path}"/resultMNO.capacitor --overwrite=true

gqui from "flatten(textproto:${modb_path}, mno.mvno.data.mobile_apn)" \
  proto "${proto_path}":MobileOperatorDB \
  "select mno.mvno.data.uuid, mno.mvno.data.mobile_apn.apn," \
    "mno.mvno.data.mobile_apn.username, mno.mvno.data.mobile_apn.password," \
    "mno.mvno.data.country, mno.mvno.data.localized_name[0].name," \
    "mno.data.country" \
  --outfile=capacitor:"${cns_path}"/resultMVNO.capacitor --overwrite=true

gqui from "flatten(textproto:${modb_path}, mno.data.mccmnc )" \
  proto "${proto_path}":MobileOperatorDB \
  "select mno.data.mccmnc, mno.data.country," \
    "mno.data.localized_name[0].name" \
  --outfile=capacitor:"${cns_path}"/countries.capacitor --overwrite=true
