# API Gateway

A production-grade API Gateway built in Go from scratch — no frameworks, pure `net/http`. Designed to sit in front of any microservices backend, handling auth, rate limiting, circuit breaking, routing, and observability in a single binary.

Built as a demonstration of systems programming and distributed systems knowledge, targeting deployment on AKS as a pod with a LoadBalancer service.

---

## Architecture

Request flow through the middleware pipeline:

```
Client Request
      │
      ▼
 ┌─────────────────────────────────────────┐
 │              HTTP Mux                   │
 │  /metrics ──► Prometheus Handler        │
 │  /*       ──► Router                    │
 └─────────────────────────────────────────┘
      │
      ▼
 ┌─────────────────────────────────────────┐
 │           Trie Router                   │
 │  Matches path prefix → Route config     │
 └─────────────────────────────────────────┘
      │
      ▼
 ┌─────────────────────────────────────────┐
 │         Middleware Chain                │
 │                                         │
 │  1. Metrics       (always)              │
 │  2. JWT Auth      (if auth_required)    │
 │  3. Rate Limiter  (if rate_limit set)   │
 └─────────────────────────────────────────┘
      │
      ▼
 ┌─────────────────────────────────────────┐
 │         Proxy Handler                   │
 │                                         │
 │  HTTP  ──► Round Robin → Circuit        │
 │             Breaker   → Upstream        │
 │                                         │
 │  WS    ──► Round Robin → Circuit        │
 │             Breaker   → TCP Dial        │
 │                          → Hijack       │
 │                          → Bidirectional│
 │                            io.Copy      │
 └─────────────────────────────────────────┘
```

---

## Components

### JWT Authentication (RS256)

Validates Bearer tokens using asymmetric RS256 signatures. Public keys are fetched from a JWKS endpoint and cached with a configurable TTL.

**Key design decisions:**
- Keys are cached in a `sync.RWMutex`-protected map. On cache miss, double-checked locking prevents multiple goroutines from simultaneously triggering a JWKS fetch.
- Forced refresh on unknown `kid` — if a key ID isn't in the cache, the cache is invalidated and JWKS is re-fetched once. This handles key rotation without a service restart.
- Claims (`sub`, `exp`, `iat`) are extracted and injected into the request context via `context.WithValue` so downstream middleware and handlers can read them without re-parsing the token.
- All JWT components are decoded with `base64.RawURLEncoding` — no padding, URL-safe alphabet, as required by the JWT spec.

---

### Rate Limiter (Token Bucket)

Per-client rate limiting using the token bucket algorithm. Client identity is resolved in order: JWT `sub` claim if auth passed, falling back to remote IP.

**Key design decisions:**
- Buckets are lazily initialized — a bucket is only created when a client makes its first request. This avoids allocating memory for clients that never appear.
- Double-checked locking on bucket creation: read lock first to check existence, write lock only if the bucket is missing, then re-check under the write lock to avoid duplicate creation under goroutine contention.
- Rate and burst are configurable per route in the YAML config, not globally. Different routes can have different limits.

---

### Circuit Breaker

Per-upstream state machine that stops forwarding requests to a failing upstream and allows recovery probing.

**States:**
```
Closed ──(failures > threshold)──► Open ──(recovery window elapsed)──► Half-Open
  ▲                                                                          │
  └──────────────────(success recorded)────────────────────────────────────┘
                      Half-Open ──(failure)──► Open
```

**Key design decisions:**
- Failure counting is windowed — failures older than `failure_window` are discarded. This prevents a burst of old errors from keeping the circuit open indefinitely.
- In Half-Open state, exactly one request is allowed through. If it succeeds, the circuit closes. If it fails, it immediately re-opens. This is enforced by setting state back to Open as soon as `Allow()` is called in Half-Open — the probe request either redeems it or it trips again.
- Each upstream instance has its own `CircuitBreaker` struct. A failure on one upstream does not affect others.

---

### Trie Router

Path-prefix matching using a trie data structure. Incoming paths are matched against the longest registered prefix.

**Key design decisions:**
- Trie over a linear scan because prefix matching on a trie is O(k) where k is the length of the path, versus O(n*k) for a linear scan over all routes. At scale with many routes, this matters.
- Prefix-based, not exact match — `/api/users/123` matches a route registered as `/api/users`.

---

### Load Balancer (Round Robin)

Distributes requests across multiple upstream instances per route using round-robin selection.

**Key design decisions:**
- Per-route atomic counter incremented on each request, modulo the number of upstream instances. Lock-free for the common case.
- Round robin is intentionally simple — no weighted routing, no least-connections. For the use case (homogeneous upstream instances), this is sufficient and has zero overhead.

---

### WebSocket Proxying

Transparent WebSocket upgrade forwarding. The gateway detects upgrade requests, dials the upstream over raw TCP, performs the handshake, hijacks the client connection, and bridges both sides with bidirectional `io.Copy`.

**Key design decisions:**
- Raw TCP dial to upstream instead of using `http.ReverseProxy` — the standard reverse proxy does not support WebSocket upgrades cleanly.
- `http.Hijacker` is used to take ownership of the client TCP connection from the HTTP server, allowing raw reads and writes after the 101 handshake.
- Two goroutines with a shared `done` channel handle bidirectional proxying. The first goroutine to finish signals done; the second is waited on before cleanup to avoid a goroutine leak.
- WebSocket connections go through the same middleware chain as HTTP (auth, rate limiting) before the upgrade is handed off.

---

### Prometheus Metrics

Exposes a `/metrics` endpoint scraped by Prometheus. The following metrics are tracked:

| Metric | Type | Labels |
|---|---|---|
| `http_requests_total` | Counter | method, path, status |
| `http_request_duration_seconds` | Histogram | method, path |
| `auth_failures_total` | Counter | — |
| `rate_limit_hits_total` | Counter | route |
| `circuit_breaker_trips_total` | Counter | — |

Upstream requests are tracked separately from gateway-level requests using `method=UPSTREAM` and `method=WS` labels, allowing per-upstream latency and error rate to be monitored independently.
