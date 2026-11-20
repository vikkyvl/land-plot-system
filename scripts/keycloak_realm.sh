#!/bin/sh

set -eu

if [ -f .env ]; then
	set -a
	. ./.env
	set +a
fi

realm=${KEYCLOAK_REALM:-land-plot}
realm_file=infra/keycloak/realm-land-plot.json
settings='{displayName, loginTheme, registrationAllowed, resetPasswordAllowed, internationalizationEnabled, supportedLocales, defaultLocale, smtpServer}'
shown_fields='displayName,loginTheme,registrationAllowed,resetPasswordAllowed,internationalizationEnabled,supportedLocales,defaultLocale,smtpServer(host,port,from,auth)'

for command_name in docker jq; do
	if ! command -v "$command_name" >/dev/null 2>&1; then
		echo "required command is missing: $command_name" >&2
		exit 1
	fi
done

kcadm() {
	docker compose exec -T keycloak \
		bash -c '/opt/keycloak/bin/kcadm.sh "$@" --config /tmp/kcadm.config' kcadm "$@"
}

if ! docker compose exec -T keycloak bash -c '/opt/keycloak/bin/kcadm.sh config credentials \
	--server http://localhost:8080 --realm master \
	--user "$KC_BOOTSTRAP_ADMIN_USERNAME" --password "$KC_BOOTSTRAP_ADMIN_PASSWORD" \
	--config /tmp/kcadm.config' >/dev/null 2>&1; then
	echo "cannot log in to the Keycloak admin API; is the keycloak container running and healthy?" >&2
	exit 1
fi

jq -c "$settings" "$realm_file" | kcadm update "realms/$realm" -f - >/dev/null

echo "realm $realm updated from $realm_file:"
kcadm get "realms/$realm" --fields "$shown_fields" 2>/dev/null | jq -c .
