# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""A program that establish SSH port forwarding."""

import argparse
import os
from pathlib import Path
import select
import socketserver
import sys

import paramiko


# pylint: disable=wrong-import-position
if __name__ == "__main__" and __package__ is None:
    # Add project root into the python path so lib importing works.
    sys.path.append(str(Path(__file__).resolve().parents[1]))

from bin import utils


# pylint: enable=wrong-import-position


class Handler(socketserver.BaseRequestHandler):
    """The sockert server handler."""

    def handle(self):
        try:
            ssh_channel = self.ssh_transport.open_channel(
                "direct-tcpip",
                (self.remote_host, self.remote_port),
                self.request.getpeername(),
            )
        except Exception as e:
            print(
                "Failed to open SSH channel to "
                + f"{self.remote_host}:{self.remote_port}: {e}"
            )
            return
        if ssh_channel is None:
            print(
                "Cannot open SSH channel to "
                + f"{self.remote_host}:{self.remote_port}"
            )
            return

        while True:
            r, _, _ = select.select(
                [self.request, ssh_channel, self.read_fd], [], []
            )
            if self.request in r:
                data = self.request.recv(1024)
                if len(data) == 0:
                    break
                ssh_channel.send(data)
            if ssh_channel in r:
                data = ssh_channel.recv(1024)
                if len(data) == 0:
                    break
                self.request.send(data)
            if self.read_fd in r:
                break
        self.request.close()
        ssh_channel.close()


class LocalForwarding:
    """A class to manage local port forwarding using SSH.

    This class provides methods to initiate and terminate local port forwarding
    tunnels using the `ssh` command with the `-N` and `-L` flags.
    """

    def __init__(
        self,
        remote_host: str,
        remote_port: int,
        local_port: int,
        ssh_transport: paramiko.Transport,
    ):
        """Initializes an SSHLocalForwarding object.

        Args:
            remote_host: The hostname or IP address of the remote server.
            remote_port: The port number on the remote server to forward
                               traffic from.
            local_port: The port number on the local machine to listen on.
            ssh_transport: The ssh tranport used for port forwarding.
        """
        self.remote_host = remote_host
        self.remote_port = remote_port
        self.local_port = local_port
        self.ssh_transport = ssh_transport
        self.terminate_fd = None

    def start(self):
        """Starts local port forwarding.

        Establishes a connection to the remote server.
        """
        output_fd, self.terminate_fd = os.pipe()

        class ParameterizedHandler(Handler):
            """Socket handler."""

            remote_host = self.remote_host
            remote_port = self.remote_port
            ssh_transport = self.ssh_transport
            read_fd = output_fd

        socketserver.ThreadingTCPServer.allow_reuse_address = True
        socketserver.ThreadingTCPServer.daemon_threads = True
        tcp_server = socketserver.ThreadingTCPServer(
            ("", self.local_port), ParameterizedHandler
        )

        tcp_server.serve_forever()

    def stop(self):
        """Stops local port forwarding.

        Terminates the SSH process established for the port forwarding tunnel.
        """
        if self.terminate_fd:
            os.close(self.terminate_fd)


def parse_arguments(argv):
    """Parse command line arguments

    Args:
        argv: argument list to parse

    Returns:
        Tuple of parsed arguments

    Raises:
        SystemExit if arguments are malformed, or required arguments
            are not present.
    """
    parser = argparse.ArgumentParser(
        description="Run SSH local port forwarding"
    )
    parser.add_argument(
        "localport", type=int, action="store", help="local listening port"
    )
    parser.add_argument(
        "remotehost", type=str, action="store", help="remote host"
    )
    parser.add_argument(
        "remoteport", type=int, action="store", help="remote port"
    )
    parser.add_argument(
        "target",
        type=str,
        action="store",
        help='the DUT SSH connection spec of the form "[user@]host[:port]"',
    )
    parser.add_argument(
        "keyfile", type=str, action="store", help="private key file"
    )
    args = parser.parse_args(argv)

    try:
        ssh_specs = utils.parse_ssh_host_spec(args.target)
    except ValueError as e:
        print(f"Invalid target: {e}")
        parser.print_help()
        sys.exit(1)

    return args, ssh_specs


def main(argv):
    arguments, ssh_specs = parse_arguments(argv)

    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())

    hostname = ssh_specs["hostname"]
    port = ssh_specs["port"]
    username = ssh_specs["user"]
    try:
        client.connect(
            hostname=hostname,
            username=username,
            port=port,
            key_filename=arguments.keyfile,
        )
    except Exception as e:
        print(f"Failed to connect to {username}@{hostname}:{port}: {e}")
        sys.exit(1)

    forward = LocalForwarding(
        arguments.remotehost,
        arguments.remoteport,
        arguments.localport,
        client.get_transport(),
    )
    try:
        forward.start()
    except KeyboardInterrupt:
        sys.exit(0)


if __name__ == "__main__":
    main(sys.argv[1:])
