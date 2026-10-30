#!/bin/sh

set -eu

gateway_url=${GATEWAY_URL:-http://localhost:8080}
consul_port=${CONSUL_HTTP_PORT:-8500}
consul_url=${CONSUL_URL:-http://localhost:${consul_port}}
rabbitmq_port=${RABBITMQ_MANAGEMENT_PORT:-15672}
rabbitmq_url=${RABBITMQ_MANAGEMENT_URL:-http://localhost:${rabbitmq_port}}
rabbitmq_user=${RABBITMQ_USER:-land_app}
rabbitmq_password=${RABBITMQ_PASSWORD:-rabbitmq_dev_password}
demo_password=${DEMO_PASSWORD:-DemoPassword123}
demo_tmp=$(mktemp -d)
stopped_container=

cleanup() {
	if [ -n "$stopped_container" ]; then
		docker start "$stopped_container" >/dev/null 2>&1 || true
	fi
	rm -rf "$demo_tmp"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

require_command() {
	if ! command -v "$1" >/dev/null 2>&1; then
		echo "required command is missing: $1" >&2
		exit 1
	fi
}

wait_for_gateway() {
	attempt=0
	while [ "$attempt" -lt 30 ]; do
		if curl -fsS "$gateway_url/health" >/dev/null; then
			return
		fi
		attempt=$((attempt + 1))
		sleep 1
	done
	echo "gateway did not become healthy" >&2
	exit 1
}

healthy_instances() {
	curl -fsS "$consul_url/v1/health/service/$1?passing=true" | jq 'length'
}

wait_for_healthy_count() {
	service_name=$1
	expected=$2
	attempt=0
	while [ "$attempt" -lt 40 ]; do
		actual=$(healthy_instances "$service_name")
		if [ "$actual" -eq "$expected" ]; then
			return
		fi
		attempt=$((attempt + 1))
		sleep 1
	done
	echo "healthy $service_name instance count did not become $expected" >&2
	exit 1
}

queue_consumers() {
	curl -fsS \
		--user "$rabbitmq_user:$rabbitmq_password" \
		"$rabbitmq_url/api/queues/%2F/report.plot-events" |
		jq '.consumers'
}

wait_for_queue_consumers() {
	expected=$1
	attempt=0
	while [ "$attempt" -lt 40 ]; do
		if [ "$(queue_consumers)" -eq "$expected" ]; then
			return
		fi
		attempt=$((attempt + 1))
		sleep 1
	done
	echo "report.plot-events consumer count did not become $expected" >&2
	exit 1
}

create_plot() {
	curl -fsS \
		--request POST \
		--header "Authorization: Bearer $token" \
		--header 'Content-Type: application/json' \
		--data "{\"name\":\"$1\",\"points\":[{\"lat\":50.45,\"lon\":30.50},{\"lat\":50.45,\"lon\":30.51},{\"lat\":50.46,\"lon\":30.51},{\"lat\":50.46,\"lon\":30.50}]}" \
		"$gateway_url/api/v1/plots" |
		jq -er '.id'
}

collect_processed_events() {
	docker compose logs --no-color report-service >"$demo_tmp/report-logs.txt"
	: >"$demo_tmp/competing.txt"
	for competing_plot_id in $competing_plot_ids; do
		grep '"msg":"plot event processed"' "$demo_tmp/report-logs.txt" |
			grep "\"plot_id\":\"$competing_plot_id\"" |
			sed -n "1s/^\([^ |]*\).*/\1 $competing_plot_id/p" >>"$demo_tmp/competing.txt"
	done
	wc -l <"$demo_tmp/competing.txt" | tr -d '[:space:]'
}

summary_version() {
	docker compose exec -T postgres psql -U report_user -d report_db -Atc \
		"SELECT version FROM plot_summaries WHERE plot_id = '$plot_id';" |
		tr -d '[:space:]'
}

wait_for_summary_version() {
	expected=$1
	attempt=0
	while [ "$attempt" -lt 20 ]; do
		actual=$(summary_version)
		if [ "$actual" = "$expected" ]; then
			return
		fi
		attempt=$((attempt + 1))
		sleep 1
	done
	echo "plot summary did not reach version $expected" >&2
	exit 1
}

processed_event_count() {
	docker compose exec -T postgres psql -U report_user -d report_db -Atc \
		"SELECT count(*) FROM processed_events WHERE event_id = '$event_id';" |
		tr -d '[:space:]'
}

publish_event() {
	event_payload=$(jq -cn \
		--arg event_id "$event_id" \
		--arg occurred_at "$occurred_at" \
		--arg plot_id "$plot_id" \
		--arg owner_id "$owner_id" \
		--arg name "$updated_name" \
		--argjson area_m2 "$area_m2" \
		'{schema_version:1,event_id:$event_id,event_type:"plot.updated",occurred_at:$occurred_at,plot_id:$plot_id,owner_id:$owner_id,name:$name,area_m2:$area_m2,version:2}')
	publish_request=$(jq -cn --arg payload "$event_payload" \
		'{properties:{content_type:"application/json",delivery_mode:2},routing_key:"plot.updated",payload:$payload,payload_encoding:"string"}')
	result=$(curl -fsS \
		--user "$rabbitmq_user:$rabbitmq_password" \
		--header 'Content-Type: application/json' \
		--data "$publish_request" \
		"$rabbitmq_url/api/exchanges/%2F/land.events/publish")
	if [ "$(printf '%s' "$result" | jq -r '.routed')" != "true" ]; then
		echo "RabbitMQ did not route the demonstration event" >&2
		exit 1
	fi
}

for command_name in docker curl jq; do
	require_command "$command_name"
done

echo "1. Starting all components with three land-plot replicas"
docker compose up -d --build --scale land-plot-service=3
wait_for_gateway
wait_for_healthy_count land-plot-service 3
docker compose ps

echo "2. Checking every service health endpoint"
curl -fsS "$gateway_url/health" | jq .
docker compose exec -T identity-service wget -qO- http://127.0.0.1:8081/health
docker compose exec -T --index 1 land-plot-service wget -qO- http://127.0.0.1:8082/health
docker compose exec -T report-service wget -qO- http://127.0.0.1:8083/health

echo "3. Checking Consul registrations"
curl -fsS "$consul_url/v1/health/state/passing" |
	jq -r '.[] | select(.ServiceName != "" and .ServiceName != "consul") | [.ServiceName,.ServiceID,.Status] | @tsv'

demo_email="lr2-demo-$(date +%s)-$$@example.com"
curl -fsS \
	--request POST \
	--header 'Content-Type: application/json' \
	--data "{\"email\":\"$demo_email\",\"password\":\"$demo_password\"}" \
	"$gateway_url/api/v1/auth/register" |
	jq .

login_response=$(curl -fsS \
	--request POST \
	--header 'Content-Type: application/json' \
	--data "{\"email\":\"$demo_email\",\"password\":\"$demo_password\"}" \
	"$gateway_url/api/v1/auth/login")
token=$(printf '%s' "$login_response" | jq -er '.access_token')

unauthorized_status=$(curl -sS \
	--output "$demo_tmp/unauthorized.json" \
	--write-out '%{http_code}' \
	"$gateway_url/api/v1/plots")
if [ "$unauthorized_status" != "401" ]; then
	echo "GET /api/v1/plots without JWT returned $unauthorized_status instead of 401" >&2
	exit 1
fi
echo "4. Protected request without JWT returned 401"

invalid_polygon_status=$(curl -sS \
	--output "$demo_tmp/invalid-polygon.json" \
	--write-out '%{http_code}' \
	--request POST \
	--header "Authorization: Bearer $token" \
	--header 'Content-Type: application/json' \
	--data '{"name":"Invalid polygon","points":[{"lat":0,"lon":0},{"lat":1,"lon":1},{"lat":0,"lon":1},{"lat":1,"lon":0}]}' \
	"$gateway_url/api/v1/plots")
if [ "$invalid_polygon_status" != "422" ]; then
	echo "self-intersecting polygon returned $invalid_polygon_status instead of 422" >&2
	exit 1
fi
echo "   Self-intersecting polygon returned 422"

plot_response=$(curl -fsS \
	--request POST \
	--header "Authorization: Bearer $token" \
	--header 'Content-Type: application/json' \
	--data '{"name":"LR2 demonstration plot","points":[{"lat":50.45,"lon":30.50},{"lat":50.45,"lon":30.51},{"lat":50.46,"lon":30.51},{"lat":50.46,"lon":30.50}]}' \
	"$gateway_url/api/v1/plots")
plot_id=$(printf '%s' "$plot_response" | jq -er '.id')
owner_id=$(printf '%s' "$plot_response" | jq -er '.owner_id')
area_m2=$(printf '%s' "$plot_response" | jq -er '.area_m2')
printf '%s' "$plot_response" | jq -e '.area_m2 > 0 and .perimeter_m > 0 and .version == 1' >/dev/null
echo "5. Plot created with calculated area and perimeter"
printf '%s' "$plot_response" | jq '{id,area_m2,perimeter_m,version}'

echo "6. Checking Round Robin across three land-plot instances"
: >"$demo_tmp/served-by.txt"
request_number=1
while [ "$request_number" -le 6 ]; do
	curl -fsS \
		--dump-header "$demo_tmp/headers.txt" \
		--output /dev/null \
		--header "Authorization: Bearer $token" \
		"$gateway_url/api/v1/plots/$plot_id"
	sed -n 's/^[Xx]-[Ss]erved-[Bb]y: *//p' "$demo_tmp/headers.txt" |
		tr -d '\r' >>"$demo_tmp/served-by.txt"
	request_number=$((request_number + 1))
done
sort -u "$demo_tmp/served-by.txt"
served_count=$(sort -u "$demo_tmp/served-by.txt" | wc -l | tr -d '[:space:]')
if [ "$served_count" -ne 3 ]; then
	echo "Round Robin reached $served_count instances instead of 3" >&2
	exit 1
fi

echo "7. Checking synchronous gateway -> report -> land flow"
curl -fsS "$gateway_url/api/v1/demo/chain?message=lr2-demo" |
	jq -e '.path | length == 3 and .[0].service == "api-gateway" and .[1].service == "report-service" and .[2].service == "land-plot-service"'
report_response=$(curl -fsS \
	--request POST \
	--header "Authorization: Bearer $token" \
	--header 'Content-Type: application/json' \
	--data "{\"plot_id\":\"$plot_id\"}" \
	"$gateway_url/api/v1/reports")
printf '%s' "$report_response" | jq -e '.area_m2 > 0 and .area_ares > 0 and .area_hectares > 0 and .plot_version == 1'

echo "8. Checking asynchronous create and update projections"
wait_for_summary_version 1
updated_name="LR2 demonstration plot updated"
curl -fsS \
	--request PATCH \
	--header "Authorization: Bearer $token" \
	--header 'Content-Type: application/json' \
	--data "{\"name\":\"$updated_name\"}" \
	"$gateway_url/api/v1/plots/$plot_id" |
	jq -e '.version == 2'
wait_for_summary_version 2
docker compose exec -T postgres psql -U report_user -d report_db -c \
	"SELECT plot_id, name, area_m2, version FROM plot_summaries WHERE plot_id = '$plot_id';"

echo "9. Checking idempotency with the same event_id twice"
event_id=$(docker compose exec -T postgres psql -U report_user -d report_db -Atc 'SELECT gen_random_uuid();' | tr -d '[:space:]')
occurred_at=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
publish_event
publish_event
attempt=0
while [ "$attempt" -lt 20 ] && [ "$(processed_event_count)" != "1" ]; do
	attempt=$((attempt + 1))
	sleep 1
done
if [ "$(processed_event_count)" != "1" ]; then
	echo "duplicate event_id was not processed idempotently" >&2
	exit 1
fi
echo "processed_events rows for $event_id: 1"

echo "10. Stopping one replica and checking health-aware failover"
stopped_container=$(docker compose ps -q land-plot-service | sed -n '1p')
stopped_hostname=$(docker inspect --format '{{.Config.Hostname}}' "$stopped_container")
stopped_instance="land-plot-service-$stopped_hostname"
docker stop "$stopped_container" >/dev/null
wait_for_healthy_count land-plot-service 2
request_number=1
while [ "$request_number" -le 4 ]; do
	curl -fsS \
		--dump-header "$demo_tmp/failover-headers.txt" \
		--output /dev/null \
		--header "Authorization: Bearer $token" \
		"$gateway_url/api/v1/plots/$plot_id"
	served_by=$(sed -n 's/^[Xx]-[Ss]erved-[Bb]y: *//p' "$demo_tmp/failover-headers.txt" | tr -d '\r')
	if [ "$served_by" = "$stopped_instance" ]; then
		echo "request was routed to stopped instance $stopped_instance" >&2
		exit 1
	fi
	echo "$served_by"
	request_number=$((request_number + 1))
done
docker start "$stopped_container" >/dev/null
stopped_container=
wait_for_healthy_count land-plot-service 3

echo "11. Scaling report-service to two competing consumers"
docker compose up -d --scale land-plot-service=3 --scale report-service=2
wait_for_healthy_count report-service 2
wait_for_healthy_count land-plot-service 3
wait_for_queue_consumers 2
docker compose ps report-service
competing_plot_ids=
plot_number=1
while [ "$plot_number" -le 6 ]; do
	competing_plot_ids="$competing_plot_ids $(create_plot "LR2 competing consumers plot $plot_number")"
	plot_number=$((plot_number + 1))
done
attempt=0
while [ "$attempt" -lt 20 ] && [ "$(collect_processed_events)" != "6" ]; do
	attempt=$((attempt + 1))
	sleep 1
done
if [ "$(collect_processed_events)" != "6" ]; then
	echo "not every competing consumers event was processed" >&2
	exit 1
fi
echo "   Instance that processed plot.created for each new plot:"
sort "$demo_tmp/competing.txt"
echo "   Events processed per instance:"
cut -d ' ' -f 1 "$demo_tmp/competing.txt" | sort | uniq -c
consumer_count=$(cut -d ' ' -f 1 "$demo_tmp/competing.txt" | sort -u | wc -l | tr -d '[:space:]')
if [ "$consumer_count" -ne 2 ]; then
	echo "events were processed by $consumer_count report-service instances instead of 2" >&2
	exit 1
fi

echo "LR2 demonstration completed successfully"
