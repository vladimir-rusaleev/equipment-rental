#!/usr/bin/env bash
set -euo pipefail
: "${APP_DB_PASSWORD:?APP_DB_PASSWORD is required}"
psql --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" --set=ON_ERROR_STOP=1 --set=app_password="$APP_DB_PASSWORD" <<'SQL'
CREATE USER rent_rental PASSWORD :'app_password';
CREATE USER rent_inventory PASSWORD :'app_password';
CREATE USER rent_notifier PASSWORD :'app_password';
CREATE DATABASE rent_rental OWNER rent_rental;
CREATE DATABASE rent_inventory OWNER rent_inventory;
CREATE DATABASE rent_notifier OWNER rent_notifier;
REVOKE ALL ON DATABASE rent_rental FROM PUBLIC;
REVOKE ALL ON DATABASE rent_inventory FROM PUBLIC;
REVOKE ALL ON DATABASE rent_notifier FROM PUBLIC;
SQL
if [[ "${CREATE_TEST_DATABASES:-false}" == "true" ]]; then
 psql --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" --set=ON_ERROR_STOP=1 <<'SQL'
CREATE DATABASE rent_rental_tests OWNER rent_rental;
CREATE DATABASE rent_inventory_tests OWNER rent_inventory;
CREATE DATABASE rent_notifier_tests OWNER rent_notifier;
REVOKE ALL ON DATABASE rent_rental_tests FROM PUBLIC;
REVOKE ALL ON DATABASE rent_inventory_tests FROM PUBLIC;
REVOKE ALL ON DATABASE rent_notifier_tests FROM PUBLIC;
SQL
fi
