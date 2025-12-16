# Gas Town Architecture

Gas Town is a multi-agent workspace manager that coordinates AI coding agents working on software projects. It provides the infrastructure for running swarms of agents, managing their lifecycle, and coordinating their work through mail and issue tracking.

## Core Concepts

### Town

A **Town** is a complete Gas Town installation - the workspace where everything lives. A town contains:
- Town configuration (`config/` directory)
- Mayor's home (`mayor/` directory at town level)
- One or more **Rigs** (managed project repositories)

### Rig

A **Rig** is a managed project repository with its associated agents. Each rig is a git clone of a project that Gas Town manages. Within each rig:
- The project's actual code lives at the rig root
- Agent directories are git-ignored via `.git/info/exclude`
- Each rig has its own Witness, Refinery, and Polecats
- Mayor has a clone in each rig for rig-specific work

### Agents

Gas Town has four agent roles:

| Agent | Scope | Responsibility |
|-------|-------|----------------|
| **Mayor** | Town-wide | Global coordination, swarm dispatch, cross-rig decisions |
| **Witness** | Per-rig | Worker lifecycle, nudging, pre-kill verification, session cycling |
| **Refinery** | Per-rig | Merge queue processing, PR review, integration |
| **Polecat** | Per-rig | Implementation work on assigned issues |

### Mail

Agents communicate via **mail** - JSONL-based inboxes for asynchronous messaging. Each agent has an inbox at `mail/inbox.jsonl`. Mail enables:
- Work assignment (Mayor → Refinery → Polecat)
- Status reporting (Polecat → Witness → Mayor)
- Session handoff (Agent → Self for context cycling)
- Escalation (Witness → Mayor for stuck workers)

### Beads

**Beads** is the issue tracking system. Gas Town agents use beads to:
- Track work items (`bd ready`, `bd list`)
- Create issues for discovered work (`bd create`)
- Claim and complete work (`bd update`, `bd close`)
- Sync state to git (`bd sync`)

Polecats have direct beads write access and file their own issues.

## Directory Structure

### Town Level

```
~/ai/                              # Town root
├── config/                        # Town configuration (VISIBLE, not hidden)
│   ├── town.json                  # {"type": "town", "name": "..."}
│   ├── rigs.json                  # Registry of managed rigs
│   └── federation.json            # Remote machine config (future)
│
├── mayor/                         # Mayor's HOME at town level
│   ├── CLAUDE.md                  # Mayor role prompting
│   ├── mail/inbox.jsonl           # Mayor's inbox
│   └── state.json                 # Mayor state
│
├── wyvern/                        # A rig (project repository)
└── beads/                         # Another rig
```

### Rig Level

```
wyvern/                            # Rig = clone of project repo
├── .git/
│   └── info/exclude               # Contains: polecats/ refinery/ witness/ mayor/
├── .beads/                        # Beads (if project uses it)
├── [project files]                # Clean project code on main branch
│
├── polecats/                      # Worker clones (gitignored)
│   ├── Nux/                       # Each polecat has a full clone
│   ├── Toast/
│   └── Capable/
│
├── refinery/                      # Refinery agent
│   ├── rig/                       # Refinery's working clone
│   ├── state.json
│   └── mail/inbox.jsonl
│
├── witness/                       # Witness agent (per-rig pit boss)
│   ├── rig/                       # Witness's working clone
│   ├── state.json
│   └── mail/inbox.jsonl
│
└── mayor/                         # Mayor's presence in this rig
    ├── rig/                       # Mayor's rig-specific clone
    └── state.json
```

### Why Decentralized?

Agents live IN rigs rather than in a central location:
- **Locality**: Each agent works in the context of its rig
- **Independence**: Rigs can be added/removed without restructuring
- **Parallelism**: Multiple rigs can have active swarms simultaneously
- **Simplicity**: Agent finds its context by looking at its own directory

## Agent Responsibilities

### Mayor

The Mayor is the global coordinator:
- **Swarm dispatch**: Decides which rigs need swarms, what work to assign
- **Cross-rig coordination**: Routes work between rigs when needed
- **Escalation handling**: Resolves issues Witnesses can't handle
- **Strategic decisions**: Architecture, priorities, integration planning

**NOT Mayor's job**: Per-worker cleanup, session killing, nudging workers

### Witness

The Witness is the per-rig "pit boss":
- **Worker monitoring**: Track polecat health and progress
- **Nudging**: Prompt workers toward completion
- **Pre-kill verification**: Ensure git state is clean before killing sessions
- **Session lifecycle**: Kill sessions, update worker state
- **Self-cycling**: Hand off to fresh session when context fills
- **Escalation**: Report stuck workers to Mayor

**Key principle**: Witness owns ALL per-worker cleanup. Mayor is never involved in routine worker management.

### Refinery

The Refinery manages the merge queue:
- **PR review**: Check polecat work before merging
- **Integration**: Merge completed work to main
- **Conflict resolution**: Handle merge conflicts
- **Quality gate**: Ensure tests pass, code quality maintained

### Polecat

Polecats are the workers that do actual implementation:
- **Issue completion**: Work on assigned beads issues
- **Self-verification**: Run decommission checklist before signaling done
- **Beads access**: Create issues for discovered work, close completed work
- **Clean handoff**: Ensure git state is clean for Witness verification

## Key Workflows

### Swarm Dispatch

```
Mayor                     Refinery                    Polecats
  │                          │                           │
  ├─── [dispatch swarm] ────►│                           │
  │                          ├─── [assign issues] ──────►│
  │                          │                           │ (work)
  │                          │◄── [PR ready] ────────────┤
  │                          │ (review/merge)            │
  │◄── [swarm complete] ─────┤                           │
```

### Worker Cleanup (Witness-Owned)

```
Polecat                   Witness                     Mayor
  │                          │                           │
  │ (completes work)         │                           │
  ├─── [done signal] ───────►│                           │
  │                          │ (capture git state)       │
  │                          │ (assess cleanliness)      │
  │◄── [nudge if dirty] ─────┤                           │
  │ (fixes issues)           │                           │
  ├─── [done signal] ───────►│                           │
  │                          │ (verify clean)            │
  │                          │ (kill session)            │
  │                          │                           │
  │                          │ (if stuck 3x) ───────────►│
  │                          │                           │ (escalation)
```

### Session Cycling (Mail-to-Self)

When an agent's context fills, it hands off to its next session:

1. **Recognize**: Notice context filling (slow responses, losing track of state)
2. **Capture**: Gather current state (active work, pending decisions, warnings)
3. **Compose**: Write structured handoff note
4. **Send**: Mail handoff to own inbox
5. **Exit**: End session cleanly
6. **Resume**: New session reads handoff, picks up where old session left off

## Key Design Decisions

### 1. Witness Owns Worker Cleanup

**Decision**: Witness handles all per-worker cleanup. Mayor is never involved.

**Rationale**:
- Separation of concerns (Mayor strategic, Witness operational)
- Reduced coordination overhead
- Faster shutdown
- Cleaner escalation path

### 2. Polecats Have Direct Beads Access

**Decision**: Polecats can create, update, and close beads issues directly.

**Rationale**:
- Simplifies architecture (no proxy through Witness)
- Empowers workers to file discovered work
- Faster feedback loop
- Beads v0.30.0+ handles multi-agent conflicts

### 3. Session Cycling via Mail-to-Self

**Decision**: Agents mail handoff notes to themselves when cycling sessions.

**Rationale**:
- Consistent pattern across all agent types
- Timestamped and logged
- Works with existing inbox infrastructure
- Clean separation between sessions

### 4. Decentralized Agent Architecture

**Decision**: Agents live in rigs (`<rig>/witness/rig/`) not centralized (`mayor/rigs/<rig>/`).

**Rationale**:
- Agents work in context of their rig
- Rigs are independent units
- Simpler role detection
- Cleaner directory structure

### 5. Visible Config Directory

**Decision**: Use `config/` not `.gastown/` for town configuration.

**Rationale**: AI models often miss hidden directories. Visible is better.

## Configuration

### town.json

```json
{
  "type": "town",
  "version": 1,
  "name": "stevey-gastown",
  "created_at": "2024-01-15T10:30:00Z"
}
```

### rigs.json

```json
{
  "version": 1,
  "rigs": {
    "wyvern": {
      "git_url": "https://github.com/steveyegge/wyvern",
      "added_at": "2024-01-15T10:30:00Z"
    }
  }
}
```

### Per-Rig Beads Config

Each rig can configure where polecats file beads:

```json
{
  "beads": {
    "repo": "local",       // "local" | "/path/to/beads" | "git-url"
    "prefix": "wyv"
  }
}
```

- `"local"`: Use project's `.beads/` (default, for your own projects)
- Path: Use beads at specific location (for OSS contributions)
- Git URL: Clone and use shared team beads

## CLI Commands

### Town Management

```bash
gt install [path]      # Install Gas Town at path
gt doctor              # Check workspace health
gt doctor --fix        # Auto-fix issues
```

### Agent Operations

```bash
gt status              # Overall town status
gt rigs                # List all rigs
gt polecats <rig>      # List polecats in a rig
```

### Communication

```bash
gt inbox               # Check inbox
gt send <addr> -s "Subject" -m "Message"
gt inject <polecat> "Message"    # Direct injection to session
gt capture <polecat> "<cmd>"     # Run command in polecat session
```

### Session Management

```bash
gt spawn --issue <id>  # Start polecat on issue
gt kill <polecat>      # Kill polecat session
gt wake <polecat>      # Mark polecat as active
gt sleep <polecat>     # Mark polecat as inactive
```

## Future: Federation

Federation enables work distribution across multiple machines via SSH. Not yet implemented, but the architecture supports:
- Machine registry (local, ssh, gcp)
- Extended addressing: `[machine:]rig/polecat`
- Cross-machine mail routing
- Remote session management

## Implementation Status

Gas Town is being ported from Python (gastown-py) to Go (gastown). The Go port (GGT) is in development:

- **Epic**: gt-u1j (Port Gas Town to Go)
- **Scaffolding**: gt-u1j.1 (Go scaffolding - blocker for implementation)
- **Management**: gt-f9x (Town & Rig Management: install, doctor, federation)

See beads issues with `bd list --status=open` for current work items.
