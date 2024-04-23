# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Program global configuration."""

import cros_ca_lib.utils.logging as lg


class Config:
    """Global configuration class."""

    def __init__(self, **kwargs):
        self.__dict__.update(kwargs)

    def update(self, **kwargs):
        self.__dict__.update(kwargs)

    def set(self, c: "Config"):
        self.__dict__ = c.__dict__.copy()


config: Config = Config()


def args_to_config(args: Config):
    """Parse arguments to config"""
    config.set(args)
    if args.no_console_output:
        # Remove the standard output from the local runner to avoid the logs
        # flushing into SSH pipe. Logs will be written to files.
        lg.logger.removeHandler(lg.stream_handler)
    if args.verbose:
        lg.logger.setLevel(lg.logging.DEBUG)
        lg.logger.debug("Running with verbose mode.")
    if args.run_id:
        lg.logger.info("Run id: %s.", args.run_id)


def config_to_args():
    arg_str = []
    args = config
    if args.no_console_output:
        arg_str.append("--no_console_output")
    if args.verbose:
        arg_str.append("--verbose")
    if args.run_id:
        escaped = args.run_id.replace('"', '\\"')
        arg_str.append(f'--run_id="{escaped}"')

    return " ".join(arg_str)
