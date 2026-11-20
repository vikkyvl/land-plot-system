#!/bin/sh

set -eu

if [ -f .env ]; then
	set -a
	. ./.env
	set +a
fi

gateway_url=${GATEWAY_URL:-http://localhost:8080}
jaeger_url=${JAEGER_URL:-http://localhost:${JAEGER_UI_PORT:-16686}}
token_url=${OIDC_TOKEN_URL:-http://localhost:${KEYCLOAK_PORT:-8180}/realms/land-plot/protocol/openid-connect/token}
demo_user=${DEMO_USER:-taras.melnyk@example.com}
demo_user_password=${TARAS_MELNYK_PASSWORD:?set TARAS_MELNYK_PASSWORD in .env}
wait_seconds=${SAGA_TRACE_WAIT_SECONDS:-60}
points='[{"lat":50.45,"lon":30.52},{"lat":50.451,"lon":30.52},{"lat":50.451,"lon":30.522}]'
work_dir=$(mktemp -d)
confirmed_plot=""

cleanup() {
	if [ -n "$confirmed_plot" ]; then
		curl -s -o /dev/null -X DELETE -H "Authorization: Bearer $token" "$gateway_url/api/v1/plots/$confirmed_plot" || true
	fi

	rm -rf "$work_dir"
}
trap cleanup EXIT

token=$(curl -fsS -X POST "$token_url" \
	--data-urlencode grant_type=password \
	--data-urlencode client_id=land-plot-dev-cli \
	--data-urlencode "username=$demo_user" \
	--data-urlencode "password=$demo_user_password" \
	--data-urlencode "scope=openid plots:read plots:write" | jq -r .access_token)

create_plot() {
	curl -fsS -D "$work_dir/headers" -o "$work_dir/body" -X POST "$gateway_url/api/v1/plots" \
		-H "Authorization: Bearer $token" -H 'Content-Type: application/json' \
		-d "{\"name\":\"$1\",\"points\":$points}"
	tr -d '\r' <"$work_dir/headers" | awk 'tolower($1) == "x-trace-id:" {print $2}'
}

# Waits until the trace in Jaeger has every expected span and prints it.
wait_for_trace() {
	trace_id=$1
	shift
	elapsed=0

	while :; do
		curl -s "$jaeger_url/api/traces/$trace_id" >"$work_dir/trace" || true
		missing=""
		for expected in "$@"; do
			if ! jq -e --arg expected "$expected" \
				'.data[0] as $t | any($t.spans[]; ($t.processes[.processID].serviceName + " | " + .operationName) == $expected)' \
				"$work_dir/trace" >/dev/null 2>&1; then
				missing="$missing
   $expected"
			fi
		done

		if [ -z "$missing" ]; then
			break
		fi

		if [ "$elapsed" -ge "$wait_seconds" ]; then
			echo "   trace $trace_id misses:$missing"
			exit 1
		fi

		sleep 2
		elapsed=$((elapsed + 2))
	done

	jq -r '.data[0] as $t | $t.spans | sort_by(.startTime)[]
		| select(.operationName | test("^(POST|GET|PATCH|DELETE|publish|process|saga) "))
		| "   \($t.processes[.processID].serviceName) | \(.operationName)\(if any(.tags[]; .key == "error" and .value == true) then " (error)" else "" end)"' \
		"$work_dir/trace"
	echo "   one trace: $trace_id, $(jq '.data[0].spans | length' "$work_dir/trace") spans, services: $(jq -r '[.data[0].processes[].serviceName] | unique | join(", ")' "$work_dir/trace")"
}

echo "1. Saga that confirms the plot: POST /plots -> plot.created -> report -> plot.summary.applied -> CONFIRMED"
confirmed_trace=$(create_plot "Saga trace check")
confirmed_plot=$(jq -r .id "$work_dir/body")
wait_for_trace "$confirmed_trace" \
	"api-gateway | POST /api/v1/plots" \
	"land-plot-service | POST /api/v1/plots" \
	"land-plot-service | publish plot.created" \
	"report-service | process report.plot-events" \
	"report-service | publish plot.summary.applied" \
	"land-plot-service | process land-plot.saga-events"
echo "   plot status: $(curl -fsS -H "Authorization: Bearer $token" "$gateway_url/api/v1/plots/$confirmed_plot" | jq -r .status)"

echo
echo "2. Saga that is compensated: report-service rejects a plot named FAIL..., the dead letter cancels it"
failed_trace=$(create_plot "FAIL saga trace check")
failed_plot=$(jq -r .id "$work_dir/body")
wait_for_trace "$failed_trace" \
	"api-gateway | POST /api/v1/plots" \
	"land-plot-service | publish plot.created" \
	"report-service | process report.plot-events" \
	"land-plot-service | process land-plot.saga-events" \
	"land-plot-service | publish plot.cancelled"
echo "   plot status: $(curl -s -o /dev/null -w '%{http_code}' -H "Authorization: Bearer $token" "$gateway_url/api/v1/plots/$failed_plot") (cancelled plots are not found)"

echo
echo "Both sagas are single traces in Jaeger: $jaeger_url/trace/$confirmed_trace and $jaeger_url/trace/$failed_trace"
