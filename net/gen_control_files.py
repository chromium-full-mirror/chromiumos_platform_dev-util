# Copyright 2022 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Loop over carriers to generate cellular control files

Since cellular control files need to be repeated per carrier, generate
them. Also, spilt cellular tests into multiple control files so that
stable tests are run immediately after reboot.
"""

import argparse
import os


prefix = """# Copyright 2023 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# Do not edit manually. Generated using ~/chromiumos/src/platform/dev/net/gen_control_files.py

AUTHOR = 'Chromium OS team'
NAME = 'tast.{primary_suite}-{carrier}-{tag}'
METADATA = {{
    "contacts": ["chromeos-cellular-team@google.com"],
    "bug_component": "b:167157", # ChromeOS > Platform > Connectivity > Cellular
    }}
TIME = 'MEDIUM'
TEST_TYPE = 'Server'
ATTRIBUTES = '{suites}'
MAX_RESULT_SIZE_KB = 1024 * 1024
PY_VERSION = 3
DEPENDENCIES = "carrier:{carrier}{extra_dependencies}"

DOC = \'\'\'

See https://chromium.googlesource.com/chromiumos/platform/tast/ for
more information.

See http://go/tast-failures for information about investigating failures.
\'\'\'

import json
import tempfile
import yaml

def run(machine):
    host = hosts.create_host(machine)
    with tempfile.NamedTemporaryFile(suffix='.yaml', mode='w+') as temp_file:
        host_info = host.host_info_store.get()
        yaml.safe_dump({{'autotest_host_info_labels':
                        json.dumps(host_info.labels)}},
                        stream=temp_file)
        temp_file.flush()
"""

suffix = """
parallel_simple(run, machines)
"""

"""
Writes a control file at out_dir/control.suite-carrier-tag. The argument l_tests should specify the tests that need to be run in the control file. If l_suite contains more than one element, the l_suite[0] is used to name the control file.
"""


def write_control_file(
    l_out_dir, l_suite, l_carrier, l_extra_dependencies, l_tag, l_tests
):
    primary_suite = l_suite[0]  # used to name the file and autotest
    suites = ""
    for i, s in enumerate(l_suite):
        if i == 0:
            suites = "suite:" + s
        else:
            suites = suites + ", suite:" + s

    with open(
        os.path.join(l_out_dir, f"control.{primary_suite}-{l_carrier}-{l_tag}"),
        "w",
        encoding="utf-8",
    ) as control_file:
        control_file.write(
            prefix.format(
                primary_suite=primary_suite,
                carrier=l_carrier,
                extra_dependencies=l_extra_dependencies,
                tag=l_tag,
                suites=suites,
            )
            + l_tests
            + suffix
        )


single_test_template = """
        host.reboot()
        job.run_test('tast',
                    host=host,
                    clear_tpm = False,
                    test_exprs={test_exprs},
                    ignore_test_failures=True, max_run_sec=10800,
                    command_args=args,
                    varsfiles=[temp_file.name]
                    )
"""


parser = argparse.ArgumentParser()
parser.add_argument(
    "-o", "--out_dir", help="Output directory for control files"
)
args = parser.parse_args()
out_dir = (
    args.out_dir
    if args.out_dir
    else os.path.join(
        os.path.expanduser("~"),
        "chromiumos/src/third_party/autotest/files/server/site_tests/tast/",
    )
)

for carrier in [
    "verizon",
    "tmobile",
    "att",
    "amarisoft",
    "vodafone",
    "rakuten",
    "ee",
    "kddi",
    "docomo",
    "softbank",
    "fi",
    "roger",
    "bell",
    "telus",
]:
    # used by the cellular dashboard to identify bad DUTs
    tests = single_test_template.format(test_exprs="['cellular.IsModemUp']")
    write_control_file(
        out_dir,
        ["cellular_ota", "cellular_repair"],
        carrier,
        "",
        "is_modem_up",
        tests,
    )

    extra_dependencies = ", cellular_modem_state:NORMAL"

    exclude_sms = "" if carrier in ["tmobile", "att"] else ' && !"cellular_sms"'
    tests = single_test_template.format(
        test_exprs=f'[\'("group:cellular" && "cellular_sim_active" && "cellular_unstable" && !"cellular_run_isolated" && !"cellular_e2e" && (!"cellular_carrier_dependent" || "cellular_carrier_{carrier}")'
        + exclude_sms
        + ")']"
    )
    write_control_file(
        out_dir,
        ["cellular_ota_flaky", "cellular_repair"],
        carrier,
        extra_dependencies,
        "platform",
        tests,
    )

    tests = single_test_template.format(
        test_exprs=f'[\'("group:cellular" && "cellular_sim_active" && !"cellular_unstable" && !"cellular_run_isolated" && !"cellular_e2e" && (!"cellular_carrier_dependent" || "cellular_carrier_{carrier}")'
        + exclude_sms
        + ")']"
    )
    write_control_file(
        out_dir,
        ["cellular_ota", "cellular_repair"],
        carrier,
        extra_dependencies,
        "platform",
        tests,
    )
    tests = single_test_template.format(
        test_exprs='[\'("group:cellular_crosbolt" && "cellular_crosbolt_perf_nightly")\']'
    )
    write_control_file(
        out_dir,
        ["cellular_ota_perf_flaky"],
        carrier,
        extra_dependencies,
        "perf",
        tests,
    )

    # limit carrier independent cq tests to tmobile, verizon, and att to save lab time.
    if carrier in ("tmobile", "verizon", "att"):
        tests = single_test_template.format(
            test_exprs=f'[\'("group:cellular" && "cellular_sim_active" && "cellular_cq" && (!"cellular_carrier_dependent" || "cellular_carrier_{carrier}"))\']'
        )
        write_control_file(
            out_dir, ["cellular-cq"], carrier, extra_dependencies, "cq", tests
        )
