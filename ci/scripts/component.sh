#!/bin/bash -eux

pushd dp-download-service
  make docker-test-component
popd
