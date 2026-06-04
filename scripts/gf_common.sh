#!/bin/bash
# GlacierFlow Common Library

# ── Shared Constants ──────────────────────────────────────────────────────────

GF_INFERENCE_URL="http://localhost:8080"

GF_INFERENCE_SERVICE_UP_STATUS='../data/shared/inference_status.txt'
GF_INFERENCE_SERVICE_UP_HASH='../data/shared/inference_up_hash.txt'

GF_INFERENCE_PRESET_NAME='../data/shared/inference_preset_name.txt'
GF_INFERENCE_PRESET_NAME_DESIRED='../data/shared/inference_preset_name_target.txt'

GF_INFERENCE_COMPOSE_FILE_SOURCE='../data/shared/inference_config.yml'
GF_INFERENCE_COMPOSE_FILE_TARGET='../docker/llama_cpp/compose.override.yml'

GF_INFERENCE_PRESETS_DIR_LOCAL='../data/shared/inference_presets/local/'
GF_INFERENCE_PRESETS_DIR_EXAMPLE='../data/shared/inference_presets/examples/'

GF_BENCHMARK_PRESETS_DIR='./../data/shared/benchmarks/'

GF_STACK_NAME="glacierflow"

# ── Utility Functions ─────────────────────────────────────────────────────────

tty_clear() {
	[[ -t 0 ]] && clear
}

has_binary() {
	if [ -f "$(which "$1")" ]; then
		echo "1"
	else
		echo "0"
	fi
}

fail_preflight_on_missing_binary() {
	if [ "$(has_binary "$1")" -ne "1" ]; then
		echo "Preflight check failed, missing binary from the system: $1"
		exit 127
	fi
}

add_minimal_padding() {
	local var="$1"
	local min_len="${2:-2}"
	local align="${3:-right}" # 'left' or 'right' (default)

	if [[ "$align" == "right" ]]; then
		printf "%-${min_len}s" "$var"
	else
		printf "%*s" "$min_len" "$var"
	fi
}
