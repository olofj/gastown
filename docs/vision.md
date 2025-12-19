# Gas Town Vision

> Work is fractal. Every piece of work can contain other work, recursively.
> Work history is proof of capability. Your CV is your chain.

## The Big Picture

Gas Town is more than an AI coding agent orchestrator. It's a **work execution engine** built on a universal ledger of work - where every task, every completion, every validation is recorded with cryptographic integrity.

The system is designed to evolve from "coding agent coordinator" to "universal work allocation platform" without changing its fundamental architecture.

## Core Insights

### 1. Git is Already a Blockchain

Git provides:
- **Merkle tree** - Cryptographic hashes linking history
- **Distributed consensus** - Push/pull with conflict resolution
- **Immutability** - History cannot be rewritten (without force)
- **Auditability** - Every change attributed to an author

We don't need to build a new blockchain. Git, combined with Beads, gives us the ledger infrastructure for free.

### 2. Work is a Universal Protocol

Every piece of structured work can be expressed as:
- **Identity** - Who is doing the work
- **Specification** - What needs to be done
- **Acceptance criteria** - How we know it's done
- **Validation** - Who approved the completion
- **Provenance** - What work led to this work

This applies equally to:
- Code commits and PRs
- Design documents
- Bug fixes
- Research tasks
- Any structured human or AI work

### 3. Your Work History IS Your CV

Instead of curated resumes:
- Every completed task is recorded
- Quality signals are captured (acceptance rate, revision count, review feedback)
- Skills are derived from demonstrated capability, not claimed expertise
- Reputation is earned through work, not credentials

This is "proof-of-stake" for work:
- Stake = accumulated reputation
- Claim work → stake your reputation
- Complete well → reputation grows
- Fail → reputation diminished (but recoverable)

### 4. Molecules Crystallize Workflows

Molecules are reusable workflow patterns that encode:
- What steps a workflow contains
- How steps depend on each other
- What quality gates must pass
- How work can be parallelized

Key properties:
- **Deterministic structure** - Same molecule, same step graph
- **Nondeterministic execution** - Any worker can execute any ready step
- **Idempotent progress** - Completed steps stay completed

This enables the "engineer in a box" - AI agents that follow rigorous workflows with built-in quality gates, not just "do the task."

### 5. Federation Creates the World Chain

The recursive structure:
```
World
├── Platform (GitHub, enterprise systems, ...)
│   ├── Organization
│   │   ├── Project
│   │   │   ├── Epic
│   │   │   │   └── Task chains
│   │   │   │       └── Entity contribution records
```

Each level has its own chain. Work rolls up. Skills aggregate. The world gets a unified view of capability.

## The Technical Foundation

### Beads as Ledger

| Concept | Beads Implementation |
|---------|---------------------|
| Transaction | Issue/task/work item |
| Address | Entity identifier |
| Smart Contract | Work specification + acceptance criteria |
| Validation | Merge/review/approval |
| Stake | Accumulated reputation chain |
| Gas | Effort estimation |

### The OS Metaphor

Gas Town is an operating system for work:

| OS Concept | Gas Town |
|------------|----------|
| Kernel | Daemon |
| Process scheduler | Ready work + dependencies |
| Timer interrupts | Timed beads |
| Semaphores | Resource beads |
| Background services | Pinned beads |
| Process templates | Molecules |
| IPC | Mail beads |

### Entity Chains

Every entity (human or agent) accumulates a work chain:
- Work they created
- Work they claimed
- Work they completed
- Work they validated
- Quality signals at each step

Format: `entity://<platform>/<org>/<id>`

Examples:
- `entity://gastown/steveyegge/polecat-nux`
- `entity://github/acme-corp/jane-doe`

The chain IS the CV. Skills are derived from demonstrated work, not claimed.

### Skill Vectors

Work items carry skill embeddings:
- What capabilities does this work demonstrate?
- What prior knowledge does it require?
- How complex is it?

Matching displaced workers to unmet demand becomes a vector similarity problem:
- Worker's demonstrated skills (from their chain)
- Job requirements (from work specification)
- Match = high similarity + acceptable reputation

## Public Framing

These capabilities can be explained without revealing the full vision:

| Capability | Public Explanation |
|------------|-------------------|
| Entity tracking | "Agent accountability and work attribution" |
| Recursive structure | "Enterprise-scale hierarchical tracking" |
| Skill vectors | "Capability-based task routing" |
| Federation | "Multi-repo / org-wide visibility" |
| Validation | "Review workflows and quality gates" |
| Mail/threading | "Agent coordination for distributed teams" |

All true - just not the full story.

## The Execution Model

### Refineries as Validators

Refineries don't just merge code - they're validator nodes:
- Verify work meets acceptance criteria
- Record validation in the ledger
- Gate entry to the canonical chain (main branch)

### Polecats as Workers

Polecats aren't just coding agents - they're work executors with chains:
- Each polecat has an identity
- Work history accumulates
- Success rate is tracked
- Demonstrated skills emerge

### Molecules as Contracts

Molecules aren't just workflows - they're smart contracts for work:
- Specify exactly what must happen
- Encode acceptance criteria per step
- Enable deterministic verification
- Support nondeterministic execution

## Where This Goes

### Phase 1: Gas Town v1 (Now)
- Coding agent orchestrator
- Beads-backed work tracking
- Molecule-based workflows
- Local federation ready

### Phase 2: Federation
- Cross-machine outposts
- Multi-rig coordination
- Git-based sync everywhere

### Phase 3: Entity Chains
- Persistent agent identities
- Work history accumulation
- Skill derivation from work

### Phase 4: Platform of Platforms
- Adapters for external work sources
- Cross-platform skill matching
- The world chain emerges

## Design Principles

1. **Git as blockchain** - Don't build new consensus; use git
2. **Federation not global consensus** - Each platform validates its own work
3. **Skill embeddings as native** - Work items carry capability vectors
4. **Human-readable** - Beads is Markdown; auditable, trustworthy
5. **Incremental evolution** - Current architecture grows into the full vision

## The Redemption Arc

The system doesn't judge - it tracks demonstrated capability and matches it to demand.

- Someone with a troubled past can rebuild their chain
- Skills proven through work matter more than credentials
- Every completion is a step toward redemption
- The ledger is honest but not cruel

This is capability matching at scale. The work speaks for itself.

---

*"Work is fractal. Money is crystallized labor. The world needs a ledger."*
