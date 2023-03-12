#!/usr/bin/env bash

set -eo pipefail

[[ ! $# -eq 1 ]] && echo "Usage: $0 backup_number" && exit 1
[[ -z "${BACKUP_DIR}" ]] && echo "Required 'BACKUP_DIR' env not set" && exit 1;
[[ -z "${JENKINS_HOME}" ]] && echo "Required 'JENKINS_HOME' env not set" && exit 1;

backup_number=$1
echo "Running restore backup with backup number #${backup_number}"

tar -C ${JENKINS_HOME} -zxf "${BACKUP_DIR}/${backup_number}.tar.gz"

echo Done
exit 0
