# Sequence: Config Change Lifecycle (MCP)

```
Admin UI       MCP API        MCP DB       Kafka              Data-plane services
   │               │              │            │                        │
   │  PATCH market │              │            │                        │
   │  (e.g. fee)   │              │            │                        │
   │──────────────▶│              │            │                        │
   │               │  auth + authz│            │                        │
   │               │  write       │            │                        │
   │               │─────────────▶│            │                        │
   │               │  control.audit           │                        │
   │               │─────────────────────────▶│                        │
   │               │  control.config          │                        │
   │               │─────────────────────────▶│  consume                │
   │               │              │            │───────────────────────▶│
   │               │              │            │                        │ apply new
   │  200 OK       │              │            │                        │ config
   │◀──────────────│              │            │                        │
```

- Admin changes desired state (e.g. market params, feature flags) via MCP API/UI.
- MCP persists to DB, publishes to `control.audit` (immutable) and `control.config`.
- Data-plane services consume `control.config` and apply new configuration (e.g. matching engine updates fee or tick size).
