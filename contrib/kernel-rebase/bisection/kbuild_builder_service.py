# Copyright 2023 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# pylint: disable=no-name-in-module
# pylint: disable=protected-access
# pylint: disable=broad-except
# pylint: disable=too-many-function-args
# pylint: disable=not-context-manager

"""Kernel builder service"""

from concurrent.futures import as_completed
from concurrent.futures import ThreadPoolExecutor
from multiprocessing import Queue
from select import select
from socket import AF_INET
from socket import SOCK_DGRAM
from socket import socket

import grpc
from kbuild_builder_pb2 import BisectKbuildRequest
from kbuild_builder_pb2 import BisectKbuildResponse
from kbuild_builder_pb2 import DutProvisionRequest
from kbuild_builder_pb2 import DutProvisionResponse
from kbuild_builder_pb2 import Status
from kbuild_builder_pb2_grpc import add_KbuildBuilderServicer_to_server
from kbuild_builder_pb2_grpc import KbuildBuilderServicer
import sh


DIAG_PORT = 50050
CTRL_PORT = 50051
KERNEL_PATH = "/home/builder/chromiumos/src/third_party/kernel/upstream"
KERNEL_TARGET = "chromeos-kernel-upstream"
CROS_CMD_PREFIX = "bash --login -c"


class Intf(KbuildBuilderServicer):
    """Implements server side of kbuild builder interface"""

    def __init__(self, q_rx, q_ctrl):
        self.q_rx = q_rx
        self.q_ctrl = q_ctrl

    def process_msg(self, request):
        """Processes received message"""

        print("received request message:\n")
        print(type(request))
        print(request)

        # send received message to control thread for processing
        self.q_ctrl.put(request)

        # wait for response
        msg = self.q_rx.get()

        print("sending response message:\n")
        print(type(msg))
        print(msg)
        return msg

    def CreateBisectKbuild(self, request, context):
        return self.process_msg(request)


def network_thread(q_rx, q_ctrl):
    """Network thread

    Network thread is reponsible for receiving kbuild_builder.proto requests and
    sending replies after request processing is completed by a builder instance
    """

    print("starting network thread")

    # We allow kernel builder to process one request at a time that's why we set
    # maximum_concurrent_rpcs to 1. This simplifies processing on the kernel builder side.
    # If a request is received when another is being  processed then RESOURCE_EXHAUSTED error
    # will be returned. It is responsibility of kernel build disaptcher to send message to kernel
    # builder and wait for response before sending another message.
    server = grpc.server(ThreadPoolExecutor(), maximum_concurrent_rpcs=1)
    add_KbuildBuilderServicer_to_server(Intf(q_rx, q_ctrl), server)
    server.add_insecure_port("[::]:" + str(CTRL_PORT))
    server.start()
    server.wait_for_termination()

    print("stopping network thread")


def is_from_net(msg):
    "Checks if message is from network thread"

    if isinstance(msg, BisectKbuildRequest):
        return True
    if isinstance(msg, DutProvisionRequest):
        return True
    return False


def is_from_builder(msg):
    "Checks if message is from builder thread"

    if isinstance(msg, BisectKbuildResponse):
        return True
    if isinstance(msg, DutProvisionResponse):
        return True
    return False


def control_thread(q_rx, q_net, q_builder):
    """Control thread

    Control thread is reponsible for passing requests to builder thread and reponses
    from builder thread to network thread. It is also responsible for storing and
    providing builder diagnostic information, for example builder's current state,
    serviced requests with their completion status and execution times
    """

    print("starting control thread")

    # create socket for retrieving diagnostic information
    sock = socket(family=AF_INET, type=SOCK_DGRAM)
    sock.bind(("127.0.0.1", DIAG_PORT))

    while True:
        rlist, _, _ = select([sock, q_rx._reader], [], [])

        if sock in rlist:
            print("received msg from diagnostic socket")
            _, _ = sock.recvfrom(1024)

        elif q_rx._reader in rlist:
            msg = q_rx.get()
            if is_from_net(msg):
                print("received msg from net thread\n")
                q_builder.put(msg)
            elif is_from_builder(msg):
                print("received msg from builder thread\n")
                q_net.put(msg)

    print("stopping control thread")


def builder_thread(q_rx, q_ctrl):
    """Builder thread

    Builder thread receives kernel build and DUT update requests and executes
    the requests in cros_sdk. After completion it returns status of the completed
    request to control thread.
    """

    print("starting builder thread")

    while True:
        msg = q_rx.get()

        if isinstance(msg, BisectKbuildRequest):

            with sh.pushd(KERNEL_PATH):

                # Checkout branch branch_name that we are requested to build from,
                # if the branch is not found then do 'git fetch' and retry, finally
                # checkout to the requested sha which should be located on the branch_name
                print("checkout to branch: " + msg.branch_name)
                sh.git(
                    "checkout",
                    "remotes/cros/merge/continuous/" + msg.branch_name,
                )

                # Run cros_workon-* for requested board_name to make sure that kernel
                # will be built from source in src/third_party/kernel/upstream.
                # It is also important to check if /build/board_name directory exists
                # in the cros_sdk chroot, if it does not exist then it needs to be setup
                # by running 'setup_board -b board_name'
                cros_sdk_cmd = sh.Command("cros_sdk")
                print("cros workon: " + msg.board_name)
                ret = cros_sdk_cmd(
                    "bash",
                    "--login",
                    "-c",
                    "cros_workon-" + msg.board_name + " start " + KERNEL_TARGET,
                )

                # Build the kernel
                print("build: " + KERNEL_TARGET)
                ret = cros_sdk_cmd(
                    "bash",
                    "--login",
                    "-c",
                    "emerge-" + msg.board_name + " " + KERNEL_TARGET,
                )
                print("build completed")
                print(ret)

            # When kernel build is successfully completed then archive build artifacts
            # into board_name+branch_name+commit_sha.tgz, for example
            # hatch-kernelnext_chromeos-kernelupstream_6.2-rc3_
            # d1678bf45f21fa5ae4a456f821858679556ea5f8.tgz and upload it to gs://kcr-bisection
            # bucket, script to archive kernel artifacts can be found in b\258652964
            msg = BisectKbuildResponse(
                kbuild_status=Status.COMPLETED,
                kbuild_archive_path=(
                    "gs://kcr-bisection/volteer-kernelnext_chromeos-"
                    "kernelupstream-6.2-rc3_"
                    "d1678bf45f21fa5ae4a456f821858679556ea5f8.tgz"
                ),
            )

            q_ctrl.put(msg)

        else:
            print("received unknown msg: ")
            print(type(msg))
            print(msg)

    print("stopping builder thread")


# Create queues for communication between threads
q_network = Queue()
q_control = Queue()
q_kbuilder = Queue()

with ThreadPoolExecutor() as ex:
    futures = []
    futures.append(ex.submit(network_thread, q_network, q_control))
    futures.append(ex.submit(control_thread, q_control, q_network, q_kbuilder))
    futures.append(ex.submit(builder_thread, q_kbuilder, q_control))

    for future in as_completed(futures):
        try:
            result = future.result()
        except Exception as e:
            print(e)
            break
