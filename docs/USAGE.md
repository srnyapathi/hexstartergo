# Hexgen usage guide

Hexgen creates a new Go project from the bundled hexagonal-architecture templates.
The generated project has a standard-library HTTP entrypoint and an application
readiness endpoint. It does not create business logic, persistence, authentication,
or deployment configuration.

## Prerequisites

- Go 1.27 or later.
- A writable parent directory for the generated project.
- A valid Go module path, such as `example.com/orders`.

The repository vendors its module-validation dependency. The generator itself can
therefore be tested and built without downloading dependencies.

## Build the generator

From the repository root:

```powershell
go test ./...
go build -o ./bin/hexgen.exe ./cmd/hexgen
```

On Unix-like systems, omit the `.exe` suffix:

```sh
go build -o ./bin/hexgen ./cmd/hexgen
```

You can also run the generator directly during development:

```powershell
go run ./cmd/hexgen --help
```

## Check command help

```powershell
./bin/hexgen.exe --help
```

The help output shows the complete command shape:

```text
hexgen new NAME --module MODULE --entrypoints http --http stdlib
```

## Generate a project

Run the command from the directory that should contain the new project:

```powershell
./bin/hexgen.exe new orders `
  --module example.com/orders `
  --entrypoints http `
  --http stdlib
```

Equivalent Unix shell syntax:

```sh
./bin/hexgen new orders \
  --module example.com/orders \
  --entrypoints http \
  --http stdlib
```

The command creates `orders` as a direct child of the current directory and prints
the location of its generated README. The destination must not already exist,
including as an empty directory.

## Options

| Option | Required when | Values | Description |
| --- | --- | --- | --- |
| `--module` | Always | A valid Go module path | Sets the generated module declaration. |
| `--entrypoints` | Always | `http`, `grpc`, `consumer`, `scheduler` | Comma-separated entrypoint selection. The current generator renders only `http`. |
| `--http` | `http` is selected | `stdlib`, `chi` | Selects the HTTP adapter. The current generator renders only `stdlib`. |
| `--database` | Optional | `postgres` | Validated for planned output, but not generated in this milestone. |
| `--broker` | `consumer` is selected | `rabbitmq`, `kafka` | Validated for planned output, but not generated in this milestone. |
| `--schedule` | `scheduler` is selected | `cron` | Validated for planned output, but not generated in this milestone. |

Each option may be specified only once. Adapter options must match their
entrypoint; for example, `--http stdlib` requires `http` in `--entrypoints`.

### Currently supported command

This is the runnable combination:

```text
--entrypoints http --http stdlib
```

The following combinations are validated but fail before creating a destination:

- `chi` HTTP output.
- gRPC output.
- PostgreSQL output.
- RabbitMQ or Kafka consumer output.
- Cron scheduler output.
- Multiple entrypoints.

This keeps unsupported output explicit instead of producing a partial project.

## Generated project layout

A successful generation currently creates:

```text
orders/
├── .env.example
├── .gitignore
├── README.md
├── go.mod
├── cmd/
│   └── http/
│       └── main.go
└── internal/
    ├── adapters/inbound/http/
    │   ├── http.go
    │   └── http_test.go
    ├── application/
    │   ├── ports/README.md
    │   └── readiness.go
    ├── config/http.go
    └── domain/README.md
```

The generated `.gitignore` excludes build artifacts, local caches, local
environment files, and Go workspace files while keeping `.env.example` tracked.

## Build and run the generated project

Change into the generated directory:

```powershell
cd orders
go mod download
go test ./...
go build ./cmd/http
go run ./cmd/http
```

The server listens on `127.0.0.1:8080` by default. Verify readiness in another
terminal:

```powershell
Invoke-WebRequest http://127.0.0.1:8080/readyz
```

The endpoint returns HTTP `200` with the body `ready` followed by a newline.

To use another address, set `HTTP_ADDR` before starting the server:

```powershell
$env:HTTP_ADDR = "127.0.0.1:9090"
go run ./cmd/http
```

```sh
HTTP_ADDR=127.0.0.1:9090 go run ./cmd/http
```

`.env.example` documents the variable. It is an example only and is not loaded
automatically by the generated program.

## Safety and failure behavior

- Project names must match `[A-Za-z][A-Za-z0-9_-]*` and cannot be Windows device
  names such as `CON` or `NUL`.
- Module paths are checked with Go's module-path validation.
- Extra positional arguments, repeated flags, duplicate entrypoints, and unknown
  values are rejected.
- Rendering and Go formatting complete before the destination is claimed.
- Existing destinations are never overwritten.
- If writing fails, cleanup removes only files and directories created unchanged by
  that invocation. Foreign or replaced content is preserved and cleanup failures
  are reported.

If generation is interrupted, inspect the destination before attempting to reuse
it. The generator intentionally refuses to claim an existing destination.

## Troubleshooting

### `claim destination` or destination already exists

Choose a new project name or remove the existing destination after confirming it
does not contain work you need. Hexgen never overwrites an existing destination.

### `module: ...`

Use a valid Go module path, for example `example.com/orders` or
`github.com/acme/orders`.

### `this milestone supports only http with stdlib and no database`

Use the supported selection:

```text
--entrypoints http --http stdlib
```

The other adapter and entrypoint values describe planned capabilities but are not
yet rendered.

### The server does not respond

Confirm that the process is running, that `HTTP_ADDR` is available, and that the
request uses the same address and port. The expected check is `GET /readyz`.

## Scope of this milestone

Hexgen currently establishes project structure and readiness plumbing. Business
operations, persistence, authentication, message consumers, scheduling, external
service checks, and deployment automation are outside the generated output.
