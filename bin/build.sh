#!/bin/bash -eu

source /build-common.sh

BINARY_NAME="bup"
COMPILE_IN_DIRECTORY="cmd/bup"
# BINTRAY_PROJECT="function61/bup"

# vendor dir contains non-gofmt code..
GOFMT_TARGETS="cmd/ pkg/"

# INCLUDE_WINDOWS="true"

standardBuildProcess
