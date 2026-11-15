#!/bin/sh

set -eu

gateway_url=${GATEWAY_URL:-http://localhost:8080}
consul_url=${CONSUL_URL:-http://localhost:${CONSUL_HTTP_PORT:-8500}}
saga_timeout=${SAGA_TIMEOUT:-30s}
processing_timeout=${MESSAGE_PROCESSING_TIMEOUT:-5s}
demo_password=${DEMO_PASSWORD:-DemoPassword123}
polygon='[{"lat":50.45,"lon":30.50},{"lat":50.45,"lon":30.51},{"lat":50.46,"lon":30.51},{"lat":50.46,"lon":30.50}]'
dead_queue=report.plot-events.dead
stopped_service=
lock_pid=

cleanup() {
	if [ -n "$lock_pid" ]; then
		kill "$lock_pid" >/dev/null 2>&1 || true
	fi
	if [ -n "$stopped_service" ]; then
		docker compose start "$stopped_service" >/dev/null 2>&1 || true
	fi
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

seconds_of() {
	seconds=${1%s}
	if [ "$seconds" = "$1" ] || [ -z "$seconds" ] || [ -n "$(printf '%s' "$seconds" | tr -d '0-9')" ]; then
		echo "duration $1 must be whole seconds, for example 30s" >&2
		exit 1
	fi
	echo "$seconds"
}

plot_sql() {
	docker compose exec -T postgres psql -U plot_user -d plot_db "$@"
}

report_sql() {
	docker compose exec -T postgres psql -U report_user -d report_db "$@"
}

healthy_instances() {
	curl -fsS "$consul_url/v1/health/service/$1?passing=true" | jq 'length'
}

wait_for_healthy() {
	attempt=0
	while [ "$attempt" -lt 60 ]; do
		if [ "$(healthy_instances "$1" 2>/dev/null || echo 0)" -ge 1 ]; then
			return
		fi
		attempt=$((attempt + 1))
		sleep 1
	done
	echo "$1 did not become healthy" >&2
	exit 1
}

wait_for_container_healthy() {
	attempt=0
	while [ "$attempt" -lt 60 ]; do
		if [ "$(docker inspect --format '{{.State.Health.Status}}' "$(docker compose ps -q "$1")")" = healthy ]; then
			return
		fi
		attempt=$((attempt + 1))
		sleep 1
	done
	echo "$1 container did not become healthy" >&2
	exit 1
}

dead_letter_count() {
	docker compose exec -T rabbitmq rabbitmqctl list_queues --quiet --no-table-headers name messages |
		awk -v queue="$dead_queue" '$1 == queue { print $2 }'
}

create_plot() {
	curl -fsS \
		--request POST \
		--header "Authorization: Bearer $token" \
		--header 'Content-Type: application/json' \
		--data "{\"name\":\"$1\",\"points\":$polygon}" \
		"$gateway_url/api/v1/plots"
}

stored_status() {
	plot_sql -Atc "SELECT status FROM plots WHERE id = '$1';" | tr -d '[:space:]'
}

wait_for_stored_status() {
	plot=$1
	expected=$2
	attempts=$3
	attempt=0
	while [ "$attempt" -lt "$attempts" ]; do
		if [ "$(stored_status "$plot")" = "$expected" ]; then
			return
		fi
		attempt=$((attempt + 1))
		sleep 1
	done
	echo "plot $plot did not reach status $expected" >&2
	exit 1
}

require_stored_status() {
	actual=$(stored_status "$1")
	if [ "$actual" != "$2" ]; then
		echo "plot $1 has status $actual instead of $2" >&2
		exit 1
	fi
}

get_plot_status_code() {
	curl -sS \
		--output /dev/null \
		--write-out '%{http_code}' \
		--header "Authorization: Bearer $token" \
		"$gateway_url/api/v1/plots/$1"
}

show_saga() {
	plot_sql -c "SELECT plot_id, kind, status, reason, new_version FROM plot_sagas WHERE plot_id = '$1' ORDER BY created_at;"
}

for command_name in docker curl jq awk; do
	require_command "$command_name"
done

saga_timeout_seconds=$(seconds_of "$saga_timeout")
processing_timeout_seconds=$(seconds_of "$processing_timeout")
lock_seconds=$((processing_timeout_seconds + 3))
if [ "$lock_seconds" -gt $((saga_timeout_seconds - 10)) ]; then
	echo "SAGA_TIMEOUT must exceed MESSAGE_PROCESSING_TIMEOUT by at least 13 seconds" >&2
	exit 1
fi

echo "0. Starting all components"
docker compose up -d --build
for service_name in identity-service land-plot-service report-service api-gateway; do
	wait_for_healthy "$service_name"
done

echo "1. Registering a user and receiving a JWT"
demo_email="lr3-demo-$(date +%s)-$$@example.com"
credentials="{\"email\":\"$demo_email\",\"password\":\"$demo_password\"}"
curl -fsS --request POST --header 'Content-Type: application/json' --data "$credentials" \
	"$gateway_url/api/v1/auth/register" >/dev/null
token=$(curl -fsS --request POST --header 'Content-Type: application/json' --data "$credentials" \
	"$gateway_url/api/v1/auth/login" | jq -er '.access_token')

echo "2. Successful saga: the plot is PENDING and becomes CONFIRMED"
plot_response=$(create_plot "Поле 1")
confirmed_plot=$(printf '%s' "$plot_response" | jq -er '.id')
printf '%s' "$plot_response" | jq -e '.status == "PENDING"' >/dev/null
printf '%s' "$plot_response" | jq '{id,name,version,status}'
wait_for_stored_status "$confirmed_plot" CONFIRMED 20
curl -fsS --header "Authorization: Bearer $token" "$gateway_url/api/v1/plots/$confirmed_plot" |
	jq '{id,name,version,status}'

echo "3. Outbox rows are published and the saga is COMPLETED"
plot_sql -c "SELECT id, routing_key, published_at IS NOT NULL AS published, attempts FROM outbox_events WHERE payload->>'plot_id' = '$confirmed_plot' ORDER BY position;"
show_saga "$confirmed_plot"

echo "4. Write failure in report-service: compensation by the DLQ message"
failed_plot=$(create_plot "FAIL поле" | jq -er '.id')
wait_for_stored_status "$failed_plot" CANCELLED 20
echo "   GET after compensation returned $(get_plot_status_code "$failed_plot")"
show_saga "$failed_plot"
docker compose logs --no-color report-service | grep '"msg":"process plot event"' | grep "$failed_plot" | tail -1

echo "5. The rejected event is in the dead letter queue"
echo "   $dead_queue messages: $(dead_letter_count)"

echo "6. report-service is stopped: compensation by timeout"
docker compose stop report-service
stopped_service=report-service
timeout_plot=$(create_plot "Поле без відповіді" | jq -er '.id')
echo "   Waiting up to $((saga_timeout_seconds + 15)) seconds for the saga deadline"
wait_for_stored_status "$timeout_plot" CANCELLED $((saga_timeout_seconds + 15))
show_saga "$timeout_plot"
docker compose start report-service
stopped_service=
wait_for_healthy report-service

echo "7. Update with a failing name is compensated with the previous data"
curl -fsS \
	--request PATCH \
	--header "Authorization: Bearer $token" \
	--header 'Content-Type: application/json' \
	--data '{"name":"FAIL нова"}' \
	"$gateway_url/api/v1/plots/$confirmed_plot" |
	jq '{name,version,status}'
attempt=0
while [ "$attempt" -lt 20 ] && [ "$(plot_sql -Atc "SELECT version FROM plots WHERE id = '$confirmed_plot';" | tr -d '[:space:]')" != "3" ]; do
	attempt=$((attempt + 1))
	sleep 1
done
curl -fsS --header "Authorization: Bearer $token" "$gateway_url/api/v1/plots/$confirmed_plot" |
	jq -e '.name == "Поле 1" and .version == 3 and .status == "CONFIRMED"'
show_saga "$confirmed_plot"

echo "8. RabbitMQ is stopped: the event waits in the outbox"
step_started=$(date +%s)
docker compose stop rabbitmq
stopped_service=rabbitmq
outbox_plot_response=$(create_plot "Поле під час збою брокера")
outbox_plot=$(printf '%s' "$outbox_plot_response" | jq -er '.id')
printf '%s' "$outbox_plot_response" | jq '{id,status}'
sleep 8
require_stored_status "$outbox_plot" PENDING
plot_sql -c "SELECT id, routing_key, attempts, published_at, left(last_error, 60) AS last_error FROM outbox_events ORDER BY position DESC LIMIT 3;"
waiting=$(plot_sql -Atc "SELECT count(*) FROM outbox_events WHERE payload->>'plot_id' = '$outbox_plot' AND published_at IS NULL;" | tr -d '[:space:]')
if [ "$waiting" != "1" ]; then
	echo "the plot.created event is not waiting in the outbox" >&2
	exit 1
fi
docker compose start rabbitmq
stopped_service=
wait_for_container_healthy rabbitmq
elapsed=$(($(date +%s) - step_started))
echo "   RabbitMQ is healthy again after $elapsed seconds"
if [ "$elapsed" -gt $((saga_timeout_seconds - 5)) ]; then
	echo "RabbitMQ стартував довше за таймаут саги, збільште SAGA_TIMEOUT" >&2
	exit 1
fi
wait_for_stored_status "$outbox_plot" CONFIRMED 30
plot_sql -c "SELECT routing_key, published_at IS NOT NULL AS published, attempts FROM outbox_events WHERE payload->>'plot_id' = '$outbox_plot';"
report_sql -c "SELECT plot_id, name, version FROM plot_summaries WHERE plot_id = '$outbox_plot';"
echo "   Plot status: $(stored_status "$outbox_plot")"

echo "9. Temporary report_db failure: the event is requeued and the saga is not cancelled"
dead_before=$(dead_letter_count)
echo "   Locking plot_summaries for $lock_seconds seconds (MESSAGE_PROCESSING_TIMEOUT is $processing_timeout)"
report_sql -c "BEGIN; LOCK TABLE plot_summaries IN ACCESS EXCLUSIVE MODE; SELECT pg_sleep($lock_seconds); COMMIT;" >/dev/null &
lock_pid=$!
sleep 1
locked_plot=$(create_plot "Поле під час блокування" | jq -er '.id')
wait "$lock_pid"
lock_pid=
wait_for_stored_status "$locked_plot" CONFIRMED $((saga_timeout_seconds + 5))
docker compose logs --no-color report-service | grep '"msg":"requeue plot event after temporary failure"' | grep "$locked_plot" | tail -2
dead_after=$(dead_letter_count)
echo "   $dead_queue messages before: $dead_before, after: $dead_after"
if [ "$dead_after" != "$dead_before" ]; then
	echo "a temporary failure was sent to the dead letter queue" >&2
	exit 1
fi
show_saga "$locked_plot"
echo "   Plot status: $(stored_status "$locked_plot")"

echo "10. No outbox row was abandoned"
abandoned=$(($(plot_sql -Atc 'SELECT count(*) FROM outbox_events WHERE failed_at IS NOT NULL;' | tr -d '[:space:]') +
	$(report_sql -Atc 'SELECT count(*) FROM outbox_events WHERE failed_at IS NOT NULL;' | tr -d '[:space:]')))
echo "   outbox_events with failed_at in plot_db and report_db: $abandoned"
if [ "$abandoned" -ne 0 ]; then
	echo "outbox rows were abandoned" >&2
	exit 1
fi

echo "LR3 demonstration completed successfully"
