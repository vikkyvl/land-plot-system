#!/bin/sh
# Imports the credentials (values come from the environment and never reach a file in git),
# imports the incident workflow and publishes it. Runs once before n8n starts.

set -eu

credentials=$(mktemp)
trap 'rm -f "$credentials"' EXIT

node /import/credentials.js >"$credentials"
n8n import:credentials --input="$credentials"
n8n import:workflow --input=/import/incident-response.json
n8n publish:workflow --id=landPlotIncident
