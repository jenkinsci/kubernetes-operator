#!/usr/bin/env bash
# Common utils

# Explicitly specify default tmp prefix used by mktemp for compatibility
# see https://www.gnu.org/software/autogen/mktemp.html
BACKUP_TMP_PREFIX="tmp"
BACKUP_TMP_PATTERN="$BACKUP_TMP_PREFIX.XXXXXXXXXX"

_log() {
    local level="$1"
    local message="$2"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    if [[ "$level" =~ ^(ERROR|ERR|error|err)$ ]]; then
        echo "${timestamp} - ${level} - ${message}" > /proc/1/fd/2
    else
        echo "${timestamp} - ${level} - ${message}" > /proc/1/fd/1
        echo "${timestamp} - ${level} - ${message}" >&2
    fi
}
