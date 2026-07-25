# HANDOFF — start here

> **Read this first.** It orients you to the current state of this repo in under
> two minutes and points to the detailed docs. For anything deeper, follow the
> links at the bottom.

## Where you are

- **Branch:** `custom/v3.5.0` (local + `agnitum` remote), HEAD `b508819d`.
- **Base:** upstream **3x-ui v3.5.0** (`4e928a1c`) + 4 migration commits.
- **Git history is healthy:** this branch shares ancestry with `origin/main`,
  so `git merge origin/main` works normally. (The old customization branch had
  a detached history — that was fixed by rebuilding on top of v3.5.0.)

## What's customized (3 features, all absent from upstream)

1. **Per-direction speed limits** — separate upload/download caps per client
   (`upSpeedLimit` / `downSpeedLimit`, MB/s in UI, bytes/s in xray config).
2. **Per-client session limits** — max concurrent proxied links per client
   (`sessionLimit`, integer; 0 = unlimited).
3. **QR/share links use the configured public endpoint** — `preferPublicHost`
   prefers the configured Sub/Web Domain unconditionally so links work even
   when the panel is managed over localhost/internal addresses.

These depend on a **custom Xray-core fork** (next section).

## Xray-core fork dependency

`go.mod` line 120 redirects xray-core to a custom fork that implements the
directional rate limiters and the session counter:

```
replace github.com/xtls/xray-core => github.com/agnitum2009/Xray-core v0.0.0-20260725032009-747831e74ddc
```

- Fork repo: `https://github.com/agnitum2009/Xray-core`, branch
  `custom/ag-v26.7.11`, based on upstream **Xray-core v26.7.11**.
- Local checkout: `/home/umax/work/3xui/xray-core`.
- If upstream xray-core bumps again, rebase this fork (see "Maintenance" in
  `docs/v3.5.0-migration-note.md` — historically only
  `app/dispatcher/default.go` needs a manual merge).

## Critical constraints (don't break these)

1. **0 = unlimited** for all three limit fields. Clearing a limit needs
   `UpdateColumns`, not a plain save (see `client_crud.go`).
2. **Downlink fallback** (`DownSpeedLimit==0 → use SpeedLimit`) must stay
   consistent across 5 places — see the data-flow diagram in
   `docs/v3.5.0-migration-note.md`.
3. **React Hook Form**: `ClientFormModal.tsx` uses RHF. New fields go through
   `<FormField>` + both `onSubmit` whitelists.
4. **protobuf append-only**: fork proto field numbers 6/7/8 never renumber.
5. **Tool configs are not code**: `.trellis/`, `.agents/`, `.codex/`,
   `.zcode/`, `AGENTS.md` are untracked local AI-tooling configs — exclude
   them from all commits and diffs.

## Rollback (at any time)

```
# Back to the pre-migration 3.4.2 custom state (local + remote tag)
git reset --hard backup/pre-v3.5.0-custom

# If the local repo is unusable
git fetch agnitum && git checkout agnitum/tags/backup/pre-v3.5.0-custom

# Drop customizations → vanilla upstream
git checkout v3.5.0   # or v3.4.2
```

`backup/pre-v3.5.0-custom` is an immutable tag (local + `agnitum`). Do **not**
use the stale local `main` branch for rollback.

## Detailed docs

| Doc | Read it for |
|---|---|
| `docs/v3.5.0-migration-note.md` | **Authoritative migration record**: what/why/how-verified, fork pin, file landing points, data flow, env requirements, maintenance. |
| `docs/session-limit-handoff.md` | **Design semantics + ops manual**: field meanings, recommended values, verified test results, full build/upgrade/rollback guide for live servers. |

## Build & verify (quick)

```
# Backend
go build ./...  &&  go test ./internal/database/... ./internal/web/service/... ./internal/xray/...
# Frontend (Node 22.20+ recommended for gen:api strip-types)
cd frontend && npm ci && npm run gen && npm run typecheck && npm run lint && npm run test && npm run build
```

Go 1.26.5 required. See "Environment requirements" in the migration note.
