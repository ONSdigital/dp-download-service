#!/bin/bash -eux

cwd=$(pwd)

pushd $cwd/dp-download-service
  go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.52.2
  make lint
popd
