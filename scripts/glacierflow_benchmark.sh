#!/bin/bash
sum=0
hasBc=0
bcBin=$(which bc);
if [ -f "$bcBin" ]; then
    hasBc=1
fi

# TODO: bench presets, storage per model/preset


echo "# ----- BENCH START -----"
for i in {1..5}; do
    jsonResponse=$(curl http://localhost:8080/v1/chat/completions \
		-H "Content-Type: application/json" \
		-d @./../data/shared/benchmark_presets/coding_example.json -s
	)
	modelName=$(echo "$jsonResponse" | jq '.model' | xargs)
	predictionTps=$(echo "$jsonResponse" | jq '.timings.predicted_per_second' | xargs)
	echo "# $modelName Pass #$i TPS: $predictionTps"
	if [ "$hasBc" -eq 1 ]; then
		sum=$(echo "$sum + $predictionTps" | bc -l)
	fi
done
if [ "$hasBc" -eq 1 ]; then
	mean=$(echo "$sum/5" | bc -l)
	echo "# AVERAGE TPS: $mean"
fi
echo "# ----- BENCH DONE -----"
