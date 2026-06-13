---
title: "Quick start"
description: "Run your first epmc command."
weight: 30
---

Once `epmc` is on your `PATH`:

```bash
epmc --help       # see the command tree
epmc version      # build info
```

This is a fresh scaffold, so the command tree is just `version` for now. Add
your first real command in `cli/`, build on the `europepmc` library package,
and document it here.

A good first command usually fetches one thing and prints it as JSON, so the
output pipes straight into `jq` and the rest of your tools.
