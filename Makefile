SERVICES := api-gateway identity-service land-plot-service report-service
MODULES := $(addprefix services/,$(SERVICES)) ai tools/stripcomments
WEB_DIR := apps/web
STRIP_COMMENTS := go -C tools/stripcomments run . $(abspath services) $(abspath ai) $(abspath tools)

# Runs a command in every Go module and stops at the first failure.
define for_each_module
	@for module in $(MODULES); do \
		printf '%s %s\n' "$(1)" "$$module"; \
		(cd $$module && $(2)) || exit 1; \
	done
endef

PROMETHEUS_IMAGE := prom/prometheus:v3.15.1
PROMETHEUS_CONFIG := $(abspath infra/observability/prometheus)

POSTGRES_PORT ?= 55432
PLOT_TEST_DATABASE_URL ?= postgres://plot_user:plot_dev_password@localhost:$(POSTGRES_PORT)/plot_db?sslmode=disable
REPORT_TEST_DATABASE_URL ?= postgres://report_user:report_dev_password@localhost:$(POSTGRES_PORT)/report_db?sslmode=disable
INTEGRATION_TEST := go test -count=1 -tags=integration ./internal/repository/postgres/...

.PHONY: fmt lint lint-fix strip-comments test test-integration vet check check-all web-check compose-config demo demo-lr3 demo-lr4 demo-lr5 demo-web ai-mcp-inspector obs-up obs-check saga-trace-check rules-check

# Formats code: gofumpt, import groups (gci) and long lines (golines).
fmt:
	$(call for_each_module,Formatting,golangci-lint fmt ./...)

# Checks formatting, whitespace (wsl_v5) and code quality using .golangci.yml,
# and that Go code has no comments except tool directives.
lint:
	$(call for_each_module,Linting,golangci-lint run ./...)
	@printf 'Checking comments\n'
	@$(subst run .,run . -check,$(STRIP_COMMENTS))

# Removes every comment except tool directives, applies formatters
# and every automatic lint fix.
lint-fix: strip-comments
	$(call for_each_module,Fixing,golangci-lint fmt ./... && golangci-lint run --fix ./...)

strip-comments:
	@$(STRIP_COMMENTS)

test:
	$(call for_each_module,Testing,go test ./...)

# Runs repository tests against the PostgreSQL from docker compose (docker compose up -d postgres).
# Each test works in its own temporary schema and drops it afterwards.
test-integration:
	cd services/land-plot-service && TEST_DATABASE_URL='$(PLOT_TEST_DATABASE_URL)' $(INTEGRATION_TEST)
	cd services/report-service && TEST_DATABASE_URL='$(REPORT_TEST_DATABASE_URL)' $(INTEGRATION_TEST)

vet:
	$(call for_each_module,Vetting,go vet ./...)

check: lint test vet compose-config rules-check

# Checks the Prometheus configuration and rules and runs the alert tests with promtool from the pinned image.
rules-check:
	docker run --rm -v "$(PROMETHEUS_CONFIG):/etc/prometheus:ro" --entrypoint sh $(PROMETHEUS_IMAGE) -c \
		'promtool check config /etc/prometheus/prometheus.yml && promtool test rules /etc/prometheus/tests/*.yml'

# Installs the web app dependencies from package-lock.json when they are missing or outdated.
$(WEB_DIR)/node_modules: $(WEB_DIR)/package.json $(WEB_DIR)/package-lock.json
	cd $(WEB_DIR) && npm ci
	@touch $@

# Lints (including layer boundaries), tests and builds the web app.
web-check: $(WEB_DIR)/node_modules
	cd $(WEB_DIR) && npm run lint && npm run test && npm run build

check-all: check web-check

compose-config:
	docker compose config --quiet
	docker compose --profile ai config --quiet
	docker compose --profile observability config --quiet

demo:
	./scripts/demo.sh

demo-lr3:
	./scripts/demo_lr3.sh

# Checks MCP and both A2A agents without the LLM, then runs every LLM scenario once when LLM_API_KEY is set.
demo-lr4:
	./scripts/demo_lr4.sh

# Checks CORS, Keycloak tokens, scopes and Zero Trust through the gateway and the services.
demo-lr5:
	./scripts/demo_lr5.sh

# Shows preview, validation details, deletion, version history, report listing, Mailpit, area accuracy,
# terrain and climate in reports, PDF and GeoJSON downloads, the plot summary, a report without Open-Meteo
# and the preview rate limit.
demo-web:
	./scripts/demo_web.sh

# Starts the system with the observability stack.
obs-up:
	docker compose --profile observability up -d --wait

# Shows the health of every Prometheus scrape target.
obs-check:
	./scripts/observability_check.sh

# Creates a confirmed and a compensated plot through the gateway and checks in Jaeger that each saga is one trace.
saga-trace-check:
	./scripts/saga_trace_check.sh

# Opens MCP Inspector with the MCP server over stdio; the stack must be running (docker compose up -d).
ai-mcp-inspector:
	go -C ai build -o $(abspath ai/bin)/ ./cmd/mcp-server
	@set -a; [ -f .env ] && . ./.env; set +a; \
	npx -y @modelcontextprotocol/inspector \
		-e GATEWAY_URL=http://localhost:$${GATEWAY_PORT:-8080} \
		-e OIDC_TOKEN_URL="$${OIDC_TOKEN_URL:-http://localhost:8180/realms/land-plot/protocol/openid-connect/token}" \
		-e AI_CLIENT_ID="$${AI_CLIENT_ID:-land-plot-ai}" \
		-e AI_CLIENT_SECRET="$${AI_CLIENT_SECRET:-local_development_ai_client_secret}" \
		-e AI_SCOPES="$${AI_SCOPES:-plots:read plots:write reports:read reports:write}" \
		$(abspath ai/bin)/mcp-server -transport stdio
