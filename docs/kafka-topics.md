# Kafka Topic Design

## Data plane

| Topic             | Producer      | Consumers                    | Purpose                          |
|-------------------|---------------|------------------------------|----------------------------------|
| orders.created    | API           | Matching engine              | New limit/market orders          |
| orders.cancelled  | API           | Matching, Indexer            | Order cancellations              |
| orders.matched    | Matching      | Settlement, Indexer, Notify  | Fills and remaining size         |
| orders.rejected   | Matching      | Indexer, Notify              | Failed risk/validation           |
| trades.settled    | Settlement    | Indexer, Notification        | On-chain settlement done         |
| balances.updated  | Settlement    | Indexer, Notification        | Balance changes                  |
| notifications.user| Various       | Notification service         | User-targeted notifications      |

## Control plane

| Topic           | Producer           | Consumers              | Purpose                          |
|-----------------|--------------------|-------------------------|----------------------------------|
| control.config  | Control plane      | All data-plane services | Config, feature flags, markets   |
| control.health  | Data-plane services| Control plane           | Heartbeats / health              |
| control.audit   | Control plane      | —                       | Immutable audit log              |
| control.commands| Control plane      | Data-plane services     | Pause market, safe mode, etc.    |

## Consumer groups

- **Matching engine**: single consumer group for `orders.created`.
- **Settlement**: single consumer group for `orders.matched`.
- **Indexer**: one group for chain + `trades.settled` / `balances.updated`.
- **Notification**: one group for domain topics + `notifications.user`.
- **Data-plane services**: each subscribes to `control.config` and `control.commands` with its own group.
