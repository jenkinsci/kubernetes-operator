#!/usr/bin/env bash

set -eo pipefail
source "$(dirname "$0")/utils.sh"

[[ ! $# -eq 1 ]] && _log "ERROR" "Usage: $0 <backup number>" && exit 1
[[ -z "${BACKUP_DIR}" ]] && _log "ERROR" "Required 'BACKUP_DIR' env not set" && exit 1
[[ -z "${JENKINS_HOME}" ]] && _log "ERROR" "Required 'JENKINS_HOME' env not set" && exit 1
BACKUP_NUMBER=$1
RETRY_COUNT=${RETRY_COUNT:-3}
RETRY_INTERVAL=${RETRY_INTERVAL:-60}

# --> Check if another restore process is running (operator restart/crash)
TRAP_FILE="${BACKUP_DIR}/_restore_${BACKUP_NUMBER}_is_running"
trap "rm -f ${TRAP_FILE}" SIGINT SIGTERM

for ((i=0; i<RETRY_COUNT; i++)); do
    [[ ! -f "${TRAP_FILE}" ]] && _log "INFO" "Restore: No other process are running, restoring" && break
    _log "INFO" "Restore is already running. Waiting for ${RETRY_INTERVAL} seconds..."
    sleep "${RETRY_INTERVAL}"
done
[[ -f "${TRAP_FILE}" ]] && { _log "ERROR" "Restore is stil running after waiting ${RETRY_COUNT} time ${RETRY_INTERVAL}s. Exiting."; exit 1; }
# --< Done

_log "INFO" "Running restore backup with backup number #${BACKUP_NUMBER}"
touch "${TRAP_FILE}"
BACKUP_FILE="${BACKUP_DIR}/${BACKUP_NUMBER}"

if [[ -f "$BACKUP_FILE.tar.gz" ]]; then
    _log "INFO" "Restore: ld format tar.gz found, restoring it"
    OPTS=""
    EXT="tar.gz"
elif [[ -f "$BACKUP_FILE.tar.zstd" ]]; then
    _log "INFO" "Restore: Backup file found, proceeding"
    OPTS="--zstd"
    EXT="tar.zstd"
else
  _log "ERROR" "Restore: Backup file not found: $BACKUP_FILE"
  exit 1
fi

tar $OPTS -C "${JENKINS_HOME}" -xf "${BACKUP_DIR}/${BACKUP_NUMBER}.${EXT}"

_log "INFO" "Restore: ${BACKUP_NUMBER} Done"
exit 0
