#!/bin/bash

set -x

# Defines the list of inputs and the number of concurrent requests.
SINGLE_ARCH_LIST=./config/singlearch_imagelist.txt
MULTI_ARCH_LIST=./config/multiarch_imagelist.txt
REGISTRY=$1
REPO=$2
THREAD=$3

# Build named pipes for process management.
FIFO=/tmp/$$.fifo
mkfifo ${FIFO}
exec 6<>${FIFO}
rm -rf ${FIFO}

# Put a new line into the pipe to start the initial process.
for((i=1;i<=$THREAD;i++))
do
    echo >&6;
done

# Merge multiple single-schema images to manifest and push it to target repo.
cat ${SINGLE_ARCH_LIST} | while read line
do
    # The child process can not be started until '\n' are read from the pipe.
    read -u6
    {
        set +e

        repo="" amd64="" arm64="" 

        # Shard different schemas in each line.
        eval $(echo $line | awk '{ printf("raw_amd64=%s;raw_arm64=%s",$1,$2) }')

        # Separate the repo and image parts and complete the tag (arch amd).
        space_amd64=$(echo $raw_amd64 | tr '/' ' ')
        eval $(echo $space_amd64 | awk '{ printf("repo=%s;amd64=%s",$1,$2) }')
        if [[ $(echo $amd64) == "" ]] ; then
            amd64="${repo}"
        fi 
        if [[ $(echo $amd64 | grep ":") == "" ]] ; then
            amd64="${amd64}:latest"
        fi 

        # Separate the repo and image parts and complete the tag (arch arm).
        space_arm64=$(echo $raw_arm64 | tr '/' ' ')
        eval $(echo $space_arm64 | awk '{ printf("repo=%s;arm64=%s",$1,$2) }')
        if [[ $(echo $arm64) == "" ]] ; then
            arm64="${repo}"
        fi 
        if [[ $(echo $arm64 | grep ":") == "" ]] ; then
            arm64="${arm64}:latest"
        fi 
 
        # Cut out the tag for later inspection.
        http_code="" image_without_tag="" image_and_tag=$(echo $amd64 | tr ':' ' ')
        eval $(echo $image_and_tag | awk '{ printf("image_without_tag=%s;tag=%s",$1,$2) }')

        # Check whether the target repo already has this tag.
        eval $(curl -I -s -m 10 https://hub.docker.com/v2/repositories/${REPO}/$image_without_tag/tags/$tag | grep HTTP | awk '{ printf("http_code=%s;",$2) }')
        if [[ $(echo $http_code) != "200" ]] ; then
            # Call docker to pull the image and push the single schema version to the remote repository (arch amd).
            docker pull ${raw_amd64}
            docker tag  ${raw_amd64} ${REGISTRY}/${REPO}/${amd64}-amd64
            docker push ${REGISTRY}/${REPO}/${amd64}-amd64
            docker image rm ${REGISTRY}/${REPO}/${amd64}-amd64
            docker image rm ${raw_amd64}

            # Call docker to pull the image and push the single schema version to the remote repository (arch arm).
            docker pull ${raw_arm64}
            docker tag  ${raw_arm64} ${REGISTRY}/${REPO}/${amd64}-arm64
            docker push ${REGISTRY}/${REPO}/${amd64}-arm64
            docker image rm ${REGISTRY}/${REPO}/${amd64}-arm64
            docker image rm ${raw_arm64}

            # Build the manifest and push it to the target repo.
            docker manifest create ${REGISTRY}/${REPO}/${amd64} \
            --amend ${REGISTRY}/${REPO}/${amd64}-amd64 \
            --amend ${REGISTRY}/${REPO}/${amd64}-arm64
            docker manifest push ${REGISTRY}/${REPO}/${amd64}
            docker manifest rm ${REGISTRY}/${REPO}/${amd64}    
        fi

        set -e

        # Write a new line to the pipe to indicate the end of one process and start the next.
        # Whatever happens when the child exits, write the pipe!
        echo >&6
    } &
done

# For an image that is already multi-architecture, it is directly synchronized to the target repo.
cat ${MULTI_ARCH_LIST} | while read raw_image
do 
    # The child process can not be started until a line are read from the pipe.
    read -u6
    {
        set +e
        
        repo="" image="" space_image=$(echo $raw_image | tr '/' ' ')

        # Separate the repo and image parts and complete the tag.
        eval $(echo $space_image | awk '{ printf("repo=%s;image=%s",$1,$2) }')
        if [[ $(echo $image) == "" ]] ; then
            image="${repo}"
        fi 
        if [[ $(echo $image | grep ":") == "" ]] ; then
            image="${image}:latest"
        fi

        # Cut out the tag for later inspection.
        http_code="" image_without_tag="" image_and_tag=$(echo $image | tr ':' ' ')
        eval $(echo $image_and_tag | awk '{ printf("image_without_tag=%s;tag=%s",$1,$2) }')

        # Check whether the target repo already has this tag.
        eval $(curl -I -s -m 10 https://hub.docker.com/v2/repositories/$1/$image_without_tag/tags/$tag | grep HTTP | awk '{ printf("http_code=%s;",$2) }')
        if [[ $(echo $http_code) != "200" ]] ; then
            docker pull ${raw_image} --platform amd64
            docker tag  ${raw_image} ${REGISTRY}/${REPO}/${image}-amd64
            docker push ${REGISTRY}/${REPO}/${image}-amd64
            docker image rm ${REGISTRY}/${REPO}/${image}-amd64
            docker image rm ${raw_image}

            docker pull ${raw_image} --platform arm64
            docker tag  ${raw_image} ${REGISTRY}/${REPO}/${image}-arm64
            docker push ${REGISTRY}/${REPO}/${image}-arm64
            docker image rm ${REGISTRY}/${REPO}/${image}-arm64
            docker image rm ${raw_image}

            docker manifest create ${REGISTRY}/${REPO}/${image} \
            --amend ${REGISTRY}/${REPO}/${image}-amd64 \
            --amend ${REGISTRY}/${REPO}/${image}-arm64
            docker manifest push ${REGISTRY}/${REPO}/${image}
            docker manifest rm ${REGISTRY}/${REPO}/${image}
        fi

        set -e

        # Write a new line to the pipe to indicate the end of one process and start the next.
        # Whatever happens when the child exits, write the pipe!
        echo >&6
    } &
done

# The parent process needs to wait for the child process to finish executing.
wait
# Close the file descriptor
exec 6>&-
exec 6<&-

echo "sync images success"
exit 0
