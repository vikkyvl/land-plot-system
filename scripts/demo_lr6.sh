#!/bin/sh

set -eu

if [ -f .env ]; then
	set -a
	. ./.env
	set +a
fi

gateway_url=${GATEWAY_URL:-http://localhost:8080}
prometheus_url=${PROMETHEUS_URL:-http://localhost:${PROMETHEUS_PORT:-19090}}
jaeger_url=${JAEGER_URL:-http://localhost:${JAEGER_UI_PORT:-16686}}
grafana_url=${GRAFANA_URL:-http://localhost:${GRAFANA_PORT:-13000}}
mailpit_url=${MAILPIT_URL:-http://localhost:${MAILPIT_UI_PORT:-8025}}
n8n_url=${N8N_URL:-http://localhost:${N8N_PORT:-5678}}
token_url=${OIDC_TOKEN_URL:-http://localhost:${KEYCLOAK_PORT:-8180}/realms/land-plot/protocol/openid-connect/token}
demo_user=${DEMO_USER:-taras.melnyk@example.com}
demo_user_password=${TARAS_MELNYK_PASSWORD:-TarasDevPassword123}
all_scopes='plots:read plots:write reports:read reports:write'
dashboard_url=$grafana_url/d/land-plot-overview/land-plot-system
blackhole_open_meteo=http://10.255.255.1/v1
alert_wait_seconds=${DEMO_ALERT_WAIT_SECONDS:-480}
ticket_wait_seconds=${DEMO_TICKET_WAIT_SECONDS:-180}
latency_wait_seconds=${DEMO_LATENCY_WAIT_SECONDS:-720}
slow_report_pause_seconds=15
plots_during_broker_outage=3
demo_tmp=$(mktemp -d)

kyiv_square='[{"lat":50.45021,"lon":30.52011},{"lat":50.45048,"lon":30.52342},{"lat":50.44962,"lon":30.5245},{"lat":50.44871,"lon":30.52388},{"lat":50.44853,"lon":30.52144}]'
hourglass='[{"lat":50.4509,"lon":30.52},{"lat":50.4509,"lon":30.5214},{"lat":50.45,"lon":30.52},{"lat":50.45,"lon":30.5214}]'
small_triangle='[{"lat":50.45,"lon":30.52},{"lat":50.451,"lon":30.52},{"lat":50.451,"lon":30.522}]'

report_service_changed=false
rabbitmq_stopped=false

restore_system() {
	if [ "$rabbitmq_stopped" = true ]; then
		echo "   restoring RabbitMQ"
		docker compose start rabbitmq >/dev/null 2>&1 || true
		rabbitmq_stopped=false
	fi
	if [ "$report_service_changed" = true ]; then
		echo "   restoring report-service with its normal configuration"
		docker compose up -d report-service >/dev/null 2>&1 || true
		report_service_changed=false
	fi
}

cleanup() {
	status=$?
	if [ "$status" -ne 0 ]; then
		echo
		echo "The demo stopped with status $status, returning the system to its normal state"
	fi
	restore_system
	rm -rf "$demo_tmp"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

fail() {
	echo "$1" >&2
	exit 1
}

require_command() {
	if ! command -v "$1" >/dev/null 2>&1; then
		fail "required command is missing: $1"
	fi
}

user_token() {
	curl -fsS \
		--data-urlencode grant_type=password \
		--data-urlencode client_id=land-plot-dev-cli \
		--data-urlencode "username=$demo_user" \
		--data-urlencode "password=$demo_user_password" \
		--data-urlencode "scope=openid $all_scopes" \
		"$token_url" | jq -er '.access_token'
}

request() {
	curl -sS --dump-header "$demo_tmp/headers.txt" --output "$demo_tmp/body.json" --write-out '%{http_code}' "$@"
}

api() {
	method=$1
	path=$2
	shift 2
	request --request "$method" --header "Authorization: Bearer $token" --header 'Content-Type: application/json' \
		"$@" "$gateway_url/api/v1$path"
}

body() {
	jq -c "$1" "$demo_tmp/body.json"
}

header_value() {
	tr -d '\r' <"$demo_tmp/headers.txt" | grep -i "^$1:" | cut -d' ' -f2- || true
}

expect_status() {
	if [ "$1" != "$2" ]; then
		fail "$3 returned $1 instead of $2: $(cat "$demo_tmp/body.json" 2>/dev/null)"
	fi
}

preview() {
	while :; do
		status=$(request --request POST --header 'Content-Type: application/json' \
			--data "{\"points\":$1}" "$gateway_url/api/v1/plots/preview")
		if [ "$status" != 429 ]; then
			echo "$status"
			return
		fi
		retry_after=$(header_value Retry-After | tr -dc '0-9')
		sleep "${retry_after:-1}"
	done
}

create_plot() {
	status=$(api POST /plots --data "{\"name\":\"$1\",\"points\":$2}")
	expect_status "$status" 201 "POST /plots $1"
	jq -er .id "$demo_tmp/body.json"
}

wait_for_plot_status() {
	attempt=0
	while [ "$attempt" -lt 30 ]; do
		api GET "/plots/$1" >/dev/null
		if [ "$(jq -r .status "$demo_tmp/body.json")" = "$2" ]; then
			return
		fi
		attempt=$((attempt + 1))
		sleep 1
	done
	fail "plot $1 did not become $2"
}

wait_for_container_healthy() {
	attempt=0
	while [ "$attempt" -lt 90 ]; do
		if [ "$(docker compose ps "$1" --format '{{.Health}}' 2>/dev/null)" = healthy ]; then
			return
		fi
		attempt=$((attempt + 1))
		sleep 2
	done
	fail "$1 did not become healthy"
}

prom_value() {
	curl -fsS --get --data-urlencode "query=$1" "$prometheus_url/api/v1/query" |
		jq -r '[.data.result[].value[1] | tonumber] | if length == 0 then "none" else (max * 100 | round / 100 | tostring) end'
}

alert_state() {
	curl -fsS "$prometheus_url/api/v1/alerts" |
		jq -r --arg name "$1" --arg service "$2" \
			'[.data.alerts[] | select(.labels.alertname == $name and .labels.service == $service) | .state] | if length == 0 then "inactive" else .[0] end'
}

wait_for_alert() {
	name=$1
	service=$2
	expected=$3
	limit=$4
	elapsed=0
	last=""
	while :; do
		state=$(alert_state "$name" "$service")
		if [ "$state" != "$last" ]; then
			echo "   $(date +%H:%M:%S) $name{service=$service}: $state"
			last=$state
		fi
		if [ "$state" = "$expected" ]; then
			return
		fi
		[ "$elapsed" -lt "$limit" ] || fail "$name for $service did not become $expected within $limit s"
		sleep 5
		elapsed=$((elapsed + 5))
	done
}

mark_mails() {
	curl -fsS "$mailpit_url/api/v1/messages?limit=200" | jq -r '.messages[].ID' >"$demo_tmp/mails_$1"
}

new_mail_subjects() {
	curl -fsS "$mailpit_url/api/v1/messages?limit=200" |
		jq -r --rawfile seen "$demo_tmp/mails_$1" \
			'($seen | split("\n")) as $old | .messages | reverse[] | select(.ID as $id | $old | index($id) | not) | .Subject'
}

wait_for_mail() {
	marker=$1
	pattern=$2
	elapsed=0
	while :; do
		subject=$(new_mail_subjects "$marker" | grep -E "$pattern" | head -n 1 || true)
		if [ -n "$subject" ]; then
			echo "$subject"
			return
		fi
		[ "$elapsed" -lt "$ticket_wait_seconds" ] || fail "no mail matching «$pattern» arrived in Mailpit"
		sleep 5
		elapsed=$((elapsed + 5))
	done
}

ticket_key() {
	echo "$1" | sed -nE 's/.*\(([A-Z][A-Z0-9]*-[0-9]+)\)$/\1/p'
}

jira_issue() {
	if [ -z "${JIRA_BASE_URL:-}" ] || [ -z "${JIRA_EMAIL:-}" ] || [ -z "${JIRA_API_TOKEN:-}" ]; then
		return 1
	fi
	printf 'user = "%s:%s"\n' "$JIRA_EMAIL" "$JIRA_API_TOKEN" |
		curl -fsS -K - "${JIRA_BASE_URL%/}/rest/api/3/issue/$1?fields=status,priority,labels,comment"
}

show_ticket() {
	if ! jira_issue "$1" >"$demo_tmp/issue.json" 2>/dev/null; then
		echo "   $1: Jira was not queried (JIRA_* not set)"
		return
	fi
	jq -r '"   \(.key): status \(.fields.status.name), priority \(.fields.priority.name), labels \(.fields.labels | join(" "))"' "$demo_tmp/issue.json"
	jq -r '.fields.comment.comments[]? | [.. | .text? // empty] | join("") | "   comment: \(.)"' "$demo_tmp/issue.json"
}

wait_for_done() {
	elapsed=0
	while :; do
		if jira_issue "$1" >"$demo_tmp/issue.json" 2>/dev/null; then
			if [ "$(jq -r '.fields.status.statusCategory.key' "$demo_tmp/issue.json")" = done ]; then
				show_ticket "$1"
				return
			fi
		else
			echo "   $1: Jira was not queried (JIRA_* not set)"
			return
		fi
		[ "$elapsed" -lt "$ticket_wait_seconds" ] || fail "$1 was not moved to Done"
		sleep 5
		elapsed=$((elapsed + 5))
	done
}

expect_comment() {
	jq -e --arg text "$2" 'any(.fields.comment.comments[]?; [.. | .text? // empty] | join("") | contains($text))' \
		"$demo_tmp/issue.json" >/dev/null || fail "$1 has no comment with «$2»"
}

loki_query() {
	encoded=$(printf '%s' "$1" | jq -sRr @uri)
	docker compose exec -T grafana wget -qO- \
		"http://loki:3100/loki/api/v1/query_range?query=$encoded&limit=200&since=15m" 2>/dev/null
}

tickets=""

record_ticket() {
	tickets="$tickets $1"
}

for command_name in docker curl jq; do
	require_command "$command_name"
done

echo "1. Starting the system with the observability stack; every Prometheus target must be up"
docker compose --profile observability up -d --build --wait >/dev/null
attempt=0
until ./scripts/observability_check.sh >"$demo_tmp/targets.txt" 2>&1; do
	attempt=$((attempt + 1))
	[ "$attempt" -lt 12 ] || { sed 's/^/   /' "$demo_tmp/targets.txt"; fail "not every Prometheus target is up"; }
	sleep 5
done
sed 's/^/   /' "$demo_tmp/targets.txt"
echo "   firing alerts before the demo: $(curl -fsS "$prometheus_url/api/v1/alerts" | jq -r '[.data.alerts[] | select(.state == "firing") | .labels.alertname + "{" + .labels.service + "}"] | if length == 0 then "none" else join(", ") end')"
token=$(user_token)
mark_mails demo

echo
echo "2. Load: previews, plots and one report (the only report of the demo that calls Open-Meteo)"
valid=0
invalid=0
for index in 1 2 3 4 5 6 7 8 9 10; do
	if [ $((index % 3)) -eq 0 ]; then
		status=$(preview "$hourglass")
		invalid=$((invalid + 1))
	else
		status=$(preview "$kyiv_square")
		valid=$((valid + 1))
	fi
	expect_status "$status" 200 "preview $index"
done
echo "   previews: $valid valid, $invalid self-intersecting"
load_plot=""
for name in "Поле біля ставка" "Ділянка біля лісу" "Город за школою" "Сад над річкою"; do
	load_plot=$(create_plot "$name" "$kyiv_square")
done
wait_for_plot_status "$load_plot" CONFIRMED
echo "   plots: 4 created, the last one $load_plot CONFIRMED"
status=$(api POST /reports --data "{\"plot_id\":\"$load_plot\"}")
expect_status "$status" 201 "report for the load plot"
echo "   report: $status, $(body '{terrain: .terrain.status, climate: .climate.status}')"

echo
echo "3. One request across the services: trace_id in the response, in the logs of three services and in Jaeger"
status=$(api POST /plots --data "{\"name\":\"Трасування запиту\",\"points\":$small_triangle}")
expect_status "$status" 201 "POST /plots for the trace"
trace_id=$(header_value X-Trace-ID)
request_id=$(header_value X-Request-ID)
trace_plot=$(jq -er .id "$demo_tmp/body.json")
[ -n "$trace_id" ] || fail "the response has no X-Trace-ID"
echo "   POST /api/v1/plots: $status, X-Trace-ID $trace_id, X-Request-ID $request_id"
wait_for_plot_status "$trace_plot" CONFIRMED
attempt=0
while :; do
	loki_query "{service=~\".+\"} | trace_id=\"$trace_id\"" >"$demo_tmp/logs.json" || true
	services=$(jq -r '[.data.result[]?.stream.service] | unique | join(" ")' "$demo_tmp/logs.json" 2>/dev/null || true)
	found=0
	for service_name in api-gateway land-plot-service report-service; do
		case " $services " in
		*" $service_name "*) found=$((found + 1)) ;;
		esac
	done
	[ "$found" -lt 3 ] || break
	attempt=$((attempt + 1))
	[ "$attempt" -lt 24 ] || fail "Loki has no lines of all three services for $trace_id (found: $services)"
	sleep 5
done
jq -r '[.data.result[] | .stream.service as $service | .values[] | {time: .[0], service: $service, line: (.[1] | fromjson? // {msg: .})}]
	| sort_by(.time)[] | "   \(.service) | \(.line.msg)\(if .line.route then " " + .line.route else "" end)\(if .line.status then " " + (.line.status | tostring) else "" end)"' \
	"$demo_tmp/logs.json" | head -n 12
echo "   Loki: lines of $services with trace_id $trace_id"
attempt=0
while :; do
	curl -s "$jaeger_url/api/traces/$trace_id" >"$demo_tmp/trace.json" || true
	http_spans=$(jq '[.data[0].spans[]? | select(any(.tags[]; .key | startswith("http.")))] | length' "$demo_tmp/trace.json" 2>/dev/null || echo 0)
	db_spans=$(jq '[.data[0].spans[]? | select(any(.tags[]; .key | startswith("db.")))] | length' "$demo_tmp/trace.json" 2>/dev/null || echo 0)
	messaging_spans=$(jq '[.data[0].spans[]? | select(any(.tags[]; .key | startswith("messaging.")))] | length' "$demo_tmp/trace.json" 2>/dev/null || echo 0)
	traced_services=$(jq '[.data[0].processes[]?.serviceName] | unique | map(select(. == "api-gateway" or . == "land-plot-service" or . == "report-service")) | length' "$demo_tmp/trace.json" 2>/dev/null || echo 0)
	if [ "$traced_services" -eq 3 ] && [ "$http_spans" -ge 3 ] && [ "$db_spans" -gt 0 ] && [ "$messaging_spans" -ge 4 ]; then
		break
	fi
	attempt=$((attempt + 1))
	[ "$attempt" -lt 30 ] || fail "the trace $trace_id in Jaeger misses HTTP, database or RabbitMQ spans"
	sleep 2
done
echo "   Jaeger: $(jq '.data[0].spans | length' "$demo_tmp/trace.json") spans of $(jq -r '[.data[0].processes[].serviceName] | unique | join(", ")' "$demo_tmp/trace.json"): $http_spans HTTP, $db_spans PostgreSQL, $messaging_spans RabbitMQ"
jq -r '.data[0] as $t | $t.spans | sort_by(.startTime)[]
	| select(.operationName | test("^(POST|GET|publish|process) "))
	| "   \($t.processes[.processID].serviceName) | \(.operationName)"' "$demo_tmp/trace.json"

echo
echo "4. ServiceDown: report-service is stopped, a ticket is opened, the service starts again and the ticket is closed"
mark_mails service_down
report_service_changed=true
docker compose stop report-service >/dev/null 2>&1
echo "   $(date +%H:%M:%S) report-service stopped"
wait_for_alert ServiceDown report-service firing "$alert_wait_seconds"
subject=$(wait_for_mail service_down '^\[critical\] ServiceDown — report-service \(')
key=$(ticket_key "$subject")
[ -n "$key" ] || fail "the ServiceDown mail has no ticket: $subject"
record_ticket "$key"
echo "   mail: $subject"
show_ticket "$key"
docker compose start report-service >/dev/null 2>&1
wait_for_container_healthy report-service
report_service_changed=false
echo "   $(date +%H:%M:%S) report-service started"
wait_for_alert ServiceDown report-service inactive "$alert_wait_seconds"
echo "   mail: $(wait_for_mail service_down "^Resolved: ServiceDown — report-service \\($key\\)")"
wait_for_done "$key"

echo
echo "5. ServiceUnhealthy: /health of report-service fails, autoheal restarts it and the ticket names autoheal"
mark_mails unhealthy
report_service_changed=true
DEMO_FAULTS_ENABLED=true docker compose up -d report-service >/dev/null 2>&1
wait_for_container_healthy report-service
started_before=$(docker inspect --format '{{.State.StartedAt}}' "$(docker compose ps -q report-service)")
docker compose exec -T report-service wget -qO- --post-data= http://127.0.0.1:9464/debug/unhealthy >/dev/null
echo "   $(date +%H:%M:%S) POST /debug/unhealthy on the internal port: /health answers $(docker compose exec -T report-service wget -S -qO- http://127.0.0.1:8083/health 2>&1 | awk '/HTTP\//{print $2; exit}')"
wait_for_alert ServiceUnhealthy report-service firing "$alert_wait_seconds"
attempt=0
until [ "$(docker inspect --format '{{.State.StartedAt}}' "$(docker compose ps -q report-service)")" != "$started_before" ]; do
	attempt=$((attempt + 1))
	[ "$attempt" -lt 60 ] || fail "autoheal did not restart report-service"
	sleep 2
done
echo "   $(date +%H:%M:%S) autoheal: $(docker compose --profile observability logs --since 5m autoheal 2>/dev/null | grep report-service | tail -n 1 | sed 's/^[^|]*| //')"
wait_for_container_healthy report-service
subject=$(wait_for_mail unhealthy '^\[warning\] ServiceUnhealthy — report-service \(')
key=$(ticket_key "$subject")
[ -n "$key" ] || fail "the ServiceUnhealthy mail has no ticket: $subject"
record_ticket "$key"
echo "   mail: $subject"
wait_for_alert ServiceUnhealthy report-service inactive "$alert_wait_seconds"
echo "   mail: $(wait_for_mail unhealthy "^Resolved: ServiceUnhealthy — report-service \\($key\\)")"
wait_for_done "$key"
if [ -f "$demo_tmp/issue.json" ] && jq -e .key "$demo_tmp/issue.json" >/dev/null 2>&1; then
	expect_comment "$key" "autoheal restarted the container"
fi
docker compose up -d report-service >/dev/null 2>&1
wait_for_container_healthy report-service
report_service_changed=false
echo "   report-service is back without DEMO_FAULTS_ENABLED"

echo
echo "6. ReplicationLagHigh: RabbitMQ is stopped, the outbox grows, RabbitMQ starts and the backlog is published"
mark_mails lag
rabbitmq_stopped=true
docker compose stop rabbitmq >/dev/null 2>&1
echo "   $(date +%H:%M:%S) RabbitMQ stopped"
index=0
while [ "$index" -lt "$plots_during_broker_outage" ]; do
	index=$((index + 1))
	create_plot "Без брокера $index" "$small_triangle" >/dev/null
done
echo "   $plots_during_broker_outage plots created while the broker is down"
elapsed=0
while [ "$(alert_state ReplicationLagHigh land-plot-service)" != firing ]; do
	echo "   $(date +%H:%M:%S) outbox pending $(prom_value 'service:outbox_pending:max'), oldest $(prom_value 'max(service:outbox_oldest_age_seconds:max)') s, ReplicationLagHigh $(alert_state ReplicationLagHigh land-plot-service)"
	[ "$elapsed" -lt "$alert_wait_seconds" ] || fail "ReplicationLagHigh did not fire"
	sleep 15
	elapsed=$((elapsed + 15))
done
echo "   $(date +%H:%M:%S) ReplicationLagHigh{service=land-plot-service}: firing"
subject=$(wait_for_mail lag '^\[warning\] ReplicationLagHigh — land-plot-service \(')
key=$(ticket_key "$subject")
[ -n "$key" ] || fail "the ReplicationLagHigh mail has no ticket: $subject"
record_ticket "$key"
echo "   mail: $subject"
docker compose start rabbitmq >/dev/null 2>&1
wait_for_container_healthy rabbitmq
rabbitmq_stopped=false
echo "   $(date +%H:%M:%S) RabbitMQ started"
elapsed=0
until [ "$(prom_value 'max(service:outbox_pending:max)')" = 0 ]; do
	[ "$elapsed" -lt "$alert_wait_seconds" ] || fail "the outbox was not published after RabbitMQ came back"
	sleep 5
	elapsed=$((elapsed + 5))
done
echo "   $(date +%H:%M:%S) outbox published, pending 0"
wait_for_alert ReplicationLagHigh land-plot-service inactive "$alert_wait_seconds"
echo "   mail: $(wait_for_mail lag "^Resolved: ReplicationLagHigh — land-plot-service \\($key\\)")"
wait_for_done "$key"
for extra in $(new_mail_subjects lag | grep -E '^\[' | grep -v ReplicationLagHigh | sed -nE 's/.*\(([A-Z][A-Z0-9]*-[0-9]+)\)$/\1/p'); do
	record_ticket "$extra"
done
new_mail_subjects lag | grep -E '^\[' | grep -v ReplicationLagHigh | sed 's/^/   also: /' || true

echo
echo "7. HighLatencyP99: Open-Meteo hangs on an unroutable address, every report waits for the 3 s timeout"
mark_mails latency
report_service_changed=true
OPEN_METEO_ELEVATION_URL=$blackhole_open_meteo/elevation OPEN_METEO_ARCHIVE_URL=$blackhole_open_meteo/archive \
	docker compose up -d report-service >/dev/null 2>&1
wait_for_container_healthy report-service
attempt=0
until [ "$(docker compose exec -T report-service printenv OPEN_METEO_ARCHIVE_URL 2>/dev/null)" = "$blackhole_open_meteo/archive" ]; do
	attempt=$((attempt + 1))
	[ "$attempt" -lt 30 ] || fail "report-service did not restart with the unroutable Open-Meteo address"
	sleep 2
done
echo "   $(date +%H:%M:%S) report-service uses $blackhole_open_meteo"
elapsed=0
slow_reports=0
while [ "$(alert_state HighLatencyP99 report-service)" != firing ]; do
	status=$(api POST /reports --data "{\"plot_id\":\"$load_plot\"}")
	expect_status "$status" 201 "slow report"
	slow_reports=$((slow_reports + 1))
	if [ $((slow_reports % 4)) -eq 1 ]; then
		echo "   $(date +%H:%M:%S) report $slow_reports: $(body '{terrain: .terrain.status, climate: .climate.status}'), p99 report-service $(prom_value 'service:http_server_request_duration_seconds:p99_5m{service="report-service"}') s, HighLatencyP99 $(alert_state HighLatencyP99 report-service)"
	fi
	[ "$elapsed" -lt "$latency_wait_seconds" ] || fail "HighLatencyP99 did not fire"
	sleep "$slow_report_pause_seconds"
	elapsed=$((elapsed + slow_report_pause_seconds + 3))
done
echo "   $(date +%H:%M:%S) HighLatencyP99{service=report-service}: firing after $slow_reports slow reports"
subject=$(wait_for_mail latency '^\[warning\] HighLatencyP99 — report-service \(')
key=$(ticket_key "$subject")
[ -n "$key" ] || fail "the HighLatencyP99 mail has no ticket: $subject"
record_ticket "$key"
echo "   mail: $subject"
show_ticket "$key"
docker compose up -d report-service >/dev/null 2>&1
wait_for_container_healthy report-service
report_service_changed=false
echo "   $(date +%H:%M:%S) report-service is back with the real Open-Meteo endpoints; the alert resolves once the slow reports leave the 5-minute window"
wait_for_alert HighLatencyP99 report-service inactive "$alert_wait_seconds"
echo "   mail: $(wait_for_mail latency "^Resolved: HighLatencyP99 — report-service \\($key\\)")"
wait_for_done "$key"
for extra in $(new_mail_subjects latency | grep -E '^\[' | grep -v 'HighLatencyP99 — report-service' | sed -nE 's/.*\(([A-Z][A-Z0-9]*-[0-9]+)\)$/\1/p'); do
	record_ticket "$extra"
done
new_mail_subjects latency | grep -E '^\[' | grep -v 'HighLatencyP99 — report-service' | sed 's/^/   also: /' || true

echo
echo "8. Where to look"
echo "   dashboard: $dashboard_url"
echo "   trace of step 3: $jaeger_url/trace/$trace_id"
echo "   n8n executions: $n8n_url (workflow «Incident response»)"
echo "   Mailpit: $mailpit_url"
echo "   Jira tickets of this demo:$tickets"
echo "   still firing: $(curl -fsS "$prometheus_url/api/v1/alerts" | jq -r '[.data.alerts[] | select(.state == "firing") | .labels.alertname + "{" + .labels.service + "}"] | if length == 0 then "none" else join(", ") + " (they resolve within 5 minutes, once the slow calls leave the window)" end')"

echo
echo "Observability demonstration completed successfully"
