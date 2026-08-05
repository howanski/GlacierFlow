#!/bin/bash
GF_STABLE_DIFFUSION_BIN_DIR="binaries"
GF_STABLE_DIFFUSION_RELEASE_FILE="release.zip"
GF_STABLE_DIFFUSION_SOURCE="https://github.com/leejet/stable-diffusion.cpp/releases/download/master-812-ea7f0c8/sd-master-ea7f0c8-bin-Linux-Ubuntu-24.04-x86_64-vulkan.zip"
GF_STABLE_DIFFUSION_VERSION_FILE="current_source.txt"
# GF_STABLE_DIFFUSION_SOURCE is overriden in .env
# see: https://github.com/leejet/stable-diffusion.cpp/releases
if [ -f "../../.env" ]; then
	set -a
	source "../../.env"
	set +a
fi

touch "$GF_STABLE_DIFFUSION_VERSION_FILE"

oldVersion=$(cat "$GF_STABLE_DIFFUSION_VERSION_FILE")

if [[ "$oldVersion" == "$GF_STABLE_DIFFUSION_SOURCE" ]]; then
	echo "StableDiffusion.cpp binaries up to date"
else
	rm -f "$GF_STABLE_DIFFUSION_RELEASE_FILE"
	if ! wget -c "$GF_STABLE_DIFFUSION_SOURCE" -O "$GF_STABLE_DIFFUSION_RELEASE_FILE"; then
		echo "ERROR: failed to download $GF_STABLE_DIFFUSION_SOURCE"
		exit 1
	fi

	echo "$GF_STABLE_DIFFUSION_SOURCE" >"$GF_STABLE_DIFFUSION_VERSION_FILE"
	rm -rf "$GF_STABLE_DIFFUSION_BIN_DIR"
	mkdir -p "$GF_STABLE_DIFFUSION_BIN_DIR"
	unzip "$GF_STABLE_DIFFUSION_RELEASE_FILE" -d "$GF_STABLE_DIFFUSION_BIN_DIR"
	echo "StableDiffusion.cpp binaries updated successfully"
fi
