#!/bin/bash -eux

cwd=$(pwd)

pushd $cwd/dp-download-service
  make test
popd
