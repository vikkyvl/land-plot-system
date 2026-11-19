#!/bin/sh

set -eu

if [ -f .env ]; then
	set -a
	. ./.env
	set +a
fi

cadvisor_url=${CADVISOR_URL:-http://localhost:${CADVISOR_PORT:-8088}}
wait_seconds=${CADVISOR_WAIT_SECONDS:-60}
service_label=container_label_com_docker_compose_service
metrics=$(mktemp)
trap 'rm -f "$metrics"' EXIT

echo "Waiting for cAdvisor at $cadvisor_url"
elapsed=0
until curl -fsS "$cadvisor_url/healthz" >/dev/null 2>&1; do
	if [ "$elapsed" -ge "$wait_seconds" ]; then
		echo "cAdvisor did not become healthy in ${wait_seconds}s"
		docker compose --profile cadvisor logs --tail 30 cadvisor
		exit 1
	fi
	sleep 2
	elapsed=$((elapsed + 2))
done

sleep 20
curl -fsS "$cadvisor_url/metrics" >"$metrics"

echo
echo "Memory and CPU per compose service"
services=$(grep '^container_memory_working_set_bytes{' "$metrics" | grep -o "$service_label=\"[^\"]*\"" | cut -d'"' -f2 | grep -v '^$' | sort -u)
for service in $services; do
	memory=$(grep '^container_memory_working_set_bytes{' "$metrics" | grep "$service_label=\"$service\"" | awk '{sum += $2} END {printf "%.0f", sum / 1048576}')
	cpu=$(grep '^container_cpu_usage_seconds_total{' "$metrics" | grep "$service_label=\"$service\"" | awk '{sum += $2} END {printf "%.1f", sum}')
	echo "   $service: ${memory} MiB, CPU ${cpu} s"
done

if [ -z "$services" ]; then
	echo "   none: cAdvisor runs but does not see the containers of the project"
	echo "   the dashboards fall back to process_cpu_seconds_total and process_resident_memory_bytes of the Go services"
	exit 1
fi

echo
echo "cAdvisor sees $(echo "$services" | wc -l | tr -d ' ') compose services"
