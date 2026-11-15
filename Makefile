SERVICES := api-gateway identity-service land-plot-service report-service
MODULES := $(addprefix services/,$(SERVICES)) ai tools/stripcomments
STRIP_COMMENTS := go -C tools/stripcomments run . $(abspath services) $(abspath ai) $(abspath tools)

# Runs a command in every Go module and stops at the first failure.
define for_each_module
	@for module in $(MODULES); do \
		printf '%s %s\n' "$(1)" "$$module"; \
		(cd $$module && $(2)) || exit 1; \
	done
endef

POSTGRES_PORT ?= 55432
PLOT_TEST_DATABASE_URL ?= postgres://plot_user:plot_dev_password@localhost:$(POSTGRES_PORT)/plot_db?sslmode=disable
REPORT_TEST_DATABASE_URL ?= postgres://report_user:report_dev_password@localhost:$(POSTGRES_PORT)/report_db?sslmode=disable
INTEGRATION_TEST := go test -count=1 -tags=integration ./internal/repository/postgres/...

.PHONY: fmt lint lint-fix strip-comments test test-integration vet check compose-config demo demo-lr3 demo-lr4 ai-mcp-inspector

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

check: lint test vet compose-config

compose-config:
	docker compose config --quiet
	docker compose --profile ai config --quiet

demo:
	./scripts/demo.sh

demo-lr3:
	./scripts/demo_lr3.sh

# Checks MCP and both A2A agents without the LLM, then runs every LLM scenario once when LLM_API_KEY is set.
demo-lr4:
	./scripts/demo_lr4.sh

# Opens MCP Inspector with the MCP server over stdio; the stack must be running (docker compose up -d).
ai-mcp-inspector:
	go -C ai build -o $(abspath ai/bin)/ ./cmd/mcp-server
	@set -a; [ -f .env ] && . ./.env; set +a; \
	npx -y @modelcontextprotocol/inspector \
		-e GATEWAY_URL=http://localhost:$${GATEWAY_PORT:-8080} \
		-e AI_USER_EMAIL="$${AI_USER_EMAIL:-ai-agent@example.com}" \
		-e AI_USER_PASSWORD="$${AI_USER_PASSWORD:-AiAgentPassword123}" \
		$(abspath ai/bin)/mcp-server -transport stdio
