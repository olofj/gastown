+++
name = "blog-update"
description = "Regenerate and deploy the Gas Town static blog"
version = 1

[gate]
type = "cooldown"
duration = "1h"

[tracking]
labels = ["plugin:blog-update", "category:blog"]
digest = true

[execution]
timeout = "2m"
notify_on_failure = true
severity = "medium"
+++

# Blog Update

Deploys the Gas Town blog by syncing `blog/site/` to the nginx document root.

## What it does

1. Validates that `blog/site/index.html` exists
2. Copies `blog/site/` contents to `/var/www/gastown-blog/`
3. Tests that nginx can serve the site (health check on port 80)
4. Records success/failure as a wisp

## Prerequisites

- nginx installed and configured with `blog/nginx.conf`
- `/var/www/gastown-blog/` directory exists and is writable

## Usage

```bash
./run.sh                    # Normal deploy
./run.sh --dry-run          # Show what would be deployed
./run.sh --check-only       # Just verify current deployment
```
