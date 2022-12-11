#!/bin/bash

set -x
set -e
set -u

cat ./config/singlearch_imagelist.txt | while read line
do
    raw_amd64="" 
    raw_arm64=""
    eval $(echo $line | awk '{ printf("raw_amd64=%s;raw_arm64=%s",$1,$2) }')

    repo=""
    amd64=""
    arm64=""

    space_amd64=$(echo $raw_amd64 | tr '/' ' ')
    eval $(echo $space_amd64 | awk '{ printf("repo=%s;amd64=%s",$1,$2) }')
    if [[ $(echo $amd64) == "" ]] ; then
        amd64="${repo}"
    fi 
    if [[ $(echo $amd64 | grep ":") == "" ]] ; then
        amd64="${amd64}:latest"
    fi 

    space_arm64=$(echo $raw_arm64 | tr '/' ' ')
    eval $(echo $space_arm64 | awk '{ printf("repo=%s;arm64=%s",$1,$2) }')
    if [[ $(echo $arm64) == "" ]] ; then
        arm64="${repo}"
    fi 
    if [[ $(echo $arm64 | grep ":") == "" ]] ; then
        arm64="${arm64}:latest"
    fi 

    docker pull ${raw_amd64}
    docker tag  ${raw_amd64} $1/${amd64}-amd64
    docker push $1/${amd64}-amd64
    docker image rm $1/${amd64}-amd64
    docker image rm ${raw_amd64}

    docker pull ${raw_arm64}
    docker tag  ${raw_arm64} $1/${amd64}-arm64
    docker push $1/${amd64}-arm64
    docker image rm $1/${amd64}-arm64
    docker image rm ${raw_arm64}

    docker manifest create $1/${amd64} \
    --amend $1/${amd64}-amd64 \
    --amend $1/${amd64}-arm64
    docker manifest push $1/${amd64}
    docker manifest rm $1/${amd64}
done

cat ./config/multiarch_imagelist.txt | while read raw_image
do
    repo=""
    image=""
    space_image=$(echo $raw_image | tr '/' ' ')

    eval $(echo $space_image | awk '{ printf("repo=%s;image=%s",$1,$2) }')
    if [[ $(echo $image) == "" ]] ; then
        image="${repo}"
    fi 

    if [[ $(echo $image | grep ":") == "" ]] ; then
        image="${image}:latest"
    fi 

    echo $image

    docker pull ${raw_image} --platform amd64
    docker tag  ${raw_image} $1/${image}-amd64
    docker push $1/${image}-amd64
    docker image rm $1/${image}-amd64
    docker image rm ${raw_image}

    docker pull ${raw_image} --platform arm64
    docker tag  ${raw_image} $1/${image}-arm64
    docker push $1/${image}-arm64
    docker image rm $1/${image}-arm64
    docker image rm ${raw_image}

    docker manifest create $1/${image} \
    --amend $1/${image}-amd64 \
    --amend $1/${image}-arm64
    docker manifest push $1/${image}
    docker manifest rm $1/${image}
done