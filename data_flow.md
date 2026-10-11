# Data Flow — Hexgen

## CLI request to generated project

**Source:** command arguments supplied to cmd/hexgen.
**Sink:** a new direct child directory and a success message naming its README, or diagnostics and a nonzero exit code.
**Transformations:** parse the new subcommand and singleton flags; validate selections; choose bundled templates; render and format Go source in memory; validate output paths; exclusively claim the destination; write using rooted filesystem operations.
**Shape at each step:** string arguments → generate.Options (Name, Module, Entrypoints, HTTP, Database, Broker, Schedule) → []generate.File (relative Path, Data bytes) → files beneath the new project directory.
**Source:** cmd/hexgen/main.go; internal/cli/cli.go; internal/generate/options.go, render.go, writer.go.

Generation performs no subprocess execution or downloads. Current output support is standard-library HTTP without a database. The wider selection contract is validated, but unimplemented output combinations fail before filesystem mutation.

## Generated HTTP readiness

**Source:** GET /readyz at the address from HTTP_ADDR (default 127.0.0.1:8080).
**Sink:** status 200 and ready plus newline, or status 503 and not ready plus newline.
**Transformations:** the executable loads its own configuration and wires application.Ready into the inbound Handler; the adapter supplies a two-second request context; the application checks context cancellation; the adapter maps the result to the fixed public response.
**Shape at each step:** http.Request → context.Context → error → HTTP status and text body. Internal error details are not serialized to clients.
**Source:** internal/generate/templates/main.go.tmpl, config.go.tmpl, http.go.tmpl, readiness.go.tmpl.

The application does not import configuration or transport packages. Domain and application/ports contain instructional documentation, with no invented business model or repository interface.
