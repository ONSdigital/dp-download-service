#!/bin/bash -eux

export cwd=$(pwd)

pushd $cwd/dp-download-service
  make audit
popd