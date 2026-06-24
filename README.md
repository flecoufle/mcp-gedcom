# mcp-gedcom

[![CI](https://github.com/flecoufle/mcp-gedcom/actions/workflows/ci.yml/badge.svg)](https://github.com/flecoufle/mcp-gedcom/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.26-blue)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

MCP (Model Context Protocol) server for reading genealogical data from GEDCOM files.

## Features

14 MCP tools for querying and analyzing GEDCOM genealogical data:
`search_person`, `search_surnames`, `search_by_date_range`,
`get_person_details`, `get_family_details`, `get_children`, `get_parents`, `get_relatives`,
`get_ancestors`, `get_descendants`, `get_statistics`,
`find_relationship_path`, `find_all_relationships`, `load_gedcom_file`

## Quick Start

### Docker
```bash
docker build -t mcp-gedcom .
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' | docker run -i --rm mcp-gedcom
```

### Local
```bash
go build -o mcp-gedcom ./cmd/mcp-gedcom/server
./mcp-gedcom -gedcom-file sample/simpsons.ged
```

## Usage

The server uses JSON-RPC 2.0 over stdio. Example session:

```json
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}
{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"search_person","arguments":{"pattern":"Homer"}}}
{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"get_statistics","arguments":{}}}
```

## License

MIT
