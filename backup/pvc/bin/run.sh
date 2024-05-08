#!/usr/bin/env bash

set -eo pipefail

INTERVAL=60

# Ensure required environment variables are set
check_env_var() {
    if [[ -z "${!1}" ]]; then
        echo "Required '$1' environment variable is not set"
        exit 1
    fi
}

# Function to find exceeding backups
find_exceeding_backups() {
    local backup_dir="$1"
    local backup_count="$2"
    find "${backup_dir}"/*.tar.zstd -maxdepth 0 -exec basename {} \; | sort -gr | tail -n +$((backup_count +1))
}

check_env_var "BACKUP_DIR"
check_env_var "JENKINS_HOME"

if [[ -z "${BACKUP_COUNT}" ]]; then
    echo "ATTENTION! No BACKUP_COUNT set, it means you MUST delete old backups manually or by custom script"
else
    echo "Retaining only the ${BACKUP_COUNT} most recent backups, cleanup occurs every ${INTERVAL} seconds"
fi

while true;
do
    sleep $INTERVAL
    if [[ -n "${BACKUP_COUNT}" ]]; then
        exceeding_backups=$(find_exceeding_backups "${BACKUP_DIR}" "${BACKUP_COUNT}")
        if [[ -n "$exceeding_backups" ]]; then
            echo "Removing backups: $(echo "$exceeding_backups" | tr '\n' ', ' | sed 's/,$//')"
            echo "$exceeding_backups" | while read -r file; do
                rm "${BACKUP_DIR}/${file}"
            done
        fi
    fi
done
