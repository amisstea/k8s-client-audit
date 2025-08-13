## k8s-client-audit

Lightweight Go static analyzers focused on common Kubernetes client anti-patterns. This repository provides:

- A multi-checker CLI to lint Go codebases for Kubernetes client misuse
- An optional helper to clone all repositories in a GitHub organization
- A wrapper script to run the linter across many repositories/modules

### Included analyzers
All analyzers live under `internal/analyzers`. The following are currently wired into the main multi-checker (enabled), unless noted as disabled.

- clientreuse: flags constructing Kubernetes clients inside loops or hot paths; prefer a singleton client
- qpsburst: flags `rest.Config` QPS/Burst that are zero/unlimited or extremely high
- missinginformer: flags direct `Watch` calls when no shared informer/cache usage is detected
- listinloop: flags `List`/`Watch` calls inside loops (prefer informers/cache or move outside loops)
- manualpolling: flags loops that poll with `List` + `sleep`/ticker; prefer `Watch`/informers
- unboundedqueue: flags workqueue construction without a rate limiter
- requeuebackoff: flags controller-runtime `Reconcile` paths that requeue immediately without backoff
- noselectors: flags `List` calls that lack label/field selectors or options
- widenamespace: flags all-namespaces listing heuristics like `InNamespace("")` or typed `Pods("").List`
- largepages: flags excessively large `ListOptions.Limit` values
- tighterrorloops: flags tight loops retrying errors around Kubernetes API calls without backoff
- missingcontext: flags client calls using `context.Background/TODO` instead of propagated context
- leakywatch: flags `Watch` result channels that are never stopped/cancelled
- restconfigdefaults: flags `rest.Config` initialization missing timeouts or UserAgent
- dynamicoveruse: flags use of dynamic/unstructured clients when typed clients appear available
- unstructuredeverywhere: flags pervasive use of `unstructured.Unstructured` rather than typed objects
- webhook_nocontext: flags webhook handlers that use `context.Background/TODO` instead of request context
- discoveryflood: flags repeated discovery/RESTMapper creations inside loops
- restmapper_not_cached: flags creation of discovery-based RESTMapper without a caching wrapper
- webhook_timeouts: flags webhook HTTP clients/servers without timeouts — disabled by default (not registered in `cmd/k8s-client-audit/main.go`)

Note: Analyzer names above match the `analysis.Analyzer.Name` used in output.

### Commands

There are two optional utilities under `cmd/`:

- `cmd/k8s-client-audit`: the multi-checker linter binary (primary tool)
- `cmd/clone-github-org`: helper to clone or update all repositories in a GitHub organization

An optional wrapper script is available in `scripts/`:

- `scripts/lint-github-org.sh`: finds all Go modules under a directory and runs the linter on each

### Build

Build the linter:

```bash
go build -o k8s-client-audit ./cmd/k8s-client-audit
```

Build the GitHub org cloner (optional):

```bash
go build -o clone-github-org ./cmd/clone-github-org
```

### Usage

Run the linter in the current module:

```bash
./k8s-client-audit ./...
```

Run the linter excluding tests:

```bash
./k8s-client-audit -test=false ./...
```

Get linter help:

```bash
./k8s-client-audit -h
```

Run the GitHub org cloner (optional):

```bash
./clone-github-org -org YOUR_ORG -dest sources
```

Environment for cloner:

- `GITHUB_TOKEN` (optional): increases rate limits and accesses private repos if permitted

Cloner help:

```bash
./clone-github-org -h
```

Run the wrapper script over many repos/modules (optional):

```bash
./scripts/lint-github-org.sh sources
```

Notes for the wrapper:

- Expects `k8s-client-audit` to be in `PATH`
- Argument defaults to `sources` if omitted
- Recursively discovers `go.mod` files (excluding vendor) and runs `k8s-client-audit -test=false ./...` in each module

### Help and flags

All commands support `-h`/`--help` to display usage and flags. The linter accepts standard `multichecker` flags (e.g., `-json`, `-vettool` style integration) in addition to its positional package patterns like `./...`.

