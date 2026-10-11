# State Machines — Hexgen

## Generation ownership

**States:** validating, rendering, claiming, writing, complete, cleaning, failed, incomplete.
**Initial:** validating.
**Transitions:**
- validating → rendering when the project name, module path, and selections are valid.
- rendering → claiming when selected templates render and Go source formats successfully, and output paths pass portable validation, including file/directory collisions and conflicting case spellings of shared ancestors.
- claiming → writing when exclusive destination creation succeeds; an existing destination is preserved and produces failed.
- writing → complete after every selected file is exclusively created, written, and closed.
- writing → cleaning on a filesystem failure after destination creation, including failure to open its rooted handle.
- cleaning → failed after removing unchanged owned files and directories in reverse order, then the empty owned destination.
- cleaning → incomplete when foreign/replaced content or filesystem errors prevent safe removal; preserve that content and report cleanup errors with the original error.
- validating, rendering, claiming → failed on an error before ownership is obtained, without modifying an existing destination.
**Source:** internal/generate/options.go, render.go, writer.go; approved design filesystem safety contract.

Cleanup keeps the rooted filesystem handle open until owned-file removal finishes. Cleanup never recursively deletes a destination. A crash can leave incomplete output; subsequent generation refuses that existing destination.

## Generated HTTP executable

**States:** starting, serving, stopping, stopped, failed.
**Initial:** starting.
**Transitions:**
- starting → serving when the listener opens.
- starting, serving → failed on an unexpected server error.
- serving → stopping on a supported termination signal.
- stopping → stopped after graceful shutdown and normal server closure.
- stopping → failed on shutdown failure, after forced closure releases resources.
**Source:** internal/generate/templates/main.go.tmpl.

This file describes the implemented HTTP foundation. Planned provider lifecycles remain in the approved design until implemented.
