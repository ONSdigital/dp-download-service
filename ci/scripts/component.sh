#!/bin/bash -eux

pushd dp-download-service
  make test-component
popd
