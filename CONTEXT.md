# Hexagonal Project Generation

Vocabulary for a flag-driven generator of Go projects with hexagonal dependency boundaries.

## Language

**Generator**:
The command-line tool that creates a new Go project from bundled templates and explicit selections.
_Avoid_: Folder maker, generated service

**Generated project**:
The Go module produced by the generator, containing the selected executable entry points and shared application locations.
_Avoid_: Template, generator

**Entry-point selection**:
The set of independently runnable HTTP, gRPC, consumer, and scheduler executables requested for a generated project.
_Avoid_: Service type when it implies one type per project

**Consumer**:
The executable that receives messages from the selected RabbitMQ or Kafka broker and invokes application behavior.
_Avoid_: Worker without qualification

**Scheduler**:
The executable that invokes application behavior on a cron schedule.
_Avoid_: Worker without qualification

**Technology selection**:
A supported router, database, broker, or scheduler choice that determines which integration is generated.
_Avoid_: Plugin

**Business placeholder**:
An intentionally unfinished application operation that cannot report successful processing until implemented.
_Avoid_: Working business feature, sample use case
