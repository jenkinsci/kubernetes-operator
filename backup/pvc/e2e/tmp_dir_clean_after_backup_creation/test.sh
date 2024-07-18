#!/bin/bash
set -eo pipefail

echo "Running tmp_dir_clean_after_backup_creation e2e test..."

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
mkdir -p ${BACKUP_DIR}

# Create an instance of the container under testing
cid="$(docker run -e BACKUP_CLEANUP_INTERVAL=1 -e JENKINS_HOME=${JENKINS_HOME} -v ${JENKINS_HOME}:${JENKINS_HOME}:ro -e BACKUP_DIR=${BACKUP_DIR} -v ${BACKUP_DIR}:${BACKUP_DIR}:rw -d ${docker_image})"
echo "Docker container ID '${cid}'"

# Remove test directory and container afterwards
trap "docker rm -vf $cid > /dev/null;rm -rf ${BACKUP_DIR}" EXIT

backup_number=1
docker exec ${cid} /home/user/bin/backup.sh ${backup_number}

[ "$(docker exec ${cid} ls /tmp | grep 'tmp')" ] && echo "tmp directory not empty" && exit 1;
# We should also check backup directory, since after #1000 we create temp directory at backup filesystem
[ "$(docker exec ${cid} ls ${BACKUP_DIR} | grep 'tmp')" ] && echo "backup dir consists temp directory" && exit 1;

backup_file="${BACKUP_DIR}/${backup_number}.tar.zstd"
[[ ! -f ${backup_file} ]] && echo "Backup file ${backup_file} not found" && exit 1;

echo "tmp directory empty, backup in backup directory present"
echo PASS
