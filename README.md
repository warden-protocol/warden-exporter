# Warden Exporter

Prometheus exporter for Warden protocol specific metrics

## Configuration

Exporter are configured through ENV vars

| ENV                  | Type   | Default                                |
| -------------------- | ------ | -------------------------------------- |
| GRPC_ADDR            | String | grpc.buenavista.wardenprotocol.org:443 |
| GRPC_TLS_ENABLED     | bool   | true                                   |
| GRPC_TIMEOUT_SECONDS | int    | 5                                      |
| CHAIN_ID             | String | buenavista-1                           |

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
```
