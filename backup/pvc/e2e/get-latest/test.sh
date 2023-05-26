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
mkdir -p ${BACKUP_DIR}
mkdir -p ${JENKINS_HOME}

mkdir -p ${BACKUP_DIR}/lost+found
touch ${BACKUP_DIR}/1.tar.zstd
touch ${BACKUP_DIR}/2.tar.zstd
touch ${BACKUP_DIR}/3.tar.zstd
touch ${BACKUP_DIR}/4.tar.zstd
touch ${BACKUP_DIR}/5.tar.zstd
touch ${BACKUP_DIR}/6.tar.zstd
touch ${BACKUP_DIR}/7.tar.zstd
touch ${BACKUP_DIR}/8.tar.zstd
touch ${BACKUP_DIR}/9.tar.zstd
touch ${BACKUP_DIR}/10.tar.zstd
touch ${BACKUP_DIR}/11.tar.zstd

# Create an instance of the container under testing
cid="$(docker run -e JENKINS_HOME=${JENKINS_HOME} -v ${JENKINS_HOME}:${JENKINS_HOME}:ro -e BACKUP_DIR=${BACKUP_DIR} -v ${BACKUP_DIR}:${BACKUP_DIR}:rw -d ${docker_image})"
echo "Docker container ID '${cid}'"

# Remove test directory and container afterwards
trap "docker rm -vf $cid > /dev/null;rm -rf ${BACKUP_DIR};rm -rf ${JENKINS_HOME}" EXIT

latest=$(docker exec ${cid} /bin/bash -c "JENKINS_HOME=${RESTORE_FOLDER};/home/user/bin/get-latest.sh")
rm ${BACKUP_DIR}/*.tar.zstd
empty_latest=$(docker exec ${cid} /bin/bash -c "JENKINS_HOME=${RESTORE_FOLDER};/home/user/bin/get-latest.sh")

if [[ "${DEBUG}" ]]; then
    docker logs ${cid}
    ls -la ${BACKUP_DIR}
fi

if [[ ! "${latest}" == "11" ]]; then
    echo "Latest backup number should be '11' but is '${latest}'"
    exit 1
fi
if [[ ! "${empty_latest}" == "-1" ]]; then
    echo "Latest backup number should be '-1' but is '${empty_latest}'"
    exit 1
fi

echo PASS
