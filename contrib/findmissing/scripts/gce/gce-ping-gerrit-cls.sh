#!/bin/bash
#
# Copyright 2022 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.


WORKSPACE="${HOME}/findmissing_workspace"
GERRIT="${WORKSPACE}/chromite/bin/gerrit"
FINDMISSING="${WORKSPACE}/dev-util/contrib/findmissing"
cd "${FINDMISSING}" || exit
LOG_FILE="/var/log/findmissing/findmissing.log"
ROTATED_REVIEWERS="tzungbi@chromium.org"
GERRIT_INTERFACE="./env/bin/python gerrit_interface.py"

# Thresholds, unit: day(s).
NO_UPDATE_THRESHOLD=7
CQ_PLUS1_RETRY_THRESHOLD=1
# Limits
CQ_PLUS1_LIMIT_PER_DAY=2
CQ_PLUS1_LIMIT=4

has_elapsed() {
    # Return 0 if `threshold` has elapsed from `from`.
    local from threshold now diff

    if [[ -z "$1" ]] || [[ -z "$2" ]]; then
        return 1
    fi
    from="$1"
    threshold="$2"

    now="$(date +%s)"
    diff="$((now - from))"

    if [[ ${diff} -lt ${threshold} ]]; then
        return 1
    fi
    return 0
}

get_chromium_reviewers() {
    local cl_num

    cl_num="$1"

    # Filter out commit-bot@chromium.org.
    "${GERRIT}" --json inspect "${cl_num}" |
        jq -c -r '.[].currentPatchSet.approvals | .[] | select(.type == "CRVW") | .by.email |
                    select(contains("chromium.org"))' |
        grep -v "commit-bot"
}

url_encoded() {
        python3 -c 'import urllib.parse, sys; print(urllib.parse.quote(sys.argv[1]))' "$1"
}

send_ping_msg() {
    local record title link branch q url

    record="$1"
    title="$(echo "${record}" | jq -r .subject)"
    link="$(echo "${record}" | jq -r .url)"
    branch="$(echo "${record}" | jq -r .branch)"

    q="component=167278"
    q="${q}&format=PLAIN"
    q="${q}&type=BUG&priority=P2&severity=S2"
    q="${q}&hotlistIds=5435871"
    q="${q}&title=$(url_encoded "Kernel: fix build failure for (\"${title}\")")"

    read -r -d '' description <<EOF
Fix build failure for ("${title}").

Link: ${link}
Branch: ${branch}
EOF
    q="${q}&description=$(url_encoded "${description}")"
    url="http://b.corp.google.com/createIssue?${q}"

    read -r -d '' msg <<EOF
Dear Reviewer,

You have been selected because this bot believes you are the best owner for this CL.

- If you think you are not the right person for handling the CL:
    - Please add chromeos-kernel-findmissing-reviewers@google.com as a reviewer.
    - Please remove yourself from the reviewer list.
- If the CL is irrelevant to ChromeOS, feel free to abandon it.
- If the CL makes sense to you:
    - If the CQ tests passed and the robot has marked V+1 and AS+1, please provide CR+2 and CQ+2.
    - If the CQ tests failed due to build failures, please provide V-1 and file a bug via [the template](${url}).
- Otherwise, please provide CR-1 and your comments.

Thank you for contributing to ChromeOS kernel stability.
EOF
    "${GERRIT}" message "${cl_num}" "${msg}"
}

check_no_update() {
    local record cl_num last_updated

    record="$1"
    cl_num="$(echo "${record}" | jq -r .cl_num)"
    last_updated="$(echo "${record}" | jq -r .last_updated)"

    if has_elapsed "${last_updated}" "$((86400 * NO_UPDATE_THRESHOLD))"; then
        echo "..Hasn't been changed for more than ${NO_UPDATE_THRESHOLD} day(s)"

        send_ping_msg "${record}" "${cl_num}"

        if [[ -z "$(get_chromium_reviewers "${cl_num}")" ]]; then
            "${GERRIT}" reviewers "${cl_num}" "${ROTATED_REVIEWERS}"
        fi
    fi
}

check_cq_plus1() {
    local record cl_num change_id branch cq v has_failed message date mdate

    record="$1"
    cl_num="$(echo "${record}" | jq -r .cl_num)"
    change_id="$(echo "${record}" | jq -r .change_id)"
    branch="$(echo "${record}" | jq -r .branch | cut -d- -f2)"
    cq="$(echo "${record}" | jq -r .cq)"
    v="$(echo "${record}" | jq -r .v)"

    has_failed=0

    if [[ ${cq} != "0" ]] || [[ ${v} == "1" ]]; then
        echo "..Already handled"
        return
    fi

    if [[ ${v} == "-1" ]]; then
        echo "..Already handled (V-1)"
        return
    fi

    # Reading all messages of the CL.
    while read -r msg; do
        message="$(echo "${msg}" | jq -r .message)"

        # If a new PS has uploaded, retry CQ+1 anyway.
        if echo "${message}" | grep -q "Uploaded patch set"; then
            echo "..Mark CQ+1 because of a new patchset"
            "${GERRIT}" label-cq "${cl_num}" 1
            return
        fi

        if echo "${message}" | grep -q "This CL has passed the run"; then
            # Make sure the latest result passes CQ.
            if [[ ${has_failed} -eq 0 ]]; then
                echo "..CQ+1 has passed"
                echo "..Mark V+1 and AS+1"
                "${GERRIT}" label-v "${cl_num}" 1
                "${GERRIT}" label-as "${cl_num}" 1
                return
            fi
        fi

        if echo "${message}" | grep -q "This CL has failed the run"; then
            if echo "${message}" | grep -q "buildtest-cq"; then
                echo "..CQ failed on buildtest"
                echo "..Mark V-1"
                "${GERRIT}" label-v "${cl_num}" -1
                return
            fi

            has_failed="$((has_failed + 1))"
            if [[ ${has_failed} -ge ${CQ_PLUS1_LIMIT} ]]; then
                echo "..Do nothing because the retries are over the limit"
                return
            fi

            date="$(echo "${msg}" | jq -r .date)"
            mdate="$(date --date="${date}" +%s)"

            if ! has_elapsed "${mdate}" "$((86400 * CQ_PLUS1_RETRY_THRESHOLD))"; then
                if [[ ${has_failed} -ge ${CQ_PLUS1_LIMIT_PER_DAY} ]]; then
                    echo "..Do nothing because the retries are over the limit today"
                    return
                fi
            fi
        fi
    done < <(${GERRIT_INTERFACE} "${change_id}" "${branch}" |
                jq -c -r '
                    reverse |
                    .[] |
                    {
                        author: .author._account_id,
                        date, message
                    }
                ' 2>/dev/null)

    echo "..Mark CQ+1"
    "${GERRIT}" label-cq "${cl_num}" 1
}

{
    echo "Triggered ping Gerrit CLs at $(date)"

    "${GERRIT}" --json mine |
        jq -c -r '
            .[] |
            .currentPatchSet.approvals as $flags |
            [$flags[] | select(.type == "CRVW")] as $crs |
            [$flags[] | select(.type == "COMR")] as $cqs |
            [$flags[] | select(.type == "VRIF")] as $vs |
            . += {
                cr: (
                    if (isempty($crs | .[])) then
                        "0"
                    else
                        ($crs | max_by(.value) | .value)
                    end
                ),
                cq: (
                    if (isempty($cqs | .[])) then
                        "0"
                    else
                        ($cqs | max_by(.value) | .value)
                    end
                ),
                v: (
                    if (isempty($vs | .[])) then
                        "0"
                    else
                        ($vs | min_by(.value) | .value) as $min |
                        ($vs | max_by(.value) | .value) as $max |
                        if ($min | tonumber) < 0 then
                            $min
                        else
                            $max
                        end
                    end
                )
            } |
            {
                cl_num: .number,
                last_updated: .lastUpdated,
                change_id: .id,
                subject: .subject,
                url: .url,
                branch: .branch,
                cr, cq, v,
            }' |
        while read -r record; do
            echo "Checking CL $(echo "${record}" | jq -r .cl_num)"
            check_no_update "${record}"
            check_cq_plus1 "${record}"
        done

    echo "End of ping Gerrit CLs at $(date)"
} >> "${LOG_FILE}" 2>&1
