#!/usr/bin/env python3
# Copyright 2023 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""SuiteSet Proto generation script."""

import os
import pathlib
import sys
from typing import List, NamedTuple, Set


sys.path.insert(1, str(pathlib.Path(__file__).parent.resolve() / "../../"))

from src.tools import proto_utils  # pylint: disable=wrong-import-position


sys.path.insert(
    1,
    str(
        pathlib.Path(__file__).parent.resolve()
        / "../../../../../../../../config/python"
    ),
)

from chromiumos.test.api import (  # noqa: E402 pylint: disable=import-error,wrong-import-position
    suite_set_pb2,
)
from chromiumos.test.api import (  # noqa: E402 pylint: disable=import-error,wrong-import-position
    test_case_metadata_pb2,
)
from chromiumos.test.api import (  # noqa: E402 pylint: disable=import-error,wrong-import-position
    test_case_pb2,
)


_CWD = os.path.dirname(os.path.abspath(__file__))
_CROS_SRC = os.path.join(_CWD, "../../../../../../../../")
_CONFIG_PROTO_DIR = os.path.join(
    _CROS_SRC, "config", "test", "suite_sets", "generated"
)
_CONFIG_INTERNAL_PROTO_DIR = os.path.join(
    _CROS_SRC, "config-internal", "test", "suite_sets", "generated"
)
_SUITE_PROTO_FILES = [
    os.path.join(_CONFIG_PROTO_DIR, "suites.jsonpb"),
    os.path.join(_CONFIG_INTERNAL_PROTO_DIR, "suites.jsonpb"),
]
_SUITE_SET_PROTO_FILES = [
    os.path.join(_CONFIG_PROTO_DIR, "suite_sets.jsonpb"),
    os.path.join(_CONFIG_INTERNAL_PROTO_DIR, "suite_sets.jsonpb"),
]


class _Suite(NamedTuple):
    """Represents a collection of Tests."""

    suite_id: str
    owners: Set[str]
    bug_component: str
    criteria: str
    tests: Set[str]


class _SuiteSet(NamedTuple):
    """Represents a collection of Suites."""

    suite_set_id: str
    owners: Set[str]
    bug_component: str
    criteria: str
    suite_sets: Set[str]
    suites: Set[str]


def copy_suites(dest: str, test_metadata_dir: str) -> None:
    """Aggregates all Suites and writes them to the given file.

    The Suites are aggregated from protos defined in the config and
    config-internal repos. The test metadata is used to filter out tests not
    relevant to the board being built.

    Args:
        dest: File path to write the resulting SuiteList proto to.
        test_metadata_dir: Path to directory containing test metadata proto
            files.
    """
    suites = _load_suites()

    # Merge Suites with the same name. This is needed support Suites that are
    # defined in both config and config-internal as a result of the Suite
    # containing both public and private tests.
    suites = _union_suites(suites)
    # Because not all tests apply to all boards, we need to look in the test
    # metadata generated for that board and filter out tests from the Suites
    # that are not part of the test metadata.
    test_ids = _load_test_ids(test_metadata_dir)
    suites = _remove_tests_not_in_list(suites, test_ids)

    _write_suites_to_file(suites, dest)


def copy_suite_sets(dest: str):
    """Aggregates all SuiteSets and writes them to the given file.

    The SuiteSets are aggregated from protos defined in the config and
    config-internal repos.

    Args:
        dest: File path to write the resulting SuiteSetList proto to.
    """
    suite_sets = _load_suite_sets()

    # Merge SuiteSets with the same name. This is needed support SuiteSets that
    # are defined in both config and config-internal as a result of the SuiteSet
    # containing both public and private Suite(Set)s.
    suite_sets = _union_suite_sets(suite_sets)

    _write_suite_sets_to_file(suite_sets, dest)


def _load_test_ids(test_metadata_dir):
    """Gets a list of test ids from the test metadata files in the given dir."""
    test_metadata_list_protos = proto_utils.load_protos_from_dir(
        test_metadata_dir,
        test_case_metadata_pb2.TestCaseMetadataList,
        proto_utils.text_proto_parser,
    )
    return _convert_to_test_ids(test_metadata_list_protos)


def _load_suites() -> List[_Suite]:
    """Gets a list of Suites from the protos defined in the config repos."""
    suite_list_protos = proto_utils.load_protos_from_files(
        _SUITE_PROTO_FILES,
        suite_set_pb2.SuiteList,
        proto_utils.json_proto_parser,
    )
    return _convert_to_suites(suite_list_protos)


def _load_suite_sets():
    """Gets a list of SuiteSets from the protos defined in the config repos."""
    suite_set_list_protos = proto_utils.load_protos_from_files(
        _SUITE_SET_PROTO_FILES,
        suite_set_pb2.SuiteSetList,
        proto_utils.json_proto_parser,
    )
    return _convert_to_suite_sets(suite_set_list_protos)


def _convert_to_test_ids(
    test_metadata_list_protos: List[test_case_pb2.TestCase],
) -> Set[str]:
    """Creates a set of test ids using the given test metadata."""
    return [
        test_metadata_proto.test_case.id.value
        for test_metadata_list_proto in test_metadata_list_protos
        for test_metadata_proto in test_metadata_list_proto.values
    ]


def _convert_to_suites(
    suite_list_protos: List[suite_set_pb2.SuiteList],
) -> List[_Suite]:
    """Converts Suite protos to an internal representations of Suites."""
    return [
        _convert_to_suite(suite_proto)
        for suite_list_proto in suite_list_protos
        for suite_proto in suite_list_proto.suites
    ]


def _convert_to_suite_sets(
    suite_set_list_protos: List[suite_set_pb2.SuiteSetList],
) -> List[_SuiteSet]:
    """Converts SuiteSet protos to an internal representations of SuiteSets."""
    return [
        _convert_to_suite_set(suite_set_proto)
        for suite_set_list_proto in suite_set_list_protos
        for suite_set_proto in suite_set_list_proto.suite_sets
    ]


def _convert_to_suite(suite_proto: suite_set_pb2.Suite) -> _Suite:
    """Converts a Suite proto to an internal representations of Suites."""
    return _Suite(
        suite_id=suite_proto.id.value,
        owners={owner.email for owner in suite_proto.metadata.owners},
        bug_component=suite_proto.metadata.bug_component.value,
        criteria=suite_proto.metadata.criteria.value,
        tests={test.value for test in suite_proto.tests},
    )


def _convert_to_suite_set(suite_set_proto: suite_set_pb2.SuiteSet) -> _SuiteSet:
    """Converts a SuiteSet proto to an internal representations of SuiteSets."""
    return _SuiteSet(
        suite_set_id=suite_set_proto.id.value,
        owners={owner.email for owner in suite_set_proto.metadata.owners},
        bug_component=suite_set_proto.metadata.bug_component.value,
        criteria=suite_set_proto.metadata.criteria.value,
        suite_sets={
            suite_set.value for suite_set in suite_set_proto.suite_sets
        },
        suites={suite.value for suite in suite_set_proto.suites},
    )


def _union_suites(suites: List[_Suite]) -> List[_Suite]:
    """Dedupes Suites with the same id by unioning their test lists."""
    suite_id_to_suite = {}
    for suite in suites:
        if suite.suite_id not in suite_id_to_suite:
            suite_id_to_suite[suite.suite_id] = suite
        else:
            existing_suite = suite_id_to_suite[suite.suite_id]
            existing_suite.tests.update(suite.tests)
    return suite_id_to_suite.values()


def _union_suite_sets(suite_sets: List[_SuiteSet]) -> List[_SuiteSet]:
    """Dedupes SuiteSets with the same id by unioning their Suite(Set) lists."""
    suite_set_id_to_suite_set = {}
    for suite_set in suite_sets:
        if suite_set.suite_set_id not in suite_set_id_to_suite_set:
            suite_set_id_to_suite_set[suite_set.suite_set_id] = suite_set
        else:
            existing_suite_set = suite_set_id_to_suite_set[
                suite_set.suite_set_id
            ]
            existing_suite_set.suite_sets.update(suite_set.suite_sets)
            existing_suite_set.suites.update(suite_set.suites)
    return suite_set_id_to_suite_set.values()


def _remove_tests_not_in_list(
    suites: List[_Suite], test_ids: Set[str]
) -> List[_Suite]:
    """Removes the tests from the Suites that are not in the test list."""
    return [
        suite._replace(tests=suite.tests.intersection(test_ids))
        for suite in suites
    ]


def _write_suites_to_file(suites: List[_Suite], dest: str) -> None:
    """Writes the Suites to the given file as a SuiteList text proto."""
    suite_list_proto = _convert_to_suite_list_proto(suites)
    with open(dest, "wb") as f:
        f.write(suite_list_proto.SerializeToString())


def _write_suite_sets_to_file(suite_sets: List[_SuiteSet], dest: str) -> None:
    """Writes the SuiteSets to the given file as a SuiteSetList text proto."""
    suite_set_list_proto = _convert_to_suite_set_list_proto(suite_sets)
    with open(dest, "wb") as f:
        f.write(suite_set_list_proto.SerializeToString())


def _convert_to_suite_list_proto(
    suites: List[_Suite],
) -> suite_set_pb2.SuiteList:
    """Converts the internal representation of Suites to a proto message."""
    return suite_set_pb2.SuiteList(
        suites=[_convert_to_suite_proto(suite) for suite in suites]
    )


def _convert_to_suite_set_list_proto(
    suite_sets: List[_SuiteSet],
) -> suite_set_pb2.SuiteSetList:
    """Converts the internal representation of SuiteSets to a proto message."""
    return suite_set_pb2.SuiteSetList(
        suite_sets=[
            _convert_to_suite_set_proto(suite_set) for suite_set in suite_sets
        ]
    )


def _convert_to_suite_proto(suite: _Suite) -> suite_set_pb2.Suite:
    """Converts the internal representation of a Suite to a proto message."""
    return suite_set_pb2.Suite(
        id=suite_set_pb2.Suite.Id(value=suite.suite_id),
        metadata=suite_set_pb2.Metadata(
            owners=[
                test_case_metadata_pb2.Contact(email=owner)
                for owner in suite.owners
            ],
            bug_component=test_case_metadata_pb2.BugComponent(
                value=suite.bug_component
            ),
            criteria=test_case_metadata_pb2.Criteria(value=suite.criteria),
        ),
        tests=[test_case_pb2.TestCase.Id(value=test) for test in suite.tests],
    )


def _convert_to_suite_set_proto(suite_set: _SuiteSet) -> suite_set_pb2.SuiteSet:
    """Converts the internal representation of a SuiteSet to a proto message."""
    return suite_set_pb2.SuiteSet(
        id=suite_set_pb2.SuiteSet.Id(value=suite_set.suite_set_id),
        metadata=suite_set_pb2.Metadata(
            owners=[
                test_case_metadata_pb2.Contact(email=owner)
                for owner in suite_set.owners
            ],
            bug_component=test_case_metadata_pb2.BugComponent(
                value=suite_set.bug_component
            ),
            criteria=test_case_metadata_pb2.Criteria(value=suite_set.criteria),
        ),
        suite_sets=[
            suite_set_pb2.SuiteSet.Id(value=suite_set_id)
            for suite_set_id in suite_set.suite_sets
        ],
        suites=[
            suite_set_pb2.Suite.Id(value=suite_id)
            for suite_id in suite_set.suites
        ],
    )
