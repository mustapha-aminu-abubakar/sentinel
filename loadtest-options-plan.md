# Loadtest Resource Exhaustion — Options

## Problem

`make loadtest` spins up the full Docker stack (Postgres, Redis, Kafka, Zookeeper, nginx, **3 API replicas**) plus a k6 container — too much for a laptop. The machine freezes before the test completes.

## How results are captured

Artifacts go to `./loadtest/` regardless of how k6 runs:
- **`./loadtest/results.json`** — full time-series metrics (from `--out json=/scripts/results.json`)
- **`./loadtest/report.html`** — exported web dashboard (from `K6_WEB_DASHBOARD_EXPORT`)

So the question is how to get k6 to finish so it writes those files.

---

## Option A — Run k6 locally (recommended first try)

k6 is a single Go binary with negligible CPU/memory. Installing it locally avoids Docker-in-Docker overhead.

```sh
# 1. Install k6
#    Debian/Ubuntu:
sudo apt install k6
#    Or from https://k6.io/docs/get-started/installation/

# 2. Start the stack with 1 API replica (not 3)
docker compose up --build -d --scale sentinel-api=1

# 3. Run k6 locally (not in Docker)
k6 run --out=web-dashboard --out json=./loadtest/results.json loadtest/scenario.js
```

Dashboard at `http://localhost:5660`. Results land in `./loadtest/` on the host directly.

**Pros:** Zero extra containers; 1 API replica halves database contention.
**Cons:** May still freeze at 2000 VUs since the whole stack runs on your laptop.

---

## Option B — Reduce the load profile

If Option A still freezes, edit `loadtest/scenario.js` to lower the peak:

| Current | Reduced |
|---------|---------|
| 2000 VUs peak | 200–500 VUs peak |
| 11 min duration | 3–5 min duration |
| 3 API replicas | 1 API replica |

Then run via Option A. You still get `results.json` + `report.html`.

**Pros:** Quick, no infra needed.
**Cons:** Doesn't hit target scale (10k checks/s), but gives working results.

---

## Option C — Cloud VM

Spin up a cloud VM and run the full stack there. The k6 web dashboard can be accessed via the VM's IP.

```sh
# On the VM:
git clone <repo>
docker compose up --build -d --scale sentinel-api=3
docker compose --profile loadtest up k6
# Then download artifacts: ./loadtest/results.json and ./loadtest/report.html
```

Render is available via MCP if you want to provision infra from here.

**Pros:** No laptop strain; real target scale.
**Cons:** Costs a few dollars; takes time to set up.

---

## Option D — E2E smoke test (quick sanity check)

`scripts/e2e.sh` is a curl-based smoke test that verifies the flow without heavy load.

```sh
make docker-run
make e2e-script
```

Not a load test, but confirms the system works.

---

## Recommendation

1. **Try Option A first** (local k6 + 1 API replica). It's the simplest change and k6 itself is negligible.
2. If it still freezes, **fall back to Option B** (reduce VUs).
3. For genuine scale validation, **Option C** (cloud VM).
