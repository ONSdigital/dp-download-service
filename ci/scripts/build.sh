#!/bin/bash -eux

cwd=$(pwd)

export GOPATH=$cwd/go

pushd $GOPATH/src/github.com/ONSdigital/dp-download-service
  make build && cp build/dp-download-service $cwd/build
popd
