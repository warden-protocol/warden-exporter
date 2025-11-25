# Warden Exporter

Prometheus exporter for Warden protocol specific metrics

## Configuration

Exporter are configured through ENV vars

| ENV                  | Type   | Default                                |
| -------------------- | ------ | -------------------------------------- |
| GRPC_ADDR            | string | grpc.buenavista.wardenprotocol.org:443 |
| GRPC_TLS_ENABLED     | bool   | true                                   |
| GRPC_TIMEOUT_SECONDS | int    | 5                                      |
| ENV_FILE             | string |                                        |
| TTL                  | int    | 60                                     |
| CHAIN_ID             | string | buenavista-1                           |
| WARDEN_METRICS       | bool   | true                                   |
| VALIDATOR_METRICS    | bool   | true                                   |
| WALLET_ADDRESSES     | string |                                        |
| DENOM                | string | uward                                  |
| WARP_METRICS         | bool   | true                                   |
| WARP_DATABASE        | string |                                        |
| WARP_DATABASE_USER   | string |                                        |
| WARP_DATABASE_PASS   | string |                                        |
| WARP_DATABASE_HOST   | string |                                        |
| VENICE_METRICS       | bool   | false                                  |
| VENICE_API_KEY       | string |                                        |
| MESSARI_METRICS      | bool   | false                                  |
| MESSARI_API_KEY      | string |                                        |
| BASE_METRICS         | bool   | false                                  |
| BASE_RPC_URL         | string |                                        |
| BASE_ADDRESSES       | string |                                        |
| BNB_METRICS          | bool   | false                                  |
| BNB_RPC_URL          | string |                                        |
| BNB_ADDRESSES        | string |                                        |
| COINGECKO_METRICS    | bool   | false                                  |
| COINGECKO_API_KEY    | string |                                        |
| XAI_METRICS          | bool   | false                                  |
| XAI_API_KEY          | string |                                        |
| XAI_TEAM_ID          | string |                                        |

## Metrics

Returns these metrics of Warden Protocol

```
- Spaces
- Keys
    - ECDSA
    - EDDSA
    - Pending
- Keychains
- Accounts
- Actions
- Wallet balances
- Validator metrics
- WARP metrics
- Venice API metrics
- Messari API metrics
- Base blockchain wallet balances
- BNB blockchain wallet balances
- CoinGecko API usage metrics
    - Rate limit per minute
    - Monthly call credit
    - Current total monthly calls
    - Current remaining monthly calls
- X.AI API metrics
    - Usage (monthly and daily cost in USD)
    - Postpaid spending limits (hard limit auto, effective hard limit, soft limit, effective limit)
    - Prepaid balance (total balance in USD)
```
