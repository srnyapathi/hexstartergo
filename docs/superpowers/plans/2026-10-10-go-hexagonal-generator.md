# Go Hexagonal Project Generator Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver a flag-driven Go generator that creates compilable hexagonal projects with independently runnable HTTP, gRPC, consumer, and scheduler executables.

**Architecture:** Embed a curated set of templates in one generator binary. Parse and validate selections, render and format an in-memory file set, then create a new destination without overwriting anything. Generated adapters call inward; executable entry points own configuration and resource lifecycles.

**Tech Stack:** Go 1.27 (installed 1.27.1), standard-library flag/embed/text-template/go-format packages, golang.org/x/mod/module for module-path validation; generated options use net/http or Chi, gRPC-Go/protobuf, pgx/v5, RabbitMQ amqp091-go or kafka-go, and robfig/cron/v3.

---

## Approved inputs and execution constraints

- Design: `docs/superpowers/specs/2026-10-10-go-hexagonal-generator-design.md`, approved by the user on 2026-10-10.
- Vocabulary: `CONTEXT.md`.
- Workspace began empty; only design documentation currently exists. No Git repository exists. Do not manufacture a remote URL or publish anything. Use local module name `hexgen` for the generator; generated modules use the required user-supplied module path.
- Verified prerequisites: Go 1.27.1, Git 2.45.1, protoc 36.2. Docker client exists but cannot connect to a running daemon; its user config is also inaccessible in this environment. Runtime infrastructure verification is an execution prerequisite, not a passed check.
- The normal terminal tool failed to start. Read/write access and direct process execution worked through Node. Commands below can run through the terminal once repaired or through `child_process.spawn` with an argument array, `windowsHide: true`, and an explicit working directory. Do not concatenate user values into shell commands.
- Keep all temporary projects, build output, dependency caches when necessary, and test fixtures within the workspace or permitted temporary directory. Use `GOTOOLCHAIN=local` to avoid unexpected toolchain downloads.
- Package versions must be resolved and pinned at execution using the available Go toolchain; this plan does not invent current versions. Generation itself never downloads or runs Go/protobuf tooling.
- This is one coupled product with sequential milestones, not several separately deployable subsystems. Complete the HTTP walking skeleton before expanding providers.
- Commit task milestones only if a Git repository has been established with authorization; otherwise record verified local changes without claiming commits.

## File ownership

| Files | Responsibility |
|---|---|
| `go.mod`, `go.sum`, `cmd/hexgen/main.go` | Generator module, dependency lock, process entry |
| `internal/cli/run.go`, `run_test.go` | Help, flag grammar, diagnostics, exit code |
| `internal/generate/options.go`, `options_test.go` | Selection contract and semantic validation |
| `internal/generate/render.go`, `render_test.go` | Embedded templates, dependency selection, deterministic formatting |
| `internal/generate/write.go`, `write_test.go` | Exclusive destination ownership and safe cleanup |
| `internal/generate/generate.go`, `generate_test.go` | End-to-end generation boundary |
| `internal/generate/templates/base/` | Shared project/module/documentation templates |
| `internal/generate/templates/http/` | HTTP executable and standard/Chi adapter templates |
| `internal/generate/templates/grpc/` | gRPC executable, adapter, source proto, generated bindings |
| `internal/generate/templates/postgres/` | Pool lifecycle template |
| `internal/generate/templates/consumer/` | Consumer executable, RabbitMQ/Kafka adapter templates |
| `internal/generate/templates/scheduler/` | Scheduler executable and adapter templates |
| `tests/acceptance/` | Black-box generation, build matrix, runtime fixture tests |
| `tests/fixtures/compose.yaml` | Disposable PostgreSQL, RabbitMQ, and Kafka test services |
| `README.md` | Generator usage, supported flags, build and verification commands |
| `CONTEXT.md`, `state_machines.md`, `data_flow.md` | Project vocabulary, ownership transitions, data/dependency flow |

Template files mirror their output names and use `.tmpl` suffixes: for example, `templates/http/main.go.tmpl` produces `cmd/http/main.go`. Variants are selected by renderer-owned explicit mappings, never by a user-supplied template path. Avoid a public plugin/manifest format.

## Task 1: Establish contracts and compile them

**Create:** `go.mod`, `internal/generate/options.go`, `internal/cli/run.go`.

- [ ] Create the generator module without a fabricated hosting namespace.

```go
module hexgen

go 1.27.0
```

- [ ] Define the generator selection and file-set shapes before rendering or writing.

```go
package generate

import "errors"

type Options struct {
    Name string
    Module string
    Entrypoints []string
    HTTP string
    Database string
    Broker string
    Schedule string
}

type File struct {
    Path string // slash-separated path relative to the new project
    Data []byte
}

var ErrNotImplemented = errors.New("generator operation not implemented")

func (o Options) Validate() error { return ErrNotImplemented }
func Render(o Options) ([]File, error) { return nil, ErrNotImplemented }
func Write(parent, name string, files []File) error { return ErrNotImplemented }
func Generate(parent string, o Options) (string, error) {
    return "", ErrNotImplemented
}
```

The temporary function bodies keep the contract compilable and fail explicitly. Move implementations to their ownership files in subsequent tasks and remove the generator sentinel when the last stub is implemented. Generated business placeholders use a separate application-owned error.

- [ ] Define the CLI boundary independently of process globals.

```go
package cli

import "io"

func Run(args []string, stdout, stderr io.Writer, parent string) int {
    return 1
}
```

- [ ] Run `go test ./internal/...`. Expected: packages compile; no claim of behavior yet.

## Task 2: Validate the selection contract

**Modify:** `internal/generate/options.go`, `go.mod`, `go.sum`.
**Create:** `internal/generate/options_test.go`.

- [ ] Add an external-package boundary test for the smallest valid request.

```go
package generate_test

import (
    "testing"
    "hexgen/internal/generate"
)

func TestOptionsValidHTTP(t *testing.T) {
    o := generate.Options{Name: "orders", Module: "example.com/orders",
        Entrypoints: []string{"http"}, HTTP: "stdlib"}
    if err := o.Validate(); err != nil { t.Fatal(err) }
}

func TestOptionsRejectInvalidSelections(t *testing.T) {
    cases := []generate.Options{
        {Name: "../orders", Module: "example.com/orders", Entrypoints: []string{"grpc"}},
        {Name: "CON", Module: "example.com/orders", Entrypoints: []string{"grpc"}},
        {Name: "orders", Module: "../bad", Entrypoints: []string{"grpc"}},
        {Name: "orders", Module: "example.com/orders"},
        {Name: "orders", Module: "example.com/orders", Entrypoints: []string{"http"}},
        {Name: "orders", Module: "example.com/orders", Entrypoints: []string{"grpc", "grpc"}},
        {Name: "orders", Module: "example.com/orders", Entrypoints: []string{"grpc"}, Broker: "kafka"},
        {Name: "orders", Module: "example.com/orders", Entrypoints: []string{"consumer"}},
        {Name: "orders", Module: "example.com/orders", Entrypoints: []string{"scheduler"}},
    }
    for i, o := range cases {
        if o.Validate() == nil { t.Errorf("case %d accepted", i) }
    }
}
```

- [ ] Run `go test ./internal/generate -run TestOptions`. Expected: the valid request fails against the stub.
- [ ] Run `go get golang.org/x/mod@latest` once during implementation; retain the resolved version and checksums. Use `module.CheckPath` rather than a home-grown module grammar.
- [ ] Implement project-name validation with `^[A-Za-z][A-Za-z0-9_-]*$`; additionally reject case-insensitive CON, PRN, AUX, NUL, COM1–COM9, and LPT1–LPT9. This documented narrow policy prevents platform-specific paths and Windows device names.
- [ ] Validate nonempty, known, unique entry points; enforce conditional required flags; allow only the approved providers and optional empty database. Include the rejected flag/value and accepted choices in errors. Preserve canonical output order `http,grpc,consumer,scheduler` without mutating caller slices.
- [ ] Extend the test table with every provider, unknown entry points, unsupported database/router/broker/schedule values, and technology flags without their entry point. Run `go test ./internal/generate -run TestOptions`; expected PASS.

## Task 3: Render the minimal HTTP project

**Create:** `internal/generate/render.go`, `render_test.go`; templates `base/go.mod.tmpl`, `base/README.md.tmpl`, `base/env.example.tmpl`, `base/domain.README.md.tmpl`, `base/ports.README.md.tmpl`, `base/readiness.go.tmpl`, `http/main.go.tmpl`, `http/config.go.tmpl`, `http/stdlib.go.tmpl`, `http/http_test.go.tmpl`.

- [ ] Add a boundary test that calls `Render` with the valid HTTP request; assert output includes `cmd/http/main.go`, `internal/application/readiness.go`, the HTTP adapter/config, `go.mod`, `.env.example`, and README files, and contains no gRPC, database, broker, or scheduler dependency. Compare two renders byte-for-byte.
- [ ] Run `go test ./internal/generate -run TestRenderHTTP`; expected FAIL against the render stub.
- [ ] Embed all template assets, including generated binding assets and documentation.

```go
//go:embed all:templates
var templates embed.FS
```

Use `text/template` with `Option("missingkey=error")`; validate first, explicitly select source/destination pairs, reject duplicate destinations, sort by destination, and run `format.Source` for every emitted `.go` file. Render quoted Go strings with `strconv.Quote` where user input appears as a literal. Return rendering/format errors before filesystem mutation.

- [ ] Define the generated application operation and HTTP adapter seam as concrete functions.

```go
// generated internal/application/readiness.go
package application

import "context"

func Ready(ctx context.Context) error { return ctx.Err() }
```

Generated HTTP contract: `func Handler(ready func(context.Context) error) http.Handler`. The standard adapter uses `http.NewServeMux` with `GET /readyz`, applies a two-second readiness context, returns status 200 plus `ready\n` on success and 503 plus `not ready\n` on failure. Do not serialize dependency errors or credentials to clients.

- [ ] Generate `HTTP_ADDR`, defaulting to `127.0.0.1:8080`, and a server with a five-second `ReadHeaderTimeout`. Main uses `signal.NotifyContext`, starts serving, and calls `Shutdown` with a fresh ten-second context; use `Close` if the graceful deadline expires. Treat `http.ErrServerClosed` as normal. Main calls an error-returning `run` function so deferred cleanup runs before `os.Exit`.
- [ ] Generate an adapter test using `httptest.NewRecorder` and `httptest.NewRequest`; inject a readiness function recording whether it was called. Verify success, failed readiness, and no unexpected business route.
- [ ] Run `go test ./internal/generate -run TestRenderHTTP`; expected PASS. No infrastructure or business model is part of this milestone.

## Task 4: Own destination creation and failure cleanup

**Create:** `internal/generate/write.go`, `write_test.go`, `generate.go`, `generate_test.go`.
**Modify:** remove corresponding stubs from `options.go`.

- [ ] Write external boundary tests for an existing empty directory, existing file, existing directory with a sentinel, and symlink where supported. Each must return an error and preserve content. Add concurrent writes to the same destination; exactly one succeeds.
- [ ] Add a failure case whose output path escapes through `../outside`; assert both destination and outside path remain absent. Also reject absolute paths, drive-qualified paths, backslashes, reserved components, duplicate paths, and file/directory collisions before claiming the destination.
- [ ] Run `go test ./internal/generate -run "TestWrite|TestGenerate"`; expected FAIL because the stubs do not create valid output.
- [ ] Implement Write with exclusive `os.Mkdir` for the destination; do not use `MkdirAll` for its root. For each validated relative file, create only required child directories and use exclusive creation flags. Use Go rooted filesystem operations (`os.OpenRoot`) to keep operations beneath the owned destination.
- [ ] Track files and directories actually created, including file identity. On failure, remove owned files only if identity still matches, then remove owned directories in reverse order using nonrecursive remove. Unknown/replaced content must prevent removal and be reported as incomplete cleanup; never call recursive delete on the destination. Join cleanup errors with the original failure.
- [ ] Keep failure injection package-private: a writer helper takes a file-write function so tests can fail the second write. This is a test seam, not a public filesystem interface. Test a partial file write and an inserted foreign file; foreign content survives.
- [ ] Implement Generate in order: Validate, Render, Write; return the joined destination path only on success. No post-write rendering, shell execution, or dependency downloads.
- [ ] Run `go test ./internal/generate -run "TestWrite|TestGenerate"`; expected PASS. Run concurrency checks with `-race` only when the local toolchain supports race instrumentation.

## Task 5: Complete the CLI-to-running-HTTP slice

**Create:** `cmd/hexgen/main.go`, `internal/cli/run_test.go`, `tests/acceptance/http_test.go`.
**Modify:** `internal/cli/run.go`.

- [ ] Define a black-box acceptance test: invoke `cli.Run` into a temporary parent, require exit 0, run the generated module tests, build its HTTP binary, start on a test-selected free port, poll `/readyz` with a deadline, assert the response, and terminate/reap the process in cleanup. Use argument-array `exec.CommandContext`, bounded output, and a test timeout.
- [ ] Add CLI tests for help, absent subcommand, absent name/module/entrypoints, extra arguments, unknown flags, duplicates in both `--http=value` and `--http value` forms, and unsupported combinations. Verify errors leave the parent empty.
- [ ] Run `go test ./internal/cli ./tests/acceptance -run "TestCLI|TestHTTPWalkingSkeleton"`; expected FAIL.
- [ ] Parse `new` and the name before passing remaining arguments to a `flag.FlagSet` with `ContinueOnError`. Register string flag values that record whether Set was already called and reject repeated singleton flags. Reject positional arguments remaining after flag parsing. Treat explicit help as exit 0; syntax/validation errors as 2; generation I/O errors as 1.
- [ ] Wire the process entry without hidden global work.

```go
package main

import (
    "fmt"
    "os"
    "hexgen/internal/cli"
)

func main() {
    parent, err := os.Getwd()
    if err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
    os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr, parent))
}
```

- [ ] Success output names the project and README; generated README lists module download/build/run commands and explicitly says `.env.example` is documentation, not automatically loaded.
- [ ] Run `go test ./internal/... ./tests/acceptance -run "TestCLI|TestHTTPWalkingSkeleton|TestOptions|TestRender|TestWrite|TestGenerate"`. Expected PASS. This is the first complete runnable milestone.

## Task 6: Add Chi and scope configuration by executable

**Create:** `templates/http/chi.go.tmpl`, `tests/acceptance/chi_test.go`.
**Modify:** renderer/dependency selection, HTTP configuration/docs templates.

- [ ] Reuse the generated HTTP Handler contract; add a Chi output test asserting its direct dependency exists only when Chi is selected.
- [ ] Build a real generated Chi project in a temporary module, resolve `github.com/go-chi/chi/v5@latest` during development, record the resulting exact version in the renderer, and test the same readiness behaviors as the standard adapter. Generated module files use the pinned version, never `latest`.
- [ ] Implement Chi routing with the same success/failure semantics and server lifecycle as stdlib; common lifecycle belongs in the generated executable, not duplicated across router adapters.
- [ ] Add generated config functions `HTTPAddress() string`, later `GRPCAddress() string`, `DatabaseURL() (string,error)`, `Consumer() (ConsumerConfig,error)`, and `Scheduler() (SchedulerConfig,error)` only as their integration tasks require them. Do not generate a global config load that requires broker variables for HTTP.
- [ ] Run `go test ./tests/acceptance -run "TestHTTPWalkingSkeleton|TestChiReadiness"` and renderer tests. Expected PASS.

## Task 7: Add gRPC with bundled protobuf bindings

**Create:** `templates/grpc/main.go.tmpl`, `adapter.go.tmpl`, `config.go.tmpl`, `readiness.proto`, `readiness.pb.go.tmpl`, `readiness_grpc.pb.go.tmpl`, `grpc_test.go.tmpl`, `tests/acceptance/grpc_test.go`.

- [ ] Define the protocol before the adapter.

```proto
syntax = "proto3";
package readiness.v1;
option go_package = "hexgen.template/api/proto;readinessv1";
service Readiness {
  rpc Check(CheckRequest) returns (CheckResponse);
}
message CheckRequest {}
message CheckResponse { bool ready = 1; }
```

- [ ] Generate the bindings with source-relative paths using installed protoc and pinned `protoc-gen-go`/`protoc-gen-go-grpc` tools installed into a workspace tool directory. Record tool versions in generated README instructions. Resolve/pin runtime dependencies `google.golang.org/grpc` and `google.golang.org/protobuf` against Go 1.27.
- [ ] Bundle actual compiler-generated code. Render the proto go_package to the requested module, and patch the generated descriptor Go-package option through protobuf descriptor serialization during generation so reflection agrees with the source. Do not hand-edit binary descriptor bytes or only replace printable import strings. Keep this transformation private to the renderer and include its protobuf dependency in the generator only when necessary for correct descriptors.
- [ ] Adapter contract: `func Register(server *grpc.Server, ready func(context.Context) error)`. Test via `bufconn`: Check calls readiness and returns ready=true; a failed check returns gRPC Unavailable without internal error details.
- [ ] Generate a separate gRPC main with `GRPC_ADDR` default `127.0.0.1:9090`, application wiring, bounded graceful stop followed by Stop, and listener/error cleanup.
- [ ] Test regeneration/source/binding agreement, including `FileOptions.GoPackage`. Remove protoc from the acceptance process PATH and verify generation, build, and the RPC still work.
- [ ] Run `go test ./tests/acceptance -run TestGRPC` and renderer tests. Expected PASS without invoking protoc during generation.

## Task 8: Add optional PostgreSQL lifecycle

**Create:** `templates/postgres/postgres.go.tmpl`, `config.go.tmpl`, `tests/acceptance/postgres_test.go`.
**Modify:** selected executable templates and dependency/docs selection.

- [ ] Define the generated outbound boundary before implementing it: `func Open(ctx context.Context, connectionString string) (*pgxpool.Pool,error)`. Use the native pool directly, not a one-implementation repository interface.
- [ ] Resolve/pin `github.com/jackc/pgx/v5`; add a fixture-backed test that Open succeeds for disposable PostgreSQL, fails under a bounded context for an unavailable endpoint, and exposes no credential string in public diagnostics.
- [ ] Open creates a pool and Pings it; on Ping failure close it before returning a sanitized error. Every selected executable reads DATABASE_URL only if PostgreSQL was selected, opens its own pool under a five-second startup context, and defers Close.
- [ ] Compose readiness in HTTP/gRPC main using a concrete closure that calls application.Ready and pool.Ping; no application import of pgx.
- [ ] Verify database omission leaves no pgx import, variable, file, or direct module requirement. Add the pool lifecycle to consumer/scheduler templates as those tasks are implemented.
- [ ] Run `go test ./tests/acceptance -run TestPostgres` with fixtures available. Expected PASS; report blocked rather than passed if Docker remains unavailable.

## Task 9: Define safe worker contracts and generated placeholders

**Create:** `templates/base/work.go.tmpl`, consumer/scheduler config templates and their generated tests.

- [ ] Define the application-owned operations only for selected workers.

```go
package application

import (
    "context"
    "errors"
)

var ErrNotImplemented = errors.New("business handler not implemented")

// Emit HandleMessage only when consumer is selected.
func HandleMessage(ctx context.Context, payload []byte) error {
    return ErrNotImplemented
}

// Emit RunScheduled only when scheduler is selected.
func RunScheduled(ctx context.Context) error {
    return ErrNotImplemented
}
```

- [ ] Generated ConsumerConfig has Enabled bool plus provider-specific fields. Parse `CONSUMER_ENABLED` with `strconv.ParseBool`, default false, reject malformed values. RabbitMQ requires RABBITMQ_URL and RABBITMQ_QUEUE; Kafka requires KAFKA_BROKERS, KAFKA_TOPIC, KAFKA_GROUP_ID. Do not create queues/topics automatically.
- [ ] SchedulerConfig has Enabled bool and Spec string. Parse `SCHEDULER_ENABLED` the same way and require a valid five-field CRON_SCHEDULE; document UTC. Configuration is validated even when processing is disabled, because startup still proves the selected wiring.
- [ ] Test business placeholders return the application sentinel; defaults disable work; malformed booleans and missing provider configuration fail without credentials in diagnostics. Verify HTTP-only output includes neither work.go nor worker variables.
- [ ] Run generated worker configuration tests through the renderer acceptance harness; expected PASS.

## Task 10: Add RabbitMQ and Kafka consumer adapters

**Create:** `templates/consumer/main.go.tmpl`, `rabbitmq.go.tmpl`, `kafka.go.tmpl`, `consumer_test.go.tmpl`, `tests/acceptance/consumer_test.go`.

- [ ] Define the native-facing callback boundary: `func Run(ctx context.Context, cfg config.ConsumerConfig, handle func(context.Context, []byte) error) error`. Return setup/processing errors so main can shut down and exit nonzero. No shared broker-provider interface.
- [ ] Resolve/pin `github.com/rabbitmq/amqp091-go` and `github.com/segmentio/kafka-go` in separate generated fixture modules. Emit exactly one direct broker client requirement.
- [ ] RabbitMQ: use bounded dial, validate queue existence without changing topology, manual acknowledgements, and prefetch 1. Disabled processing opens the connection/channel but does not register a consumer. Enabled processing calls handle then Ack only on nil error. On handler error leave the message unacknowledged, stop, and close the channel so the broker can redeliver; do not loop on the unfinished handler. Context cancellation cancels/ends consumption and closes resources.
- [ ] Kafka: validate connectivity/topic metadata during startup. Disabled processing must not join the consumer group or fetch messages. Enabled processing uses `FetchMessage` followed by `CommitMessages` only after handler success; never use `ReadMessage` auto-commit. Use synchronous commits and stop on handler error. Close the reader on exit and bound network operations.
- [ ] Test with real fixtures: publish one message, start disabled consumer and confirm it remains; enable with the unfinished handler and confirm unsuccessful processing preserves RabbitMQ delivery/Kafka group offset; inject a successful callback to prove normal acknowledgement/commit. Verify both adapters exit on cancellation.
- [ ] Generate consumer main with signal context, optional database pool, a clear disabled-state log, and cleanup before returning an error.
- [ ] Run `go test ./tests/acceptance -run "TestRabbitMQ|TestKafka"` with fixtures. Expected PASS.

## Task 11: Add the scheduled worker

**Create:** `templates/scheduler/main.go.tmpl`, `scheduler.go.tmpl`, `scheduler_test.go.tmpl`, `tests/acceptance/scheduler_test.go`.

- [ ] Define `func Run(ctx context.Context, cfg config.SchedulerConfig, job func(context.Context) error) error`. Resolve/pin `github.com/robfig/cron/v3`.
- [ ] Implement UTC five-field cron parsing; fail invalid schedules before entering the run loop. Disabled processing validates then waits for cancellation without scheduling a job. Enabled processing uses cron with overlapping runs skipped. Pass the cancellable application context to each invocation.
- [ ] A job failure, including ErrNotImplemented, reaches Run through a buffered error channel and terminates scheduling with a nonzero process result. On context cancellation or job failure stop cron; await active jobs within the ten-second shutdown deadline. Do not report failed work as success.
- [ ] Test the callback/error boundary with direct callback invocation and a package-private scheduler timing seam; do not slow unit tests by waiting a full minute. Include one real cron-trigger runtime acceptance test with an explicit time allowance. Check disabled mode and cancellation separately.
- [ ] Generate scheduler main with its own configuration, optional pool, signal handling, and log identifying disabled processing.
- [ ] Run `go test ./tests/acceptance -run TestScheduler` and generated unit tests. Expected PASS.

## Task 12: Complete the selection/build matrix and runtime fixtures

**Create:** `tests/acceptance/matrix_test.go`, `runtime_test.go`, `tests/fixtures/compose.yaml`.
**Modify:** rendering/documentation tests for full output support.

- [ ] Enumerate choices mechanically, not a hand-maintained sample list: HTTP absent/stdlib/Chi (3), gRPC absent/present (2), consumer absent/RabbitMQ/Kafka (3), scheduler absent/present (2), and database absent/PostgreSQL (2). Exclude the two combinations with no entry points. Assert exactly `3*2*3*2*2-2 = 70` valid generated projects.
- [ ] For each project, Render/Generate into an isolated temporary directory and assert expected/forbidden paths, environment variables, direct dependencies, deterministic bytes, formatted source, and inward import boundaries. Inspect imports with Go parser/AST, not fragile string matching.
- [ ] Run `go mod tidy` and `go test ./...` inside each generated project using argument-array subprocesses, a shared dependency cache, a bounded timeout, and bounded parallelism. Run `go list -m all` to verify module resolution. Generated go.mod templates must include the tidy-consistent selected module requirements for Go 1.27; do not leave builds dependent on undocumented automatic module edits.
- [ ] Dependency closure is a maintainer-generated embedded asset per distinct dependency selection if necessary; derive it from actual tidy output, retain exact versions, and avoid runtime dependency resolution. Full go.sum files can be generated/embedded with those assets to support the documented build command.
- [ ] Create test-only Compose services for PostgreSQL, RabbitMQ, and single-node Kafka, pinned to versions resolved during implementation. Bind fixture ports to loopback, use disposable named volumes/test credentials, and give each fixture project a unique name. No generated deployment stack is added.
- [ ] Gate infrastructure tests with `HEXGEN_INTEGRATION=1`; if set, missing Docker/fixtures is a failure, not a skip. Default tests report explicit integration skips. Bound startup with service health checks. Fixture cleanup acts only on the uniquely named test project.
- [ ] Start all four executables from one all-options generated project using a chosen broker; require HTTP/gRPC readiness, disabled worker startup, independent termination, and no cross-executable variable requirements. The other broker has its own provider test.
- [ ] Run `go test ./...`; then set `HEXGEN_INTEGRATION=1` and run `go test ./tests/acceptance -timeout 20m`. Expected PASS once infrastructure prerequisites are satisfied. Keep compilation-matrix tests separate from slow real cron/broker checks so failures are attributable.

## Task 13: Finish user documentation and focused verification

**Create/modify:** `README.md`, generated README/.env templates, `CONTEXT.md`, `state_machines.md`, `data_flow.md`.

- [ ] Document the two approved example commands, exact supported values, conditional flags, no-overwrite behavior, offline generation versus dependency downloads during building, and name restrictions.
- [ ] Every generated README includes only selected run commands, its environment variables, setup instructions for external services, disabled-worker behavior, where to implement business handlers, and protobuf regeneration commands only when gRPC is selected. Examples contain no credentials.
- [ ] Use the project-context skills at implementation time to establish/maintain vocabulary, generation ownership transitions (validate/render/claim/write/complete/cleanup), and dependency/runtime flows. Do not put implementation notes into the glossary.
- [ ] Build with `go build -o ./bin/hexgen.exe ./cmd/hexgen` on Windows (omit `.exe` on Unix). Run the binary help and both documented generation examples in a temporary working directory, then test their generated modules.
- [ ] Run `go test ./...` and `go vet ./...`. Confirm the 70-case generation/build matrix and runtime provider/composition checks have actually passed; do not substitute successful rendering for a successful build.
- [ ] Inspect final diff/file inventory for temporary artifacts, credentials, unsupported options, generator stubs, and unrelated changes. Retain only source, templates, required pinned assets, tests, and documentation.
- [ ] Report implemented behavior and exact verification results. If Docker/network restrictions prevent required integration checks, identify those checks as unresolved and do not declare full acceptance.

## Acceptance mapping and self-review

| Design requirement | Plan tasks |
|---|---|
| CLI flags and actionable validation | 1, 2, 5 |
| Bundled offline generation; deterministic output | 3, 7, 12 |
| Runnable HTTP walking skeleton | 3–5 |
| Standard HTTP and Chi | 3, 6 |
| gRPC without generator-time protoc | 7 |
| Optional PostgreSQL wiring | 8 |
| Safe disabled/unfinished worker behavior | 9–11 |
| RabbitMQ and Kafka | 10 |
| Cron scheduling | 11 |
| Separate executables; graceful shutdown | 3, 7, 8, 10–12 |
| Selected-only files/config/dependencies | 3, 6–12 |
| No overwrite, concurrency, safe cleanup | 4 |
| All 70 valid combinations compile | 12 |
| Runtime providers plus composition | 8, 10–12 |
| Documentation and architectural boundaries | 12, 13 |

Review result: all approved feature areas have an implementation/verification task. The material environment limitation is Docker availability; external dependency access must also be exercised during execution. The plan adds no remote templates, business model, deployment system, or provider/plugin abstraction.
