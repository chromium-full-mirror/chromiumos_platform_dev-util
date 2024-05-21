# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""GoogleMeet test case."""

import tempfile
import time
from typing import Any, Dict

import cros_ca_lib.definition as df
from cros_ca_lib.testing.lib import base_test
from cros_ca_lib.testing.local.cuj import GoogleMeet as gm
from cros_ca_lib.utils.autotest import bond_http_api


bot_video_file_path = "jamboard_three_close_video_hd.1280_720.yuv"


class GoogleMeet(base_test.BasePrepare):
    """Test case for Google Meet."""

    required_vars = {
        "typing_delay": "Key typing delay in seconds",
        f"{df.VAR_GAIA_ACCOUNT}": "The test account",
        f"{df.VAR_GAIA_PASSWORD}": "The password of the test account",
    }
    optional_vars = {
        f"{df.VAR_BOND_CREDENTIALS}": (
            "The Bond API credentials in JSON format. If not provided, the one"
            + f" in {df.default_bond_creds_file} will be used"
        ),
        "meet_ttl": "Google Meet length in minutes. "
        + f"Default is {gm.meet_ttl} minute",
        "meet_code": "Google Meet code that can be re-used. Default is None",
        "duration": "Scheduled test duration. "
        + f"Default is {gm.test_duration} minute",
        "xperf": f"If run xperf during the test. Default is {gm.xperf}",
    }
    parameters = {
        # Total participant number in the meet
        "": 2,  # Default is 2
        "2p": 2,
        "16p": 16,
    }

    def _prepare(self) -> Dict[str, Any]:
        meet_code = None
        try:
            meet_code = self.var("meet_code")
        except Exception:
            pass

        if not meet_code:
            bot_ttl = gm.meet_ttl
            try:
                bot_ttl = float(self.var("meet_ttl"))
            except Exception:
                pass
            bot_num = self.parameter - 1
            bond_credentials = None
            try:
                bond_credentials = self.var(df.VAR_BOND_CREDENTIALS)
            except Exception:
                pass
            if bond_credentials:
                with tempfile.NamedTemporaryFile(
                    delete=True, buffering=0
                ) as temp_file:
                    temp_file.write(
                        bond_credentials.encode()
                    )  # Encode content to bytes before writing
                    meet_code = self.get_meet_code(
                        bot_ttl, bot_num, temp_file.name
                    )
            else:
                meet_code = self.get_meet_code(
                    bot_ttl, bot_num, df.default_bond_creds_file
                )
        return dict(meet_code=meet_code)

    def get_meet_code(self, bots_ttl_min, bot_count, creds_file):
        """Get Meet code from Bond API

        Args:
            bots_ttl_min: Meet TTL in minutes.
            bot_count: number of bots to add to the Meet.
            creds_file: Bond API credential file.

        Returns:
            meet code.
        """
        bots_ttl_sec = bots_ttl_min * 60 + 30

        if bot_count <= 0:
            raise ValueError("Bot count must have at least 1 bot")

        bond_api = bond_http_api.BondHttpApi(creds_file)
        meet_code = bond_api.CreateConference()

        first_bot_added = False
        try_num = 3  # try 3 times
        for _ in range(try_num):
            added_bots = bond_api.AddBotsRequest(
                meet_code,
                1,
                bots_ttl_sec,
                requested_layout="SPOTLIGHT",
                video_file_path=bot_video_file_path,
            )
            if len(added_bots) == 1:
                first_bot_added = True
                break
            time.sleep(1)

        if not first_bot_added:
            raise Exception(
                f"Failed to add the first bot after {try_num} retries"
            )
        to_add = bot_count - 1
        if to_add > 0:
            for _ in range(try_num):
                added_bots = bond_api.AddBotsRequest(
                    meet_code,
                    to_add,
                    bots_ttl_sec,
                    video_file_path=bot_video_file_path,
                )
                to_add -= len(added_bots)
                if to_add == 0:
                    break
                time.sleep(1)

            if to_add > 0:
                raise Exception(
                    f"Failed to add all the bots after {try_num} retries; "
                    + f"{to_add} to be added."
                )

        self.logger.info(
            f"Meet code: {meet_code}; duration: {bots_ttl_min} minutes; "
            + f"bots: {bot_count}"
        )
        return meet_code
