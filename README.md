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
```
