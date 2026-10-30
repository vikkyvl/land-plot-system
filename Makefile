SERVICES := api-gateway identity-service land-plot-service report-service
MODULES := $(addprefix services/,$(SERVICES)) tools/stripcomments
STRIP_COMMENTS := go -C tools/stripcomments run . $(abspath services) $(abspath tools)

# Runs a command in every Go module and stops at the first failure.
define for_each_module
	@for module in $(MODULES); do \
		printf '%s %s\n' "$(1)" "$$module"; \
		(cd $$module && $(2)) || exit 1; \
	done
endef

.PHONY: fmt lint lint-fix strip-comments test vet check compose-config demo

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

vet:
	$(call for_each_module,Vetting,go vet ./...)

check: lint test vet compose-config

compose-config:
	docker compose config --quiet

demo:
	./scripts/demo.sh
