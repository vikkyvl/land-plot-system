#!/bin/sh

set -eu

if [ -f .env ]; then
	set -a
	. ./.env
	set +a
fi

gateway_url=${GATEWAY_URL:-http://localhost:8080}
token_url=${OIDC_TOKEN_URL:-http://localhost:${KEYCLOAK_PORT:-8180}/realms/land-plot/protocol/openid-connect/token}
demo_user=${DEMO_USER:-taras.melnyk@example.com}
demo_user_password=${TARAS_MELNYK_PASSWORD:-TarasDevPassword123}
requests=${OVERHEAD_REQUESTS:-500}
warmup_requests=50
go_services='identity-service land-plot-service report-service api-gateway'
work_dir=$(mktemp -d)
tracing_off=false

restore_tracing() {
	docker compose up -d $go_services >/dev/null 2>&1 || true
	tracing_off=false
}

cleanup() {
	if [ "$tracing_off" = true ]; then
		restore_tracing
	fi
	rm -rf "$work_dir"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

fail() {
	echo "$1" >&2
	exit 1
}

wait_for_healthy() {
	for service_name in $go_services; do
		attempt=0
		until [ "$(docker compose ps "$service_name" --format '{{.Health}}' 2>/dev/null)" = healthy ]; do
			attempt=$((attempt + 1))
			[ "$attempt" -lt 90 ] || fail "$service_name did not become healthy"
			sleep 2
		done
	done
	sleep 5
}

user_token() {
	curl -fsS \
		--data-urlencode grant_type=password \
		--data-urlencode client_id=land-plot-dev-cli \
		--data-urlencode "username=$demo_user" \
		--data-urlencode "password=$demo_user_password" \
		--data-urlencode 'scope=openid plots:read' \
		"$token_url" | jq -er '.access_token'
}

measure() {
	output=$1
	shift
	index=0
	while [ "$index" -lt "$warmup_requests" ]; do
		curl -s -o /dev/null "$@"
		index=$((index + 1))
	done
	index=0
	: >"$output"
	while [ "$index" -lt "$requests" ]; do
		curl -s -o /dev/null --write-out '%{time_total}\n' "$@" >>"$output"
		index=$((index + 1))
	done
}

percentiles() {
	sort -n "$1" | awk '{ values[NR] = $1 } END {
		p50 = values[int(NR * 0.50 + 0.5)]
		p95 = values[int(NR * 0.95 + 0.5)]
		printf "%.1f %.1f", p50 * 1000, p95 * 1000
	}'
}

run_series() {
	mode=$1
	token=$(user_token)
	measure "$work_dir/$mode-gateway" "$gateway_url/api/v1/plots"
	measure "$work_dir/$mode-plots" --header "Authorization: Bearer $token" "$gateway_url/api/v1/plots/summary"
}

echo "Gateway latency with and without tracing, $requests requests per series after $warmup_requests warm-up requests"
echo
echo "1. Tracing on (OTEL_EXPORTER_OTLP_ENDPOINT as configured)"
docker compose up -d $go_services >/dev/null 2>&1
wait_for_healthy
run_series on

echo "2. Tracing off (OTEL_EXPORTER_OTLP_ENDPOINT empty in the four Go services)"
tracing_off=true
OTEL_EXPORTER_OTLP_ENDPOINT= docker compose up -d $go_services >/dev/null 2>&1
wait_for_healthy
[ -z "$(docker compose exec -T api-gateway printenv OTEL_EXPORTER_OTLP_ENDPOINT 2>/dev/null)" ] || fail "the gateway still has an OTLP endpoint"
run_series off

echo "3. Restoring tracing"
restore_tracing
wait_for_healthy

echo
echo "| Request | Tracing | p50, ms | p95, ms |"
echo "|---|---|---|---|"
for series in gateway plots; do
	case $series in
	gateway) label='`GET /api/v1/plots` without a token (401 from the gateway)' ;;
	plots) label='`GET /api/v1/plots/summary` (gateway → land-plot-service → PostgreSQL)' ;;
	esac
	for mode in on off; do
		set -- $(percentiles "$work_dir/$mode-$series")
		echo "| $label | $mode | $1 | $2 |"
	done
done
