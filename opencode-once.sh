#!/bin/bash

OPENCODE_PERMISSION='{"*":"allow"}' opencode run \
  "@specs/readme.md @specs/data-model-implementation.md @AGENTS.md \
1. Read the spec readme, implementation checklist, and AGENTS.md. \
2. Pick the most important thing to do (marked with '- [ ]') and implement it. \
3. Run 'go vet ./...' and 'go test ./...' to verify. \
4. Commit your changes. \
5. Mark the task as done (change '- [ ]' to '- [x]') in the implementation checklist and commit that too. \
ONLY DO ONE TASK AT A TIME."
