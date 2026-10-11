# Dependency provenance

The golang.org/x/mod packages in this directory are unmodified copies of the upstream packages bundled in Go 1.27.1 at src/cmd/vendor/golang.org/x/mod. The version is pinned in go.mod and vendor/modules.txt: v0.36.1-0.20260813213634-8569e2639ca1.

Included packages: module, semver, internal/lazyregexp. LICENSE and PATENTS are preserved. The machine's Go and Node network requests to proxy.golang.org were blocked, so the installed Go distribution supplied these public upstream sources. Normal builds use this checked-in vendor directory and require no dependency download.
