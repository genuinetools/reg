#!/usr/bin/env bash

set -e

docker rm -f registry.test || true
docker run -it -d --name registry.test -p 5000:5000 registry:2.6.0

docker pull alpine:3.5

for repo in `seq 1 20`;
do
    for tag in `seq 1 10`;
    do
        docker tag alpine:3.5 127.0.0.1:5000/company/alpine-${repo}:${tag}
        docker push 127.0.0.1:5000/company/alpine-${repo}:${tag}
    done
done

docker logs -f registry.test
