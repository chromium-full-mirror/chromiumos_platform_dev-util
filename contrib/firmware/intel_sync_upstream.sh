#!/bin/bash
# Copyright 2020 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

SOC_LIST=(tgl jsl adl adln mtl rpl twl ptl wcl nvl )
declare -A SOC_EDK_LOCAL_DIR_MAP=( ["tgl"]="branch2-private" ["jsl"]="branch1-private" ["adl"]="branch1-private" ["adln"]="branch1-private" ["mtl"]="branch1-private" ["rpl"]="branch2-private" ["twl"]="branch1-private" ["ptl"]="branch1-private" ["wcl"]="branch1-private" ["nvl"]="branch1-private" )

# If FSP is using a staging repo that does not follow the format ${SOC}-staging,
# then add the mapping here.
declare -A SOC_FSP_STAGING_REPO_MAP=( ["adl"]="ccg-adl-generic-full" ["adln"]="adl-n-staging" ["twl"]="adl-n-staging" )

# If edk2/edk2-platforms are using a branch prefix that does not follow the format chromeos-${SOC},
# then add the mapping here.
declare -A SOC_EDK_BRANCH_PREFIX_MAP=( ["adln"]="chromeos-adl-n" )

# If edk2/edk2-platforms are using a repo name with a suffix (e.g. are not edk-staging or edk-platforms-staging)
declare -A SOC_EDK_REPO_SUFFIX_MAP=( ["mtl"]="intelcollab" ["rpl"]="intelcollab" ["twl"]="intelcollab" ["ptl"]="intelcollab" ["wcl"]="intelcollab" ["nvl"]="intelcollab" )
# Binary package repository mapping for FSP binary patch creation
declare -A SOC_BIN_PACKAGE_REPO_MAP=(
  ["ptl"]="https://github.com/otcshare/CCG-PTL-Generic-Binaries"
  ["wcl"]="https://github.com/otcshare/CCG-WCL-Generic-Binaries"
  ["nvl"]="https://github.com/otcshare/CCG-NVL-Mobile-Generic-Binaries"
)

# Binary file paths mapping for each SOC - configurable for future changes
declare -A SOC_BIN_FILE_PATHS=(
  ["ptl"]="Binaries/PantherLakeBinPkg/MemoryInit/GreenMrc.bin Binaries/PantherLakeBinPkg/MemoryInit/ReleaseGreenMrc.bin"
  ["wcl"]="Binaries/PantherLakeBinPkg/MemoryInit/GreenMrc.bin Binaries/PantherLakeBinPkg/MemoryInit/ReleaseGreenMrc.bin"
  ["nvl"]="Binaries/NovaLakeBinPkg/MemoryInit/GreenMrc.bin Binaries/NovaLakeBinPkg/MemoryInit/ReleaseGreenMrc.bin"
)

function die()
{
  if [ "$1" -ne 0 ]; then
    echo "Error: $1"
    echo "$2"
    exit 1
  fi;
}

function fix_permissions()
{
  # edk2/edk2-platform - when submodules are stored as regular files
  # we see files with 000 permissions
  local dir_type="$1"
  local staging_name="$2"
  local version="$3"

  # Only apply permission fixes for non-FSP directories
  if [[ "${dir_type}" != "fsp" ]]; then
    echo "Checking for files with 000 permissions in ${dir_type}..."

    # Find files with 000 permissions directly
    local files_with_000_perms
    files_with_000_perms=$(git diff --name-only \
      chrome-internal-tot.."${staging_name}"-"${version}" | \
      xargs -I {} find {} -perm 000 2>/dev/null)

    if [[ -n "${files_with_000_perms}" ]]; then
      echo "Found files with 000 permissions that will be fixed:"
      echo "${files_with_000_perms}"
      echo "Fixing permissions for ${dir_type}..."
      echo "${files_with_000_perms}" | xargs chmod 664
      echo "Permissions fixed."
    else
      echo "No files with 000 permissions found for ${dir_type}."
    fi
  fi
}

function prompt_and_push()
{
  local branch="$1"
  local description="$2"

  while true
  do
    read -r -p "Do you want to push your changes to cros-internal? [y/n] >" input

    case ${input} in
      [yY][eE][sS]|[yY])
        git push cros-internal HEAD:refs/for/"${branch}"
        echo "Pushed ${description}, ready for review."
        break
        ;;

      [nN][oO]|[nN])
        echo "Ready to push ${description}."
        echo "Execute 'git push cros-internal HEAD:refs/for/${branch}' to push change."
        break
        ;;

      *)
        echo "Invalid input..."
        ;;
    esac
  done
}

function update_binaries()
{
  local soc="$1"
  local version="$2"
  local repo_path="$3"
  local src_dir="${CHROMIUM_TOT_ROOT}/src/third_party/fsp/${soc}/fsp/"

  echo "Creating binary patch for SOC: ${soc}, Version: ${version}"

  # Check if destination directory exists
  [[ -d "${src_dir}" ]] || { echo "Error: Destination directory does not exist: ${src_dir}"; exit 1; }

  # Change to destination directory for git operations
  pushd "${src_dir}" > /dev/null || { echo "Cannot navigate to ${src_dir}"; exit 1; }

  # Reset repository to clean state from remote origin
  echo "Resetting repository to cros-internal/chromeos..."
  git fetch cros-internal && git reset --hard cros-internal/chromeos && git clean -fd

  # Get binary files and extract unique folders in one pass
  local bin_files=(${SOC_BIN_FILE_PATHS[${soc}]})
  local restored_folders=""

  # Process each file: restore folder (if needed) and copy file
  for file_path in "${bin_files[@]}"; do
    local rel_path="${file_path#Binaries/}"
    local folder_name="${rel_path%%/*}"

    # Restore folder if not already done
    if [[ ! " ${restored_folders} " =~ " ${folder_name} " ]]; then
      echo "Restoring previous version of ${folder_name}..."
      git checkout HEAD~1 -- "${folder_name}" 2>/dev/null && \
        echo "Successfully restored ${folder_name}" || \
        echo "No previous ${folder_name} found (first time addition)"
      restored_folders+=" ${folder_name}"
    fi

    # Copy binary file
    local src_file="${repo_path}/${file_path}"
    local dest_file="${src_dir}${rel_path}"
    local dest_dir="${dest_file%/*}"

    [[ -d "${dest_dir}" ]] || { echo "Error: Destination directory does not exist: ${dest_dir}"; exit 1; }
    echo "Copying: ${rel_path}"
    cp "${src_file}" "${dest_file}" || { echo "Failed to copy ${rel_path}"; exit 1; }
  done

  popd > /dev/null
  echo "All binary files copied successfully!"
}

function clone_repo()
{
  local repo_url="$1"
  local local_path="$2"
  local version="$3"

  # Clean up any existing directory
  rm -rf "${local_path}"

  echo "Cloning repository..."
  git clone --depth 1 --branch "${version}" "${repo_url}" "${local_path}" || {
    echo "Tag '${version}' not found. Available tags:"
    git ls-remote --tags "${repo_url}" | tail -10
    die 1 "Tag '${version}' does not exist"
  }
}

function usage()
{
  echo "Error: missing parameter."
  echo "Usage: $0 SOC [fsp|edk2|edk2-platforms|coreboot|bin] version_string bug_id"
  echo "Example: $0 tgl fsp 'TGL.2527_17' 123456"
  echo "Example: $0 ptl bin 'PTL.123.45' 123456"
  exit 1
}

# Verify param count
if [ "$#" -lt "4" ]; then
  usage
fi;
SOC="$1"
DIR="$2"
VERSION="$3"
BUG_ID="$4"

for arg in "$@"; do
  if [[ "$arg" == "--external" ]]; then
    run_external=true
    break # Exit the loop once --external is found
  fi
done

if [[ ! "${SOC_LIST[*]}" =~ ${SOC} ]]; then
  die 1 "SoC is not supported"
fi

case ${DIR} in
  fsp)
    if [ -v SOC_FSP_STAGING_REPO_MAP["${SOC}"] ]; then
        STAGING_NAME="${SOC_FSP_STAGING_REPO_MAP[${SOC}]}"
    else
        STAGING_NAME="${SOC}-staging"
    fi
    CHROMEOS_BRANCH=chromeos
    SRC_DIR="${CHROMIUM_TOT_ROOT}/src/third_party/fsp/${SOC}/${DIR}/"
    STAGING_REPO="https://chrome-internal.googlesource.com/chromeos/third_party/intel-fsp/${STAGING_NAME}"
    ;;

  edk2 | edk2-platforms)
    if [ -v SOC_EDK_REPO_SUFFIX_MAP["${SOC}"] ]; then
      STAGING_NAME="${DIR}-staging-${SOC_EDK_REPO_SUFFIX_MAP[${SOC}]}"
    else
      STAGING_NAME="${DIR}-staging"
    fi
    LOCAL_DIR="${SOC_EDK_LOCAL_DIR_MAP[${SOC}]}"
    if [ -v SOC_EDK_BRANCH_PREFIX_MAP["${SOC}"] ]; then
      CHROMEOS_BRANCH="${SOC_EDK_BRANCH_PREFIX_MAP[${SOC}]}-${LOCAL_DIR}"
    else
      CHROMEOS_BRANCH="chromeos-${SOC}-${LOCAL_DIR}"
    fi
    SRC_DIR="${CHROMIUM_TOT_ROOT}/src/third_party/fsp/${SOC}/${DIR}/${LOCAL_DIR}"
    STAGING_REPO="https://chrome-internal.googlesource.com/chromeos/third_party/intel-fsp/${STAGING_NAME}"
    ;;

  coreboot)
    STAGING_NAME="${SOC}-staging"
    CHROMEOS_BRANCH=chromeos
    SRC_DIR="${CHROMIUM_TOT_ROOT}/src/third_party/coreboot-intel-private/${SOC}"
    STAGING_REPO="https://chrome-internal.googlesource.com/chromeos/third_party/coreboot-intel-private/${STAGING_NAME}"
    ;;

  bin)
    # Check if SOC is supported for binary packages
    [[ -v SOC_BIN_PACKAGE_REPO_MAP["${SOC}"] ]] || die 1 "SOC '${SOC}' is not supported for binary package repositories"

    bin_repo_url="${SOC_BIN_PACKAGE_REPO_MAP[${SOC}]}"

    # Debug output
    echo "Processing binary package repository for SOC: ${SOC}"
    echo "Repository URL: '${bin_repo_url}'"

    # Validate URL is not empty
    [[ -n "${bin_repo_url}" ]] || die 1 "Repository URL is empty for SOC: ${SOC}"

    repo_name="${bin_repo_url##*/}"    # Extract repo name (e.g., "CCG-WCL-Generic-Binaries.git")
    repo_name="${repo_name%.git}"      # Remove .git suffix (e.g., "CCG-WCL-Generic-Binaries")
    fsp_dir="${CHROMIUM_TOT_ROOT}/src/third_party/fsp/${SOC}/fsp/"
    local_clone_path="/tmp/${repo_name}"

    # Clone repository with specific tag
    clone_repo "${bin_repo_url}" "${local_clone_path}" "${VERSION}"

    # Copy binary files and clean up in one step
    update_binaries "${SOC}" "${VERSION}" "${local_clone_path}"
    rm -rf "${local_clone_path}"

    # Navigate to FSP directory and commit changes
    pushd "${fsp_dir}" > /dev/null || die 1 "Can't find ${fsp_dir}"

    git add . || die 1 "Failed to stage binary files"

    # Generate commit message with file list efficiently
    file_list=""
    bin_files=(${SOC_BIN_FILE_PATHS[${SOC}]})
    for file_path in "${bin_files[@]}"; do
      file_list+="- ${file_path#Binaries/}"$'\n'
    done

    # Commit changes or exit gracefully if no changes
    git commit -m "Update ${SOC} binaries to version ${VERSION}"$'\n\n'"Updated binaries:"$'\n'"${file_list}"$'\n'"BUG=b:${BUG_ID}" || {
      popd > /dev/null
      echo "No changes to commit - binary package processing complete"
      exit 0
    }

    echo "Binary package processing complete, ready for upload."
    prompt_and_push "chromeos" "binary update for ${SOC}-${VERSION}"

    popd > /dev/null
    exit 0
    ;;

  *)
    usage
    ;;
esac

# Assumption is that the staging repo mirrors tags and heads under upstream/
case ${VERSION} in
  master | EDK2_Trunk_Intel | main)
    UPREV_BRANCH=remotes/${STAGING_NAME}/upstream/${VERSION}
    ;;

  *)

    UPREV_BRANCH=upstream/${VERSION}
    ;;
esac

if [ -z "${CHROMIUM_TOT_ROOT}" ]; then
  die 1 "CHROMIUM_TOT_ROOT environment variable is not set"
fi

# Clone the repo where the staging changes need to be pushed
pushd "${SRC_DIR}" > /dev/null || die 1 "Can't find ${SRC_DIR}"

# Add staging repo as remote repo to my local repo
git remote add "${STAGING_NAME}" "${STAGING_REPO}"
err=$?

# If remote already exists, that's ok, but otherwise, exit on error
if [ "${err}" -ne 0 ] && [ "${err}" -ne 3 ] && [ "${err}" -ne 128 ]; then
  die "${err}" "Can't add remote ${STAGING_NAME}"
elif [ "${err}" -eq 0 ]; then
  echo "Created remote ${STAGING_NAME}"
else
  echo "Remote ${STAGING_NAME} already exists"
fi;

git fetch --tags --force "${STAGING_NAME}"
die $? "Can't fetch ${STAGING_NAME}"

# Detach from any branch before deleting
git checkout --detach

# Removing a stale branch
git branch -D "${STAGING_NAME}"-"${VERSION}"

# Set up remote branch
git checkout "${UPREV_BRANCH}" -b "${STAGING_NAME}"-"${VERSION}"
die $? "Can't checkout upstream/${VERSION}"
echo "Checked out upstream/${VERSION} to branch ${STAGING_NAME}-${VERSION}"

# Checkout a local branch from remotes/cros-internal/${CHROMEOS_BRANCH}
git branch -D chrome-internal-tot
git checkout -b chrome-internal-tot remotes/cros-internal/"${CHROMEOS_BRANCH}"
die $? "Error checking out remotes/cros-internal/${CHROMEOS_BRANCH}"
echo "Checked out remotes/cros-internal/${CHROMEOS_BRANCH} to branch chrome-internal-tot"

# Merge from staging branch
git merge "${STAGING_NAME}-${VERSION}" --strategy-option theirs --no-ff --log
if [ $? -ne 0 ]; then
  echo "Didn't merge cleanly to ${STAGING_NAME}-${VERSION}"

  while true
  do
    read -r -p "Do you want to force sync to the tag? [y/n] >" input

    case ${input} in
      [yY][eE][sS]|[yY])
        echo "Force syncing"
        # Remove all the changes that were not in the tag
        git diff --name-only --diff-filter=U | xargs git rm

        # Clean any temporary files
        git clean -fx

        # No signoff used in FSP repo
        git commit

        if [[ $? -eq 0 ]]; then
          COMMIT_ARGS="--amend"
        else
          COMMIT_ARGS="-s"
        fi

        git diff -a chrome-internal-tot.."${STAGING_NAME}"-"${VERSION}" > /tmp/merge-to-tag.patch
        patch -p1 < /tmp/merge-to-tag.patch
        rm /tmp/merge-to-tag.patch

        # edk2/edk2-platforms - fix  permission issues when submodules handled as regular files
        # git add fails without this change
        fix_permissions "${DIR}" "${STAGING_NAME}" "${VERSION}"
        git add .

        git commit ${COMMIT_ARGS} --message "Merge branch \`${VERSION}\` into chrome-internal-tot"$'\n\n'"BUG=b:${BUG_ID}" --edit
        break
        ;;

      [nN][oO]|[nN])
        echo "Launching mergetool to manually fix issues."
        git mergetool
        die $? "Error returned from mergetool"
        git commit
        break
        ;;
      *)
        echo "Invalid input..."
        ;;
    esac
  done
fi;

DIFF_WITH_UPSTREAM=$(git diff --stat=200 "${UPREV_BRANCH}" .)
if [ -n "${DIFF_WITH_UPSTREAM}" ]; then
  echo -e "\n\033[33mWARNING: The local and upstream branches have below differences.\033[0m"
  echo -n "${DIFF_WITH_UPSTREAM}" | head -n -1
  echo -e "\033[33mWARNING: Please review the above differences before pushing.\033[0m\n"
fi

echo "Merge of ${STAGING_NAME}-${VERSION} complete, ready for upload."

# Restore the required file if deleted
FILES_TO_RESTORE=("OWNERS" "DIR_METADATA")

restore_file() {
  local file_name="$1"
  local full_path="${SRC_DIR}/${file_name}"

  if [ ! -f "${full_path}" ]; then
    echo "File '${file_name}' is missing. Restoring..."
    if git -C "${SRC_DIR}" checkout remotes/cros-internal/"${CHROMEOS_BRANCH}" -- "${file_name}"; then
      git commit --amend --no-edit > /dev/null
      echo "File '${file_name}' restored successfully."
    else
      echo -e "\033[31mError: Failed to restore '${file_name}' using git.\033[0m"
    fi
  fi
}

# Restore each file in the list
for file in "${FILES_TO_RESTORE[@]}"; do
  restore_file "${file}"
done

# Use the reusable push function
prompt_and_push "${CHROMEOS_BRANCH}" "merge of ${STAGING_NAME}-${VERSION}"
