# Claude: Gastown Go Port

Run `bd prime` for beads context.

## Project Info

This is the **Go port** of Gas Town, a multi-agent workspace manager.

- **Issue prefix**: `gt-`
- **Python version**: ~/ai/gastown-py (reference implementation)
- **Architecture**: docs/architecture.md

## Development

```bash
go build -o gt ./cmd/gt
go test ./...
```

## Key Epics

- `gt-u1j`: Port Gas Town to Go (main tracking epic)
- `gt-f9x`: Town & Rig Management (install, doctor, federation)
