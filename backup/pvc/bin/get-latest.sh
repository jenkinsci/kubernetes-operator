#!/usr/bin/env bash

set -eo pipefail
source "$(dirname "$0")/utils.sh"

is_backup_not_exist() {
    local backup_dir="$1"
    # Save the current value of 'set -e'
    local previous_e
    previous_e=$(set +e; :; echo $?)
    # Temporarily turn off 'set -e'
    set +e
    # Run ls command to check if any files matching the pattern exist
    ls "${backup_dir}"/*.tar.* 1> /dev/null 2>&1
    # Store the exit status of the ls command
    local ls_exit_status=$?
    # Restore the previous value of 'set -e'
    [ "$previous_e" = "0" ] && set -e
    # Return true if ls command succeeded (no files found), otherwise return false
    [ $ls_exit_status -ne 0 ]
}

[[ -z "${BACKUP_DIR}" ]] && { _log "ERROR" "Required 'BACKUP_DIR' env not set"; exit 1; }

# Check if we have any backup
if is_backup_not_exist "${BACKUP_DIR}"; then
  echo "-1"
  exit 0
fi

# Search for all the tar.* inside the backup dir to support the migration between gzip vs zstd
latest=$(find "${BACKUP_DIR}"/*.tar.* -maxdepth 0 -exec basename {} \; | sort -g | tail -n 1)

if [[ "${latest}" == "" ]]; then
  echo "-1"
else
  echo "${latest%%.*}"
fi
