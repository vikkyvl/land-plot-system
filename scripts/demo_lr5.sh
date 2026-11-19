#!/bin/sh

set -eu

if [ -f .env ]; then
	set -a
	. ./.env
	set +a
fi

gateway_url=${GATEWAY_URL:-http://localhost:8080}
consul_url=${CONSUL_URL:-http://localhost:${CONSUL_HTTP_PORT:-8500}}
token_url=${OIDC_TOKEN_URL:-http://localhost:${KEYCLOAK_PORT:-8180}/realms/land-plot/protocol/openid-connect/token}
demo_user=${DEMO_USER:-taras.melnyk@example.com}
demo_user_password=${TARAS_MELNYK_PASSWORD:-TarasDevPassword123}
ai_client_id=${AI_CLIENT_ID:-land-plot-ai}
ai_client_secret=${AI_CLIENT_SECRET:-local_development_ai_client_secret}
allowed_origin=https://cad.kpi.ua
foreign_origin=https://evil.example
all_scopes='plots:read plots:write reports:read reports:write'
demo_tmp=$(mktemp -d)

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
		--data-urlencode "scope=openid $1" \
		"$token_url" | jq -er '.access_token'
}

client_token() {
	curl -fsS \
		--data-urlencode grant_type=client_credentials \
		--data-urlencode "client_id=$ai_client_id" \
		--data-urlencode "client_secret=$ai_client_secret" \
		--data-urlencode "scope=$1" \
		"$token_url" | jq -er '.access_token'
}

base64url_decode() {
	segment=$(printf '%s' "$1" | tr '_-' '/+')
	case $((${#segment} % 4)) in
	2) segment="$segment==" ;;
	3) segment="$segment=" ;;
	esac
	printf '%s' "$segment" | base64 -d 2>/dev/null
}

base64url_encode() {
	base64 | tr -d '=\n' | tr '/+' '_-'
}

payload_of() {
	base64url_decode "$(printf '%s' "$1" | cut -d. -f2)"
}

request() {
	curl -sS --dump-header "$demo_tmp/headers.txt" --output "$demo_tmp/body.json" --write-out '%{http_code}' "$@"
}

header_values() {
	tr -d '\r' <"$demo_tmp/headers.txt" | grep -i "^$1:" | cut -d' ' -f2- || true
}

header_count() {
	tr -d '\r' <"$demo_tmp/headers.txt" | grep -ci "^$1:" || true
}

expect_status() {
	if [ "$1" != "$2" ]; then
		fail "$3 returned $1 instead of $2"
	fi
}

for command_name in docker curl jq openssl base64; do
	require_command "$command_name"
done

echo "0. Starting the system with Keycloak"
docker compose up -d --build
for service_name in identity-service land-plot-service report-service api-gateway; do
	wait_for_healthy "$service_name"
done

full_token=$(user_token "$all_scopes")
echo "   Token of $demo_user from Keycloak, decoded payload:"
payload_of "$full_token" | jq '{iss, aud, sub, azp, scope, exp, exp_utc: (.exp | todate)}'

echo
echo "1. Preflight from $allowed_origin"
status=$(request --request OPTIONS --header "Origin: $allowed_origin" \
	--header 'Access-Control-Request-Method: POST' \
	--header 'Access-Control-Request-Headers: authorization, content-type' \
	"$gateway_url/api/v1/plots")
tr -d '\r' <"$demo_tmp/headers.txt" | grep -iE '^(HTTP/|access-control|vary)' | sed 's/^/   /'
expect_status "$status" 204 "preflight from $allowed_origin"
[ "$(header_values Access-Control-Allow-Origin)" = "$allowed_origin" ] || fail "wrong Access-Control-Allow-Origin"
[ "$(header_values Access-Control-Allow-Methods)" = "GET, POST, PATCH, DELETE, OPTIONS" ] || fail "wrong Access-Control-Allow-Methods"

echo
echo "2. Preflight from $foreign_origin"
status=$(request --request OPTIONS --header "Origin: $foreign_origin" \
	--header 'Access-Control-Request-Method: GET' "$gateway_url/api/v1/plots")
tr -d '\r' <"$demo_tmp/headers.txt" | grep -iE '^(HTTP/|access-control|vary)' | sed 's/^/   /'
[ "$(header_count Access-Control-Allow-Origin)" = 0 ] || fail "a foreign origin got Access-Control-Allow-Origin"
expect_status "$status" 403 "preflight from $foreign_origin"

echo
echo "3. GET /plots from $allowed_origin with a token"
status=$(request --header "Origin: $allowed_origin" --header "Authorization: Bearer $full_token" "$gateway_url/api/v1/plots")
expect_status "$status" 200 "GET /plots from $allowed_origin"
echo "   status $status, Access-Control-Allow-Origin headers: $(header_count Access-Control-Allow-Origin)"
[ "$(header_count Access-Control-Allow-Origin)" = 1 ] || fail "the response must carry exactly one Access-Control-Allow-Origin"

echo
echo "4. Request without a token"
status=$(request "$gateway_url/api/v1/plots")
echo "   status $status, WWW-Authenticate: $(header_values WWW-Authenticate), body: $(jq -c '{code}' "$demo_tmp/body.json")"
expect_status "$status" 401 "GET /plots without a token"
[ "$(header_values WWW-Authenticate)" = 'Bearer error="invalid_token"' ] || fail "missing WWW-Authenticate"

echo
echo "5. Forged and HS256 tokens"
token_header=$(printf '%s' "$full_token" | cut -d. -f1)
token_signature=$(printf '%s' "$full_token" | cut -d. -f3)
forged_payload=$(payload_of "$full_token" | jq -c '.scope = "plots:read plots:write reports:read reports:write admin"' | base64url_encode)
status=$(request --header "Authorization: Bearer $token_header.$forged_payload.$token_signature" "$gateway_url/api/v1/plots")
echo "   token with a changed payload: $status"
expect_status "$status" 401 "a token with a changed payload"
hmac_header=$(printf '{"alg":"HS256","typ":"JWT"}' | base64url_encode)
hmac_payload=$(payload_of "$full_token" | base64url_encode)
hmac_signature=$(printf '%s.%s' "$hmac_header" "$hmac_payload" |
	openssl dgst -sha256 -hmac local_development_jwt_secret_change_me -binary | base64url_encode)
status=$(request --header "Authorization: Bearer $hmac_header.$hmac_payload.$hmac_signature" "$gateway_url/api/v1/plots")
echo "   HS256 token signed with the old JWT secret: $status"
expect_status "$status" 401 "an HS256 token"

echo
echo "6. Token with plots:read only"
read_token=$(user_token plots:read)
echo "   scope: $(payload_of "$read_token" | jq -r .scope)"
status=$(request --header "Authorization: Bearer $read_token" "$gateway_url/api/v1/plots")
echo "   GET /plots: $status"
expect_status "$status" 200 "GET /plots with plots:read"
status=$(request --request POST --header "Authorization: Bearer $read_token" --header 'Content-Type: application/json' \
	--data '{"name":"Без права запису","points":[{"lat":50.45,"lon":30.50},{"lat":50.45,"lon":30.51},{"lat":50.46,"lon":30.51}]}' \
	"$gateway_url/api/v1/plots")
echo "   POST /plots: $status, WWW-Authenticate: $(header_values WWW-Authenticate), body: $(jq -c '{code}' "$demo_tmp/body.json")"
expect_status "$status" 403 "POST /plots with plots:read only"

echo
echo "7. Token with all scopes: plot, saga and report"
plot_id=$(curl -fsS --request POST --header "Authorization: Bearer $full_token" --header 'Content-Type: application/json' \
	--data '{"name":"Ділянка ЛР5","points":[{"lat":50.4501,"lon":30.5234},{"lat":50.4501,"lon":30.5250},{"lat":50.4510,"lon":30.5250},{"lat":50.4510,"lon":30.5234}]}' \
	"$gateway_url/api/v1/plots" | jq -er '.id')
attempt=0
plot_status=PENDING
while [ "$attempt" -lt 30 ] && [ "$plot_status" != CONFIRMED ]; do
	sleep 1
	plot_status=$(curl -fsS --header "Authorization: Bearer $full_token" "$gateway_url/api/v1/plots/$plot_id" | jq -r .status)
	attempt=$((attempt + 1))
done
echo "   plot $plot_id is $plot_status"
[ "$plot_status" = CONFIRMED ] || fail "the saga did not confirm the plot"
curl -fsS --request POST --header "Authorization: Bearer $full_token" --header 'Content-Type: application/json' \
	--data "{\"plot_id\":\"$plot_id\"}" "$gateway_url/api/v1/reports" | jq -c '{id, plot_name, area_hectares}' | sed 's/^/   report: /'

echo
echo "8. Direct request to land-plot-service, bypassing the gateway, without a token"
direct=$(docker compose exec -T api-gateway sh -c \
	'wget -q -S -O /dev/null http://land-plot-service:8082/api/v1/plots 2>&1 | grep -m1 "HTTP/"' || true)
echo "  $direct"
printf '%s' "$direct" | grep -q ' 401 ' || fail "land-plot-service accepted a request without a token"

echo
echo "9. Profile from the token, user created in identity_db"
curl -fsS --header "Authorization: Bearer $full_token" "$gateway_url/api/v1/users/profile" | jq -c '{sub, email, name, scopes}' | sed 's/^/   /'
subject=$(payload_of "$full_token" | jq -r .sub)
stored=$(docker compose exec -T postgres psql -U identity_user -d identity_db -Atc "SELECT id || ' ' || coalesce(email, '') FROM users WHERE id = '$subject';")
echo "   identity_db: $stored"
[ -n "$stored" ] || fail "the user was not created in identity_db"

echo
echo "10. Client credentials of the producer agent (plots:read plots:write) cannot create reports"
producer_token=$(client_token 'plots:read plots:write')
echo "   client: $(payload_of "$producer_token" | jq -c '{azp, sub, scope}')"
status=$(request --request POST --header "Authorization: Bearer $producer_token" --header 'Content-Type: application/json' \
	--data "{\"plot_id\":\"$plot_id\"}" "$gateway_url/api/v1/reports")
echo "   POST /reports: $status, WWW-Authenticate: $(header_values WWW-Authenticate)"
expect_status "$status" 403 "POST /reports with producer scopes"

echo
echo "LR5 demonstration completed successfully"
