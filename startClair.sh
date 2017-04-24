#!/usr/bin/env bash

set -e

mkdir -p $PWD/clair_config

docker rm -f postgres.test || true
docker rm -f clair.test || true

curl -sfL https://raw.githubusercontent.com/coreos/clair/master/config.example.yaml -o $PWD/clair_config/config.yaml
sed -i -s "s/host=localhost/host=postgres/g" $PWD/clair_config/config.yaml
docker run -d --name postgres.test -e POSTGRES_PASSWORD="" -p 5432:5432 postgres:9.6-alpine
sleep 5
docker run -it \
       --name clair.test \
       --link postgres.test:postgres \
       -p 6060-6061:6060-6061 \
       -v $PWD/clair_config:/config \
    quay.io/coreos/clair-git:latest -config=/config/config.yaml