#!/bin/sh
set -eu

create_role_and_database() {
    database_name="$1"
    database_user="$2"
    database_password="$3"

    psql --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" --set ON_ERROR_STOP=1 \
        --set database_name="$database_name" \
        --set database_user="$database_user" \
        --set database_password="$database_password" <<'SQL'
SELECT format('CREATE ROLE %I LOGIN PASSWORD %L', :'database_user', :'database_password')
WHERE NOT EXISTS (SELECT FROM pg_roles WHERE rolname = :'database_user') \gexec
SELECT format('CREATE DATABASE %I OWNER %I', :'database_name', :'database_user')
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = :'database_name') \gexec
SQL
}

create_role_and_database "$IDENTITY_DB_NAME" "$IDENTITY_DB_USER" "$IDENTITY_DB_PASSWORD"
create_role_and_database "$PLOT_DB_NAME" "$PLOT_DB_USER" "$PLOT_DB_PASSWORD"
create_role_and_database "$REPORT_DB_NAME" "$REPORT_DB_USER" "$REPORT_DB_PASSWORD"

