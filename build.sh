#!/usr/bin/env bash

rm -rf output/
mkdir -p output/

RUN_NAME="gofwd"
DATE=$(date +'%F %T %z')
GIT_SHA=$(git rev-parse --short HEAD || echo "GitNotFound")

#go build -ldflags "-s -w -extldflags '-static' -X main.magic=${GIT_SHA} -X 'main.date=${DATE}'" -o output/${RUN_NAME}
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w -extldflags '-static' -X main.magic=${GIT_SHA} -X 'main.date=${DATE}'" -o output/${RUN_NAME}