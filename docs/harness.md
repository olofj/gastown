# Gas Town Harness

A **harness** is a Gas Town installation directory - the top-level workspace where all Gas Town components live.

## GGT Harness Location

The Go implementation of Gas Town (GGT) uses a separate harness from the Python implementation (PGT):

| Implementation | Harness Location | Status |
|----------------|------------------|--------|
| GGT (Go) | `~/gt/` | Active, canonical |
| PGT (Python) | `~/ai/` | Legacy, reference only |

## Harness Structure

A properly configured GGT harness looks like:

```
~/gt/                              # HARNESS ROOT
├── CLAUDE.md                      # Mayor role prompting
├── .beads/                        # Town-level beads (gm-* prefix)
│   ├── beads.db                   # Mayor mail, coordination
│   └── config.yaml
│
├── mayor/                         # Mayor's HOME at town level
│   ├── town.json                  # {"type": "town", "name": "..."}
│   ├── rigs.json                  # Registry of managed rigs
│   └── state.json
│
├── gastown/                       # A rig (project container)
│   ├── config.json
│   ├── .beads/ → mayor/rig/.beads # Symlink to canonical beads
│   ├── mayor/rig/                 # Mayor's clone (beads authority)
│   ├── refinery/rig/              # Refinery's clone
│   ├── witness/                   # Witness agent
│   ├── crew/                      # Overseer workspaces
│   └── polecats/                  # Worker directories
│
└── wyvern/                        # Another rig (same structure)
```

## Key Directories

### Town Level (`~/gt/`)

- **CLAUDE.md**: Mayor role prompting, loaded when Mayor starts
- **.beads/**: Town-level beads database (prefix: `gm-`)
  - Contains Mayor mail and cross-rig coordination beads
- **mayor/**: Mayor's home directory with town configuration

### Rig Level (`~/gt/<rig>/`)

Each rig is a **container directory** (NOT a git clone) that holds:

- **config.json**: Rig configuration (git_url, beads prefix)
- **.beads/**: Symlink to `mayor/rig/.beads` (canonical beads)
- **mayor/rig/**: Mayor's git clone, canonical for beads and worktrees
- **refinery/rig/**: Refinery's clone for merge operations
- **witness/**: Witness agent state
- **crew/**: Overseer's personal workspaces
- **polecats/**: Worker directories (git worktrees)

## Historical Context: PGT/GGT Separation

During initial GGT development, both implementations briefly shared `~/ai/`:

```
~/ai/ (legacy mixed harness - DO NOT USE for GGT)
├── gastown/
│   ├── .gastown/          # PGT marker
│   └── mayor/             # OLD GGT clone (deprecated)
├── mayor/                 # PGT Mayor home
└── .beads/redirect        # OLD redirect (deprecated)
```

This caused confusion because:
- `~/ai/gastown/` contained both PGT markers AND GGT code
- `~/ai/mayor/` was PGT's Mayor home, conflicting with GGT concepts
- The beads redirect pointed at GGT beads from a PGT harness

The separation was completed by creating `~/gt/` as the dedicated GGT harness.

## Cleanup Steps for ~/ai/

If you have the legacy mixed harness, clean it up:

```bash
# Remove old GGT clone (beads are no longer here)
rm -rf ~/ai/gastown/mayor/

# Remove old beads redirect
rm ~/ai/.beads/redirect

# Keep PGT-specific files:
# - ~/ai/mayor/ (PGT Mayor home)
# - ~/ai/gastown/.gastown/ (PGT marker)
```

## Creating a New Harness

To create a fresh GGT harness:

```bash
gt install ~/my-harness
```

This creates:
- Town-level directories and configuration
- Town-level beads database
- CLAUDE.md for Mayor role

Then add rigs:

```bash
cd ~/my-harness
gt rig add myproject https://github.com/user/myproject
```

## Environment Variables

When spawning agents, Gas Town sets:

| Variable | Purpose | Example |
|----------|---------|---------|
| `BEADS_DIR` | Point to rig's beads | `/path/to/rig/.beads` |
| `BEADS_NO_DAEMON` | Disable daemon for worktrees | `1` |

## See Also

- [Architecture](architecture.md) - Full system design
- [Federation Design](federation-design.md) - Remote outposts
