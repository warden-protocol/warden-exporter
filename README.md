# Warden Exporter

Prometheus exporter for Warden protocol specific metrics

## Configuration

Exporter are configured through ENV vars

| ENV                  | Type   | Default                  |
| -------------------- | ------ | ------------------------ |
| PORT                 | string | 8081                     |
| ENV_FILE             | string |                          |
| GRPC_ADDR            | string | grpc.wardenprotocol.org:443 |
| GRPC_TLS_ENABLED     | bool   | true                     |
| GRPC_TIMEOUT_SECONDS | int    | 45                       |
| HTTP_TIMEOUT_SECONDS | int    | 10                       |
| TTL                  | int    | 60                       |
| CHAIN_ID             | string | warden_8765-1            |
| DENOM                | string | award                    |
| EXPONENT             | int    | 18                       |
| BLOCK_WINDOW         | int    | 200                      |
| VALIDATOR_METRICS    | bool   | true                     |
| MINT_METRICS         | bool   | true                     |
| WALLET_ADDRESSES     | string |                          |
| VENICE_METRICS       | bool   | false                    |
| VENICE_API_KEY       | string |                          |
| MESSARI_METRICS      | bool   | false                    |
| MESSARI_API_KEY      | string |                          |
| BASE_METRICS         | bool   | false                    |
| BASE_RPC_URL         | string |                          |
| BASE_ADDRESSES       | string |                          |
| BNB_METRICS          | bool   | false                    |
| BNB_RPC_URL          | string |                          |
| BNB_ADDRESSES        | string |                          |
| COINGECKO_METRICS    | bool   | false                    |
| COINGECKO_API_KEY    | string |                          |
| XAI_METRICS          | bool   | false                    |
| XAI_API_KEY          | string |                          |
| XAI_TEAM_ID          | string |                          |
| OPENAI_METRICS       | bool   | false                    |
| OPENAI_API_KEY       | string |                          |
| TAVILY_METRICS       | bool   | false                    |
| TAVILY_API_KEY       | string |                          |
| OPENROUTER_METRICS   | bool   | false                    |
| OPENROUTER_API_KEY   | string |                          |
| COMPOSIO_METRICS     | bool   | false                    |
| COMPOSIO_API_KEY     | string |                          |

## Metrics

Returns these metrics

```
- Validator metrics
    - Missed blocks within the last `BLOCK_WINDOW` blocks
    - Blocks proposed
    - Average block time
    - Bonded tokens
    - Delegator shares
- Mint metrics
    - Inflation
    - Annual provisions
    - Total supply
- Wallet balances (`WALLET_ADDRESSES` accepts a comma-separated list)
- Venice API metrics (`VENICE_API_KEY` accepts a comma-separated list of keys for multiple accounts)
    - Billing balance
    - Usage
- Messari API credits (allocated and remaining)
- Base blockchain wallet balances (`BASE_ADDRESSES` accepts a comma-separated list)
- BNB blockchain wallet balances (`BNB_ADDRESSES` accepts a comma-separated list)
- CoinGecko API usage metrics
    - Rate limit per minute
    - Monthly call credit
    - Current total monthly calls
    - Current remaining monthly calls
- X.AI API metrics
    - Usage (monthly and daily cost in USD)
    - Postpaid spending limits (hard limit auto, effective hard limit, soft limit, effective limit)
    - Prepaid balance (total balance in USD)
- OpenAI API metrics
    - Monthly costs in USD
- Tavily API metrics
    - Plan usage (credits consumed in current billing cycle)
    - Plan limit (credit ceiling for current billing cycle)
- OpenRouter API metrics (`OPENROUTER_API_KEY` accepts a comma-separated list of keys)
    - Usage in USD (total, daily, weekly, monthly)
    - Spending limit and remaining for the configured period
    - Account purchased credits and total credit usage in USD
- Composio API metrics (requires an organization-level API key, `x-org-api-key`)
    - Org metering usage quantity month-to-date by entity_type (e.g. tool_calls, sessions, premium_tool_calls)
    - Org metering event count month-to-date by entity_type
    - tool_calls usage breakdown by tool_slug (top 100)
    - Total project count in the organization
```
