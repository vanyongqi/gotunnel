# gotunnel Typical Bugs and Fixes

**This document summarizes core bug cases found during gotunnel development and real-world deployment: their root causes, code changes, and why the fix works.**

---

## 1. Data channel prematurely closed (causes HTTP 502)

### Symptoms
- Using the example scenario: server exposes port 10086, client local web service on 8080.
- Both client and server log "data channel established (e.g. 17ms)", but visiting http://server-ip:10086 returns HTTP 502 Bad Gateway.
- Client log shows `[DEBUG][client] Starting relay: local port 8080` immediately followed by `[DEBUG][client] Relay finished: local port 8080` — relay should persist until HTTP completes.

### Cause
- In server `cmd/server/main.go`, `handleControlConn` mistakenly used `defer conn.Close()` even for data_channel connections.
- When returning from handler after channel registration, `defer` closed the TCP connection too early.
- The connection placed into `mapping.DataChan` was already closed before `RelayConn` could use it.

### Solution
- Remove `defer conn.Close()` for the data_channel registration branch; retain it for control channel only.
- Only close data channel on validation/channel/queue failure, not on normal registration. Let RelayConn control lifecycle.

### Why This Fix
- Ownership principle: whichever component uses a connection, controls closing it. Data channels are owned by RelayConn; control channels by handleControlConn.
- This guarantees the channel stays open until all user HTTP or TCP operations are completed, no unexpected resets or closes.

---

## 2. Data channel timeout too short (channel setup fails)

### Symptoms
- Server logs `[WARN][server] data channel connection timeout: port 10086`.
- Channel setup fails on slow networks or when client takes a bit longer to respond.

### Cause
- Server's listenAndForwardWithStop/select used a hardcoded 10s timeout.
- Channel setup involves multiple network roundtrips: notify client, client opens new TCP, registers, handshake, etc.

### Solution
- Increase timeout from 10s → 60s.
- Log and monitor actual channel setup latency.

### Why This Fix
- 60s covers even slow/long-path networks and proxy scenarios, with no significant resource cost.
- Log lets us further optimize if real setup is slow for many users.

---

## 3. Duration values in logs missing (`<no value>ms`)

### Symptoms
- Log shows: `[DEBUG][client] Data channel dialed: <no value>ms` (should be a number).

### Cause
- Logging's `argsToMap` function didn't fully support int64 (from `Milliseconds()`); no value appeared in template.

### Solution
- Add type handling for both int and int64 in argsToMap; always forward duration values as ms integer.

### Why This Fix
- Accurate logging means relay and setup delays can be debugged and optimized; helps with user feedback on where issues happen.

---

**Result:**
With these fixes, example port forwarding, HTTP, and SSH relays now work robustly and continuously, and logs provide complete end-to-end traceability and diagnostics.

