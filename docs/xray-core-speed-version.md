# Patched Xray-core speed-limit artifact

Real per-client speed enforcement requires a patched Xray-core binary. Official
XTLS/Xray-core release binaries do not include this local `speedLimit` /
`deviceLimit` patch.

Current local development source:

- fork target: `agnitum2009/Xray-core`
- upstream base commit: `d7fa207`
- local patched checkout: `/home/agnitum/3xui/xray-core`
- local smoke binary: `/home/agnitum/3xui/xray-core/xray-speed`
- intended release tag for packaging: `v26.6.27-speed`

`DockerInit.sh` defaults to the fork release URL. For local smoke before a
GitHub release exists, run it with:

```bash
XRAY_LOCAL_BIN=/home/agnitum/3xui/xray-core/xray-speed bash DockerInit.sh amd64
./build/bin/xray-linux-amd64 version
```

Rollback:

1. set `XRAY_CORE_REPO=XTLS/Xray-core` and `XRAY_CORE_VERSION=v26.6.27`, or
2. restore the previous official Xray binary in the deployment, then restart
   3x-ui/Xray.

E2E proof still required on a deployed node:

1. create/edit a client with `speedLimit > 0`;
2. connect through that client;
3. measure sustained throughput below the configured bytes/s ceiling;
4. set `speedLimit = 0`, restart/reload the client, and verify throttling is
   removed.
