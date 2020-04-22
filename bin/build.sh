#!/bin/bash -eu

if [ ! -L "/usr/local/bin/sto" ]; then
	ln -s /workspace/rel/sto_linux-amd64 /usr/local/bin/sto
fi

function docsDeployerSpec {
	if [ -n "${FASTBUILD:-}" ]; then
		return # skip non-essential step
	fi

	cd misc/docs-website-deployerspec/

	deployer package "$FRIENDLY_REV_ID" ../../rel/docs-website-deployerspec.zip
}

function updateServer {
	if [ -n "${FASTBUILD:-}" ]; then
		return # skip non-essential step
	fi

	cd misc/varasto-updateserver/

	# this is a file that will be deployed (by function61/deployer) at:
	#     https://function61.com/varasto/updateserver/latest-version.json
	# this won't be deployed on all commits however - deployments will be initiated manually
	# when we want users to update to the released version
	echo -n "{\"LatestVersion\": \"$FRIENDLY_REV_ID\"}" > latest-version.json

	tar -czf "updateserver.tar.gz" latest-version.json

	# so it won't end up in deployerspec zip
	rm latest-version.json

	deployer package "$FRIENDLY_REV_ID" ../../rel/updateserver-deployerspec.zip

	# clean up generated files
	rm updateserver.tar.gz
}

# make sure parent dir exits, under which FUSE projector will mount itself
mkdir -p /mnt/stofuse

source /build-common.sh

BINARY_NAME="sto"
COMPILE_IN_DIRECTORY="cmd/sto"

# vendor dir contains non-gofmt code..
GOFMT_TARGETS="cmd/ pkg/"

standardBuildProcess

(docsDeployerSpec)

(updateServer)
