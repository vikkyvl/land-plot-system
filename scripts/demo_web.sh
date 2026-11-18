#!/bin/sh

set -eu

if [ -f .env ]; then
	set -a
	. ./.env
	set +a
fi

gateway_url=${GATEWAY_URL:-http://localhost:8080}
consul_url=${CONSUL_URL:-http://localhost:${CONSUL_HTTP_PORT:-8500}}
mailpit_url=${MAILPIT_URL:-http://localhost:${MAILPIT_UI_PORT:-8025}}
token_url=${OIDC_TOKEN_URL:-http://localhost:${KEYCLOAK_PORT:-8180}/realms/land-plot/protocol/openid-connect/token}
realm=${KEYCLOAK_REALM:-land-plot}
demo_user=${DEMO_USER:-taras.melnyk@example.com}
demo_user_password=${TARAS_MELNYK_PASSWORD:-TarasDevPassword123}
allowed_origin=https://cad.kpi.ua
all_scopes='plots:read plots:write reports:read reports:write'
max_preview_requests=300
demo_tmp=$(mktemp -d)

square='[{"lat":50.45021,"lon":30.52011},{"lat":50.45048,"lon":30.52342},{"lat":50.44962,"lon":30.5245},{"lat":50.44871,"lon":30.52388},{"lat":50.44853,"lon":30.52144}]'
square_extended='[{"lat":50.45021,"lon":30.52011},{"lat":50.45048,"lon":30.52342},{"lat":50.44962,"lon":30.5245},{"lat":50.44871,"lon":30.52388},{"lat":50.44853,"lon":30.52144},{"lat":50.44926,"lon":30.51996}]'
hourglass='[{"lat":50.4509,"lon":30.52},{"lat":50.4509,"lon":30.5214},{"lat":50.45,"lon":30.52},{"lat":50.45,"lon":30.5214}]'
broken='[{"lat":50,"lon":30},{"lat":95,"lon":30},{"lat":50,"lon":30}]'

cleanup() {
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

wait_for_healthy() {
	attempt=0
	while [ "$attempt" -lt 90 ]; do
		if [ "$(curl -fsS "$consul_url/v1/health/service/$1?passing=true" 2>/dev/null | jq 'length' 2>/dev/null || echo 0)" -ge 1 ]; then
			return
		fi
		attempt=$((attempt + 1))
		sleep 2
	done
	fail "$1 did not become healthy"
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

preview() {
	request --request POST --header 'Content-Type: application/json' \
		--data "{\"points\":$1}" "$gateway_url/api/v1/plots/preview"
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

expect_body() {
	if ! jq -e "$1" "$demo_tmp/body.json" >/dev/null; then
		fail "$2: $(cat "$demo_tmp/body.json")"
	fi
}

wait_for_status() {
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

create_plot() {
	api POST /plots --data "{\"name\":\"$1\",\"points\":$2}" >/dev/null
	jq -er .id "$demo_tmp/body.json"
}

mail_count() {
	curl -fsS "$mailpit_url/api/v1/messages?limit=1" | jq -r '.total'
}

for command_name in docker curl jq; do
	require_command "$command_name"
done

echo "0. Starting the system with Mailpit and applying the realm settings"
docker compose up -d --build
for service_name in identity-service land-plot-service report-service api-gateway; do
	wait_for_healthy "$service_name"
done
./scripts/keycloak_realm.sh | sed 's/^/   /'
token=$(user_token)

echo
echo "1. Preview of a valid plot without a token"
status=$(preview "$square")
expect_status "$status" 200 "preview of a valid plot"
body '{valid, area_m2: (.area_m2|round), area_ares: (.area_ares*10|round/10), area_hectares: (.area_hectares*100|round/100), perimeter_m: (.perimeter_m|round), errors}' | sed 's/^/   /'
expect_body '.valid and .area_hectares > 0 and .errors == []' "the valid plot was not measured"

echo
echo "2. Preview of a self-intersecting plot: crossing sides and the points to swap (0-based)"
status=$(preview "$hourglass")
expect_status "$status" 200 "preview of a self-intersecting plot"
body '{valid, area_m2, errors}' | sed 's/^/   /'
expect_body '.valid == false and .area_m2 == null and .errors[0].details == {"segments":[[1,2],[3,0]],"swap":[2,3]}' \
	"the self-intersection details are wrong"

echo
echo "3. Preview with a coordinate out of range and a duplicate point"
status=$(preview "$broken")
expect_status "$status" 200 "preview of broken points"
body '[.errors[] | {code, details}]' | sed 's/^/   /'
expect_body '[.errors[].code] == ["invalid_coordinate","duplicate_point"]' "the point problems were not listed"

echo
echo "4. Saving the self-intersecting plot is rejected with the same details"
status=$(api POST /plots --data "{\"name\":\"Пісочний годинник\",\"points\":$hourglass}")
echo "   status $status, $(body '{code, details}')"
expect_status "$status" 422 "POST /plots with a self-intersection"

echo
echo "5. Plot history: a plot, a changed boundary and two confirmed versions"
plot_id=$(create_plot "Поле біля ставка" "$square")
wait_for_status "$plot_id" CONFIRMED
status=$(api PATCH "/plots/$plot_id" --data "{\"points\":$square_extended}")
expect_status "$status" 200 "PATCH /plots/$plot_id"
wait_for_status "$plot_id" CONFIRMED
status=$(api GET "/plots/$plot_id/versions")
expect_status "$status" 200 "GET /plots/$plot_id/versions"
body '.[] | {version, area_hectares: (.area_hectares*100|round/100), perimeter_m: (.perimeter_m|round), points: (.points|length)}' | sed 's/^/   /'
expect_body 'length == 2 and .[0].version > .[1].version' "the history must contain two versions, newest first"

echo
echo "6. Reports: list of the plot, limit and an invalid query"
first_report=$(api POST /reports --data "{\"plot_id\":\"$plot_id\"}" >/dev/null && jq -er .id "$demo_tmp/body.json")
second_report=$(api POST /reports --data "{\"plot_id\":\"$plot_id\"}" >/dev/null && jq -er .id "$demo_tmp/body.json")
status=$(api GET "/reports?plot_id=$plot_id")
expect_status "$status" 200 "GET /reports?plot_id"
body '.[] | {id, plot_version, plot_deleted}' | sed 's/^/   /'
expect_body "[.[].id] == [\"$second_report\", \"$first_report\"]" "the reports must be listed newest first"
status=$(api GET "/reports?limit=1")
echo "   limit=1: $status, $(body '[.[].id]')"
expect_body 'length == 1' "limit=1 returned more than one report"
status=$(api GET "/reports?limit=500")
echo "   limit=500: $status, $(body '{code}')"
expect_status "$status" 400 "GET /reports?limit=500"

echo
echo "7. Deletion: a pending plot, the confirmed plot, 410 afterwards and the marked report"
pending_id=$(create_plot "Ще обробляється" "$square")
status=$(api DELETE "/plots/$pending_id")
echo "   DELETE of a PENDING plot: $status, $(body '{code}')"
expect_status "$status" 409 "DELETE of a pending plot"
status=$(api DELETE "/plots/$plot_id")
echo "   DELETE of the confirmed plot: $status"
expect_status "$status" 204 "DELETE /plots/$plot_id"
for path in "/plots/$plot_id" "/plots/$plot_id/versions"; do
	status=$(api GET "$path")
	echo "   GET $path: $status, $(body '{code}')"
	expect_status "$status" 410 "GET $path after deletion"
done
attempt=0
until api GET "/reports/$first_report" >/dev/null && [ "$(jq -r .plot_deleted "$demo_tmp/body.json")" = true ]; do
	attempt=$((attempt + 1))
	[ "$attempt" -lt 30 ] || fail "the report was not marked after plot.deleted"
	sleep 1
done
echo "   report after plot.deleted: $(body '{id, plot_name, plot_deleted}')"
status=$(api POST /reports --data "{\"plot_id\":\"$plot_id\"}")
echo "   new report for the deleted plot: $status, $(body '{code}')"
expect_status "$status" 410 "POST /reports for a deleted plot"

echo
echo "8. CORS preflight for DELETE from $allowed_origin"
status=$(request --request OPTIONS --header "Origin: $allowed_origin" \
	--header 'Access-Control-Request-Method: DELETE' "$gateway_url/api/v1/plots/$plot_id")
echo "   status $status, Access-Control-Allow-Methods: $(header_value Access-Control-Allow-Methods)"
expect_status "$status" 204 "preflight for DELETE"
header_value Access-Control-Allow-Methods | grep -q DELETE || fail "DELETE is not allowed by CORS"

echo
echo "9. Password reset email from Keycloak arrives in Mailpit ($mailpit_url)"
mails_before=$(mail_count)
docker compose exec -T keycloak bash -c '
	kcadm() { /opt/keycloak/bin/kcadm.sh "$@" --config /tmp/kcadm.config; }
	kcadm config credentials --server http://localhost:8080 --realm master \
		--user "$KC_BOOTSTRAP_ADMIN_USERNAME" --password "$KC_BOOTSTRAP_ADMIN_PASSWORD" >/dev/null 2>&1
	user_id=$(kcadm get users -r '"$realm"' -q email='"$demo_user"' -q exact=true --fields id --format csv --noquotes)
	kcadm update "users/$user_id/execute-actions-email" -r '"$realm"' -b "[\"UPDATE_PASSWORD\"]" >/dev/null 2>&1'
attempt=0
until [ "$(mail_count)" -gt "$mails_before" ]; do
	attempt=$((attempt + 1))
	[ "$attempt" -lt 15 ] || fail "no email arrived in Mailpit"
	sleep 1
done
curl -fsS "$mailpit_url/api/v1/messages?limit=1" | jq -c '.messages[0] | {to: [.To[].Address], subject: .Subject}' | sed 's/^/   /'

echo
echo "10. Preview rate limit on the gateway"
count=0
status=200
while [ "$count" -lt "$max_preview_requests" ] && [ "$status" != 429 ]; do
	status=$(preview "$square")
	count=$((count + 1))
done
expect_status "$status" 429 "preview after $count requests"
echo "   request $count: $status, Retry-After: $(header_value Retry-After), $(body '{code}')"

echo
echo "Web backend demonstration completed successfully"
