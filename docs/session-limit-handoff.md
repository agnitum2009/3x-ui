# Handoff: Directional Speed Limits and Per-Client Session Limits

> **Currency note (2026-07-25):** This handoff was refreshed after the v3.5.0
> migration. The commit hashes, fork pin, and build steps below reflect the
> current `custom/v3.5.0` branch (based on upstream 3x-ui **v3.5.0** and
> Xray-core **v26.7.11**). For the full migration record see
> `docs/v3.5.0-migration-note.md`; for a quick orientation see `HANDOFF.md` at
> the repo root. Operational behavior (verified results, field semantics,
> upgrade/rollback steps) is unchanged in spirit but the artifacts are newer.

## Scope

This handoff covers the agnitum fork work that adds:

1. Per-client directional speed limits:
   - `upSpeedLimit`: server inbound / client upload limit, bytes per second.
   - `downSpeedLimit`: server outbound / client download limit, bytes per second.
2. Per-client concurrent session limit:
   - `sessionLimit`: maximum active proxied links for one client.
   - `0` means unlimited.
3. 3x-ui panel support for editing, storing, displaying, and emitting these fields into Xray config.

Delivery targets only:

- `https://github.com/agnitum2009/Xray-core`
- `https://github.com/agnitum2009/3x-ui`

Do not push this work to upstream `XTLS/Xray-core` or `MHSanaei/3x-ui` unless explicitly approved later.

## Current GitHub state

### Xray-core fork

Repository:

```text
https://github.com/agnitum2009/Xray-core
```

Branch + required commit:

```text
branch: custom/ag-v26.7.11
commit: 747831e74ddc
```

This commit is a rebase of the original fork work onto upstream Xray-core
**v26.7.11** (commit `50231eaff98c`). The original 0627-era commits
(`5f45a0be11a3` etc.) are superseded — do not build from them.

Important implementation notes (8 files changed vs upstream v26.7.11):

- `common/protocol/user.proto` appends `up_speed_limit = 6`,
  `down_speed_limit = 7`, `session_limit = 8` (append-only, no renumbering).
- `common/protocol/user.pb.go` is regenerated.
- `common/protocol/user.go` round-trips SpeedLimit/UpSpeedLimit/DownSpeedLimit/SessionLimit in `MemoryUser`.
- `common/protocol/user_json.go` parses `speedLimit`/`upSpeedLimit`/`downSpeedLimit`/`sessionLimit` (camelCase) from JSON user configs.
- `app/dispatcher/default.go` enforces:
  - per-direction speed limits through link readers/writers;
  - session limits before outbound dispatch;
  - session counter release on link writer close/error, not on request-context return.
  - The only manual merge needed during the v26.7.11 rebase was here: upstream renamed `stats.GetOrRegisterCounter` from a free function to a stats-manager method; the fork's insertion points did not overlap with that rename.
- Tests: `app/dispatcher/session_limit_test.go`, `common/protocol/user_json_test.go`, `common/protocol/user_session_json_test.go`.

### 3x-ui fork

Repository:

```text
https://github.com/agnitum2009/3x-ui
```

Branch + required commit:

```text
branch: custom/v3.5.0
commit: b508819d
```

This branch is based on upstream 3x-ui **v3.5.0** (`4e928a1c`) plus the 4
migration commits. The `go.mod` replacement points at the v26.7.11-based fork:

```text
replace github.com/xtls/xray-core => github.com/agnitum2009/Xray-core v0.0.0-20260725032009-747831e74ddc
```

## Verified behavior

### Local panel smoke

Local test panel URL:

```text
http://localhost:18080/test/
```

A client update through the panel API wrote:

```json
"sessionLimit": 2
```

The generated Xray config contained the same client-level field.

### Session-limit pressure test

Test shape:

- Temporary VLESS/TCP server with `sessionLimit=2`.
- Temporary SOCKS client forwarding through that server.
- Three concurrent downloads through the same client identity.

Observed result after the fix:

- Two concurrent requests were accepted.
- The third request was rejected before outbound dispatch and received `0` bytes.

Representative rejected result:

```text
curl=3 http=000 bytes=0
TLS unexpected eof while reading
```

### 100-concurrency sample

Test shape:

- Temporary VLESS/TCP server with `sessionLimit=100`.
- 100 concurrent 1 MB downloads through Cloudflare speed endpoint.

Observed sample:

```text
requests=100
ok=86
fail=14
total_bytes=99,471,761
max_cpu=2.9%
max_rss=43,056 KB ~= 42.0 MB
```

The failed requests were timeout failures during the 20 second test window, not an Xray crash.

## Operational meaning of the new fields

### Speed limit fields

The panel displays speed limit values in MB/s for human input, but Xray config stores bytes per second.

- `upSpeedLimit`: server inbound / client upload.
- `downSpeedLimit`: server outbound / client download.
- `speedLimit`: legacy compatibility field; treat as downlink fallback only.
- `0`: unlimited.

### Session limit field

`sessionLimit` limits active proxied links for a single client identity.

It is closer to an active connection/session limit than an HTTP request limit.

Recommended starting values:

```text
Normal user:          50-100
Developer/Codex user: 200-300
Build/CI host:        500
Trusted admin:        0 for unlimited
```

Do not set very low values for developer machines. Five Codex sessions plus browsers, instant messaging, package downloads, and background sync can easily reach 80-200 active connections during bursts.

## Upgrade guide for an already installed 3x-ui website

This guide assumes the common Linux installation layout:

```text
Panel binary: /usr/local/x-ui/x-ui
Panel data:   /etc/x-ui/x-ui.db
Service:      x-ui
Xray binary:  /usr/local/x-ui/bin/xray-linux-<arch>
CLI script:   /usr/bin/x-ui
```

If your installation uses custom paths, inspect them first and substitute the correct paths.

### 0. Read this before starting

This fork currently has Git commits, not a published release archive. The safest production path is:

1. Build the panel and Xray binaries from the agnitum forks.
2. Back up the current installation.
3. Stop the service.
4. Replace only the panel binary and Xray binary.
5. Start the service.
6. Verify the panel, Xray version, generated config, and one test client.

Do not run the upstream installer during this upgrade, because it downloads upstream release assets and may overwrite the forked binaries.

### 1. Identify architecture and current paths

Run on the server:

```bash
uname -m
systemctl status x-ui --no-pager || service x-ui status
command -v x-ui || true
ls -lah /usr/local/x-ui /usr/local/x-ui/bin /etc/x-ui
```

Map `uname -m` to the 3x-ui/Xray binary suffix:

```text
x86_64 / amd64       -> amd64
arm64 / aarch64      -> arm64
armv7 / armv7l       -> armv7
armv6                -> armv6
386 / i386 / i686    -> 386
```

For most VPS servers, the suffix is `amd64` and the Xray binary is:

```text
/usr/local/x-ui/bin/xray-linux-amd64
```

### 2. Back up the live installation

Run as root:

```bash
set -euo pipefail
TS="$(date +%Y%m%d-%H%M%S)"
BACKUP="/root/x-ui-backup-$TS"
mkdir -p "$BACKUP"

cp -a /usr/local/x-ui "$BACKUP/usr-local-x-ui"
cp -a /etc/x-ui "$BACKUP/etc-x-ui"

if [ -f /etc/default/x-ui ]; then cp -a /etc/default/x-ui "$BACKUP/etc-default-x-ui"; fi
if [ -f /etc/systemd/system/x-ui.service ]; then cp -a /etc/systemd/system/x-ui.service "$BACKUP/x-ui.service"; fi
if [ -f /usr/bin/x-ui ]; then cp -a /usr/bin/x-ui "$BACKUP/usr-bin-x-ui"; fi

echo "Backup created at: $BACKUP"
```

Keep this backup until the upgraded panel has been verified under real traffic.

### 3. Build the forked binaries

You can build on the production server or on another Linux machine with the same CPU architecture.

Required tools:

```text
Go compatible with this repository
Node.js >= 22
npm >= 10
git
```

#### Build Xray-core

```bash
set -euo pipefail
mkdir -p /opt/agnitum-build
cd /opt/agnitum-build

rm -rf Xray-core
git clone https://github.com/agnitum2009/Xray-core.git
cd Xray-core
git checkout custom/ag-v26.7.11   # commit 747831e74ddc, based on upstream v26.7.11

GOPROXY=https://goproxy.cn,direct go build -o /opt/agnitum-build/xray-linux-$(case "$(uname -m)" in x86_64|amd64) echo amd64;; aarch64|arm64) echo arm64;; armv7*|armv7l) echo armv7;; armv6*) echo armv6;; i386|i686) echo 386;; *) uname -m;; esac) ./main
```

Verify:

```bash
/opt/agnitum-build/xray-linux-* version | head -3
```

#### Build 3x-ui

```bash
set -euo pipefail
cd /opt/agnitum-build

rm -rf 3x-ui
git clone https://github.com/agnitum2009/3x-ui.git
cd 3x-ui
git checkout custom/v3.5.0   # commit b508819d, based on upstream v3.5.0

# Confirm the fork replace pin is present
grep 'agnitum2009/Xray-core v0.0.0-20260725032009-747831e74ddc' go.mod

npm --prefix frontend ci
npm --prefix frontend run build
GONOSUMDB=github.com/agnitum2009/Xray-core GOPROXY=https://goproxy.cn,direct go build -o /opt/agnitum-build/x-ui .
```

Verify:

```bash
ls -lah /opt/agnitum-build/x-ui /opt/agnitum-build/xray-linux-*
```

### 4. Stop the existing service

```bash
systemctl stop x-ui 2>/dev/null || service x-ui stop
```

Confirm it is stopped:

```bash
pgrep -af 'x-ui|xray' || true
```

If there are non-3x-ui Xray processes on the same host, do not kill them blindly. Only stop the process that belongs to `/usr/local/x-ui`.

### 5. Replace binaries

Set the architecture suffix first:

```bash
ARCH="amd64"   # change this if your server is not amd64
```

Install the panel binary:

```bash
install -m 755 /opt/agnitum-build/x-ui /usr/local/x-ui/x-ui
```

Install the patched Xray binary:

```bash
install -m 755 "/opt/agnitum-build/xray-linux-$ARCH" "/usr/local/x-ui/bin/xray-linux-$ARCH"
```

If your installation also has `/usr/local/x-ui/bin/xray`, update it too:

```bash
if [ -e /usr/local/x-ui/bin/xray ]; then
  install -m 755 "/opt/agnitum-build/xray-linux-$ARCH" /usr/local/x-ui/bin/xray
fi
```

Do not delete `/etc/x-ui/x-ui.db`.

### 6. Start the service

```bash
systemctl start x-ui 2>/dev/null || service x-ui start
sleep 3
systemctl status x-ui --no-pager || service x-ui status
```

### 7. Verify the upgraded panel

Check the panel settings:

```bash
/usr/local/x-ui/x-ui setting -show true || x-ui settings
```

Open the existing panel URL in the browser and log in.

Expected UI behavior on the client edit page:

- Upload speed limit is separate from download speed limit.
- Session limit is a separate field.
- `0` means unlimited.
- Invalid numeric input falls back to `0`.

### 8. Verify Xray uses the patched binary

Find the active Xray process:

```bash
pgrep -af '/usr/local/x-ui/bin/xray|xray-linux'
```

Check version:

```bash
/usr/local/x-ui/bin/xray-linux-$ARCH version | head -3
```

The version line should report **`Xray 26.7.11`** (the upstream base of this
fork). The important part is that the binary was built from
`agnitum2009/Xray-core` branch `custom/ag-v26.7.11` (commit `747831e74ddc`).

### 9. Verify generated Xray config includes limits

Create or edit one test client in the panel:

```text
upSpeedLimit:   any nonzero value if testing upload throttling
downSpeedLimit: any nonzero value if testing download throttling
sessionLimit:   2
```

Restart Xray from the panel or run:

```bash
systemctl restart x-ui
```

Then inspect the generated Xray config. The exact config path depends on runtime settings, but common locations are under `/usr/local/x-ui/bin` or the configured XUI bin folder.

Try:

```bash
find /usr/local/x-ui -name 'config.json' -print
```

Then:

```bash
jq '.. | objects | select(.email? == "YOUR_TEST_CLIENT_EMAIL") | {email, upSpeedLimit, downSpeedLimit, speedLimit, sessionLimit}' /usr/local/x-ui/bin/config.json
```

Expected shape:

```json
{
  "email": "YOUR_TEST_CLIENT_EMAIL",
  "upSpeedLimit": 1048576,
  "downSpeedLimit": 1048576,
  "speedLimit": 1048576,
  "sessionLimit": 2
}
```

### 10. Verify session-limit behavior

A simple production-safe check:

1. Pick a test client, not a real user.
2. Set `sessionLimit=2`.
3. Connect with that client.
4. Start three large downloads at the same time through that client.
5. Expected result: two active transfers continue; the third connection is refused/closed early.

Do not run a large pressure test on a production server during peak traffic.

### 11. Rollback

If the panel fails to start or proxy traffic breaks, restore the backup created in step 2:

```bash
systemctl stop x-ui 2>/dev/null || service x-ui stop

BACKUP="/root/x-ui-backup-YYYYMMDD-HHMMSS"  # replace with the actual backup path
rm -rf /usr/local/x-ui
cp -a "$BACKUP/usr-local-x-ui" /usr/local/x-ui

rm -rf /etc/x-ui
cp -a "$BACKUP/etc-x-ui" /etc/x-ui

if [ -f "$BACKUP/etc-default-x-ui" ]; then cp -a "$BACKUP/etc-default-x-ui" /etc/default/x-ui; fi
if [ -f "$BACKUP/x-ui.service" ]; then cp -a "$BACKUP/x-ui.service" /etc/systemd/system/x-ui.service; systemctl daemon-reload; fi
if [ -f "$BACKUP/usr-bin-x-ui" ]; then cp -a "$BACKUP/usr-bin-x-ui" /usr/bin/x-ui; chmod +x /usr/bin/x-ui; fi

systemctl start x-ui 2>/dev/null || service x-ui start
```

## Notes for future development

- Keep speed limits and session limits separate. They control different resource axes.
- Do not merge upload and download speed limits into one input.
- Keep `0` as the only unlimited value.
- Session limit enforcement must happen before outbound dispatch.
- Session counters must remain active until the proxied link closes or errors.
- If a release artifact is created later, update this handoff to replace the source-build upgrade path with release-download commands.

### v3.5.0-specific constraints (added during the migration)

- **React Hook Form**: `ClientFormModal.tsx` migrated to RHF in v3.5.0. New
  form fields must go through `<FormField>` and be added to both the
  `safeParse` whitelist and the `clientPayload` whitelist in `onSubmit`.
- **0-value persistence**: clearing a limit (set to 0 = unlimited) over a
  previously-set value requires `UpdateColumns` (see `client_crud.go Update`);
  a plain save will not write the zero.
- **Downlink fallback consistency**: the `DownSpeedLimit == 0 → use SpeedLimit`
  fallback is applied in 5 places (model helper, xray.go, api.go AddUser,
  client_crud Update, frontend edit-load). Keep them in sync.
- **protobuf append-only**: fork proto field numbers 6/7/8 must never be
  renumbered, or wire compatibility with deployed cores breaks.
- **Generated artifacts**: after any Go model change that surfaces in the API,
  run `npm run gen` (regenerates `openapi.json` + `src/generated/*`) before
  `npm run build`. Never hand-merge those files.

## Known gaps

- No official GitHub release artifact exists yet for the agnitum fork build.
- The 100-concurrency sample used an external Cloudflare endpoint, so network variance affected completion count.
- Full `infra/conf` Xray test package was not run because this checkout lacks `resources/geoip.dat`; targeted VLESS config tests were run instead.
- The v3.5.0 upgrade guide above was written from the v3.4.2 guide and updated
  for the new commit/pin values, but has **not been re-run end-to-end on a live
  server** since the v3.5.0 migration. The panel built and tested cleanly in
  CI (go build/test, npm build/test all green), but a production smoke test is
  still recommended before deploying.
