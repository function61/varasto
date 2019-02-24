#!/bin/bash -eu

if [ ! -L "/usr/local/bin/varasto" ]; then
	ln -s /go/src/github.com/function61/varasto/rel/varasto_linux-amd64 /usr/local/bin/varasto
fi

source /build-common.sh

BINARY_NAME="varasto"
COMPILE_IN_DIRECTORY="cmd/varasto"

# vendor dir contains non-gofmt code..
GOFMT_TARGETS="cmd/ pkg/"

standardBuildProcess
