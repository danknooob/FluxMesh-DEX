# Sequence: Config Change Lifecycle (Control Plane)

```
Admin UI       Gateway       Control API    Control DB   Kafka              Data-plane     MongoDB
   │              │               │              │           │                  │              │
   │ PATCH market │               │              │           │                  │              │
   │ (e.g. fee)   │               │              │           │                  │              │
   │─────────────▶│               │              │           │                  │              │
   │              │ JWT (admin)   │              │           │                  │              │
   │              │ Rate limit    │              │           │                  │              │
   │              │──────────────▶│              │           │                  │              │
   │              │               │ auth + authz │           │                  │              │
   │              │               │ write        │           │                  │              │
   │              │               │─────────────▶│           │                  │              │
   │              │               │ control.audit            │                  │              │
   │              │               │─────────────────────────▶│                  │              │
   │              │               │ control.config           │                  │              │
   │              │               │─────────────────────────▶│  consume         │              │
   │              │               │              │           │─────────────────▶│              │
   │              │               │              │           │                  │ apply new    │
   │              │               │              │           │                  │ config       │
   │              │               │              │           │                  │              │
   │              │               │              │           │ Event Log consumes             │
   │              │               │              │           │──────────────────────────────▶  │
   │              │               │              │           │                  │   persist    │
   │              │               │              │           │                  │   + title    │
   │  200 OK      │               │              │           │                  │              │
   │◀─────────────│◀──────────────│              │           │                  │              │
```

### Steps

1. Admin changes desired state (market params, feature flags) via admin UI.
2. **Gateway** validates JWT (admin role required), rate limits, proxies to control plane.
3. **Control Plane** persists to DB, publishes to `control.audit` (immutable) and `control.config`.
4. **Data-plane services** consume `control.config` and apply new configuration.
5. **Event Log** persists both `control.audit` and `control.config` events to MongoDB with titles like "Config change: update BTC-USDC".
