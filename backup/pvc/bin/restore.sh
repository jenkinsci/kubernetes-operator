#!/usr/bin/env bash

set -eo pipefail

[[ ! $# -eq 1 ]] && echo "Usage: $0 backup_number" && exit 1
[[ -z "${BACKUP_DIR}" ]] && echo "Required 'BACKUP_DIR' env not set" && exit 1;
[[ -z "${JENKINS_HOME}" ]] && echo "Required 'JENKINS_HOME' env not set" && exit 1;

backup_number=$1
backup_file="${BACKUP_DIR}/${backup_number}"
echo "Running restore backup with backup number #${backup_number}"

if [[ -f "$backup_file.tar.gz" ]]; then
    echo "Old format tar.gz found, restoring it"
    OPTS=""
elif [[ -f "$backup_file.tar.zstd" ]]; then
    echo "Backup file found, proceeding"
    OPTS="--zstd"
else
  echo "ERR: Backup file not found: $backup_file"
  exit 1
fi

tar $OPTS -C ${JENKINS_HOME} -xf "${BACKUP_DIR}/${backup_number}.tar.zstd"

echo Done
exit 0
