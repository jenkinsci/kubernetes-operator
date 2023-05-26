#!/bin/bash
set -eo pipefail

[[ "${DEBUG}" ]] && set -x

# set current working directory to the directory of the script
cd "$(dirname "$0")"

docker_image=$1

if ! docker inspect ${docker_image} &> /dev/null; then
    echo "Image '${docker_image}' does not exists"
    false
fi

JENKINS_HOME="$(pwd)/jenkins_home"
BACKUP_DIR="$(pwd)/backup"
RESTORE_FOLDER="$(pwd)/restore"
JENKINS_HOME_AFTER_RESTORE="$(pwd)/jenkins_home_after_restore"
mkdir -p ${BACKUP_DIR}
mkdir -p ${RESTORE_FOLDER}

# Create an instance of the container under testing
cid="$(docker run -e JENKINS_HOME=${JENKINS_HOME} -v ${JENKINS_HOME}:${JENKINS_HOME}:ro -e BACKUP_DIR=${BACKUP_DIR} -v ${BACKUP_DIR}:${BACKUP_DIR}:rw -e RESTORE_FOLDER=${RESTORE_FOLDER} -v ${RESTORE_FOLDER}:${RESTORE_FOLDER}:rw -d ${docker_image})"
echo "Docker container ID '${cid}'"

# Remove test directory and container afterwards
trap "docker rm -vf $cid > /dev/null;rm -rf ${BACKUP_DIR};rm -rf ${RESTORE_FOLDER}" EXIT

backup_number=1
docker exec ${cid} /home/user/bin/backup.sh ${backup_number}

backup_file="${BACKUP_DIR}/${backup_number}.tar.zstd"
[[ ! -f ${backup_file} ]] && echo "Backup file ${backup_file} not found" && exit 1;

docker exec ${cid} /bin/bash -c "JENKINS_HOME=${RESTORE_FOLDER};/home/user/bin/restore.sh ${backup_number}"

echo "Compare directories"
diff --brief --recursive "${RESTORE_FOLDER}" "${JENKINS_HOME_AFTER_RESTORE}"
echo "Directories are the same"
echo PASS
