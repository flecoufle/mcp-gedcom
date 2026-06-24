#!/bin/bash

cd "$(dirname "$0")"

# can be disabled later
rm mcp-gedcom 2>/dev/null

if [ ! -f ./mcp-gedcom ]; then
    echo "Local Building mcp-gedcom..."
    go build -o mcp-gedcom ./cmd/mcp-gedcom/server

    if [ $? -ne 0 ]; then
        echo "Build failed!"
        exit 1
    fi
fi

echo "Running mcp-gedcom tests..."

PASS=0
FAIL=0

run_json() {
    echo "$1" | ./mcp-gedcom 2>/dev/null
}

JSON=$(run_json '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{"roots":{"listChanged":true},"sampling":{}},"clientInfo":{"name":"test","version":"1.0.0"}}}')
if echo "$JSON" | jq -r '.result.protocolVersion' 2>/dev/null | grep -q "2024-11-05"; then
    echo "✓ Initialize"
    ((PASS++))
else
    echo "✗ Initialize"
    ((FAIL++))
fi

TOOLS=$(run_json '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}')
TOOL_COUNT=$(echo "$TOOLS" | jq '.result.tools | length' 2>/dev/null)
if [ "$TOOL_COUNT" = "14" ]; then
    echo "✓ List tools (14 tools)"
    ((PASS++))
else
    echo "✗ List tools (expected 14, got $TOOL_COUNT)"
    ((FAIL++))
fi

SEARCH_WILLIAMS='{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"search_person","arguments":{"pattern":"Eugene"}}}'
RESPONSE=$(run_json "$SEARCH_WILLIAMS")
if echo "$RESPONSE" | jq -r '.result.content[0].text' 2>/dev/null | grep -qi "Eugene"; then
    echo "✓ search_person Eugene"
    ((PASS++))
else
    echo "✗ search_person Eugene"
    ((FAIL++))
fi

SEARCH_WILLIAMS_SURNAME='{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"search_person","arguments":{"pattern":"Robert Eugene Williams"}}}'
RESPONSE=$(run_json "$SEARCH_WILLIAMS_SURNAME")
if echo "$RESPONSE" | jq -r '.result.content[0].text' 2>/dev/null | grep -qi "Robert Eugene Williams"; then
    echo "✓ search_person Robert Eugene Williams"
    ((PASS++))
else
    echo "✗ search_person Robert Eugene Williams"
    ((FAIL++))
fi

SEARCH_PERSON_BIRTHYEAR='{"jsonrpc":"2.0","id":25,"method":"tools/call","params":{"name":"search_person","arguments":{"pattern":"Williams","birthYear":1910}}}'
RESPONSE=$(run_json "$SEARCH_PERSON_BIRTHYEAR")
if echo "$RESPONSE" | jq -r '.result.content[0].text' 2>/dev/null | grep -qi "Williams"; then
    echo "✓ search_person with birthYear 1910"
    ((PASS++))
else
    echo "✗ search_person with birthYear 1910"
    ((FAIL++))
fi

GET_I1='{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"get_person_details","arguments":{"id":"I1"}}}'
RESPONSE=$(run_json "$GET_I1")
if echo "$RESPONSE" | jq -r '.result.content[0].text' 2>/dev/null | grep -qi "williams"; then
    echo "✓ get_person_details I1"
    ((PASS++))
else
    echo "✗ get_person_details I1"
    ((FAIL++))
fi

GET_I1_CURLY='{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"get_person_details","arguments":{"id":"@I1@"}}}'
RESPONSE=$(run_json "$GET_I1_CURLY")
if echo "$RESPONSE" | jq -r '.result.content[0].text' 2>/dev/null | grep -qi "williams"; then
    echo "✓ get_person_details @I1@"
    ((PASS++))
else
    echo "✗ get_person_details @I1@"
    ((FAIL++))
fi

RELATIVES=$(run_json '{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"get_person_details","arguments":{"id":"I1"}}}')
if echo "$RELATIVES" | jq -r '.result.content[0].text' 2>/dev/null | grep -q "Spouse"; then
    echo "✓ get_person_details returns relatives"
    ((PASS++))
else
    echo "✗ get_person_details returns relatives"
    ((FAIL++))
fi

GET_FAMILY='{"jsonrpc":"2.0","id":8,"method":"tools/call","params":{"name":"get_family_details","arguments":{"id":"F1"}}}'
RESPONSE=$(run_json "$GET_FAMILY")
if echo "$RESPONSE" | jq -r '.result.content[0].text' 2>/dev/null | grep -q "Robert Eugene"; then
    echo "✓ get_family_details"
    ((PASS++))
else
    echo "✗ get_family_details"
    ((FAIL++))
fi

GET_FAMILY_NOT_FOUND='{"jsonrpc":"2.0","id":9,"method":"tools/call","params":{"name":"get_family_details","arguments":{"id":"F999"}}}'
RESPONSE=$(run_json "$GET_FAMILY_NOT_FOUND")
IS_ERROR=$(echo "$RESPONSE" | jq -r '.result.isError' 2>/dev/null)
if [ "$IS_ERROR" = "true" ]; then
    echo "✓ get_family_details not found"
    ((PASS++))
else
    echo "✗ get_family_details not found"
    ((FAIL++))
fi

BIRTH_RANGE='{"jsonrpc":"2.0","id":9,"method":"tools/call","params":{"name":"search_by_date_range","arguments":{"startYear":1800,"endYear":1850,"event":"birth"}}}'
RESPONSE=$(run_json "$BIRTH_RANGE")
if echo "$RESPONSE" | jq -r '.result.content[0].text' 2>/dev/null | grep -qi "Williams"; then
    echo "✓ search_by_date_range birth 1800-1850"
    ((PASS++))
else
    echo "✗ search_by_date_range birth 1800-1850"
    ((FAIL++))
fi

DEATH_RANGE='{"jsonrpc":"2.0","id":10,"method":"tools/call","params":{"name":"search_by_date_range","arguments":{"startYear":1900,"endYear":1950,"event":"death"}}}'
RESPONSE=$(run_json "$DEATH_RANGE")
if echo "$RESPONSE" | jq -r '.result.content[0].text' 2>/dev/null | grep -q "Individuals found"; then
    echo "✓ search_by_date_range death 1900-1950"
    ((PASS++))
else
    echo "✗ search_by_date_range death 1900-1950"
    ((FAIL++))
fi

NOT_FOUND='{"jsonrpc":"2.0","id":11,"method":"tools/call","params":{"name":"search_person","arguments":{"pattern":"NonExistentPerson"}}}'
RESPONSE=$(run_json "$NOT_FOUND")
IS_ERROR=$(echo "$RESPONSE" | jq -r '.result.isError' 2>/dev/null)
if [ "$IS_ERROR" = "true" ]; then
    echo "✓ Not found returns isError"
    ((PASS++))
else
    echo "✗ Not found returns isError"
    ((FAIL++))
fi

LASTNAME='{"jsonrpc":"2.0","id":16,"method":"tools/call","params":{"name":"search_surnames","arguments":{"pattern":"Wil"}}}'
RESPONSE=$(run_json "$LASTNAME")
if echo "$RESPONSE" | jq -r '.result.content[0].text' 2>/dev/null | grep -qi "Williams"; then
    echo "✓ search_surnames Wil"
    ((PASS++))
else
    echo "✗ search_surnames Wil"
    ((FAIL++))
fi

LASTNAME_PAGINATED='{"jsonrpc":"2.0","id":17,"method":"tools/call","params":{"name":"search_surnames","arguments":{"pattern":"Wil","offset":0,"limit":10}}}'
RESPONSE=$(run_json "$LASTNAME_PAGINATED")
if echo "$RESPONSE" | jq -r '.result.structuredContent.pagination.total' 2>/dev/null | grep -q "2"; then
    echo "✓ search_surnames pagination"
    ((PASS++))
else
    echo "✗ search_surnames pagination"
    ((FAIL++))
fi

SEARCH_INDIVIDUALS_NO_NAME='{"jsonrpc":"2.0","id":18,"method":"tools/call","params":{"name":"search_person","arguments":{}}}'
RESPONSE=$(run_json "$SEARCH_INDIVIDUALS_NO_NAME")
IS_ERROR=$(echo "$RESPONSE" | jq -r '.result.isError' 2>/dev/null)
if [ "$IS_ERROR" = "true" ] && echo "$RESPONSE" | jq -r '.result.content[0].text' 2>/dev/null | grep -q "Missing required parameter: pattern"; then
    echo "✓ search_person missing pattern returns error"
    ((PASS++))
else
    echo "✗ search_person missing pattern returns error"
    ((FAIL++))
fi

SEARCH_SURNAMES_NO_PATTERN='{"jsonrpc":"2.0","id":19,"method":"tools/call","params":{"name":"search_surnames","arguments":{}}}'
RESPONSE=$(run_json "$SEARCH_SURNAMES_NO_PATTERN")
IS_ERROR=$(echo "$RESPONSE" | jq -r '.result.isError' 2>/dev/null)
if [ "$IS_ERROR" = "true" ] && echo "$RESPONSE" | jq -r '.result.content[0].text' 2>/dev/null | grep -q "Missing required parameter: pattern"; then
    echo "✓ search_surnames missing pattern returns error"
    ((PASS++))
else
    echo "✗ search_surnames missing pattern returns error"
    ((FAIL++))
fi

GET_INDIVIDUAL_NO_ID='{"jsonrpc":"2.0","id":20,"method":"tools/call","params":{"name":"get_person_details","arguments":{}}}'
RESPONSE=$(run_json "$GET_INDIVIDUAL_NO_ID")
IS_ERROR=$(echo "$RESPONSE" | jq -r '.result.isError' 2>/dev/null)
if [ "$IS_ERROR" = "true" ] && echo "$RESPONSE" | jq -r '.result.content[0].text' 2>/dev/null | grep -q "Missing required parameter: id"; then
    echo "✓ get_person_details missing id returns error"
    ((PASS++))
else
    echo "✗ get_person_details missing id returns error"
    ((FAIL++))
fi

GET_PERSON_DETAILS_NO_SPOUSE='{"jsonrpc":"2.0","id":26,"method":"tools/call","params":{"name":"get_person_details","arguments":{"id":"I1","withSpouse":false}}}'
RESPONSE=$(run_json "$GET_PERSON_DETAILS_NO_SPOUSE")
if echo "$RESPONSE" | jq -r '.result.content[0].text' 2>/dev/null | grep -qv "Spouse"; then
    echo "✓ get_person_details without spouse"
    ((PASS++))
else
    echo "✗ get_person_details without spouse"
    ((FAIL++))
fi

GET_PERSON_DETAILS_NO_CHILDREN='{"jsonrpc":"2.0","id":27,"method":"tools/call","params":{"name":"get_person_details","arguments":{"id":"I1","withChildren":false}}}'
RESPONSE=$(run_json "$GET_PERSON_DETAILS_NO_CHILDREN")
if echo "$RESPONSE" | jq -r '.result.content[0].text' 2>/dev/null | grep -qv "Children"; then
    echo "✓ get_person_details without children"
    ((PASS++))
else
    echo "✗ get_person_details without children"
    ((FAIL++))
fi

GET_RELATIVES='{"jsonrpc":"2.0","id":28,"method":"tools/call","params":{"name":"get_relatives","arguments":{"id":"I1"}}}'
RESPONSE=$(run_json "$GET_RELATIVES")
if echo "$RESPONSE" | jq -r '.result.content[0].text' 2>/dev/null | grep -qi "Relatives of"; then
    echo "✓ get_relatives"
    ((PASS++))
else
    echo "✗ get_relatives"
    ((FAIL++))
fi

FIND_PATH='{"jsonrpc":"2.0","id":29,"method":"tools/call","params":{"name":"find_relationship_path","arguments":{"id1":"I1","id2":"I3"}}}'
RESPONSE=$(run_json "$FIND_PATH")
if echo "$RESPONSE" | jq -r '.result.content[0].text' 2>/dev/null | grep -qi "Relationship"; then
    echo "✓ find_relationship_path"
    ((PASS++))
else
    echo "✗ find_relationship_path"
    ((FAIL++))
fi

FIND_PATH_NOT_FOUND='{"jsonrpc":"2.0","id":30,"method":"tools/call","params":{"name":"find_relationship_path","arguments":{"id1":"I1","id2":"I999"}}}'
RESPONSE=$(run_json "$FIND_PATH_NOT_FOUND")
if echo "$RESPONSE" | jq -r '.result.isError' 2>/dev/null | grep -q "true"; then
    echo "✓ find_relationship_path not found returns error"
    ((PASS++))
else
    echo "✗ find_relationship_path not found returns error"
    ((FAIL++))
fi

FIND_PATH_ALL='{"jsonrpc":"2.0","id":31,"method":"tools/call","params":{"name":"find_relationship_path","arguments":{"id1":"I1","id2":"I2","AncestorsOnly":false}}}'
RESPONSE=$(run_json "$FIND_PATH_ALL")
if echo "$RESPONSE" | jq -r '.result.content[0].text' 2>/dev/null | grep -qi "yes relationship found"; then
    echo "✓ find_relationship_path AncestorsOnly=false"
    ((PASS++))
else
    echo "✗ find_relationship_path AncestorsOnly=false"
    ((FAIL++))
fi

FIND_PATH_TRUE='{"jsonrpc":"2.0","id":32,"method":"tools/call","params":{"name":"find_relationship_path","arguments":{"id1":"I1","id2":"I2","AncestorsOnly":true}}}'
RESPONSE=$(run_json "$FIND_PATH_TRUE")
if echo "$RESPONSE" | jq -r '.result.content[0].text' 2>/dev/null | grep -qi "no relationship found"; then
    echo "✓ find_relationship_path AncestorsOnly=true not found"
    ((PASS++))
else
    echo "✗ find_relationship_path AncestorsOnly=true not found"
    ((FAIL++))
fi

UNKNOWN_TOOL='{"jsonrpc":"2.0","id":12,"method":"tools/call","params":{"name":"unknown_tool","arguments":{}}}'
RESPONSE=$(run_json "$UNKNOWN_TOOL")
IS_ERROR=$(echo "$RESPONSE" | jq -r '.result.isError' 2>/dev/null)
if [ "$IS_ERROR" = "true" ]; then
    echo "✓ Unknown tool returns isError"
    ((PASS++))
else
    echo "✗ Unknown tool returns isError"
    ((FAIL++))
fi

GET_CHILDREN='{"jsonrpc":"2.0","id":13,"method":"tools/call","params":{"name":"get_children","arguments":{"id":"I1"}}}'
RESPONSE=$(run_json "$GET_CHILDREN")
if echo "$RESPONSE" | jq -r '.result.content[0].text' 2>/dev/null | grep -qi "Joe"; then
    echo "✓ get_children"
    ((PASS++))
else
    echo "✗ get_children"
    ((FAIL++))
fi

GET_PARENTS='{"jsonrpc":"2.0","id":14,"method":"tools/call","params":{"name":"get_parents","arguments":{"id":"I3"}}}'
RESPONSE=$(run_json "$GET_PARENTS")
if echo "$RESPONSE" | jq -r '.result.content[0].text' 2>/dev/null | grep -q "Parents of"; then
    echo "✓ get_parents"
    ((PASS++))
else
    echo "✗ get_parents"
    ((FAIL++))
fi

GET_PARENTS_STRUCTURED='{"jsonrpc":"2.0","id":24,"method":"tools/call","params":{"name":"get_parents","arguments":{"id":"I3"}}}'
RESPONSE=$(run_json "$GET_PARENTS_STRUCTURED")
if echo "$RESPONSE" | jq -r '.result.structuredContent.families' 2>/dev/null | grep -q "F1"; then
    echo "✓ get_parents structuredContent has families"
    ((PASS++))
else
    echo "✗ get_parents structuredContent has families"
    ((FAIL++))
fi

GET_CHILDREN_NOT_FOUND='{"jsonrpc":"2.0","id":15,"method":"tools/call","params":{"name":"get_children","arguments":{"id":"I3"}}}'
RESPONSE=$(run_json "$GET_CHILDREN_NOT_FOUND")
if echo "$RESPONSE" | jq -r '.result.content[0].text' 2>/dev/null | grep -q "No children found"; then
    echo "✓ get_children no children"
    ((PASS++))
else
    echo "✗ get_children no children"
    ((FAIL++))
fi

GET_ANCESTORS='{"jsonrpc":"2.0","id":21,"method":"tools/call","params":{"name":"get_ancestors","arguments":{"id":"I1"}}}'
RESPONSE=$(run_json "$GET_ANCESTORS")
if echo "$RESPONSE" | jq -r '.result.content[0].text' 2>/dev/null | grep -q "Ancestors of"; then
    echo "✓ get_ancestors"
    ((PASS++))
else
    echo "✗ get_ancestors"
    ((FAIL++))
fi

GET_DESCENDANTS='{"jsonrpc":"2.0","id":22,"method":"tools/call","params":{"name":"get_descendants","arguments":{"id":"I1"}}}'
RESPONSE=$(run_json "$GET_DESCENDANTS")
if echo "$RESPONSE" | jq -r '.result.content[0].text' 2>/dev/null | grep -q "Descendants of"; then
    echo "✓ get_descendant"
    ((PASS++))
else
    echo "✗ get_descendants"
    ((FAIL++))
fi

GET_STATISTICS='{"jsonrpc":"2.0","id":23,"method":"tools/call","params":{"name":"get_statistics","arguments":{}}}'
RESPONSE=$(run_json "$GET_STATISTICS")
if echo "$RESPONSE" | jq -r '.result.content[0].text' 2>/dev/null | grep -q "Total individuals"; then
    echo "✓ get_statistics"
    ((PASS++))
else
    echo "✗ get_statistics"
    ((FAIL++))
fi

GET_ALL_RELATIONSHIPS='{"jsonrpc":"2.0","id":33,"method":"tools/call","params":{"name":"find_all_relationships","arguments":{"id1":"I9393","id2":"I9251"}}}'
RESPONSE=$(echo "$GET_ALL_RELATIONSHIPS" | ./mcp-gedcom -gedcom-path /home/another/github/ 2>/dev/null)
TEXT=$(echo "$RESPONSE" | jq -r '.result.content[0].text' 2>/dev/null)
if echo "$TEXT" | grep -q "Parenté"; then
    echo "✓ find_all_relationships"
    ((PASS++))
else
    echo "✗ find_all_relationships"
    ((FAIL++))
fi

LOAD_GEDCOM='{"jsonrpc":"2.0","id":35,"method":"tools/call","params":{"name":"load_gedcom_file","arguments":{"path":"./sample/simpsons.ged"}}}'
RESPONSE=$(echo "$LOAD_GEDCOM" | ./mcp-gedcom -gedcom-path /home/another/github/ 2>/dev/null)
TEXT=$(echo "$RESPONSE" | jq -r '.result.content[0].text' 2>/dev/null)
if echo "$TEXT" | grep -q "GEDCOM file loaded" && echo "$RESPONSE" | jq -r '.result.structuredContent.total_individuals' 2>/dev/null | grep -q "11"; then
    echo "✓ load_gedcom_file simpsons.ged"
    ((PASS++))
else
    echo "✗ load_gedcom_file simpsons.ged"
    ((FAIL++))
fi

LOAD_GEDCOM_MISSING_PATH='{"jsonrpc":"2.0","id":36,"method":"tools/call","params":{"name":"load_gedcom_file","arguments":{}}}'
RESPONSE=$(echo "$LOAD_GEDCOM_MISSING_PATH" | ./mcp-gedcom -gedcom-path /home/another/github/ 2>/dev/null)
IS_ERROR=$(echo "$RESPONSE" | jq -r '.result.isError' 2>/dev/null)
if [ "$IS_ERROR" = "true" ] && echo "$RESPONSE" | jq -r '.result.content[0].text' 2>/dev/null | grep -q "Missing required parameter: path"; then
    echo "✓ load_gedcom_file missing path returns error"
    ((PASS++))
else
    echo "✗ load_gedcom_file missing path returns error"
    ((FAIL++))
fi

LOAD_GEDCOM_BAD_PATH='{"jsonrpc":"2.0","id":37,"method":"tools/call","params":{"name":"load_gedcom_file","arguments":{"path":"./nonexistent.ged"}}}'
RESPONSE=$(echo "$LOAD_GEDCOM_BAD_PATH" | ./mcp-gedcom -gedcom-path /home/another/github/ 2>/dev/null)
IS_ERROR=$(echo "$RESPONSE" | jq -r '.result.isError' 2>/dev/null)
if [ "$IS_ERROR" = "true" ]; then
    echo "✓ load_gedcom_file bad path returns error"
    ((PASS++))
else
    echo "✗ load_gedcom_file bad path returns error"
    ((FAIL++))
fi

CHECK_MULTIPLE_LINKS='{"jsonrpc":"2.0","id":34,"method":"tools/call","params":{"name":"find_all_relationships","arguments":{"id1":"I6708","id2":"I1337","maxDepth":15}}}'
RESPONSE=$(echo "$CHECK_MULTIPLE_LINKS" | ./mcp-gedcom -gedcom-path /home/another/github/ 2>/dev/null)
TEXT=$(echo "$RESPONSE" | jq -r '.result.content[0].text' 2>/dev/null)
if echo "$TEXT" | grep -q "Parenté" && echo "$TEXT" | grep -q "DELEBECQUE"; then
    echo "✓ check_multiple_links"
    ((PASS++))
else
    echo "✗ check_multiple_links"
    ((FAIL++))
fi

FIND_ALL_REL_GIVENNAME='{"jsonrpc":"2.0","id":38,"method":"tools/call","params":{"name":"find_all_relationships","arguments":{"id1":"I9393","id2":"I9251","givenName": false}}}'
RESPONSE=$(echo "$FIND_ALL_REL_GIVENNAME" | ./mcp-gedcom -gedcom-path /home/another/github/ 2>/dev/null)
TEXT=$(echo "$RESPONSE" | jq -r '.result.content[0].text' 2>/dev/null)
if echo "$TEXT" | grep -q "Parenté"; then
    echo "✓ find_all_relationships givenName=false"
    ((PASS++))
else
    echo "✗ find_all_relationships givenName=false"
    ((FAIL++))
fi

echo ""
echo "Results: $PASS passed, $FAIL failed"

[ $FAIL -gt 0 ] && exit 1
exit 0