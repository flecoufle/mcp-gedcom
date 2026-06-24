# Release Process

1. Ensure CI is green on `main`.
2. Run the release script with the desired version:

```bash
./release.sh v0.1.0
```

This will create and push a signed tag, triggering the GitHub Actions release workflow.

## Versioning

This project follows [SemVer](https://semver.org/).
The tag must follow the `v<major>.<minor>.<patch>` pattern (e.g. `v0.1.0`, `v1.0.0`).
