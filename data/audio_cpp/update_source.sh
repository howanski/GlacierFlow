#!/bin/bash
GF_AUDIO_CPP_VERSION_FILE="../last_build.txt"
GF_AUDIO_REPO=https://github.com/0xShug0/audio.cpp.git
GF_AUDIO_REPO_DIR="audio.cpp"
if [ -f "../../.env" ]; then
	set -a
	source "../../.env"
	set +a
fi

if [ ! -d "$GF_AUDIO_REPO_DIR" ]; then
	git clone "$GF_AUDIO_REPO"
fi

if [ ! -d "$GF_AUDIO_REPO_DIR" ]; then
	echo "Failed to clone repo"
	exit 1
fi

cd "$GF_AUDIO_REPO_DIR"
mkdir -p build/linux-vulkan-release/bin/
git pull

currentVersion=$(git rev-parse --verify HEAD)

touch "$GF_AUDIO_CPP_VERSION_FILE"

oldVersion=$(cat "$GF_AUDIO_CPP_VERSION_FILE")

if [[ "$oldVersion" == "$currentVersion" ]]; then
	echo "audio.cpp binaries up to date"
else
	chmod +x scripts/build_linux.sh
	./scripts/build_linux.sh --native-model-manager --deployment-build --backend vulkan --target audiocpp_server
	if [ $? -ne 0 ]; then
		echo "FAIL" > "$GF_AUDIO_CPP_VERSION_FILE"
		echo "Build failed"
		exit 1
	fi
	echo "$currentVersion" > "$GF_AUDIO_CPP_VERSION_FILE"
fi
