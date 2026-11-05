#!/usr/bin/env bash
set -uo pipefail

cd "$(dirname "$0")"

BIN="${BIN:-/out/mcp-gedcom}"
GEDCOM_FILE="${GEDCOM_FILE:-./gedcom.ged}"

if [ ! -x "$BIN" ]; then
    echo "Binary not found or not executable: $BIN" >&2
    echo "Run this script inside the Docker test stage or set BIN explicitly." >&2
    exit 1
fi

if ! command -v jq >/dev/null 2>&1; then
    echo "jq is required for integration tests." >&2
    exit 1
fi

if [ ! -f "$GEDCOM_FILE" ]; then
    echo "GEDCOM file not found: $GEDCOM_FILE" >&2
    exit 1
fi

PASS=0
FAIL=0

pass() {
    echo "✓ $1"
    PASS=$((PASS + 1))
}

fail() {
    echo "✗ $1"
    FAIL=$((FAIL + 1))
}

run_json() {
    echo "$1" | "$BIN" -gedcom-file "$GEDCOM_FILE" 2>/dev/null
}

check_text() {
    local label="$1" response="$2" pattern="$3"
    if printf '%s' "$response" | jq -r '.result.content[0].text' 2>/dev/null | grep -qi -- "$pattern"; then
        pass "$label"
    else
        fail "$label"
    fi
}

check_text_absent() {
    local label="$1" response="$2" pattern="$3"
    if printf '%s' "$response" | jq -r '.result.content[0].text' 2>/dev/null | grep -qi -- "$pattern"; then
        fail "$label"
    else
        pass "$label"
    fi
}

check_error() {
    local label="$1" response="$2"
    if [ "$(printf '%s' "$response" | jq -r '.result.isError' 2>/dev/null)" = "true" ]; then
        pass "$label"
    else
        fail "$label"
    fi
}

check_error_message() {
    local label="$1" response="$2" pattern="$3"
    if [ "$(printf '%s' "$response" | jq -r '.result.isError' 2>/dev/null)" = "true" ] \
        && printf '%s' "$response" | jq -r '.result.content[0].text' 2>/dev/null | grep -qi -- "$pattern"; then
        pass "$label"
    else
        fail "$label"
    fi
}

check_jq_equals() {
    local label="$1" response="$2" filter="$3" expected="$4"
    if [ "$(printf '%s' "$response" | jq -r "$filter" 2>/dev/null)" = "$expected" ]; then
        pass "$label"
    else
        fail "$label"
    fi
}

check_jq_contains() {
    local label="$1" response="$2" filter="$3" pattern="$4"
    if printf '%s' "$response" | jq -r "$filter" 2>/dev/null | grep -q -- "$pattern"; then
        pass "$label"
    else
        fail "$label"
    fi
}

echo "Running mcp-gedcom tests..."

INIT=$(run_json '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{"roots":{"listChanged":true},"sampling":{}},"clientInfo":{"name":"test","version":"1.0.0"}}}')
check_jq_equals "Initialize" "$INIT" '.result.protocolVersion' "2024-11-05"

TOOLS=$(run_json '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}')
check_jq_equals "List tools (14 tools)" "$TOOLS" '.result.tools | length' "14"

RESPONSE=$(run_json '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"search_person","arguments":{"pattern":"Eugene"}}}')
check_text "search_person Eugene" "$RESPONSE" "Eugene"

RESPONSE=$(run_json '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"search_person","arguments":{"pattern":"Robert Eugene Williams"}}}')
check_text "search_person Robert Eugene Williams" "$RESPONSE" "Robert Eugene Williams"

RESPONSE=$(run_json '{"jsonrpc":"2.0","id":25,"method":"tools/call","params":{"name":"search_person","arguments":{"pattern":"Williams","birthYear":1910}}}')
check_text "search_person with birthYear 1910" "$RESPONSE" "Williams"

RESPONSE=$(run_json '{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"get_person_details","arguments":{"id":"I1"}}}')
check_text "get_person_details I1" "$RESPONSE" "williams"

RESPONSE=$(run_json '{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"get_person_details","arguments":{"id":"@I1@"}}}')
check_text "get_person_details @I1@" "$RESPONSE" "williams"

RESPONSE=$(run_json '{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"get_person_details","arguments":{"id":"I1"}}}')
check_text "get_person_details returns relatives" "$RESPONSE" "Spouse"

RESPONSE=$(run_json '{"jsonrpc":"2.0","id":8,"method":"tools/call","params":{"name":"get_family_details","arguments":{"id":"F1"}}}')
check_text "get_family_details" "$RESPONSE" "Robert Eugene"

RESPONSE=$(run_json '{"jsonrpc":"2.0","id":9,"method":"tools/call","params":{"name":"get_family_details","arguments":{"id":"F999"}}}')
check_error "get_family_details not found" "$RESPONSE"

RESPONSE=$(run_json '{"jsonrpc":"2.0","id":9,"method":"tools/call","params":{"name":"search_by_date_range","arguments":{"startYear":1800,"endYear":1850,"event":"birth"}}}')
check_text "search_by_date_range birth 1800-1850" "$RESPONSE" "Williams"

RESPONSE=$(run_json '{"jsonrpc":"2.0","id":10,"method":"tools/call","params":{"name":"search_by_date_range","arguments":{"startYear":1900,"endYear":1950,"event":"death"}}}')
check_text "search_by_date_range death 1900-1950" "$RESPONSE" "Individuals found"

RESPONSE=$(run_json '{"jsonrpc":"2.0","id":11,"method":"tools/call","params":{"name":"search_person","arguments":{"pattern":"NonExistentPerson"}}}')
check_error "Not found returns isError" "$RESPONSE"

RESPONSE=$(run_json '{"jsonrpc":"2.0","id":16,"method":"tools/call","params":{"name":"search_surnames","arguments":{"pattern":"Wil"}}}')
check_text "search_surnames Wil" "$RESPONSE" "Williams"

RESPONSE=$(run_json '{"jsonrpc":"2.0","id":17,"method":"tools/call","params":{"name":"search_surnames","arguments":{"pattern":"Wil","offset":0,"limit":10}}}')
check_jq_equals "search_surnames pagination" "$RESPONSE" '.result.structuredContent.pagination.total' "2"

RESPONSE=$(run_json '{"jsonrpc":"2.0","id":18,"method":"tools/call","params":{"name":"search_person","arguments":{}}}')
check_error_message "search_person missing pattern returns error" "$RESPONSE" "Missing required parameter: pattern"

RESPONSE=$(run_json '{"jsonrpc":"2.0","id":19,"method":"tools/call","params":{"name":"search_surnames","arguments":{}}}')
check_error_message "search_surnames missing pattern returns error" "$RESPONSE" "Missing required parameter: pattern"

RESPONSE=$(run_json '{"jsonrpc":"2.0","id":20,"method":"tools/call","params":{"name":"get_person_details","arguments":{}}}')
check_error_message "get_person_details missing id returns error" "$RESPONSE" "Missing required parameter: id"

RESPONSE=$(run_json '{"jsonrpc":"2.0","id":26,"method":"tools/call","params":{"name":"get_person_details","arguments":{"id":"I1","withSpouse":false}}}')
check_text_absent "get_person_details without spouse" "$RESPONSE" "- Spouse:"

RESPONSE=$(run_json '{"jsonrpc":"2.0","id":27,"method":"tools/call","params":{"name":"get_person_details","arguments":{"id":"I1","withChildren":false}}}')
check_text_absent "get_person_details without children" "$RESPONSE" "- Children:"

RESPONSE=$(run_json '{"jsonrpc":"2.0","id":28,"method":"tools/call","params":{"name":"get_relatives","arguments":{"id":"I1"}}}')
check_text "get_relatives" "$RESPONSE" "Relatives of"

RESPONSE=$(run_json '{"jsonrpc":"2.0","id":29,"method":"tools/call","params":{"name":"find_relationship_path","arguments":{"id1":"I1","id2":"I3"}}}')
check_text "find_relationship_path" "$RESPONSE" "Relationship"

RESPONSE=$(run_json '{"jsonrpc":"2.0","id":30,"method":"tools/call","params":{"name":"find_relationship_path","arguments":{"id1":"I1","id2":"I999"}}}')
check_error "find_relationship_path not found returns error" "$RESPONSE"

RESPONSE=$(run_json '{"jsonrpc":"2.0","id":31,"method":"tools/call","params":{"name":"find_relationship_path","arguments":{"id1":"I1","id2":"I2","AncestorsOnly":false}}}')
check_text "find_relationship_path AncestorsOnly=false" "$RESPONSE" "yes relationship found"

RESPONSE=$(run_json '{"jsonrpc":"2.0","id":32,"method":"tools/call","params":{"name":"find_relationship_path","arguments":{"id1":"I1","id2":"I2","AncestorsOnly":true}}}')
check_text "find_relationship_path AncestorsOnly=true not found" "$RESPONSE" "no relationship found"

RESPONSE=$(run_json '{"jsonrpc":"2.0","id":12,"method":"tools/call","params":{"name":"unknown_tool","arguments":{}}}')
check_error "Unknown tool returns isError" "$RESPONSE"

RESPONSE=$(run_json '{"jsonrpc":"2.0","id":13,"method":"tools/call","params":{"name":"get_children","arguments":{"id":"I1"}}}')
check_text "get_children" "$RESPONSE" "Joe"

RESPONSE=$(run_json '{"jsonrpc":"2.0","id":14,"method":"tools/call","params":{"name":"get_parents","arguments":{"id":"I3"}}}')
check_text "get_parents" "$RESPONSE" "Parents of"

RESPONSE=$(run_json '{"jsonrpc":"2.0","id":24,"method":"tools/call","params":{"name":"get_parents","arguments":{"id":"I3"}}}')
check_jq_contains "get_parents structuredContent has families" "$RESPONSE" '.result.structuredContent.families' "F1"

RESPONSE=$(run_json '{"jsonrpc":"2.0","id":15,"method":"tools/call","params":{"name":"get_children","arguments":{"id":"I3"}}}')
check_text "get_children no children" "$RESPONSE" "No children found"

RESPONSE=$(run_json '{"jsonrpc":"2.0","id":21,"method":"tools/call","params":{"name":"get_ancestors","arguments":{"id":"I1"}}}')
check_text "get_ancestors" "$RESPONSE" "Ancestors of"

RESPONSE=$(run_json '{"jsonrpc":"2.0","id":22,"method":"tools/call","params":{"name":"get_descendants","arguments":{"id":"I1"}}}')
check_text "get_descendants" "$RESPONSE" "Descendants of"

RESPONSE=$(run_json '{"jsonrpc":"2.0","id":23,"method":"tools/call","params":{"name":"get_statistics","arguments":{}}}')
check_text "get_statistics" "$RESPONSE" "Total individuals"

echo ""
echo "Results: $PASS passed, $FAIL failed"

if [ "$FAIL" -gt 0 ]; then
    exit 1
fi
exit 0
