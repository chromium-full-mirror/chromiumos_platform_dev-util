# Copyright 2023 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""SuiteSet Proto generation script."""

import os
import pathlib
import sys
from typing import Dict, List, NamedTuple, Optional, Set, Union

from google.protobuf import json_format  # pylint: disable=import-error


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
_TEST_PREFIXES = ["tast.", "tauto.", "crosier."]


class Suite(NamedTuple):
    """Represents a collection of Tests."""

    suite_id: str
    owners: Set[str]
    bug_component: str
    criteria: str
    tests: Set[str]


class SuiteSet(NamedTuple):
    """Represents a collection of Suites."""

    suite_set_id: str
    owners: Set[str]
    bug_component: str
    criteria: str
    suite_sets: Set[str]
    suites: Set[str]


CentralizedSuite = Union[SuiteSet, Suite]


def prep_centralized_suites(
    suites_input: List[str],
    suites_output: str,
    suite_sets_input: List[str],
    suite_sets_output: str,
    test_metadata_dir: str,
) -> None:
    """Aggregates all CSuites and writes them to the given out files.

    The Suites/SuiteSets are aggregated from protos defined in
    suite_input/suite_set_input files. The test metadata is used to
    filter out tests not relevant to the board being built.

    Args:
        suites_input: Files to read the jsonpb SuiteList protos from.
        suites_output: File path to write the resulting SuiteList proto to.
        suite_sets_input: Files to read the jsonpb SuiteSetList protos from.
        suite_sets_output: File path to write the resulting SuiteSetList
            proto to.
        test_metadata_dir: Path to directory containing test metadata proto
            files.
    """
    suites = _aggregate_suites(suites_input, test_metadata_dir)
    suite_sets = _aggregate_suite_sets(suite_sets_input)
    _write_suites_to_file(suites, suites_output)
    _write_suite_sets_to_file(suite_sets, suite_sets_output)


def _aggregate_suites(src: List[str], test_metadata_dir: str) -> List[Suite]:
    """Aggregates all Suites across input files."""
    suites = load_suites(src)

    # Merge Suites with the same name. This is needed support Suites that are
    # defined in both config and config-internal as a result of the Suite
    # containing both public and private tests.
    suites = _union_suites(suites)
    # Because not all tests apply to all boards, we need to look in the test
    # metadata generated for that board and filter out tests from the Suites
    # that are not part of the test metadata.
    test_ids = _load_test_ids(test_metadata_dir)
    return _remove_tests_not_in_list(suites, test_ids)


def _aggregate_suite_sets(src: List[str]) -> List[SuiteSet]:
    """Aggregates all SuiteSets across input files."""
    suite_sets = load_suite_sets(src)

    # Merge SuiteSets with the same name. This is needed support SuiteSets that
    # are defined in both config and config-internal as a result of the SuiteSet
    # containing both public and private Suite(Set)s.
    return _union_suite_sets(suite_sets)


def _load_test_ids(test_metadata_dir: str) -> Set[str]:
    """Gets a list of test ids from the test metadata files in the given dir."""
    test_metadata_list_protos = proto_utils.load_protos_from_dir(
        test_metadata_dir,
        test_case_metadata_pb2.TestCaseMetadataList,
        proto_utils.text_proto_parser,
    )
    return _convert_to_test_ids(test_metadata_list_protos)


def load_suites(files: List[str]) -> List[Suite]:
    """Gets a list of Suites from the protos defined in the given files.

    Args:
        files: Files to read the jsonpb SuiteList protos from.

    Returns:
        A list that is the concatenation of all the suite lists.
    """
    suite_list_protos = proto_utils.load_protos_from_files(
        files,
        suite_set_pb2.SuiteList,
        proto_utils.json_proto_parser,
    )
    return _convert_to_suites(suite_list_protos)


def load_suite_sets(files: List[str] = None):
    """Gets a list of SuiteSets from the protos defined in the given files.

    Args:
        files: Files to read the jsonpb SuiteList protos from.

    Returns:
        A list that is the concatenation of all the suite lists.
    """

    suite_set_list_protos = proto_utils.load_protos_from_files(
        files,
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
) -> List[Suite]:
    """Converts Suite protos to an internal representations of Suites."""
    return [
        _convert_to_suite(suite_proto)
        for suite_list_proto in suite_list_protos
        for suite_proto in suite_list_proto.suites
    ]


def _convert_to_suite_sets(
    suite_set_list_protos: List[suite_set_pb2.SuiteSetList],
) -> List[SuiteSet]:
    """Converts SuiteSet protos to an internal representations of SuiteSets."""
    return [
        _convert_to_suite_set(suite_set_proto)
        for suite_set_list_proto in suite_set_list_protos
        for suite_set_proto in suite_set_list_proto.suite_sets
    ]


def _convert_to_suite(suite_proto: suite_set_pb2.Suite) -> Suite:
    """Converts a Suite proto to an internal representations of Suites."""
    validation_err = _validate_suite(suite_proto)
    if validation_err:
        raise RuntimeError(validation_err)

    return Suite(
        suite_id=suite_proto.id.value,
        owners={owner.email for owner in suite_proto.metadata.owners},
        bug_component=suite_proto.metadata.bug_component.value,
        criteria=suite_proto.metadata.criteria.value,
        tests={test.value for test in suite_proto.tests},
    )


def _convert_to_suite_set(suite_set_proto: suite_set_pb2.SuiteSet) -> SuiteSet:
    """Converts a SuiteSet proto to an internal representations of SuiteSets."""
    validation_err = _validate_suite_set(suite_set_proto)
    if validation_err:
        raise RuntimeError(validation_err)

    return SuiteSet(
        suite_set_id=suite_set_proto.id.value,
        owners={owner.email for owner in suite_set_proto.metadata.owners},
        bug_component=suite_set_proto.metadata.bug_component.value,
        criteria=suite_set_proto.metadata.criteria.value,
        suite_sets={
            suite_set.value for suite_set in suite_set_proto.suite_sets
        },
        suites={suite.value for suite in suite_set_proto.suites},
    )


def _validate_suite(suite_proto: suite_set_pb2.Suite) -> Optional[str]:
    """Validates the Suite is well-formed."""
    suite_str = json_format.MessageToJson(suite_proto)

    if not (suite_proto.id and suite_proto.id.value):
        return f"Suite missing ID: {suite_str}"

    metadata_err = _validate_metadata(suite_proto.metadata)
    if metadata_err:
        return f"{metadata_err} for Suite: {suite_str}"

    if not suite_proto.tests:
        return f"Suite must contain at least one test: {suite_str}"

    invalid_tests = []
    for test in suite_proto.tests:
        test_name = test.value
        if not any(test_name.startswith(prefix) for prefix in _TEST_PREFIXES):
            invalid_tests.append(test_name)
    if invalid_tests:
        return (
            "Follwing tests don't start with a known test prefix "
            f"({_TEST_PREFIXES}): {invalid_tests}"
        )


def _validate_suite_set(
    suite_set_proto: suite_set_pb2.SuiteSet,
) -> Optional[str]:
    """Validates the SuiteSet is well-formed."""
    suite_set_str = json_format.MessageToJson(suite_set_proto)

    if not (suite_set_proto.id and suite_set_proto.id.value):
        return f"SuiteSet missing ID: {suite_set_str}"

    metadata_err = _validate_metadata(suite_set_proto.metadata)
    if metadata_err:
        return f"{metadata_err} for SuiteSet: {suite_set_str}"

    if not (suite_set_proto.suite_sets or suite_set_proto.suites):
        return f"SuiteSet must contain a Suite or SuiteSet: {suite_set_str}"

    if suite_set_proto.suite_sets:
        sub_suite_sets = [s.value for s in suite_set_proto.suite_sets]
        if _has_dupes(sub_suite_sets):
            return (
                f"SuiteSet can't have duplicate sub-SuiteSets: {suite_set_str}"
            )

    if suite_set_proto.suites:
        sub_suites = [s.value for s in suite_set_proto.suites]
        if _has_dupes(sub_suites):
            return f"SuiteSet can't have duplicate sub-suites: {suite_set_str}"


def _validate_metadata(metadata: suite_set_pb2.Metadata) -> Optional[str]:
    """Validates the Metadata is well-formed."""
    if not metadata.owners:
        return "metadata missing owners"
    if not (metadata.bug_component and metadata.bug_component.value):
        return "metadata missing bug component"
    if not (metadata.criteria and metadata.criteria.value):
        return "metadata missing criteria"


def _has_dupes(l: List) -> bool:
    """Returns true if the list has duplicate elements, false otherwise."""
    return len(l) != len(set(l))


def _union_suites(suites: List[Suite]) -> List[Suite]:
    """Dedupes Suites with the same id by unioning their test lists."""
    suite_id_to_suite = {}
    for suite in suites:
        if suite.suite_id not in suite_id_to_suite:
            suite_id_to_suite[suite.suite_id] = suite
        else:
            existing_suite = suite_id_to_suite[suite.suite_id]
            existing_suite.tests.update(suite.tests)
    return suite_id_to_suite.values()


def _union_suite_sets(suite_sets: List[SuiteSet]) -> List[SuiteSet]:
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
    suites: List[Suite], test_ids: Set[str]
) -> List[Suite]:
    """Removes the tests from the Suites that are not in the test list."""
    suites_out = []
    for suite in suites:
        updated_tests = []
        for test in suite.tests:
            if test not in test_ids:
                print(f"Removing test: {test}")
            else:
                updated_tests.append(test)
        suites_out.append(suite._replace(tests=updated_tests))
    return suites_out


def validate_centralized_suites(
    csuites: List[CentralizedSuite], ensure_descendants_exist: bool = True
) -> None:
    """Validates csuite mappings are wellformed.

    Ensures all csuite ids are unique, all sub-csuites exist (if
    ensure_descendants_exist is True), and no cycles
    exist with the csuites.

    Args:
        csuites: The list of centralized suites to validate.
        ensure_descendants_exist: If an error should be thrown if a csuite
            descendant does not exist in the mappings.
    """
    mappings = {}
    for csuite in csuites:
        csuite_id = _get_csuite_id(csuite)
        if csuite_id in mappings:
            raise RuntimeError(f"csuite id not is unique: {csuite_id}")
        mappings[csuite_id] = csuite
    for csuite_id in mappings:
        _assert_acyclic_csuite(csuite_id, mappings, ensure_descendants_exist)


def _get_csuite_id(csuite: CentralizedSuite) -> str:
    """Returns the id of the given csuite."""
    if isinstance(csuite, Suite):
        return csuite.suite_id
    elif isinstance(csuite, SuiteSet):
        return csuite.suite_set_id
    else:
        raise ValueError(f"unsupported datatype: {type(csuite)}")


def _get_sub_csuite_ids(csuite: CentralizedSuite) -> List[str]:
    """Returns the csuties contained within the given csuite."""
    if isinstance(csuite, Suite):
        return []
    elif isinstance(csuite, SuiteSet):
        return list(csuite.suite_sets) + list(csuite.suites)
    else:
        raise ValueError(f"unsupported datatype: {type(csuite)}")


def _assert_acyclic_csuite(
    csuite_id: str,
    mappings: Dict[str, CentralizedSuite],
    ensure_descendants_exist: bool,
) -> None:
    """Validates csuite is acyclic and all sub-csuites."""

    def _assert_acyclic_csuite_inner(csuite_id: str, visted: Set[str]) -> None:
        if csuite_id in visted:
            raise RuntimeError(f"cycle detected with csuite: {csuite_id}")
        if csuite_id not in mappings:
            if ensure_descendants_exist:
                raise RuntimeError(f"csuite does not exist: {csuite_id}")
            else:
                return
        # The visited list is copied so that recursive calls don't mutate each
        # other's visited list.
        new_visted = visted.copy()
        new_visted.add(csuite_id)
        csuite = mappings[csuite_id]
        for sub_csuite_id in _get_sub_csuite_ids(csuite):
            _assert_acyclic_csuite_inner(sub_csuite_id, new_visted)

    _assert_acyclic_csuite_inner(csuite_id, set())


def _write_suites_to_file(suites: List[Suite], dest: str) -> None:
    """Writes the Suites to the given file as a SuiteList text proto."""
    suite_list_proto = _convert_to_suite_list_proto(suites)
    with open(dest, "wb") as f:
        f.write(suite_list_proto.SerializeToString())


def _write_suite_sets_to_file(suite_sets: List[SuiteSet], dest: str) -> None:
    """Writes the SuiteSets to the given file as a SuiteSetList text proto."""
    suite_set_list_proto = _convert_to_suite_set_list_proto(suite_sets)
    with open(dest, "wb") as f:
        f.write(suite_set_list_proto.SerializeToString())


def _convert_to_suite_list_proto(
    suites: List[Suite],
) -> suite_set_pb2.SuiteList:
    """Converts the internal representation of Suites to a proto message."""
    return suite_set_pb2.SuiteList(
        suites=[_convert_to_suite_proto(suite) for suite in suites]
    )


def _convert_to_suite_set_list_proto(
    suite_sets: List[SuiteSet],
) -> suite_set_pb2.SuiteSetList:
    """Converts the internal representation of SuiteSets to a proto message."""
    return suite_set_pb2.SuiteSetList(
        suite_sets=[
            _convert_to_suite_set_proto(suite_set) for suite_set in suite_sets
        ]
    )


def _convert_to_suite_proto(suite: Suite) -> suite_set_pb2.Suite:
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


def _convert_to_suite_set_proto(suite_set: SuiteSet) -> suite_set_pb2.SuiteSet:
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
