# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""GoogleMeet test case."""

# The following line disable the abstract function implementation warning.
# pylint: disable=W0223

import time

from lib.testing.lib import base_test
from lib.testing.local.cuj import GoogleMeet as gm
from lib.utils.autotest import bond_http_api


bot_video_file_path = "jamboard_three_close_video_hd.1280_720.yuv"


class GoogleMeet(base_test.BaseTest):
    """Test case for Google Meet."""

    required_vars = dict(
        typing_delay="Key typing delay in seconds",
        gaia_account="The test account",
        gaia_password="The password of the test account",
    )
    optional_vars = dict(
        xperf=f"If run xperf during the test. Default is {gm.xperf}",
        meet_ttl="Google Meet length in minutes. "
        + f"Default is {gm.meet_ttl} minute",
        meet_code="Google Meet code that can be re-used. Default is None",
        duration="Scheduled test duration. "
        + f"Default is {gm.test_duration} minute",
    )
    parameters = {
        # Total participant number in the meet
        "": 2,  # Default is 2
        "2p": 2,
        "16p": 16,
    }

    def prepare(self) -> dict:
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
            meet_code = self.get_meet_code(bot_ttl, bot_num)
        return dict(meet_code=meet_code)

    def get_meet_code(self, bots_ttl_min, bot_count):
        """Get Meet code from Bond API

        Args:
            bots_ttl_min: Meet TTL in minutes.
            bot_count: number of bots to add to the Meet.

        Returns:
            meet code.
        """
        bots_ttl_sec = bots_ttl_min * 60 + 30

        if bot_count <= 0:
            raise ValueError("Bot count must have at least 1 bot")

        bond_api = bond_http_api.BondHttpApi()
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
            time.wait(1)

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
                time.wait(1)

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
