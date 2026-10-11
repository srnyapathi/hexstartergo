# Hexgen

A Go project generator using bundled templates and hexagonal dependency boundaries.

See the [detailed usage guide](docs/USAGE.md) for prerequisites, command options,
generated layout, safety behavior, and troubleshooting.

The current runnable milestone supports standard-library HTTP without a database. Chi, gRPC, PostgreSQL, consumers, and scheduling are planned but are not yet generated; unsupported output combinations fail before creating a destination. See [the implementation checkpoint](docs/superpowers/plans/implementation-progress.md).

## Build and use

Go 1.27 or later is required. The module-validation dependency is vendored, so this milestone can build without downloading dependencies.

```powershell
go test ./...
go build -o ./bin/hexgen.exe ./cmd/hexgen
./bin/hexgen.exe --help
./bin/hexgen.exe new orders --module example.com/orders --entrypoints http --http stdlib
cd orders
go test ./...
go run ./cmd/http
```

On Unix, omit the .exe suffix. GET http://127.0.0.1:8080/readyz returns status 200 with ready and a newline. Set HTTP_ADDR to choose another listen address. The generated .env.example documents configuration and is not automatically loaded.

The HTTP adapter calls application readiness. No business endpoints, persistence, authentication, or deployment system are generated.

## Destination safety

The project name must match `[A-Za-z][A-Za-z0-9_-]*` and must not be a Windows reserved device name. The module path must be valid for Go. Repeated flags, unsupported selections, and extra positional arguments are rejected.

Generation creates a new direct child of the current directory and refuses every existing destination, including an empty directory. Templates render and Go files format before creation. On a write error, cleanup removes only unchanged files and directories owned by that invocation; foreign or replaced content is preserved and cleanup failures are reported. Interrupted generation can leave an incomplete destination which must be inspected before reuse.

Generation itself does not execute tools or download dependencies. Dependency resolution and external-service verification for later providers remain pending; this milestone is not full acceptance of the planned generator.
