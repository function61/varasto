#!/bin/bash -eu

if [ ! -L "/usr/local/bin/sto" ]; then
	ln -s /workspace/rel/sto_linux-amd64 /usr/local/bin/sto
fi

# make sure parent dir exits, under which FUSE projector will mount itself
mkdir -p /mnt/stofuse

source /build-common.sh

BINARY_NAME="sto"
COMPILE_IN_DIRECTORY="cmd/sto"

# vendor dir contains non-gofmt code..
GOFMT_TARGETS="cmd/ pkg/"

standardBuildProcess
