#!/bin/sh

set -eu

if [ -f .env ]; then
	set -a
	. ./.env
	set +a
fi

realm=${KEYCLOAK_REALM:-land-plot}
alias=google

if [ -z "${GOOGLE_CLIENT_ID:-}" ] || [ -z "${GOOGLE_CLIENT_SECRET:-}" ]; then
	echo "warning: GOOGLE_CLIENT_ID or GOOGLE_CLIENT_SECRET is not set; the Google login is not configured" >&2
	exit 0
fi

export GOOGLE_CLIENT_ID GOOGLE_CLIENT_SECRET

kcadm() {
	docker compose exec -T -e GOOGLE_CLIENT_ID -e GOOGLE_CLIENT_SECRET keycloak \
		bash -c '/opt/keycloak/bin/kcadm.sh "$@" --config /tmp/kcadm.config' kcadm "$@"
}

docker compose exec -T keycloak bash -c '/opt/keycloak/bin/kcadm.sh config credentials \
	--server http://localhost:8080 --realm master \
	--user "$KC_BOOTSTRAP_ADMIN_USERNAME" --password "$KC_BOOTSTRAP_ADMIN_PASSWORD" \
	--config /tmp/kcadm.config' >/dev/null

provider_settings() {
	docker compose exec -T -e GOOGLE_CLIENT_ID -e GOOGLE_CLIENT_SECRET keycloak bash -c '
		/opt/keycloak/bin/kcadm.sh '"$1"' --config /tmp/kcadm.config -r '"$realm"' \
			-s alias='"$alias"' -s providerId=google -s enabled=true -s displayName=Google \
			-s trustEmail=true -s storeToken=false -s firstBrokerLoginFlowAlias="first broker login" \
			-s "config.clientId=$GOOGLE_CLIENT_ID" -s "config.clientSecret=$GOOGLE_CLIENT_SECRET" \
			-s config.defaultScope="openid profile email" -s config.syncMode=FORCE \
			-s config.guiOrder=1 -s config.hideOnLoginPage=false'
}

if kcadm get "identity-provider/instances/$alias" -r "$realm" >/dev/null 2>&1; then
	provider_settings "update identity-provider/instances/$alias"
	echo "identity provider $alias updated in realm $realm"
else
	provider_settings "create identity-provider/instances"
	echo "identity provider $alias created in realm $realm"
fi

echo "redirect URI for the Google OAuth client: http://localhost:${KEYCLOAK_PORT:-8180}/realms/$realm/broker/$alias/endpoint"
