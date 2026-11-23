# AI agents for the Land Plot System

This Go module lets AI models and agents use the Land Plot System API. It contains:

| Command                | What it is                                                                                  |
|------------------------|---------------------------------------------------------------------------------------------|
| `cmd/fc-client`        | Console LLM client with function calling over the REST API                                  |
| `cmd/mcp-server`       | MCP server with tools, resources and prompts over stdio and Streamable HTTP                 |
| `cmd/producer-agent`   | A2A agent over `land-plot-service`, the event producer: creates and updates plots           |
| `cmd/consumer-agent`   | A2A agent over `report-service`, the event consumer: builds reports and delegates to the producer |
| `cmd/a2a-call`         | Small A2A client that sends a task and prints its states and artifacts                      |

Every part works with the system only through `api-gateway` with an access token from Keycloak, like any other client. No part reads the databases or RabbitMQ directly, and the Go services of the system are not changed.

## Layout

```
ai/
├── cmd/                     # one main.go per command, each calls bootstrap
└── internal/
    ├── landapi/             # REST client for the gateway: client credentials token, plots, reports
    ├── tools/               # tool definitions, JSON schemas, argument validation, results
    ├── guard/               # token budget, step limit, prompt injection checks, secret redaction
    ├── llm/                 # Provider interface, Gemini adapter, fake provider for tests
    ├── agent/               # tool calling loop and system prompts (prompts/*.txt)
    ├── mcpserver/           # MCP server: tools, resources, prompts, HTTP access token
    ├── a2aagent/            # A2A executor, agent cards, producer and consumer logic
    ├── config/              # typed configuration from environment variables
    └── bootstrap/           # wiring of every command
```

The tools are described once in `tools` and are used by function calling, by the MCP server and by both A2A agents.

## Free LLM key

The agents use Google Gemini through Google AI Studio on the free tier:

1. Open <https://aistudio.google.com/apikey>, sign in with a Google account and press **Create API key**.
2. Put the key into `.env` as `LLM_API_KEY`. `.env` is ignored by git; `.env.example` keeps the value empty.
3. Check the limits of your project at <https://aistudio.google.com/rate-limit>. The default model is `gemini-3.5-flash-lite`, which has a higher daily limit than the Flash models. Limits are per project, so a new key does not reset them; the daily quota resets at midnight Pacific time.

No test and no `make check` step calls the real model. Tests use the fake provider from `internal/llm`.

## Configuration

All settings are environment variables; `.env.example` lists them with defaults.

| Variable                    | Default                  | Used by            | Meaning                                                    |
|-----------------------------|--------------------------|--------------------|------------------------------------------------------------|
| `LLM_API_KEY`               | empty                    | fc-client, agents  | Gemini key; the agents start without it and fail tasks with «LLM_API_KEY не задано» |
| `LLM_MODEL`                 | `gemini-3.5-flash-lite`  | fc-client, agents  | Gemini model                                               |
| `LLM_MAX_OUTPUT_TOKENS`     | `1024`                   | fc-client, agents  | Output limit of one model answer, thinking tokens included |
| `LLM_THINKING_LEVEL`        | `high`                   | fc-client          | Thinking level of `-mode reasoning`: `low`, `medium`, `high` |
| `LLM_MIN_INTERVAL`          | `6s`                     | fc-client, agents  | Minimum pause between model calls                          |
| `LLM_REQUEST_TIMEOUT`       | `1m`                     | fc-client, agents  | Timeout of one model call                                  |
| `AGENT_TOKEN_BUDGET`        | `30000`                  | fc-client, agents  | Tokens for one request or A2A task                         |
| `AGENT_MAX_STEPS`           | `8`                      | fc-client, agents  | Steps with tool calls for one request or task              |
| `USER_INPUT_MAX_CHARS`      | `2000`                   | fc-client, agents  | Maximum request length                                     |
| `TOOL_RESULT_MAX_BYTES`     | `4000`                   | all                | Tool results are cut to this size                          |
| `GATEWAY_URL`               | `http://localhost:8080`  | all                | Gateway address; containers use `http://api-gateway:8080`  |
| `OIDC_TOKEN_URL`            | `http://localhost:8180/realms/land-plot/protocol/openid-connect/token` | all | Keycloak token endpoint; containers use `http://keycloak:8080/...` |
| `AI_CLIENT_ID`              | `land-plot-ai`           | all                | Confidential Keycloak client of all AI programs            |
| `AI_CLIENT_SECRET`          | —                        | all                | Secret of `land-plot-ai`; the realm is imported with the same value |
| `AI_SCOPES`                 | all four API scopes      | all                | Scopes requested for the program (see below)               |
| `MCP_ACCESS_TOKEN`          | empty                    | mcp-server (HTTP)  | Static bearer token, at least 32 characters                |
| `MCP_RATE_LIMIT_PER_MINUTE` | `30`                     | mcp-server         | Tool calls per minute                                      |
| `PRODUCER_AGENT_URL`        | `http://localhost:8091`  | consumer-agent     | The only producer the consumer trusts                      |
| `A2A_CALL_TIMEOUT`          | `90s`                    | consumer-agent     | How long the consumer waits for a producer task            |

A command started on the host does not read `.env` by itself; export it first:

```sh
set -a; . ./.env; set +a
```

## Access tokens

All AI programs share the confidential Keycloak client `land-plot-ai` and get tokens through the OAuth 2.0 Client Credentials grant: `landapi` posts `grant_type=client_credentials`, the client ID and secret and the requested `scope` to `OIDC_TOKEN_URL`, caches the token until 30 seconds before it expires, and asks for a new one when the gateway answers `401`. The programs therefore act as one service account with one `sub` and see the same plots, but each requests only the scopes it needs:

| Program          | `AI_SCOPES`                               |
|------------------|-------------------------------------------|
| `producer-agent` | `plots:read plots:write`                  |
| `consumer-agent` | `plots:read reports:read reports:write`   |
| `fc-client`      | all four                                  |
| `mcp-server`     | all four                                  |

Docker Compose sets these values for the containers. If the producer tried to build a report, the gateway would answer `403 insufficient_scope`.

## Guardrails

- **Tokens.** Every model answer is limited by `LLM_MAX_OUTPUT_TOKENS`. Usage reported by the model is added to `AGENT_TOKEN_BUDGET`; when the budget is spent, the agent stops with «Бюджет токенів вичерпано» instead of calling the model again. `AGENT_MAX_STEPS` limits tool-calling steps. Every model call is logged with input, output, thinking and total tokens.
- **Free tier limits.** Model calls are at least `LLM_MIN_INTERVAL` apart. On `429` the client waits for the delay suggested by the API, or 2, 4 and 8 seconds, and retries up to 3 times; a spent daily quota is reported at once without retries.
- **Arguments.** Every tool has a JSON schema without extra properties and is validated before the API call: UUIDs, a name of 1 to 120 characters without control characters, 3 to 100 points within latitude and longitude ranges, an open polygon, `timeout_seconds` from 1 to 40. A validation error is returned to the model as a tool error and the API is not called.
- **Prompt injection.** The system prompts say that tool results are data. Tool results are wrapped in `<tool_result>…</tool_result>`. Requests with phrases such as `ignore previous instructions`, `ігноруй попередні інструкції`, `system prompt` or `api key` are rejected; the same phrases inside API data, such as plot names, are replaced with `[вилучено]` and logged. Only the tools given to an agent can be called, and the API has no destructive operations.
- **Secrets.** The model never sees the access token, the client secret or the LLM key; `landapi` gets and refreshes tokens itself. The final answer is checked for the client secret, the key, JWTs and bearer tokens.

## Function calling: `fc-client`

The stack must be running (`docker compose up -d`). From the repository root:

```sh
set -a; . ./.env; set +a
go -C ai run ./cmd/fc-client 'Знайди мою ділянку "Дача в Ірпені". Якщо її немає, створи її з точками (50.5203, 30.2471), (50.5203, 30.2486), (50.5212, 30.2486), (50.5212, 30.2471). Дочекайся підтвердження і сформуй звіт, скажи площу в гектарах.'
```

The client prints every step: the model call with its tokens, the tool with its arguments and a short result, and at the end the answer with totals. Flags:

- `-mode non-reasoning` (default): minimal thinking and a detailed step-by-step system prompt (`internal/agent/prompts/non_reasoning.txt`);
- `-mode reasoning`: thinking at `LLM_THINKING_LEVEL` and a short prompt with the goal, constraints and success criterion (`internal/agent/prompts/reasoning.txt`);
- `-v`: JSON logs with token usage on stderr.

A small budget shows the token guardrail:

```sh
AGENT_TOKEN_BUDGET=500 go -C ai run ./cmd/fc-client 'Покажи мої ділянки'
```

## MCP server

The server exposes:

- **tools:** `list_plots`, `get_plot`, `create_plot`, `update_plot`, `wait_plot_confirmed`, `create_report`, `get_report`;
- **resources:** `land://plots` and the templates `land://plots/{plot_id}` and `land://reports/{report_id}`;
- **prompts:** `analyze_plot(plot_id)` and `create_plot_from_coordinates(name, coordinates)`.

Tool errors come back as results with `isError: true`. Tool calls are limited to `MCP_RATE_LIMIT_PER_MINUTE` and logged with their duration.

### MCP Inspector (stdio)

```sh
docker compose up -d
make ai-mcp-inspector
```

The target builds `ai/bin/mcp-server` and opens [MCP Inspector](https://modelcontextprotocol.io/docs/tools/inspector) with the server over stdio and the AI client credentials from `.env`. The Tools, Resources and Prompts tabs list everything above.

### Claude Desktop (stdio)

Build the binary with `go -C ai build -o "$(pwd)/ai/bin/" ./cmd/mcp-server` and add it to `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "land-plot-system": {
      "command": "/absolute/path/to/land-plot-system/ai/bin/mcp-server",
      "args": ["-transport", "stdio"],
      "env": {
        "GATEWAY_URL": "http://localhost:8080",
        "OIDC_TOKEN_URL": "http://localhost:8180/realms/land-plot/protocol/openid-connect/token",
        "AI_CLIENT_ID": "land-plot-ai",
        "AI_CLIENT_SECRET": "<the value of AI_CLIENT_SECRET from .env>",
        "AI_SCOPES": "plots:read plots:write reports:read reports:write"
      }
    }
  }
}
```

### Streamable HTTP

`docker compose --profile ai up -d` starts the server on `http://127.0.0.1:8090/mcp` in stateless JSON mode. Every request needs the static token:

```sh
curl -s http://127.0.0.1:8090/mcp \
  -H 'Content-Type: application/json' -H 'Accept: application/json, text/event-stream' \
  -H "Authorization: Bearer $MCP_ACCESS_TOKEN" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'
```

Without the header the server answers `401`. `GET /health` needs no token.

## A2A agents

`docker compose --profile ai up -d` starts both agents:

| Agent            | URL on the host          | Skills                       | Tools of its model                                               |
|------------------|--------------------------|------------------------------|------------------------------------------------------------------|
| `producer-agent` | `http://localhost:8091`  | `create_plot`, `update_plot` | `list_plots`, `get_plot`, `create_plot`, `update_plot`, `wait_plot_confirmed` |
| `consumer-agent` | `http://localhost:8092`  | `plot_report`                | `get_plot`, `create_report`, `get_report`, `delegate_to_producer` |

Agent Cards:

```sh
curl -s http://localhost:8091/.well-known/agent-card.json | jq
curl -s http://localhost:8092/.well-known/agent-card.json | jq
```

The producer creates or updates a plot, which starts the saga of `land-plot-service`, and waits for it. Its task completes with a `plot` artifact (`plot_id`, `name`, `status`, `version`, `outcome`) only when the saga is confirmed; a compensated saga, for example after a `FAIL` name, ends the task in `TASK_STATE_FAILED` with an explanation. The outcome comes from the tool results, not from the model text.

The consumer builds area reports. When a plot is missing, its `delegate_to_producer` tool reads the producer card from `PRODUCER_AGENT_URL`, checks that the needed skill is listed and sends the task over A2A. The tool has no address argument, so the model can reach only that producer, and the producer reply goes back to the model as data inside `<tool_result>`. The consumer task completes with a `report` artifact (`report_id`, `plot_id`, `area_hectares` and more).

Each task gets its own token budget and step limit, and its text passes the same input checks as a console request.

Send a task and watch its states:

```sh
go -C ai run ./cmd/a2a-call http://localhost:8092 'Створи ділянку "Пай під Білою Церквою" з точками (49.7968, 30.1012), (49.7968, 30.1105), (49.8021, 30.1105), (49.8021, 30.1012) і сформуй звіт'
go -C ai run ./cmd/a2a-call http://localhost:8091 'Створи ділянку "FAIL поле" з точками (50.4401, 30.5134), (50.4401, 30.5150), (50.4410, 30.5150)'
```

`a2a-call` exits with `0` for a completed task and `1` otherwise. A raw JSON-RPC call needs the protocol version header:

```sh
curl -s http://localhost:8091/ -H 'Content-Type: application/json' -H 'A2A-Version: 1.0' \
  -d '{"jsonrpc":"2.0","id":1,"method":"SendMessage","params":{"message":{"messageId":"m-1","role":"ROLE_USER","parts":[{"text":"Створи ділянку \"Сад\" з точками (50.45, 30.52), (50.45, 30.53), (50.46, 30.53)"}]}}}'
```

## Demo

```sh
make demo-lr4
```

The script starts the stack with the `ai` profile and runs deterministic checks first: MCP `401` without a token, `tools/list`, a validation error, `create_plot` with `wait_plot_confirmed`, a resource and the prompts, and both Agent Cards. When `LLM_API_KEY` is set, it then runs every LLM scenario once with pauses: the function calling chains, the token budget, two prompt injection checks, both prompting modes, consumer-to-producer delegation and a failed producer task. `DEMO_LLM=0` skips the LLM part.
