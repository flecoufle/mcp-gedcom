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

The project is built and tested through Docker. Only `make`, `git` and `docker` are required on the host.

```bash
make doctor
make test
make image
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' | docker run -i --rm mcp-gedcom:latest
```

Run the server with a GEDCOM file:

```bash
make run GEDCOM_FILE=./sample/simpsons.ged
```

## Development

```bash
make test-unit        # unit tests in Docker
make test-integration # integration tests in Docker
make test             # both
make ci               # tests + runtime image
make shell            # interactive Go shell, source mounted at /src
make clean            # remove tagged images
```

There is no local Go toolchain requirement: compilation and tests always happen inside the Docker build stages.

`make test-unit` and `make test-integration` stream their output and bypass the Docker cache for the test stage, so the tests actually run on every invocation. Re-enable caching with `make test-integration INTEGRATION_NO_CACHE=`.

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
