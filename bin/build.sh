#!/bin/bash -eu

if [ ! -L "/usr/local/bin/bup" ]; then
	ln -s /go/src/github.com/function61/bup/rel/bup_linux-amd64 /usr/local/bin/bup
fi

source /build-common.sh

BINARY_NAME="bup"
COMPILE_IN_DIRECTORY="cmd/bup"

# vendor dir contains non-gofmt code..
GOFMT_TARGETS="cmd/ pkg/"

standardBuildProcess
