#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
LIMIT="${LIMIT:-5}"          # requests_allowed per window
WINDOW="${WINDOW:-60}"       # window_seconds

PASS=0
FAIL=0

pass()  { echo "  PASS: $1"; PASS=$((PASS + 1)); }
fail()  { echo "  FAIL: $1"; FAIL=$((FAIL + 1)); }
check() { if [ "$1" -eq 0 ]; then pass "$2"; else fail "$2"; fi; }

echo "=== Sentinel E2E Smoke Test ==="
echo "Target: $BASE_URL  Limit: $LIMIT req / ${WINDOW}s window"
echo ""

# ------------------------------------------------------------------
# 1. Health check
# ------------------------------------------------------------------
echo "--- 1. Health ---"
status=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/health")
check $(( status != 200 )) "GET /health (expected 200, got $status)"

# ------------------------------------------------------------------
# 2. Create a test client
# ------------------------------------------------------------------
echo "--- 2. Create client ---"
client_resp=$(curl -s -X POST "$BASE_URL/clients" \
  -H "Content-Type: application/json" \
  -d '{"name":"e2e-test-client"}')
client_id=$(echo "$client_resp" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])" 2>/dev/null || true)

if [ -z "$client_id" ]; then
  fail "POST /clients — could not extract client_id from: $client_resp"
  echo ""
  echo "=== RESULTS: $PASS passed, $FAIL failed ==="
  exit 1
fi
pass "POST /clients -> client_id=$client_id"

# ------------------------------------------------------------------
# 3. Create a rate rule
# ------------------------------------------------------------------
echo "--- 3. Create rule ---"
rule_resp=$(curl -s -X POST "$BASE_URL/rules" \
  -H "Content-Type: application/json" \
  -d "{\"client_id\":\"$client_id\",\"api\":\"e2e-test\",\"requests_allowed\":$LIMIT,\"window_seconds\":$WINDOW}")
rule_id=$(echo "$rule_resp" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])" 2>/dev/null || true)

if [ -z "$rule_id" ]; then
  fail "POST /rules — could not extract rule_id from: $rule_resp"
else
  pass "POST /rules -> rule_id=$rule_id"
fi

# ------------------------------------------------------------------
# 4. Fire checks — should be allowed (under limit)
# ------------------------------------------------------------------
echo "--- 4. Check requests (under limit) ---"
for i in $(seq 1 $LIMIT); do
  resp=$(curl -s -X POST "$BASE_URL/v1/check" \
    -H "Content-Type: application/json" \
    -d "{\"client_id\":\"$client_id\",\"api\":\"e2e-test\"}")
  allowed=$(echo "$resp" | python3 -c "import sys,json; print(json.load(sys.stdin).get('allowed',''))" 2>/dev/null || true)
  remaining=$(echo "$resp" | python3 -c "import sys,json; print(json.load(sys.stdin).get('remaining',''))" 2>/dev/null || true)

  if [ "$allowed" = "True" ]; then
    pass "check #$i: allowed (remaining=$remaining)"
  else
    fail "check #$i: unexpected — $resp"
  fi
done

# ------------------------------------------------------------------
# 5. Fire another check — should be rate-limited (over limit)
# ------------------------------------------------------------------
echo "--- 5. Check request (over limit) ---"
resp=$(curl -s -X POST "$BASE_URL/v1/check" \
  -H "Content-Type: application/json" \
    -d "{\"client_id\":\"$client_id\",\"api\":\"e2e-test\"}")
allowed=$(echo "$resp" | python3 -c "import sys,json; print(json.load(sys.stdin).get('allowed',''))" 2>/dev/null || true)
retry=$(echo "$resp" | python3 -c "import sys,json; print(json.load(sys.stdin).get('retry_after',''))" 2>/dev/null || true)

if [ "$allowed" = "False" ] && [ -n "$retry" ]; then
  pass "check #$((LIMIT+1)): rate-limited (retry_after=$retry)"
else
  fail "check #$((LIMIT+1)): expected rate-limit, got — $resp"
fi

# ------------------------------------------------------------------
# 6. Query analytics usage endpoint
# ------------------------------------------------------------------
echo "--- 6. Analytics query ---"
sleep 2

analytics_resp=$(curl -s "$BASE_URL/analytics/usage?client_id=$client_id&api=e2e-test&bucket=hour")
analytics_http=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/analytics/usage?client_id=$client_id&api=e2e-test&bucket=hour")

if analytics_count=$(echo "$analytics_resp" | python3 -c "import sys,json; d=json.load(sys.stdin); print(len(d))" 2>/dev/null); then
  pass "GET /analytics/usage -> HTTP $analytics_http, count=$analytics_count"
else
  fail "GET /analytics/usage -> unexpected response (HTTP $analytics_http): $analytics_resp"
fi

# ------------------------------------------------------------------
# Summary
# ------------------------------------------------------------------
echo ""
echo "=== RESULTS: $PASS passed, $FAIL failed ==="
exit $FAIL
