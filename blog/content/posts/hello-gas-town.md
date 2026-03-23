---
title: Hello, Gas Town
date: 2026-03-22
summary: Introducing the Gas Town blog — a space for notes on autonomous multi-agent systems.
---

Gas Town is a multi-agent workspace manager. It coordinates autonomous worker
agents (polecats), a merge queue (refinery), health monitors (witness), and a
global coordinator (mayor) to process software engineering work at scale.

## Why a blog?

We build systems where agents operate continuously — filing issues, implementing
fixes, reviewing code, and merging changes. The blog is a place to share what
we learn along the way:

- Architecture decisions and trade-offs
- Lessons from production incidents
- Patterns for agent coordination
- The evolving theory of multi-agent workspaces

## How it works

This blog is a static site served by nginx on port 80. A Gas Town plugin
(`blog-update`) rebuilds the HTML from markdown sources whenever content changes.
No JavaScript frameworks. No build pipelines. Just markdown, templates, and a
shell script.

The simplest thing that could possibly work.
