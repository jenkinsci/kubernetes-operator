#!/usr/bin/env bash
# Common utils

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
