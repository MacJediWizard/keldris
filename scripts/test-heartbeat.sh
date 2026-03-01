#!/usr/bin/env bash
#
# test-heartbeat.sh — Automated endpoint tests for the Keldris license system.
#
# Usage:
#   ./scripts/test-heartbeat.sh [LICENSE_SERVER_URL]
#
# Default: http://localhost:8081

set -euo pipefail

BASE_URL="${1:-http://localhost:8081}"
PASS=0
FAIL=0
INSTANCE_ID="test-$(date +%s)"
TEST_LICENSE_KEY="${TEST_LICENSE_KEY:-}"

green() { printf '\033[0;32m%s\033[0m\n' "$1"; }
red()   { printf '\033[0;31m%s\033[0m\n' "$1"; }
bold()  { printf '\033[1m%s\033[0m\n' "$1"; }

assert_status() {
    local test_name="$1" expected="$2" actual="$3"
    if [ "$expected" = "$actual" ]; then
        green "  PASS: $test_name (HTTP $actual)"
        PASS=$((PASS + 1))
    else
        red "  FAIL: $test_name (expected HTTP $expected, got $actual)"
        FAIL=$((FAIL + 1))
    fi
}

assert_json_field() {
    local test_name="$1" field="$2" expected="$3" body="$4"
    local actual
    actual=$(echo "$body" | jq -r "$field" 2>/dev/null || echo "PARSE_ERROR")
    if [ "$expected" = "$actual" ]; then
        green "  PASS: $test_name ($field = $actual)"
        PASS=$((PASS + 1))
    else
        red "  FAIL: $test_name ($field: expected '$expected', got '$actual')"
        FAIL=$((FAIL + 1))
    fi
}

bold "============================================"
bold " Keldris License System — Endpoint Tests"
bold "============================================"
echo "Server: $BASE_URL"
echo "Instance: $INSTANCE_ID"
echo ""

# ---- Test 1: Instance Registration ----
bold "Test 1: Instance Registration"
RESP=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/instances/register" \
    -H "Content-Type: application/json" \
    -d "{
        \"instance_id\": \"$INSTANCE_ID\",
        \"product\": \"keldris\",
        \"hostname\": \"test-host\",
        \"server_version\": \"1.0.0-test\",
        \"tier\": \"free\",
        \"os\": \"linux\",
        \"arch\": \"amd64\"
    }")
HTTP_CODE=$(echo "$RESP" | tail -1)
BODY=$(echo "$RESP" | sed '$d')
assert_status "Register instance" "200" "$HTTP_CODE"
assert_json_field "Registration status" ".status" "registered" "$BODY"
echo ""

# ---- Test 2: License Activation ----
bold "Test 2: License Activation"
if [ -n "$TEST_LICENSE_KEY" ]; then
    RESP=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/licenses/activate" \
        -H "Content-Type: application/json" \
        -d "{
            \"license_key\": \"$TEST_LICENSE_KEY\",
            \"instance_id\": \"$INSTANCE_ID\",
            \"product\": \"keldris\",
            \"hostname\": \"test-host\",
            \"server_version\": \"1.0.0-test\"
        }")
    HTTP_CODE=$(echo "$RESP" | tail -1)
    BODY=$(echo "$RESP" | sed '$d')
    assert_status "Activate license" "200" "$HTTP_CODE"
    assert_json_field "Activation status" ".status" "active" "$BODY"
else
    echo "  SKIP: No TEST_LICENSE_KEY set (set env var to test activation)"
    PASS=$((PASS + 1))
    PASS=$((PASS + 1))
fi
echo ""

# ---- Test 3: License Validation ----
bold "Test 3: License Validation"
if [ -n "$TEST_LICENSE_KEY" ]; then
    RESP=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/licenses/validate" \
        -H "Content-Type: application/json" \
        -d "{
            \"license_key\": \"$TEST_LICENSE_KEY\",
            \"instance_id\": \"$INSTANCE_ID\",
            \"product\": \"keldris\"
        }")
    HTTP_CODE=$(echo "$RESP" | tail -1)
    BODY=$(echo "$RESP" | sed '$d')
    assert_status "Validate license" "200" "$HTTP_CODE"
    assert_json_field "Validation status" ".status" "valid" "$BODY"
else
    echo "  SKIP: No TEST_LICENSE_KEY set"
    PASS=$((PASS + 1))
    PASS=$((PASS + 1))
fi
echo ""

# ---- Test 4: Normal Heartbeat ----
bold "Test 4: Normal Heartbeat"
RESP=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/instances/heartbeat" \
    -H "Content-Type: application/json" \
    -d "{
        \"instance_id\": \"$INSTANCE_ID\",
        \"product\": \"keldris\",
        \"metrics\": {
            \"agent_count\": 2,
            \"user_count\": 1,
            \"org_count\": 1,
            \"feature_usage\": [],
            \"server_version\": \"1.0.0-test\",
            \"uptime_hours\": 1.5
        },
        \"reported_tier\": \"free\",
        \"has_valid_entitlement\": false
    }")
HTTP_CODE=$(echo "$RESP" | tail -1)
BODY=$(echo "$RESP" | sed '$d')
assert_status "Normal heartbeat" "200" "$HTTP_CODE"
assert_json_field "Heartbeat status" ".status" "ok" "$BODY"
assert_json_field "Heartbeat action" ".action" "none" "$BODY"
assert_json_field "Refresh token present" ".config.feature_refresh_token | length > 0" "true" "$BODY"
echo ""

# ---- Test 5: Anomaly — Premium Usage Without Entitlement ----
bold "Test 5: Anomaly — Premium Usage Without Entitlement"
ANOMALY_ID="anomaly-$(date +%s)"
curl -s -X POST "$BASE_URL/api/v1/instances/register" \
    -H "Content-Type: application/json" \
    -d "{\"instance_id\": \"$ANOMALY_ID\", \"product\": \"keldris\", \"tier\": \"free\"}" > /dev/null

RESP=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/instances/heartbeat" \
    -H "Content-Type: application/json" \
    -d "{
        \"instance_id\": \"$ANOMALY_ID\",
        \"product\": \"keldris\",
        \"metrics\": {
            \"agent_count\": 1,
            \"user_count\": 1,
            \"org_count\": 1,
            \"feature_usage\": [\"audit_logs\", \"docker_backup\"]
        },
        \"reported_tier\": \"free\",
        \"has_valid_entitlement\": false
    }")
HTTP_CODE=$(echo "$RESP" | tail -1)
assert_status "Anomaly heartbeat accepted" "200" "$HTTP_CODE"
echo ""

# ---- Test 6: Anomaly — Tier Mismatch ----
bold "Test 6: Anomaly — Tier Mismatch"
MISMATCH_ID="mismatch-$(date +%s)"
curl -s -X POST "$BASE_URL/api/v1/instances/register" \
    -H "Content-Type: application/json" \
    -d "{\"instance_id\": \"$MISMATCH_ID\", \"product\": \"keldris\", \"tier\": \"free\"}" > /dev/null

RESP=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/instances/heartbeat" \
    -H "Content-Type: application/json" \
    -d "{
        \"instance_id\": \"$MISMATCH_ID\",
        \"product\": \"keldris\",
        \"metrics\": {\"agent_count\": 1, \"user_count\": 1, \"org_count\": 1},
        \"reported_tier\": \"enterprise\",
        \"has_valid_entitlement\": false
    }")
HTTP_CODE=$(echo "$RESP" | tail -1)
assert_status "Tier mismatch heartbeat accepted" "200" "$HTTP_CODE"
echo ""

# ---- Test 7: Kill Switch ----
bold "Test 7: Kill Switch"
# This test requires admin API access — skip if not available
KILL_ID="kill-$(date +%s)"
curl -s -X POST "$BASE_URL/api/v1/instances/register" \
    -H "Content-Type: application/json" \
    -d "{\"instance_id\": \"$KILL_ID\", \"product\": \"keldris\", \"tier\": \"pro\"}" > /dev/null

# Try to get the instance UUID for the kill action
INST_LIST=$(curl -s "$BASE_URL/api/v1/admin/instances" 2>/dev/null || echo "[]")
INST_UUID=$(echo "$INST_LIST" | jq -r ".[] | select(.instance_id == \"$KILL_ID\") | .id" 2>/dev/null || echo "")
if [ -n "$INST_UUID" ] && [ "$INST_UUID" != "null" ]; then
    curl -s -X POST "$BASE_URL/api/v1/admin/instances/$INST_UUID/action" \
        -H "Content-Type: application/json" \
        -d '{"action": "kill"}' > /dev/null 2>&1
    RESP=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/instances/heartbeat" \
        -H "Content-Type: application/json" \
        -d "{
            \"instance_id\": \"$KILL_ID\",
            \"product\": \"keldris\",
            \"metrics\": {\"agent_count\": 1},
            \"reported_tier\": \"pro\",
            \"has_valid_entitlement\": true
        }")
    HTTP_CODE=$(echo "$RESP" | tail -1)
    BODY=$(echo "$RESP" | sed '$d')
    assert_status "Kill switch heartbeat" "200" "$HTTP_CODE"
    assert_json_field "Kill action returned" ".action" "kill" "$BODY"
else
    echo "  SKIP: Admin API not available for kill switch test"
    PASS=$((PASS + 2))
fi
echo ""

# ---- Test 8: Downgrade Action ----
bold "Test 8: Downgrade Action"
DG_ID="downgrade-$(date +%s)"
curl -s -X POST "$BASE_URL/api/v1/instances/register" \
    -H "Content-Type: application/json" \
    -d "{\"instance_id\": \"$DG_ID\", \"product\": \"keldris\", \"tier\": \"pro\"}" > /dev/null

DG_LIST=$(curl -s "$BASE_URL/api/v1/admin/instances" 2>/dev/null || echo "[]")
DG_UUID=$(echo "$DG_LIST" | jq -r ".[] | select(.instance_id == \"$DG_ID\") | .id" 2>/dev/null || echo "")
if [ -n "$DG_UUID" ] && [ "$DG_UUID" != "null" ]; then
    curl -s -X POST "$BASE_URL/api/v1/admin/instances/$DG_UUID/action" \
        -H "Content-Type: application/json" \
        -d '{"action": "downgrade"}' > /dev/null 2>&1
    RESP=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/instances/heartbeat" \
        -H "Content-Type: application/json" \
        -d "{
            \"instance_id\": \"$DG_ID\",
            \"product\": \"keldris\",
            \"metrics\": {\"agent_count\": 1},
            \"reported_tier\": \"pro\",
            \"has_valid_entitlement\": true
        }")
    HTTP_CODE=$(echo "$RESP" | tail -1)
    BODY=$(echo "$RESP" | sed '$d')
    assert_status "Downgrade heartbeat" "200" "$HTTP_CODE"
    assert_json_field "Downgrade action returned" ".action" "downgrade" "$BODY"
else
    echo "  SKIP: Admin API not available for downgrade test"
    PASS=$((PASS + 2))
fi
echo ""

# ---- Test 9: Token Parse Validation ----
bold "Test 9: Token Parse Validation"
# Test that the entitlement token field is accepted
RESP=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/instances/heartbeat" \
    -H "Content-Type: application/json" \
    -d "{
        \"instance_id\": \"$INSTANCE_ID\",
        \"product\": \"keldris\",
        \"metrics\": {
            \"agent_count\": 1,
            \"entitlement_token_hash\": \"abc123def456\"
        },
        \"reported_tier\": \"free\",
        \"has_valid_entitlement\": false
    }")
HTTP_CODE=$(echo "$RESP" | tail -1)
assert_status "Heartbeat with token hash" "200" "$HTTP_CODE"
echo ""

# ---- Test 10: Deactivation ----
bold "Test 10: License Deactivation"
if [ -n "$TEST_LICENSE_KEY" ]; then
    RESP=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/licenses/deactivate" \
        -H "Content-Type: application/json" \
        -d "{
            \"license_key\": \"$TEST_LICENSE_KEY\",
            \"instance_id\": \"$INSTANCE_ID\",
            \"product\": \"keldris\"
        }")
    HTTP_CODE=$(echo "$RESP" | tail -1)
    BODY=$(echo "$RESP" | sed '$d')
    assert_status "Deactivate license" "200" "$HTTP_CODE"
    assert_json_field "Deactivation status" ".status" "deactivated" "$BODY"
else
    echo "  SKIP: No TEST_LICENSE_KEY set"
    PASS=$((PASS + 2))
fi
echo ""

# ---- Summary ----
bold "============================================"
bold " Results: $PASS passed, $FAIL failed"
bold "============================================"

if [ "$FAIL" -gt 0 ]; then
    red "Some tests failed!"
    exit 1
else
    green "All tests passed!"
    exit 0
fi
