#!/usr/bin/env bash

set -euo pipefail

source "$(dirname "$0")/gf_common.sh"

# Configuration
LOG_FILE=gf_benchy.log
MODEL_NAME="xyz"     # doesn't matter with singular model served via llama.cpp
OUTPUT_FORMAT="json" # Options: md, json, csv
PP_ADD=500           # Prompt tokens appended after the cached context during the evaluation run
TG_TOKENS=200        # Number of output tokens to generate when measuring
VALUE_TYPE_PROMPT_PROCESSING="PP"
VALUE_TYPE_TOKEN_GENERATION="TG"

if [ ! -f "$LOG_FILE" ]; then
	touch "$LOG_FILE"
fi

# $1 -> how much kv cache to fill before testing
# $2 -> how many rounds (more rounds = more time, more accurate calculation)
run_benchy() {
	local currentModelHash=$(cat "$GF_INFERENCE_SERVICE_UP_HASH")
	echo "Probing hash : $currentModelHash"
	echo "Probing depth: $1"

	local speedAchievedPromptProcessing=$(read_value $currentModelHash $VALUE_TYPE_PROMPT_PROCESSING $1)
	local speedAchievedTokenGeneration=$(read_value $currentModelHash $VALUE_TYPE_TOKEN_GENERATION $1)

	if [ ! -z "$speedAchievedPromptProcessing" ]; then
		if [ ! -z "$speedAchievedTokenGeneration" ]; then
			echo "$VALUE_TYPE_PROMPT_PROCESSING : $speedAchievedPromptProcessing"
			echo "$VALUE_TYPE_TOKEN_GENERATION : $speedAchievedTokenGeneration"
			return
		fi
	fi

	local benchResultFile=$(mktemp /tmp/benchy_XXXXXX.json)
	uvx llama-benchy \
		--base-url "$GF_INFERENCE_URL/" \
		--depth $1 \
		--enable-prefix-caching \
		--format "$OUTPUT_FORMAT" \
		--latency-mode generation \
		--model "$MODEL_NAME" \
		--pp "$PP_ADD" \
		--runs "$2" \
		--save-result "$benchResultFile" \
		--tg "$TG_TOKENS" 2>/dev/null 1>/dev/null

	speedAchievedTokenGeneration=$(jq -r '.benchmarks[1].tg_throughput.mean' "$benchResultFile")
	speedAchievedPromptProcessing=$(jq -r '.benchmarks[1].pp_throughput.mean' "$benchResultFile")

	echo "$VALUE_TYPE_PROMPT_PROCESSING : $speedAchievedPromptProcessing"
	echo "$VALUE_TYPE_TOKEN_GENERATION : $speedAchievedTokenGeneration"

	save_value $currentModelHash $VALUE_TYPE_PROMPT_PROCESSING $1 $speedAchievedPromptProcessing
	save_value $currentModelHash $VALUE_TYPE_TOKEN_GENERATION $1 $speedAchievedTokenGeneration

	rm $benchResultFile
}

# $1 hash
# $2 value type
# $3 depth
get_record_key() {
	echo "${1}_${2}_${3}_"
}

# $1 hash
# $2 value type
# $3 depth
# $4 value
# overwrites old value if exists
save_value() {
	local key=$(get_record_key $1 $2 $3)
	if grep -q "^${key} " "$LOG_FILE" 2>/dev/null; then
		sed -i "s/^${key} .*/${key} $4/" "$LOG_FILE"
	else
		echo "${key} $4" >>$LOG_FILE
	fi
}

# $1 hash
# $2 value type
# $3 depth
read_value() {
	grep "$(get_record_key $1 $2 $3)" "$LOG_FILE" | cut -d " " -f 2
}

# ── Listing ──────────────────────────────────────────────────────────────────

# $1 raw value from log (may be empty)
fmt_value() {
	if [ -z "$1" ]; then
		echo "-"
	else
		awk -v v="$1" 'BEGIN { printf "%.2f", v }'
	fi
}

list_results() {
	local presetFiles=$(find "$GF_INFERENCE_PRESETS_DIR_LOCAL" -name '*.yml' | sort -f)
	if [ -z "$presetFiles" ]; then
		echo "Listing example presets (no local presets found)"
		presetFiles=$(find "$GF_INFERENCE_PRESETS_DIR_EXAMPLE" -name '*.yml' | sort -f) || presetFiles=""
	fi

	local currentModelHash=$(cat "$GF_INFERENCE_SERVICE_UP_HASH" 2>/dev/null)

	# compute max preset name length for alignment
	local presetNameLength=1
	local presetName
	for presetFile in $presetFiles; do
		presetName=$(basename "$presetFile" ".yml")
		if [ "${#presetName}" -gt "$presetNameLength" ]; then
			presetNameLength=${#presetName}
		fi
	done
	presetNameLength=$((presetNameLength + 1))

	# header
	local printLine="  "
	printLine+=$(add_minimal_padding "Model Name" "$presetNameLength" left)
	local col
	for col in pp2k tg2k pp32k tg32k pp132k tg132k; do
		printLine+=$(add_minimal_padding "$col" 8 left)
	done
	echo "$printLine"

	# data
	for presetFile in $presetFiles; do
		presetName=$(basename "$presetFile" ".yml")
		local presetHash=$(md5sum "$presetFile" | cut -d " " -f 1)

		local prefix="   "
		if [ "$presetHash" == "$currentModelHash" ]; then
			prefix="-> "
		fi

		printLine="$prefix$(add_minimal_padding "$presetName" "$presetNameLength" left)"
		local depth pp tg
		for depth in 2000 32000 132000; do
			pp=$(read_value "$presetHash" "$VALUE_TYPE_PROMPT_PROCESSING" "$depth") || pp=""
			tg=$(read_value "$presetHash" "$VALUE_TYPE_TOKEN_GENERATION" "$depth") || tg=""
			printLine+=$(add_minimal_padding "$(fmt_value "$pp")" 8 left)
			printLine+=$(add_minimal_padding "$(fmt_value "$tg")" 8 left)
		done
		echo "$printLine"
	done
}

# ── Preset selector ──────────────────────────────────────────────────────────
inference_preset_change() {
	local selectedPreset=""
	while [ -z "$selectedPreset" ]; do
		local presetFiles=$(find "$GF_INFERENCE_PRESETS_DIR_LOCAL" -name '*.yml' | sort -f)
		if [ -z "$presetFiles" ]; then
			echo "Listing example presets (no local presets found)"
			presetFiles=$(find "$GF_INFERENCE_PRESETS_DIR_EXAMPLE" -name '*.yml' | sort -f) || presetFiles=""
		fi

		for presetFile in $presetFiles; do
			basename "$presetFile" ".yml"
		done
		echo ""
		read -p "Preset to load: " selectedPreset
	done
	"$(dirname "$0")/glacierflow_inference_select_preset" "$selectedPreset"
	sleep 1
}

# ── Main Menu ────────────────────────────────────────────────────────────────

# $1 -> how much kv cache to fill before testing
# $2 -> how many rounds
# Iterates over every preset file, switching to it before each run,
# so the whole set is benchmarked one by one without stopping.
run_benchy_on_all_presets() {
	local depth=$1
	local runs=$2

	local presetFiles=$(find "$GF_INFERENCE_PRESETS_DIR_LOCAL" -name '*.yml' | sort -f)
	if [ -z "$presetFiles" ]; then
		echo "Listing example presets (no local presets found)"
		presetFiles=$(find "$GF_INFERENCE_PRESETS_DIR_EXAMPLE" -name '*.yml' | sort -f) || presetFiles=""
	fi

	for presetFile in $presetFiles; do
		local presetName=$(basename "$presetFile" ".yml")
		local presetHash=$(md5sum "$presetFile" | cut -d " " -f 1)

		# Skip presets whose values already exist for this depth. Checking the
		# log avoids switching to (loading) a model we have already benchmarked.
		local existingPP existingTG
		existingPP=$(read_value "$presetHash" "$VALUE_TYPE_PROMPT_PROCESSING" "$depth" || true)
		existingTG=$(read_value "$presetHash" "$VALUE_TYPE_TOKEN_GENERATION" "$depth" || true)
		if [ -n "$existingPP" ] && [ -n "$existingTG" ]; then
			echo ""
			echo "▶ Skipping preset: $presetName (depth $depth) — values already present"
			continue
		fi

		echo ""
		"$(dirname "$0")/glacierflow_inference_select_preset" "$presetName"
		echo "▶ Benchmarking preset: $presetName (depth $depth)"
		echo "  Preset: $presetName"
		echo "  Depth : $depth"
		echo ""
		run_benchy "$depth" "$runs"
	done
}

tty_clear
while true; do
	echo ""
	echo "=== gf_benchy ==="
	echo " 1) Run bench with depth 2000"
	echo " 2) Run bench with depth 32000"
	echo " 3) Run bench with depth 132000"
	echo " 5) Run bench on ALL presets (depth 2000)"
	echo " 6) Run bench on ALL presets (depth 32000)"
	echo " 7) Run bench on ALL presets (depth 132000)"
	echo " L) Listing"
	echo " P) Change preset"
	echo " q) Quit"
	read -rp "Select an option: " choice || break

	case "$choice" in
	1) tty_clear && run_benchy 2000 3 && list_results ;;
	2) tty_clear && run_benchy 32000 1 && list_results ;;
	3) tty_clear && run_benchy 132000 1 && list_results ;;
	5) tty_clear && run_benchy_on_all_presets 2000 3 && list_results ;;
	6) tty_clear && run_benchy_on_all_presets 32000 1 && list_results ;;
	7) tty_clear && run_benchy_on_all_presets 132000 1 && list_results ;;
	l | L) tty_clear && list_results ;;
	p | P) inference_preset_change ;;
	q | Q) break ;;
	*) echo "Invalid option: $choice" ;;
	esac
done
