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
# unit: day(s).
THRESHOLD=7
ROTATED_REVIEWERS="tzungbi@chromium.org"
GERRIT_INTERFACE="./env/bin/python gerrit_interface.py"

to_string() {
    local delta days hours

    delta="$1"

    days="$((delta / 86400))"
    hours="$(((delta - days * 86400) / 3600))"

    echo "${days} day(s) ${hours} hours"
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

check_no_update() {
    local record cl_num last_updated now threshold diff

    record="$1"
    cl_num="$(echo "${record}" | jq -r .cl_num)"
    last_updated="$(echo "${record}" | jq -r .last_updated)"

    now="$(date +%s)"
    threshold="$((86400 * THRESHOLD))"

    diff="$((now - last_updated))"
    if [[ ${diff} -le ${threshold} ]]; then
        echo "..Age: $(to_string "${diff}")"
    else
        echo "..Hasn't been changed for more than ${THRESHOLD} day(s)"
        "${GERRIT}" message "${cl_num}" \
            "The CL hasn't been changed for more than ${THRESHOLD} day(s)"

        if [[ -z "$(get_chromium_reviewers "${cl_num}")" ]]; then
            "${GERRIT}" reviewers "${cl_num}" "${ROTATED_REVIEWERS}"
        fi
    fi
}

check_cq_plus1() {
    local record cl_num change_id branch cq v retries retry_threshold message
    local now date mdate diff

    record="$1"
    cl_num="$(echo "${record}" | jq -r .cl_num)"
    change_id="$(echo "${record}" | jq -r .change_id)"
    branch="$(echo "${record}" | jq -r .branch | cut -d- -f2)"
    cq="$(echo "${record}" | jq -r .cq)"
    v="$(echo "${record}" | jq -r .v)"

    retries=2
    retry_threshold=86400
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
            has_failed=1

            retries="$((retries - 1))"
            if [[ ${retries} -gt 0 ]]; then
                continue
            fi

            now="$(date +%s)"
            date="$(echo "${msg}" | jq -r .date)"
            mdate="$(date --date="${date}" +%s)"

            diff="$((now - mdate))"
            if [[ ${diff} -gt ${retry_threshold} ]]; then
                echo "..Mark CQ+1 because of ${retry_threshold}s has elapsed"
                "${GERRIT}" label-cq "${cl_num}" 1
                return
            else
                echo "..Do nothing because there is no more retries quota"
                return
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
                cr: ($crs | max_by(.value) | .value),
                cq: ($cqs | max_by(.value) | .value),
                v: (
                    ($vs | min_by(.value) | .value) as $min |
                    ($vs | max_by(.value) | .value) as $max |
                    if ($min | tonumber) < 0 then
                        $min
                    else
                        $max
                    end
                )
            } |
            {
                cl_num: .number,
                last_updated: .lastUpdated,
                change_id: .id,
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
