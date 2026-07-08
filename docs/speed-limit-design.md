# Per-client speed limit design for 3x-ui + Xray-core

Status: planning document, no implementation in this repository yet.

## Goal

Add real per-client speed limiting to the deployed 3x-ui stack, from panel input to Xray-core enforcement.

A successful implementation must satisfy all of these:

1. 3x-ui exposes a speed limit control on `/panel/clients`, displayed after the Traffic column.
2. 3x-ui stores the setting per client and sends it to Xray-core in both generated config and runtime AddUser API paths.
3. A patched Xray-core enforces the limit in the data path, not only in the UI.
4. The deployed server uses the patched 3x-ui binary and the matching patched Xray-core binary.
5. `0` means unlimited.

## Direct decision

The existing `speed/` code is useful as a prototype, but it must not be copied verbatim.

Use this production shape instead:

- Xray-core fork owns enforcement.
- 3x-ui fork owns UI, persistence, API payloads, config generation, and packaged binary selection.
- `limitIp` can be reused as `deviceLimit`.
- Add one new client field, `speedLimit`, for bytes per second.
- Preserve protobuf compatibility by appending fields instead of renumbering existing fields.

## Source evidence

- Go `golang.org/x/time/rate`: implements a token bucket limiter; `WaitN` waits for `n` events but errors when `n` exceeds burst size.  
  <https://pkg.go.dev/golang.org/x/time/rate>
- Linux `tc-tbf`: Token Bucket Filter is a standard traffic shaping mechanism; tokens correspond roughly to bytes and traffic waits when tokens are unavailable.  
  <https://man7.org/linux/man-pages/man8/tc-tbf.8.html>
- Xray API docs: HandlerService supports runtime user add/remove for VMess, VLESS, Trojan, and Shadowsocks; StatsService exists separately.  
  <https://xtls.github.io/en/config/api.html>
- Xray statistics docs: per-user statistics are keyed by user email and counted in bytes.  
  <https://xtls.github.io/en/config/stats.html>
- Protobuf docs: existing field numbers must not be changed once a message is in use.  
  <https://protobuf.dev/programming-guides/proto3/>
- Xray issue #3667: the submitted IP/rate limiting proposal is closed as not planned, so this is a fork-maintained feature, not an upstream-supported feature.  
  <https://github.com/XTLS/Xray-core/issues/3667>

## First-principles review

### What is the real object being controlled?

The controlled object is not the panel form field. It is bytes leaving or entering the Xray data path for a resolved authenticated user.

Therefore, any design that only changes 3x-ui is not real speed limiting.

### What is the minimum enforcement point?

The minimum enforcement point is where Xray already has:

- the authenticated user email;
- the inbound tag / session context;
- the byte stream represented as buffers;
- the ability to block or delay forwarding.

In current Xray-core, that is around dispatcher link wrapping / writers, not in 3x-ui.

### What is the minimum control plane?

The minimum control plane is one numeric value per client:

```text
speedLimit: bytes per second, 0 = unlimited
```

3x-ui must persist and send this value. Xray-core must enforce it.

### What must not be treated as truth?

- UI display is not enforcement truth.
- 3x-ui database is not enforcement truth after deployment unless the running Xray binary receives and applies the value.
- Official Xray-core does not support this field unless patched.
- The `speed/` prototype is not production-ready code.

## MECE scope breakdown

### 1. Xray-core protocol contract

Required:

```proto
message User {
  uint32 level = 1;
  string email = 2;
  xray.common.serial.TypedMessage account = 3;
  uint64 speed_limit = 4;
  uint32 device_limit = 5;
}
```

Rules:

- Do not renumber `level`, `email`, or `account`.
- Regenerate generated protobuf code from `user.proto`.
- Carry fields into `MemoryUser` and back through `ToProtoUser`.

### 2. Xray-core static config parsing

Required:

- VMess clients read `speedLimit` and `limitIp`/`deviceLimit` from JSON.
- VLESS clients read `speedLimit` and `limitIp`/`deviceLimit` from JSON.
- Trojan clients read `speedLimit` and `limitIp`/`deviceLimit` from JSON.
- Shadowsocks clients read `speedLimit` and `limitIp`/`deviceLimit` from JSON.
- Hysteria2 and WireGuard support should be explicit: either supported and tested, or documented as unsupported in first release.

Reason: 3x-ui rebuilds full Xray config on restart; runtime gRPC support alone is insufficient.

### 3. Xray-core runtime API path

Required:

- HandlerService AddUser must accept `speed_limit` and `device_limit` through `protocol.User`.
- 3x-ui's gRPC AddUser path must populate those fields.
- Hot update must refresh limiter state for changed users, or remove/re-add must rebuild the limiter.

### 4. Xray-core enforcement logic

Required:

- Key limiters by a stable scope: inbound tag + email.
- Separate upload and download limiters unless product copy explicitly says the limit is combined bidirectional bandwidth.
- Use token bucket with bytes as tokens.
- Handle buffer writes larger than burst size by chunking or using a burst no smaller than max buffer size.
- Release device/IP occupancy on connection close, not only after a timeout.
- Keep a timeout cleanup as a safety net for abnormal termination.
- Support dynamic change: if speed changes, update or recreate the per-user limiter.

Non-goals for first version:

- Distributed limit across multiple nodes.
- Global account limit across several inbound tags.
- Per-protocol custom rate behavior.
- Kernel-level QoS guarantees.

### 5. 3x-ui persistence and API

Required:

- Add `speed_limit` column to `clients`.
- Add `SpeedLimit uint64` / `speedLimit` to:
  - `model.Client`;
  - `model.ClientRecord`;
  - conversion methods `ToRecord` and `ToClient`;
  - merge logic;
  - client list slim DTO;
  - create/update payloads;
  - OpenAPI/generated schemas.
- Existing rows default to `0`.

### 6. 3x-ui UI

Required:

- On `/panel/clients`, display Speed Limit immediately after Traffic.
- Edit client modal includes speedLimit input near traffic quota and IP limit.
- Use `0 = unlimited` copy.
- Prefer human display units, but persist bytes/s.

Suggested UI copy:

```text
Speed Limit
Maximum per-direction speed. 0 = unlimited.
```

If product chooses combined bidirectional limit instead, copy must say:

```text
Combined upload + download speed. 0 = unlimited.
```

### 7. Packaging and deployment

Required:

- Build and publish patched Xray-core binaries from the fork.
- Make 3x-ui installer / Docker build download the patched binary, not official Xray-core.
- Deploy both binaries together.
- Record the exact Xray-core commit used by the 3x-ui build.

### 8. Verification

Minimum checks before calling the feature done:

- Xray-core unit test: `protocol.User` preserves speed/device fields.
- Xray-core unit test: limiter delays writes and does not error on large buffer with low limit.
- Xray-core unit test: device limit releases IP on connection close.
- 3x-ui backend test: client create/update persists `speedLimit`.
- 3x-ui backend test: generated Xray config contains `speedLimit`.
- 3x-ui backend test: gRPC AddUser populates `protocol.User.SpeedLimit`.
- Frontend test/schema check: client form accepts speedLimit and list rows display it.
- Manual e2e: create a client with low speedLimit, connect through deployed patched stack, measure throughput below limit.

## Ontology review

### Entities

- Client: panel-level account row, identified mainly by email.
- Inbound: Xray listener / protocol configuration.
- User: Xray-core runtime authenticated principal.
- Connection: one live session through Xray.
- Source IP: observed network address for device/IP limiting.
- Limiter: runtime state object that decides when bytes may pass.
- Policy: existing Xray level-based settings; not enough for per-client speed limit unless extended.
- Binary: deployed executable, either official or patched.

### Relations

- A client can attach to multiple inbounds.
- An inbound contains protocol-specific users/clients.
- Xray converts config/API users into `MemoryUser`.
- Dispatcher sees `MemoryUser` and wraps data path writers.
- Limiter state belongs to `(inboundTag, email, direction)`.
- Device state belongs to `(inboundTag, email, sourceIP)`.

### Invariants

- `speedLimit = 0` means no limiter is attached.
- `deviceLimit = 0` means no device/IP rejection.
- Protobuf field numbers 1, 2, 3 remain unchanged.
- Static config and runtime API must carry equivalent semantics.
- 3x-ui and Xray-core binaries must be version-aligned.

### Category mistakes to avoid

- Treating traffic quota as speed limit. Quota is cumulative bytes; speed is bytes per second.
- Treating Fail2ban IP blocking as Xray-core device enforcement. It is an external enforcement path.
- Treating online stats as proof of rate limiting. It is observation, not shaping.
- Treating official Xray-core as compatible with new fields. It is not until patched.

## Risk register

| Risk | Impact | Mitigation |
| --- | --- | --- |
| Proto renumbering | Breaks API/config compatibility | Append fields only |
| Low limit + large buffer | `WaitN` returns error or disconnects | Chunk writes or set safe burst |
| Shared up/down limiter | User-visible speed lower than expected | Separate per-direction limiters |
| Stale limiter after edit | Old speed remains active | Update/recreate limiter on AddUser/hot apply |
| IP occupancy leak | Users locked out after disconnect | Release on context close; timeout cleanup only as fallback |
| Official binary deployed | UI works but no real limit | Package patched Xray binary and verify version |
| Multi-node mismatch | Some nodes enforce, some do not | Node heartbeat should expose Xray fork/version later |

## Recommended phased implementation

### Phase 0: freeze semantics

- `speedLimit` unit: bytes/s.
- UI display: human readable.
- Enforcement: per direction.
- `limitIp` maps to core `device_limit`.

### Phase 1: Xray-core fork

- Add protobuf fields compatibly.
- Add limiter package with tests.
- Integrate dispatcher.
- Add config parsing for supported protocols.
- Build binary.

### Phase 2: 3x-ui fork

- Add DB/API/frontend field.
- Pass field into generated config and AddUser.
- Show Speed Limit after Traffic on `/panel/clients`.
- Point packaging to patched Xray-core artifact.

### Phase 3: deployment validation

- Deploy patched 3x-ui and Xray-core together.
- Confirm `xray -version` matches patched build.
- Create test client with a low limit.
- Measure throughput.
- Confirm `0` restores unlimited behavior.

## Final recommendation

Proceed, but treat this as a coordinated fork feature, not a small panel change.

The lazy correct implementation is:

1. one new panel field: `speedLimit`;
2. reuse `limitIp` for device limit;
3. one Xray-core enforcement point in dispatcher;
4. one packaging switch to the patched Xray binary;
5. tests only where the feature can break.
