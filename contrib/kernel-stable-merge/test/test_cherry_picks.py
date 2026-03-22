# Copyright 2026 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Test the backend for command cros-ksm-find-cherry-picks."""

from dataclasses import KW_ONLY, dataclass

import pytest

from cros_ksm.find_cherry_picks import find_cherry_picks
from cros_ksm.git import Commit, KernelVersion, Repo

type OidSubject = str

@dataclass
class Expect:

    _: KW_ONLY
    kmajor: str
    downstream_commit: OidSubject
    incoming_commit: OidSubject
    future_commit: OidSubject
    cherry_commits: list[OidSubject]

# TODO: Add more cherry pick expectations for other kernel versions and tags.
expect_list: list[Expect] = [
    Expect(
        kmajor = "6.12",
        downstream_commit =     "7b74924ef6cb02ab5ced4536efab7cae4370597a CHROMIUM: Change to branch owners",
        incoming_commit =       "c4ffbe29c40ed851601bce640d5ead48eaaae08d Linux 6.12.91",
        future_commit =         "296aabce459470a4c1b68ffd0c0c0920e563aaad Linux 6.12.95",
        cherry_commits = [
            '546fc0fe44797e07dfb17898e019e0b543628f53 Revert "perf cgroup: Update metric leader in evlist__expand_cgroup"',
            '9fdca4c4d7eb942e83c81ef0e097303730ef72a6 Revert "perf tool_pmu: Fix aggregation on duration_time"',
            '94fc25a1f82aeaf7777478234ad38d90adb9d94f Revert "perf python: Add parse_events function"',
            '709575a6ab20da63b2183ec4ac89a8e284ddbc1d Revert "perf tool_pmu: Factor tool events into their own PMU"',
            "f056f06e8068b02c3963a0630b3303638d799b46 ata: libata-scsi: do not use the deferred QC feature for ATA_DEFER_PORT",
            "91d1d273bd3f21b40a691590cf5a8504bb373649 ata: libata-scsi: do not use the deferred QC feature on PMPs with CBS",
            "ba9e12cd5b965c29f8a2c4f837372af3c6c262b5 ata: libata-scsi: do not needlessly defer commands when using PMP with FBS",
            '9454e60067ec3711a2d2a0bcf56867d59f7d3975 Revert "ice: fix double-free of tx_buf skb"',
            'a4cb26be376466335d9d94b8d7c4ec5fcbb55811 Revert "ice: Remove jumbo_remove step from TX path"',
            "d4e78c6119d7085d86021e80fb48c660837ba71f tracing: Fix the bug where bpf_get_stackid returns -EFAULT on the ARM64",
            "21a8aa77c2ff439b631ffd1856017aa1d5d32ed4 net/mlx5e: Trigger neighbor resolution for unresolved destinations",
            "8c7d9f3ca1f3d7bb62d3ccb5c4447ed66321bf57 net/mlx5e: Use ip6_dst_lookup instead of ipv6_dst_lookup_flow for MAC init",
            "77cc16f889938f3f72989e01f8cd5ae1c645ee4f x86/fgraph: Fix return_to_handler regs.rsp value",
            "5869f7eb3428ea3449311089d5b0039056a4d5a0 iommu/vt-d: Draining PRQ in sva unbind path when FPD bit set",
            "eda3d0d1b56bf2d6311ac3ebdd8efb6e3696cafb riscv: fgraph: Select HAVE_FUNCTION_GRAPH_TRACER depends on HAVE_DYNAMIC_FTRACE_WITH_ARGS",
            "7ed384db061a264bd806898f7ccab9b98b591488 riscv: fgraph: Fix stack layout to match __arch_ftrace_regs argument of ftrace_return_to_handler",
            "9d651c29530b1944970f8eced30b2637d5c9ce7e arm64: Kconfig: Remove selecting replaced HAVE_FUNCTION_GRAPH_RETVAL",
            "519fb0a42ce5d7e46935577309fb282a5f2c6ea3 ksmbd: fix SID memory leak in set_posix_acl_entries_dacl() on overflow",
            "13fb413ae22a37c69341918a6d651d19a9b0b9b7 smb: client: protect tc_count increment in smb2_find_smb_sess_tcon_unlocked()",
            "f2deaa2f409a4598eaa10f2a93a676c0632af248 cifs: Fix busy dentry used after unmounting",
            "a9f0bfd624ee8a286d6fd2bf0f796e730efb49b0 batman-adv: tp_meter: fix tp_vars reference leak in receiver shutdown",
            "39084bd89421fdf32a0b4ce26891b0087758cf77 firmware: arm_ffa: Align RxTx buffer size before mapping",
            "12a0487945c09760a5968d9333383014ea294117 btrfs: tracepoints: fix sleep while in atomic context in btrfs_sync_file()",
            "be4e99038c1603fa6b329d8ee3e364825e17c353 tracing: Avoid NULL return from hist_field_name() on truncation",
            '829808cbf8cf8a6d07a0e67a5ea2c3fcd63a9e5c net/sched: cls_fw: fix NULL dereference of "old" filters before change()',
            "bca6386dc08750fc7cdcbc7683473748ba3114b9 xfrm: move policy_bydst RCU sync from per-netns .exit to .pre_exit",
            "427913961a0367b9a541d6f0b341f704e02dd209 bcache: fix uninitialized closure object",
            "42c445f8dab9560987a32e2dd52a276507ec394a nfc: nxp-nci: i2c: use rising-edge IRQ on ACPI systems",
            "75977f5f5c4974c72bbedefbf09ad33ebbaa36f1 media: rc: fix race between unregister and urb/irq callbacks",
            "c32874cc226f7a4e4f278d79c17b33a0df7dd8b9 media: rc: ttusbir: fix inverted error logic",
            "0d880d2db9856e94127ab09331363bef59f98005 media: rc: igorplugusb: fix control request setup packet",
            "167838cf63acd9f52ed3c699756eb2b7860d4d77 USB: cdc-acm: Fix bit overlap and move quirk definitions to header",
            "d333af32e4451285e427f2d9c29de3a39f6f6d48 ksmbd: OOB read regression in smb_check_perm_dacl() ACE-walk loops",
            "2d4e59522a06fbd8c6189328cb8dd03c4013bec9 USB: serial: option: add missing RSVD(5) flag for Rolling RW135R-GL",
            "9e42d1f4d0a46a35cf3e131b0430b5f8102605d5 drm/amd/pm/si: Disregard vblank time when no displays are connected",
            "b94a6ccbaf1104dd980150a65fdeb2f69d17d2f5 rxrpc: Fix DATA decrypt vs splice() by copying data to buffer in recvmsg",
            "948f85f5155fb4996a8ff9ae8efdf68f0a564ae4 rxrpc: Fix RESPONSE packet verification to extract skb to a linear buffer",
            "03ffe1112ed88bb3a9bd0b971549bf4d64bfc59a dm cache policy smq: check allocation under invalidate lock",
            "b053fcd8912f06c30f932f5b8ec41c72de474695 ALSA: PCM: Fix wait queue list corruption in snd_pcm_drain() on linked streams",
            "b1180ff50cca48807893ecde7d1f81d573c88c85 mmc: dw_mmc-rockchip: Add missing private data for very old controllers",
            "faf6c45592c7b033dd6f8ee87170caf0293844b3 block: fix handling of dead zone write plugs",
            '5454c4109e13ec70862f231bfded7285fa3d8c90 Revert "net: bonding: fix use-after-free in bond_xmit_broadcast()"',
            "35bdb300ca7a09ea24410d47cced0295d35ba505 locking/mutex: Remove wakeups from under mutex::wait_lock",
            "5799f9bd7fee40370b93ab1ddf001cdc7017c14d locking/rtmutex: Skip remove_waiter() when waiter is not enqueued",
            "be3bfcb34bda04f6a350710db471d4133f950f2c sctp: disable BH before calling udp_tunnel_xmit_skb()",
            '50267ad1d3e176687266c862d35dc305ae94be63 Revert "PCI: qcom: Advertise Hotplug Slot Capability with no Command Completion support"',
            "ce261a20b41db522e320a41bbf1292bf85af66df apparmor: fix use-after-free in rawdata dedup loop",
            "58a5deb220bcac4c73bf58954c0845644c997487 f2fs: fix incorrect FI_NO_EXTENT handling in __destroy_extent_node()",
            "d5c81a095c86fe507c032d08f3a8cfc518444927 ksmbd: fix out-of-bounds read in smb_check_perm_dacl()",
            "7de33483ed6df148a7c8d8dd4dc70c981ccfbcf2 locking: rtmutex: Fix wake_q logic in task_blocks_on_rt_mutex",
        ],
    ),
]

def resolve_rev_subject(repo: Repo, rev_subject: OidSubject) -> Commit:
    rev, subject = rev_subject.split(maxsplit=1)
    commit = repo.resolve_refish_to_commit(rev)
    assert commit.subject.text == subject
    return commit

@pytest.fixture(params = expect_list)
def expect(request: pytest.FixtureRequest) -> Expect:
    return request.param

def test(repo: Repo, expect: Expect):
    # FIXME: The repo fixture is hardcoded to always use 6.12
    expect_kmajor = KernelVersion.parse_major_exact(expect.kmajor)
    assert repo.kernel_major_version == expect_kmajor

    actual_cherry_commits = [
        f"{commit.id} {commit.subject}"
        for commit in find_cherry_picks(
            repo,
            downstream_commit = resolve_rev_subject(repo, expect.downstream_commit),
            incoming_commit = resolve_rev_subject(repo, expect.incoming_commit),
            future_commit = resolve_rev_subject(repo, expect.future_commit),
        )
    ]

    assert actual_cherry_commits == expect.cherry_commits
