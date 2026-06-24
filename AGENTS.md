mcp-gedcom

## Implementation

MCP (Model Context Protocol) server in Go for reading genealogical information from GEDCOM file.

### Project Structure
```
mcp-gedcom/
├── cmd/
│   └── mcp-gedcom/
│       └── server/
│           └── main.go           # Server entry point
├── internal/
│   ├── gedcom/
│   │   ├── encoding.go          # Encoding detection/conversion
│   │   └── loader.go            # GEDCOM file loader
│   ├── mcp/
│   │   └── protocol.go          # MCP JSON-RPC protocol
│   └── tools/
│       └── handlers.go           # Tool implementations
├── sample/
│   └── simpsons.ged            # Sample Simpsons data
├── .github/
│   └── workflows/
│       ├── ci.yml               # CI workflow
│       └── release.yml          # Release workflow
├── Dockerfile                   # Multi-stage Docker build
├── test.sh                      # Test script (run during Docker build)
├── go.mod
├── go.sum
├── .gitignore
```

### Dependencies
- **GEDCOM Parser**: github.com/iand/gedcom
- **Encoding conversion**: golang.org/x/text

### MCP Tools

1. **search_person** - Search individuals by pattern and optional approximate birth year
   - Input: `pattern` (string, required), `birthYear` (number, optional, ±2 years)
   - Searches the full name (multi-word, case-insensitive) after stripping all / characters
   - Returns: List of matching individuals with basic info (id, name, sex, birth/death dates)

2. **search_surnames** - Search unique surnames by pattern
   - Input: `pattern` (string, required), `offset` (number, optional, default: 0), `limit` (number, optional, default: 20)
   - Searches only the surname using NamePieceSurname field
   - Returns: Surnames with counts and pagination info

3. **get_person_details** - Get full details for an individual by ID
   - Input: `id` (string, required, e.g., "I1" or "@I1@"), `withSpouse` (boolean, optional, default: true), `withChildren` (boolean, optional, default: true)
   - Returns: Full individual record (id, name, sex, events, notes, relatives)

4. **get_family_details** - Get family details by ID
   - Input: `id` (string, required, e.g., "F1" or "@F0000@")
   - Returns: Husband, wife, marriage/divorce info, children with birth dates

 5. **get_children** - Get children of an individual with pagination
    - Input: `id` (string, required), `offset` (number, optional), `limit` (number, optional)
    - Returns: Families with spouse names, birth dates, and children (with pedigree info)

6. **get_parents** - Get parents of an individual
    - Input: `id` (string, required)
    - Returns: Parents grouped by family with father/mother and optional type (adopted, foster)

7. **get_relatives** - Get family relationships (spouses and parents) for an individual
    - Input: `id` (string, required)
    - Returns: Relatives with spouse and parent family links)

8. **search_by_date_range** - Search by birth/death year range
    - Input: `startYear` (number, required), `endYear` (number, required), `event` (string: "birth" or "death", optional)
    - Returns: Matching individuals with dates

9. **get_ancestors** - Get ancestors of an individual
    - Input: `id` (string, required), `generations` (number, optional, default: 3)
    - Returns: Ancestors organized by generation with relationship types

10. **get_descendants** - Get descendants of an individual
    - Input: `id` (string, required), `generations` (number, optional, default: 3)
    - Returns: Descendants organized by generation with relationship types

11. **get_statistics** - Get statistics about the GEDCOM file
    - Input: none
    - Returns: Total individuals and families count

12. **find_relationship_path** — Calculates the shortest kinship path between two individuals using bidirectional BFS

    **Parameters:**
    - `id1` (string, required) — First individual ID (e.g., `"I1"`, `"Homer_Simpson"`)
    - `id2` (string, required) — Second individual ID
    - `AncestorsOnly` (boolean, optional, default: `true`)
      - `true` (default) — only traverses parent/ancestor links. Use for blood relationships (cousins, uncle, aunt, grandparent, siblings).
      - `false` — traverses all links (parents, spouses, children). Use for in-law or extended family relationships.

    **Result** — JSON object with these fields:

    | Field | Type | Description |
    |-------|------|-------------|
    | `id1` / `id2` | string | Normalized individual IDs |
    | `name1` / `name2` | string | Cleaned full names (without `/`) |
    | `relationship` | string | Human-readable description of the relationship |
    | `path` | array | Ordered chain of nodes from id1 to id2 |

    **`relationship` output variants:**

    | Scenario | Example output |
    |----------|---------------|
    | No path found | `"no relationship found (searched only ancestors)"` |
    | Same person | `"same person"` |
    | Direct connection | `"yes relationship found: directly related"` |
    | One intermediary | `"yes relationship found: related through one intermediary"` |
    | Longer path | `"yes relationship found: related (path length: 2)"` |
    | **Common ancestor found** | `"Common ancestor: Homer Simpson (Homer_Simpson). Bart Simpson → ancestor: father (1 gen). Lisa Simpson → ancestor: father (1 gen)."` |

    When a common blood ancestor is found, the relationship string explicitly names the relationship from each individual to that ancestor using `getRelationship()`, producing terms like `"father"`, `"grandfather"`, `"great-grandmother"`, etc., with generation count.

    **`path` node structure:**

    ```json
    {"id": "Bart_Simpson", "name": "Bart Simpson", "link_is": "child", "link_of_id": "Homer_Simpson", "isMeeting": false}
    ```

    - `link_is` semantics — contains the original link:
      - `""` — start or end node (no context needed)
      - `"parent"` — this node is the PARENT of `link_of_id`
      - `"child"` — this node is the CHILD of `link_of_id`
      - `"spouse"` — this node is the SPOUSE of `link_of_id`
      - Before the junction, `link_of_id` is the **previous** node in the path; after the junction, `link_of_id` is the **next** node
    - `link_of_id` (string) — ID of the node the `link_is` relationship refers to (parent, child, or spouse)
    - `isMeeting: true` marks the single junction node where the two BFS searches met.

    **Usage guidance for AI agents:**
    - For **blood relatives** (cousins, uncle, aunt, grandparent, sibling): keep `AncestorsOnly=true` (default). The common ancestor detection will identify the shared blood relative and name the relationship on both sides.
    - For **in-laws or spouse-mediated relationships**: set `AncestorsOnly=false` to allow traversal through marriage links.
    - When the output contains a `"Common ancestor"` line, the two per-individual relationship strings encode the exact kinship. For example, `"grandfather (2 gen)"` + `"father (1 gen)"` means id1 and id2 are uncle/nephew (the common ancestor is grandfather of one and father of the other).
    - The path can be read as a directional chain: `id1 [child of →] parent [child of →] grandparent (Junction) [parent of →] uncle [parent of →] id2`
    - If no common ancestor is found (relationships through marriage only), the generic path length message is returned.

13. **find_all_relationships** — Finds ALL kinship relationships between two individuals using BFS ancestor search. Returns grouped relationship blocks with a global kinship coefficient.

    **Parameters:**
    - `id1` (string, required) — First individual ID
    - `id2` (string, required) — Second individual ID
    - `maxDepth` (number, optional, default: 15) — Maximum generations to search
    - `maxResults` (number, optional, default: 10) — Maximum number of relationship links to return (0 = unlimited)

    **Algorithm:**
    - `GetAllAncestors` performs BFS up to `maxDepth` for each individual, following only biological parent links (FAMC with `type="birth"` or empty)
    - Uses path-based deduplication to allow multiple distinct paths to the same ancestor when they converge from different lineages (e.g., two sisters marrying into different lines that both connect back to the same ancestor)
    - `maxPathsPerAncestor = 50` prevents combinatorial explosion in deeply endogamous trees
    - `FindAllRelationships` intersects both ancestor maps, builds individual ancestor links, filters shadowed links (keeping only the closest ancestor when one lies on another's lineage), limits to the `maxResults` closest (by depth sum), then groups married couples via `groupCouples`
    - `ComputeKinshipCoefficient` sums `(0.5)^(depthA+depthB+1)` per individual ancestor
    - Link count per ancestor reflects the number of distinct path pairs: if one individual has 2 paths to the common ancestor and the other has 1 path, the ancestor gets `(2 liens de parenté)`

    **Output format (French):**
    ```
    Parenté (5 liens de parenté)

    Olivier MARCHAND est un cousin d'Emmanuelle BIENVENU.

    En effet,
    Jacques Philippe BIENVENU (1 lien de parenté)
    Marie-Louise JUBAINVILLE (1 lien de parenté)
    sont en même temps
    des grands-parents d'Emmanuelle BIENVENU
    des grands-parents d'Olivier MARCHAND

    Olivier MARCHAND est aussi un cousin d'un parent d'Emmanuelle BIENVENU.

    En effet,
    Albertine Marie-Adèle GUERARD (1 lien de parenté)
    est en même temps
    une arrière-grand-mère d'Emmanuelle BIENVENU
    une grand-mère d'Olivier MARCHAND

    Olivier MARCHAND est aussi un cousin issu de germains d'un parent d'Emmanuelle BIENVENU.
    
    En effet,
    Jacques Narcisse MARCHAND (1 lien de parenté)
    Desirée Mélanie DUVAL (1 lien de parenté)
    sont en même temps
    des ancêtres à la 4e génération d'Emmanuelle BIENVENU
    des arrière-grands-parents d'Olivier MARCHAND
    Parenté: 8,6%"
    ```

    **Relationship labels (French):**

    | Depth pair | Label |
    |------------|-------|
    | 1/1 | frère/soeur |
    | 2/2 | cousin |
    | 3/2 | cousin d'un parent |
    | 3/3 | cousin issu de germains |
    | 4/3 | cousin issu de germains d'un parent |
    | d/d (deg ≥ 2, ret = 0) | cousin au Ne degré |
    | d/d (deg ≥ 1, ret ≥ 1) | cousin au Ne degré, X fois retiré |

    **Files:**
    - `internal/gedcom/ancestors.go` — BFS `GetAllAncestors`, `FindAllRelationships`, `groupCouples`
    - `internal/gedcom/relationship.go` — `ComputeRelationLabel` (fr), `AncestorLabel` (fr), `ComputeKinshipCoefficient`
    - `internal/tools/all_relationships.go` — Handler
    - `cmd/mcp-gedcom/server/formatters.go` — `formatFindAllRelationshipsResult`, `groupIntoBlocks`

### Error Handling

All tools validate required parameters. If a required parameter is missing, returns:
```json
{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"Missing required parameter: {param_name}"}],"isError":true}}
```

Unknown tools also return an error.

## Docker

### Build

```bash
docker build -t mcp-gedcom .
```

### Run

```bash
docker run -it --rm mcp-gedcom
```

### Exemple d'appel

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' | docker run -i --rm mcp-gedcom

echo '{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"search_surnames","arguments":{"pattern":"Wil"}}}' | docker run -i --rm mcp-gedcom
```

### Local Build & Running

```bash
# Build the server
go build -o mcp-gedcom ./cmd/mcp-gedcom/server

# Run with default (first .ged found in current directory)
./mcp-gedcom

# Run with specific directory containing GEDCOM files
./mcp-gedcom -gedcom-path /sample/ -gedcom-file gedcom.ged
```

### Protocol

The server uses JSON-RPC 2.0 over stdio. Initialize with:

```json
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}
```

Then list/call tools:

```json
{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"search_person","arguments":{"pattern":"Robert"}}}
{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"search_surnames","arguments":{"pattern":"Sim","offset":0,"limit":10}}}
{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"get_person_details","arguments":{"id":"I1"}}}
```

## GivenName Feature

All tools (except `get_statistics`) support an optional `givenName` boolean parameter:

| Value | Behavior |
|-------|----------|
| `true` (default) | Display full given name (e.g., "John Michael") |
| `false` | Display usage name (quoted nickname) if present, falls back to full given name |

### Implementation Details

- `internal/gedcom/individual.go`: `ExtractGivenName`, `ExtractUsageName`, `ExtractSurname`, `ResolveName` functions; shared constructors (`MakePersonSummary`, `PersonMap`, `PersonMapWithBirth`, `ChildMap`) store `given_name`, `usage_name`, `surname` keys
- `internal/tools/handlers.go`: All 11 handlers accept `useGiven bool` parameter; `resolveNameInMap` helper resolves sub-map names; all inline maps enriched with `given_name`/`usage_name`/`surname`; no raw `Name[0].Name` remains
- `cmd/mcp-gedcom/server/main.go`: `GivenName *bool` in each dispatch struct; threads to handlers; `givenName` in tool schemas
- `cmd/mcp-gedcom/server/formatters.go`: `formatFindAllRelationshipsResult` accepts `useGiven` and resolves names via `getDisplayName` closure
- `cmd/mcp-gedcom/server/main.go` line 686 — `formatFindRelationshipPathResult` reads pre-resolved names from handler result (resolved via `ResolveName` in `HandleFindRelationshipPath`)
