# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Program global configuration."""

from typing import NamedTuple

import lib.utils.logging as lg


class Config:
    """Global configuration class."""

    verbose: bool = False


config = Config()


def args_to_config(arguments: NamedTuple):
    """Parse arguments to config"""
    config.verbose = arguments.verbose
    if config.verbose:
        lg.logger.setLevel(lg.logging.DEBUG)
        lg.logger.debug("Running with verbose mode.")


def config_to_args():
    args = ""
    if config.verbose:
        args += "--verbose"
    return args
