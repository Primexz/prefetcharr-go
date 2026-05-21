# Agent Guide

## Project Overview

prefetcharr-go is a small Go service that polls Jellyfin sessions and asks Sonarr to search upcoming seasons for watched TV series.
Core flow:

1. Load YAML config with defaults and validation from `internal/app/config.go`.
2. Create Jellyfin and Sonarr clients in `internal/app/app.go`.
3. Poll active Jellyfin sessions through `JellyfinClient.NowPlaying`.
4. Ignore non-episode sessions, resolve the series TVDB ID, then find the matching Sonarr series.
5. Compute target seasons with `targetSeasons`.
6. Skip missing, complete, or recently searched seasons.
7. Monitor the target Sonarr season and submit a Sonarr `SeasonSearch` command.

## Development Commands

Run tests:

```sh
go test ./...
```

Run the service locally:

```sh
go run ./cmd/prefetcharr -config config.yaml
```

Build the binary:

```sh
go build -o prefetcharr ./cmd/prefetcharr
```

Run linting when available:

```sh
golangci-lint run
```

Format Go code before finishing:

```sh
gofmt -w cmd internal
```

## Coding Guidelines

- Keep changes small and idiomatic Go.
- Prefer standard library APIs unless the existing dependency set already solves the problem.
- Keep service behavior centered in `internal/app`; keep `cmd/prefetcharr` limited to process setup.
- Thread `context.Context` through network calls.
- Preserve API key handling through headers/context as used by the current Jellyfin and Sonarr clients.
- Avoid logging secrets or full API URLs with credentials.
- Update `config.example.yaml` and `README.md` when config behavior changes.

## Configuration Notes

- Durations may be YAML strings such as `300s` or numeric seconds.

## Conventional Commits

Use Conventional Commits for all commit messages:

```text
<type>(optional-scope): <description>
```

Add a short commit body that explains what changed and why when creating commits.

Common types for this repo:

- `feat`: user-visible behavior or configuration additions.
- `fix`: bug fixes.
- `docs`: README, examples, or agent/contributor documentation.
- `test`: tests only.
- `refactor`: code restructuring without behavior changes.
- `chore`: maintenance, dependency, release, or tooling changes.
- `ci`: GitHub Actions or release workflow changes.

Examples:

```text
feat(prefetch): include monitored specials
fix(sonarr): handle failed season search responses
docs: clarify docker compose config path
test(config): cover numeric duration parsing
```

## Conventional Branches

Use conventional, lowercase, hyphen-separated branch names:

```text
<type>/<short-description>
```

Branch type should match the intended Conventional Commit type.

Examples:

```text
feat/include-current-season
fix/sonarr-command-error
docs/update-config-example
test/dedupe-ttl
chore/update-goreleaser
```

Keep branch names short, descriptive, and scoped to one change.
