# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Utils to manage logging."""

from cros_ca_lib.utils import logging as lg
import yaml


logger = lg.logger


gaia_account_file = "/creds/test_accounts/gaia_account.yaml"


def get_gaia_account():
    try:
        with open(gaia_account_file, "r", encoding="utf-8") as file:
            data = yaml.safe_load(file)
    except FileNotFoundError:
        logger.warning("Gaia account file %s doesn't exist.", gaia_account_file)
        raise

    return data["account"], data["password"]
