# Copyright 2023 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Utility functions for proto messages."""

import os
from typing import AnyStr, Callable, List, TypeVar

from google.protobuf import json_format  # pylint: disable=import-error
from google.protobuf.message import Message  # pylint: disable=import-error


# Represents a super class of all proto messages
_MessageVar = TypeVar("_MessageVar", bound=Message)


_PB_EXTENSION = "pb"


def load_protos_from_dir(
    directory: str,
    proto_constructor: Callable[[], _MessageVar],
    proto_parser: Callable[[AnyStr, _MessageVar], None],
) -> List[_MessageVar]:
    """Deserializes all *.pb files in the directory into proto messages.

    Args:
        directory: The path to the directory the will be walked.
        proto_constructor: A function that constructs the proto message which
            will be used to load the file contents into.
        proto_parser: A parser that loads a proto file contents into a given
            proto message.

    Returns:
        A list of proto messages.
    """
    protos = []
    for root, _, files in os.walk(directory):
        for file in files:
            if file.endswith(_PB_EXTENSION):
                path = os.path.join(root, file)
                proto = load_proto_from_file(
                    path, proto_constructor, proto_parser
                )
                protos.append(proto)
    return protos


def load_protos_from_files(
    files: List[str],
    proto_constructor: Callable[[], _MessageVar],
    proto_parser: Callable[[AnyStr, _MessageVar], None],
) -> List[_MessageVar]:
    """Deserializes the given files into proto messages.

    Args:
        files: The paths to the files that will be parsed.
        proto_constructor: A function that constructs the proto message which
            will be used to load the file contents into.
        proto_parser: A parser that loads a proto file contents into a given
            proto message.

    Returns:
        A list of proto messages.
    """
    return [
        load_proto_from_file(file, proto_constructor, proto_parser)
        for file in files
        if os.path.isfile(file)
    ]


def load_proto_from_file(
    file: str,
    proto_constructor: Callable[[], _MessageVar],
    proto_parser: Callable[[AnyStr, _MessageVar], None],
) -> _MessageVar:
    """Deserializes the given file into a proto message.

    Args:
        file: The path to the file that will be parsed.
        proto_constructor: A function that constructs the proto message which
            will be used to load the file contents into.
        proto_parser: A parser that loads a proto file contents into a given
            proto message.

    Returns:
        A proto messages.
    """
    proto = proto_constructor()
    with open(file, "rb") as f:
        try:
            proto_parser(f.read(), proto)
        except Exception as err:
            raise RuntimeError(f"error parsing proto file: {file}") from err
    return proto


def json_proto_parser(raw_input: AnyStr, proto: _MessageVar):
    """Parses the given json input into the given proto message.

    Args:
        raw_input: A json string/byte array to parse.
        proto: The proto message to populate.
    """
    json_format.Parse(raw_input, proto)


def text_proto_parser(raw_input: AnyStr, proto: _MessageVar):
    """Parses the given text proto input into the given proto message.

    Args:
        raw_input: A text proto string/byte array to parse.
        proto: The proto message to populate.
    """
    proto.ParseFromString(raw_input)
