#!/bin/bash -eu

buildFrontend() {
	source /build-common.sh

	standardBuildProcess "frontend"
}

copyF61uiStaticFiles() {
	rm -rf public/f61ui/
	cp -r frontend/f61ui/public/ public/f61ui/
}

packagePublicFiles() {
	tar -czf rel/public.tar.gz public/
}

(cd frontend/ && buildFrontend)

copyF61uiStaticFiles

packagePublicFiles
