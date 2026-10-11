# Go Hexagonal Project Generator — Design

Date: 2026-10-10
Status: Approved by the user on 2026-10-10; implementation planning authorized.

## Outcome

Build a Go command-line generator, provisionally named `hexgen`, that creates a compilable Go module from bundled templates. Flags select independently runnable HTTP, gRPC, queue-consumer, and scheduled-worker entry points plus curated infrastructure integrations. Generated applications share application and domain locations without inventing a sample business feature.

## Approved walking skeleton

1. Invoke the generator with a project name, module path, and standard-library HTTP selection.
2. Validate the request and require a new destination.
3. Render a Go module with an HTTP executable and a trivial application readiness operation.
4. Start the generated executable; its readiness handler calls the application operation and returns a fixed successful response.
5. Run `go test ./...` in the generated module and verify one HTTP request end to end.

The seams are generation inputs to project output/error, and HTTP adapter to application readiness operation. Use concrete Go functions/types unless a real substitution boundary requires an interface. There is no database requirement or invented persistence port for this slice. Extend this working path with each approved integration.

## Approach and alternatives

Use bundled templates embedded in the generator binary. This makes template selection deterministic and generation itself offline. Downloading Go dependencies when building a generated project may still require network access. Remote templates add versioning and download concerns; a single starter repository makes optional combinations harder to compose. Neither is part of this release.

The generator itself uses the Go standard library for CLI flag parsing, embedding, template rendering, and filesystem operations where suitable. It does not need to imitate the architecture it generates. Separate option validation, rendering, and filesystem ownership sufficiently to test their boundaries; do not introduce plugin interfaces or a dependency injection framework.

## CLI contract

```sh
hexgen new orders --module github.com/yourname/orders --entrypoints http,grpc,consumer,scheduler --http chi --database postgres --broker rabbitmq --schedule cron

hexgen new orders --module github.com/yourname/orders --entrypoints http --http stdlib
```

- `new` accepts one project name, used as a direct child directory of the current directory. Reject path traversal, separators, reserved platform names, and invalid names. No arbitrary output-path flag in this release.
- `--module` is required and must be a valid Go module path. Escape/render user-supplied strings safely; never execute them as shell commands.
- `--entrypoints` is a required nonempty comma-separated subset of `http,grpc,consumer,scheduler`. Reject unknown names and duplicates.
- HTTP requires `--http stdlib` or `--http chi`.
- gRPC uses gRPC-Go; there is no provider flag with only one supported value.
- Consumer requires `--broker rabbitmq` or `--broker kafka`.
- Scheduler requires `--schedule cron`. This explicit selection preserves the agreed CLI shape.
- Optional `--database postgres` selects PostgreSQL through pgx; omission means no database integration.
- Reject technology flags whose corresponding entry point is absent, unknown flags, unsupported values, repeated singleton flags, and unexpected positional arguments.
- One technology per category per generated project.
- Provide command help, actionable errors, nonzero exit status on failure, and a concise success message pointing to the generated README.

## Generated layout

```text
myservice/
  cmd/
    http/main.go
    grpc/main.go
    consumer/main.go
    scheduler/main.go
  internal/
    domain/
    application/
      ports/
    adapters/
      inbound/
        http/
        grpc/
        consumer/
        scheduler/
      outbound/
        postgres/
    config/
  api/
    proto/
  go.mod
  .env.example
  README.md
```

This is the union of supported output locations, not the output for every invocation. Include selected entry points and integrations only. Empty architectural locations contain short instructional READMEs rather than fake entities or interfaces. `api/proto` exists only with gRPC.

## Dependency boundaries

- Domain contains future business concepts and rules; it imports no application or infrastructure packages.
- Application contains use cases and the outbound ports those use cases require. It does not import inbound/outbound adapters, configuration, or transport libraries.
- Inbound adapters translate requests, messages, or triggers into application calls. Queue consumption is inbound; message publishing is outside this release.
- Outbound adapters implement actual application ports as business features are added. Initially PostgreSQL provides connection lifecycle scaffolding, not an invented repository implementation.
- Each executable loads its configuration, constructs its dependencies, and owns its lifecycle. Executables share source code, not live objects across processes.
- Avoid generic Repository, Service, Manager, or dependency-container interfaces without a concrete use case. Infrastructure setup may exist before a business persistence port exists.

## Runtime behavior

Generated modules compile, and each selected executable has real initialization and graceful shutdown wiring. Business operations remain explicit placeholders.

HTTP exposes `/readyz`; gRPC exposes a minimal readiness RPC whose `.proto` source and generated Go code are bundled. Both call the trivial application readiness operation. No business endpoints or invented business API are generated. Protobuf tooling is unnecessary during generation and initial build. Document compatible regeneration commands/tools for users changing the API.

Consumer and scheduler executables connect/setup their selected infrastructure but start with business processing disabled. Log that state clearly. A documented environment setting enables processing only after the user implements the handler. A still-unimplemented handler returns an explicit error. RabbitMQ must not acknowledge it as successful; Kafka must not commit its offset as successful. Stop processing on that placeholder error rather than creating a tight retry loop. Scheduled placeholders report failure rather than success. Broker retry policies for real business failures belong to the future application.

PostgreSQL selection supplies a shared pool-construction facility and a pool owned by each selected executable, validating connectivity at startup and closing on exit. This verifies the selected scaffold without inventing business persistence. HTTP/gRPC readiness additionally checks their configured pool when present. Broker configuration/connections belong only to the consumer executable; scheduler configuration belongs only to the scheduler.

Use environment variables for runtime configuration. `.env.example` documents variables but is not automatically loaded. Do not generate real credentials or usable secret defaults. Required configuration is validated before serving/processing, and connection/startup failures produce clear errors and a nonzero exit. Use bounded startup/shutdown waits and respond to supported termination signals. Never log credentials.

README documents entry-point commands, environment variables, required external services, business-handler locations, and infrastructure/security setup needed before external deployment. Do not imply production readiness; deployment manifests, authentication systems, and TLS automation are outside this release.

## Filesystem safety and reproducibility

Validate selections and render all output in memory before creating output files. Reject an already-existing destination, including an empty directory or symlink. Atomically claim the destination by creating it exclusively; never overwrite or merge. Ensure all generated paths remain within the claimed directory.

If writing fails, remove only files/directories created by this invocation. Never recursively delete a preexisting destination or unrecognized content. If cleanup cannot safely finish, report the incomplete output location and original failure. A crash may leave an incomplete directory; subsequent invocations must refuse it rather than overwrite it.

For identical selections and module path, generated contents are deterministic; avoid timestamps and random identifiers in output. Format generated Go source. Dependencies are version-pinned and restricted to selected technologies and their required transitive dependencies. Do not run shell commands or download dependencies during generation.

## Verification and acceptance

1. The walking-skeleton HTTP request succeeds through the application boundary.
2. Every valid selection combination can be rendered and its generated module passes `go test ./...`; include each of the 15 nonempty entry-point subsets and all applicable provider/database choices. Cache dependency downloads and enumerate the matrix mechanically.
3. Generated output contains no packages, configuration, or direct dependencies for unselected integrations. Shared dependencies of selected technologies are allowed.
4. Invalid flags/module paths/project names fail before filesystem mutation.
5. Existing directories/files/symlinks remain unchanged; concurrent generation to the same destination has one winner without overwriting. Injected write failures exercise safe cleanup.
6. Runtime integration checks cover both HTTP routers, gRPC readiness without local protoc, PostgreSQL connection/readiness failures, RabbitMQ and Kafka setup, scheduler startup, and graceful shutdown. Use disposable infrastructure fixtures; no production endpoints.
7. Consumer placeholders never acknowledge/commit success; default workers do not process work. Scheduler placeholders cannot report business success.
8. Validate generated import boundaries and formatting. README commands match actual flags and executable names.

Generation/build checks cover the full selection matrix; runtime tests cover individual providers plus an all-entry-points composition rather than duplicating every runtime combination.

## Non-goals

No interactive wizard, web UI, remote/custom templates, plugin system, multiple providers within a category, sample CRUD feature, generic repositories, migrations, queue publishing, deployment scaffolding, or modifications to existing generated projects. No application code is implemented during this brainstorming stage.

## Review notes

Workspace inspection found no existing files or Git repository. This design and glossary can be saved locally, but there is no existing repository in which to commit them. The next step after written-spec approval is an implementation plan.
