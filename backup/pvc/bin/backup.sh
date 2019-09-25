#!/usr/bin/env bash

set -eo pipefail

[[ ! $# -eq 1 ]] && echo "Usage: $0 backup_number" && exit 1;
[[ -z "${BACKUP_DIR}" ]] && echo "Required 'BACKUP_DIR' env not set" && exit 1;
[[ -z "${JENKINS_HOME}" ]] && echo "Required 'JENKINS_HOME' env not set" && exit 1;
[[ -z "${EXCLUDE_CONFIG_XML}" ]] && EXCLUDE_CONFIG_XML=true;

backup_number=$1
echo "Running backup"

CONFIG_ARG=""

if [[ ${EXCLUDE_CONFIG_XML} == true ]]; then
  CONFIG_ARG="--exclude jobs/*/config.xml"
fi

tar -C ${JENKINS_HOME} -czf "${BACKUP_DIR}/${backup_number}.tar.gz" ${CONFIG_ARG} --exclude jobs/*/workspace* -c jobs

[[ ! -s ${BACKUP_DIR}/${backup_number}.tar.gz ]] && echo "backup file '${BACKUP_DIR}/${backup_number}.tar.gz' is empty" && exit 1;

echo Done
exit 0
