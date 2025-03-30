#!/usr/bin/env bash

set -eo pipefail
source "$(dirname "$0")/utils.sh"

# Use 60 as default in case BACKUP_CLEANUP_INTERVAL did not set
BACKUP_CLEANUP_INTERVAL=${BACKUP_CLEANUP_INTERVAL:=60}

# Ensure required environment variables are set
check_env_var() {
    if [[ -z "${!1}" ]]; then
        _log "ERROR" "Required '$1' environment variable is not set"
        exit 1
    fi
}

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

# Function to find exceeding backups
find_exceeding_backups() {
    local backup_dir="$1"
    local backup_count="$2"
    # Check if we have any backup
    if is_backup_not_exist "${backup_dir}"; then
        _log "ERROR" "[run] backups not found in ${backup_dir}"
        return
    fi
    find "${backup_dir}"/*.tar.zstd -maxdepth 0 -exec basename {} \; | sort -gr | tail -n +$((backup_count +1))
}

check_env_var "BACKUP_DIR"
check_env_var "JENKINS_HOME"

if [[ -z "${BACKUP_COUNT}" ]]; then
    _log "WARNING" "[run] no BACKUP_COUNT set, it means you MUST delete old backups manually or by custom script"
else
    _log "INFO" "[run] retaining only the ${BACKUP_COUNT} most recent backups, cleanup occurs every ${BACKUP_CLEANUP_INTERVAL} seconds"
fi

while true;
do
    sleep "$BACKUP_CLEANUP_INTERVAL"
    if [[ -n "${BACKUP_COUNT}" ]]; then
        exceeding_backups=$(find_exceeding_backups "${BACKUP_DIR}" "${BACKUP_COUNT}")
        if [[ -n "$exceeding_backups" ]]; then
            _log "INFO" "[run] removing backups: $(echo "$exceeding_backups" | tr '\n' ', ' | sed 's/,$//')"
            echo "$exceeding_backups" | while read -r file; do
                rm "${BACKUP_DIR}/${file}"
            done
        fi
    fi
done
