#!/bin/bash
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GF_AUDIO_CPP_VERSION_FILE="$SCRIPT_DIR/last_build.txt"
GF_AUDIO_REPO=https://github.com/0xShug0/audio.cpp.git

if [ -f "$SCRIPT_DIR/../../.env" ]; then
	set -a
	source "$SCRIPT_DIR/../../.env"
	set +a
fi

# Re-read env var in case .env set it
GF_AUDIO_SRC_DIR="${GF_AUDIO_CPP_SRC_DIR:-$HOME/.audio.cpp-src}"

if [ ! -d "$GF_AUDIO_SRC_DIR" ]; then
	git clone "$GF_AUDIO_REPO" "$GF_AUDIO_SRC_DIR"
fi

if [ ! -d "$GF_AUDIO_SRC_DIR" ]; then
	echo "Failed to clone repo to $GF_AUDIO_SRC_DIR"
	exit 1
fi

cd "$GF_AUDIO_SRC_DIR"
git pull || {
	echo "git pull failed in $GF_AUDIO_SRC_DIR"
	exit 1
}

currentVersion=$(git rev-parse --verify HEAD)
oldVersion=$(cat "$GF_AUDIO_CPP_VERSION_FILE" 2>/dev/null)

if [[ "$oldVersion" == "$currentVersion" ]]; then
	echo "audio.cpp binaries up to date"
else
	chmod +x scripts/build_linux.sh
	./scripts/build_linux.sh --native-model-manager --deployment-build --backend vulkan --target audiocpp_cli --target audiocpp_server
	if [ $? -ne 0 ]; then
		echo "FAIL" > "$GF_AUDIO_CPP_VERSION_FILE"
		echo "Build failed"
		exit 1
	fi

	# Copy built binaries back to project
	BUILD_OUT="$GF_AUDIO_SRC_DIR/build/linux-vulkan-release/bin"
	DEST="$SCRIPT_DIR/build/bin"
	if [ ! -d "$BUILD_OUT" ]; then
		echo "Build output directory not found: $BUILD_OUT"
		echo "FAIL" > "$GF_AUDIO_CPP_VERSION_FILE"
		exit 1
	fi
	mkdir -p "$DEST"
	rm -rf "$DEST"/*
	cp -a "$BUILD_OUT"/* "$DEST"/ || {
		echo "Failed to copy binaries to $DEST"
		echo "FAIL" > "$GF_AUDIO_CPP_VERSION_FILE"
		exit 1
	}

	echo "$currentVersion" > "$GF_AUDIO_CPP_VERSION_FILE"
	echo "Binaries copied to $DEST"
fi
