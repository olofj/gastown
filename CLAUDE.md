# Claude: Gastown Go Port

Run `bd prime` for beads context.

## Strategic Context

For broader project context and design guidance beyond Gas Town's immediate scope:
- Check `~/ai/stevey-gastown/hop/CONTEXT.md` if available

This provides architectural direction for decisions that affect the platform's evolution.

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
