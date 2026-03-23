+++
name = "blog-update"
description = "Rebuild Gas Town blog when content changes"
version = 1

[gate]
type = "cooldown"
duration = "15m"

[tracking]
labels = ["plugin:blog-update", "category:blog"]
digest = true

[execution]
type = "script"
timeout = "2m"
notify_on_failure = true
severity = "low"
+++

# Blog Update

Checks for changes in `blog/content/` and rebuilds the static site.
Executed via `run.sh` — no AI interpretation.

## What it does

1. Compares content hashes against the last successful build
2. If content changed, runs `blog/build.sh` to regenerate HTML
3. Reloads nginx if the build succeeds and nginx is running
4. Records success or failure as a wisp

## Usage

```bash
./run.sh              # Normal execution (skip if unchanged)
./run.sh --force      # Rebuild even if content unchanged
```
