#!/bin/bash
# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# Terminate if any command fails
set -e

unset topic
unset skip_original

while [[ $# -gt 0 ]]; do
  case $1 in
    -c|--continue)
      skip_original=1
      shift
      ;;
    -*)
      >&2 echo Usage: "$0 [-c --continue] <topic>"
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
  >&2 echo Usage: "$0 [-c --continue] <topic>"
  exit 1
fi

# config.py is expected to have a line that defines the rebase_target global variable
target=$(grep "^rebase_target =" config.py | sed 's/^[^"]*"\([^"]\+\).*/\1/')
if [[ ${target} = "" ]]; then
  >&2 echo No rebase target defined in config.py?
  exit 1
fi

if [[ ${skip_original} -ne 1 ]]; then
  # Upload the original pre-rebase patches
  rm -f "review/${topic}/"*.patch
  cp "review/${topic}/original/"*.patch "review/${topic}/"
  git add "review/${topic}/"*.patch
  git commit -m "DO-NOT-LAND: ${target}-rebase-${topic} topic review"
  repo upload --cbr --no-verify --wip .
fi

# Upload the rebased patches (+ fixups/reverts)
rm -f "review/${topic}/"*.patch
cp "review/${topic}/new/"*.patch "review/${topic}/"
git rm --cache "review/${topic}/*.patch"
git add "review/${topic}/"*.patch
git commit --amend --no-edit
repo upload --cbr --no-verify --wip .

# Clean up
rm -f "review/${topic}/"*.patch
