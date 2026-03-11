# Sequence: Order Lifecycle

```
Client        Gateway        API          Postgres     Kafka         Matching      Settlement    EVM         MongoDB
  │              │             │              │           │              │              │           │            │
  │ POST /orders │             │              │           │              │              │           │            │
  │─────────────▶│             │              │           │              │              │           │            │
  │              │ JWT check   │              │           │              │              │           │            │
  │              │ Rate limit  │              │           │              │              │           │            │
  │              │────────────▶│              │           │              │              │           │            │
  │              │             │ validate     │           │              │              │           │            │
  │              │             │ read user    │           │              │              │           │            │
  │              │             │ from headers │           │              │              │           │            │
  │              │             │─────────────▶│           │              │              │           │            │
  │              │             │ persist order│           │              │              │           │            │
  │              │             │◀─────────────│           │              │              │           │            │
  │              │             │ orders.created           │              │              │           │            │
  │              │             │─────────────────────────▶│              │              │           │            │
  │ 202 Accepted │             │              │           │              │              │           │            │
  │◀─────────────│◀────────────│              │           │              │              │           │            │
  │              │             │              │           │  consume     │              │           │            │
  │              │             │              │           │─────────────▶│              │           │            │
  │              │             │              │           │              │ match order  │           │            │
  │              │             │              │           │              │ book (P/T)   │           │            │
  │              │             │              │           │ orders.matched│              │           │            │
  │              │             │              │           │◀──────────────│              │           │            │
  │              │             │              │           │  consume     │              │           │            │
  │              │             │              │           │──────────────────────────────▶           │            │
  │              │             │              │           │              │              │ settle    │            │
  │              │             │              │           │              │              │──────────▶│            │
  │              │             │              │           │              │              │◀──────────│            │
  │              │             │              │           │ trades.settled│              │           │            │
  │              │             │              │           │◀──────────────│              │           │            │
  │              │             │              │           │                              │           │            │
  │              │             │              │           │ Event Log consumes all topics│           │            │
  │              │             │              │           │──────────────────────────────────────────────────────▶│
  │              │             │              │           │              │              │           │  persist   │
  │              │             │              │           │              │              │           │  + title   │
  │ WebSocket   │             │              │           │              │              │           │            │
  │ order fill  │             │              │           │ Notification │              │           │            │
  │◀─────────────────────────────────────────────────────────────────────              │           │            │
```

### Steps

1. **Client** sends `POST /orders` with JWT in `Authorization` header.
2. **Gateway** validates JWT, checks rate limit, injects `X-User-ID`/`X-Role`, proxies to API.
3. **API** reads user from headers, validates order, persists to Postgres, publishes `orders.created` to Kafka.
4. **Matching Engine** consumes `orders.created`, runs price-time priority matching, publishes `orders.matched` or `orders.rejected`.
5. **Settlement** consumes `orders.matched`, batches trades, calls EVM `ExchangeCore.settleTrades`, publishes `trades.settled` and `balances.updated`.
6. **Event Log** consumes every topic and persists each event to MongoDB with a human-readable title.
7. **Notification** pushes order fill update to client via WebSocket.
