#!/bin/sh

set -eu

if [ -f .env ]; then
	set -a
	. ./.env
	set +a
fi

prometheus_url=${PROMETHEUS_URL:-http://localhost:${PROMETHEUS_PORT:-19090}}
grafana_url=${GRAFANA_URL:-http://localhost:${GRAFANA_PORT:-13000}}
jaeger_url=${JAEGER_URL:-http://localhost:${JAEGER_UI_PORT:-16686}}
alertmanager_url=${ALERTMANAGER_URL:-http://localhost:${ALERTMANAGER_PORT:-19093}}
optional_jobs=${OPTIONAL_JOBS:-cadvisor}

echo "Prometheus targets ($prometheus_url)"
targets=$(curl -fsS "$prometheus_url/api/v1/targets?state=active")
echo "$targets" | jq -r '.data.activeTargets[] | "   \(.labels.job): \(.health)\(if .lastError != "" then " (\(.lastError))" else "" end)"' | sort

down=$(echo "$targets" | jq -r --arg optional "$optional_jobs" '
	($optional | split(" ")) as $skip
	| [.data.activeTargets[] | select(.health != "up") | .labels.job | select(. as $job | $skip | index($job) | not)]
	| unique | join(" ")')

echo
echo "User interfaces"
for name_url in "Grafana $grafana_url/api/health" "Jaeger $jaeger_url/" "Alertmanager $alertmanager_url/-/ready" "Prometheus $prometheus_url/-/ready"; do
	name=${name_url%% *}
	url=${name_url#* }
	status=$(curl -s -o /dev/null -w '%{http_code}' "$url")
	echo "   $name: $status"
	if [ "$status" != 200 ]; then
		down="$down $name"
	fi
done

if [ -n "$(echo "$down" | tr -d ' ')" ]; then
	echo
	echo "Not healthy:$down"
	exit 1
fi

echo
echo "Observability stack is healthy (optional jobs: $optional_jobs)"
