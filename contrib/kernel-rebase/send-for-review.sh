#!/bin/bash
# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# Terminate if any command fails
set -e

print_usage () {
  >&2 echo Usage: "$0 [-c|--continue [-o|--original]] <topic>"
  >&2 echo "  -c|--continue : Amend the existing HEAD commit with updated new patches"
  >&2 echo "  -o|--original : Use in conjunction with -c to also re-upload original patches"
}

create_commit() {
  rm -f "review/${topic}/"*.patch
  cp "review/${topic}/$1/"*.patch "review/${topic}/"
  git rm --ignore-unmatch --cache "review/${topic}/*.patch"
  git add "review/${topic}/"*.patch
  git commit "${git_commit_args[@]}"
  repo upload --cbr --no-verify --wip .
}

unset topic
send_original=0
is_continue=0
git_commit_args=""

while [[ $# -gt 0 ]]; do
  case $1 in
    -c|--continue)
      is_continue=1
      shift
      ;;
    -o|--original)
      send_original=1
      shift
      ;;
    -*)
      print_usage
      exit 1
      ;;
    *)
      # Topic is the only supported positional argument
      if [[ ${topic} != "" ]]; then
        >&2 echo "Only one topic can be specified"
        exit 1
      fi
      topic="$1"
      shift
      ;;
  esac
done

# Check if a topic was given
if [[ ${topic} = "" ]]; then
  print_usage
  exit 1
fi

# config.py is expected to have a line that defines the rebase_target global variable
target=$(grep "^rebase_target =" config.py | sed 's/^[^"]*"\([^"]\+\).*/\1/')
if [[ ${target} = "" ]]; then
  >&2 echo No rebase target defined in config.py?
  exit 1
fi

if [[ ${send_original} -ne 0 && ${is_continue} -eq 0 ]]; then
  >&2 echo --original only allowed with --continue
  print_usage
  exit 1
fi

git_commit_args=(-m "DO-NOT-LAND: ${target}-rebase-${topic} topic review")
if [[ ${is_continue} -eq 1 ]]; then
  git_commit_args=(--amend --no-edit)
else
  send_original=1
fi

if [[ ${send_original} -eq 1 ]]; then
  # Upload the original pre-rebase patches
  create_commit "original"
  git_commit_args=(--amend --no-edit)
fi

# Upload the rebased patches (+ fixups/reverts)
create_commit "new"

# Clean up
rm -f "review/${topic}/"*.patch
