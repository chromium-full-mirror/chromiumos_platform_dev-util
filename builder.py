# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Package builder for the dev server."""

from __future__ import print_function

import logging
import os
import subprocess

import cherrypy  # pylint: disable=import-error

# This must happen before any local modules get a chance to import
# anything from chromite.
import setup_chromite  # pylint: disable=unused-import,import-error

from chromite.lib import gmerge_binhost


# Relative path to the wrapper directory inside the sysroot.
_SYSROOT_BUILD_BIN = "build/bin"


def _SysrootCmd(sysroot, cmd):
    """Path to the sysroot wrapper for |cmd|.

    Args:
        sysroot: Path to the sysroot.
        cmd: Name of the command.
    """
    return os.path.join(sysroot, _SYSROOT_BUILD_BIN, cmd)


# Module-local log function.
def _Log(message, *args):
    return logging.info(message, *args)


def _OutputOf(command):
    """Runs command, a list of arguments beginning with an executable.

    Args:
        command: A list of arguments, beginning with the executable

    Returns:
        The output of the command

    Raises:
        subprocess.CalledProcessError if the command fails
    """
    command_name = " ".join(command)
    _Log("Executing: " + command_name)

    with subprocess.Popen(command, stdout=subprocess.PIPE) as p:
        output_blob = p.communicate()[0].decode("utf-8")
        if p.returncode != 0:
            raise subprocess.CalledProcessError(p.returncode, command_name)
        return output_blob


class Builder:
    """Builds packages for the devserver."""

    def _ShouldBeWorkedOn(self, board, pkg):
        """Is pkg a package that could be worked on, but is not?"""
        if pkg in _OutputOf(["cros_workon", "--board=" + board, "list"]):
            return False

        # If it's in the list of possible workon targets, we should be working
        # on it
        return pkg in _OutputOf(
            ["cros_workon", "--board=" + board, "--all", "list"]
        )

    def SetError(self, text):
        cherrypy.response.status = 500
        _Log(text)
        return text

    def Build(self, board, pkg, additional_args):
        """Handles a build request from the cherrypy server."""
        _Log("Additional build request arguments: " + str(additional_args))

        def _AppendStrToEnvVar(env, var, additional_string):
            env[var] = env.get(var, "") + " " + additional_string
            _Log("%s flags modified to %s" % (var, env[var]))

        env_copy = os.environ.copy()
        if "use" in additional_args:
            _AppendStrToEnvVar(env_copy, "USE", additional_args["use"])

        if "features" in additional_args:
            _AppendStrToEnvVar(
                env_copy, "FEATURES", additional_args["features"]
            )

        try:
            if not additional_args.get(
                "accept_stable"
            ) and self._ShouldBeWorkedOn(board, pkg):
                return self.SetError(
                    "Package is not cros_workon'd on the devserver machine.\n"
                    "Either start working on the package or pass "
                    "--accept_stable to gmerge"
                )

            sysroot = "/build/%s/" % board
            # If user did not supply -n, we want to rebuild the package.
            usepkg = additional_args.get("usepkg")
            if not usepkg:
                rc = subprocess.call(
                    [_SysrootCmd(sysroot, "emerge"), pkg], env=env_copy
                )
                if rc != 0:
                    return self.SetError("Could not emerge " + pkg)

            # Sync gmerge binhost.
            deep = additional_args.get("deep")
            if not gmerge_binhost.update_gmerge_binhost(sysroot, [pkg], deep):
                return self.SetError("Package %s is not installed" % pkg)

            return "Success\n"
        except OSError as e:
            return self.SetError("Could not execute build command: " + str(e))
