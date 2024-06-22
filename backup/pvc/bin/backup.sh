#!/usr/bin/env bash

set -eo pipefail
source "$(dirname "$0")/utils.sh"
set -x

[[ ! $# -eq 1 ]] && _log "ERROR" "Usage: $0 BACKUP_NUMBER" && exit 1
[[ -z "${BACKUP_DIR}" ]] && _log "ERROR" "Required 'BACKUP_DIR' env not set" && exit 1
[[ -z "${JENKINS_HOME}" ]] && _log "ERROR" "Required 'JENKINS_HOME' env not set" && exit 1
RETRY_COUNT=${RETRY_COUNT:-3}
RETRY_INTERVAL=${RETRY_INTERVAL:-60}
BACKUP_NUMBER=$1
TRAP_FILE="${BACKUP_DIR}/_backup_${BACKUP_NUMBER}_is_running"

# --> Check if another backup process is running (operator restart/crash)
for ((i=0; i<RETRY_COUNT; i++)); do
    [[ ! -f "${TRAP_FILE}" ]] && _log "INFO" "Backup: no other backup process are running" && break
    _log "INFO" "Backup is already running. Waiting for ${RETRY_INTERVAL} seconds..."
    sleep "${RETRY_INTERVAL}"
done
[[ -f "${TRAP_FILE}" ]] && { _log "ERROR" "Backup is stil running after waiting ${RETRY_COUNT} time ${RETRY_INTERVAL}s. Exiting."; exit 1; }
# --< Done

_log "INFO" "Running backup ${BACKUP_NUMBER}"
touch "${TRAP_FILE}"
# create temp dir on the same filesystem with a BACKUP_DIR to be able use atomic mv enstead of copy
BACKUP_TMP_DIR=$(mktemp -d --tmpdir="${BACKUP_DIR}")

_clean(){
    test -d "${BACKUP_TMP_DIR}" && rm -fr "${BACKUP_TMP_DIR}"
    test -f "${TRAP_FILE}" && rm -f "${TRAP_FILE}"
}

_trap(){
    _clean
    _log "ERROR" "Backup: something wrong happened, check the logs"
}

trap '_trap' SIGQUIT SIGINT SIGTERM

# config.xml in a job directory is a config file that shouldn't be backed up
# config.xml in child directories is state that should. For example-
# branches/myorg/branches/myrepo/branches/master/config.xml should be retained while
# branches/myorg/config.xml should not
tar --zstd -C "${JENKINS_HOME}" -cf "${BACKUP_TMP_DIR}/${BACKUP_NUMBER}.tar.zstd" \
    --exclude jobs/*/workspace* \
    --no-wildcards-match-slash --anchored \
    --ignore-failed-read \
    --exclude jobs/*/config.xml -c jobs || ret=$?

if [[ "$ret" -eq 0 ]]; then
  _log "INFO" "Backup ${BACKUP_NUMBER} was completed without warnings"
elif [[ "$ret" -eq 1 ]]; then
  _log "INFO" "Backup ${BACKUP_NUMBER} was completed with some warnings"
fi

mv "${BACKUP_TMP_DIR}/${BACKUP_NUMBER}.tar.zstd" "${BACKUP_DIR}/${BACKUP_NUMBER}.tar.zstd"

_log "INFO" "Cleaning ${BACKUP_TMP_DIR} and trap file ${TRAP_FILE}"
_clean
[[ ! -s ${BACKUP_DIR}/${BACKUP_NUMBER}.tar.zstd ]] && _log "ERROR" "Backup file '${BACKUP_DIR}/${BACKUP_NUMBER}.tar.zstd' is empty" && exit 1

_log "INFO" "Backup ${BACKUP_NUMBER} done"
exit 0
