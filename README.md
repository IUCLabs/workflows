# workflows

Shared reusable GitHub Actions workflows for IUCLabs repos. Every third-party
action is pinned by commit SHA; Dependabot bumps them weekly.

| Workflow | What it does |
|---|---|
| `go-ci.yml` | `pre-commit`, `lint` (go vet, optional golangci-lint), `test` (optional coverage gate) and `build` jobs |
| `release-vault-plugin.yml` | goreleaser release: RPMs, raw binary, checksums signed via the Vault gpg engine, and the `ghcr.io/iuclabs/<repo>:<tag>` plugin image |
| `release-terraform-provider.yml` | goreleaser release, then PUTs the linux_amd64 zip to the artifactapi `terraform-unkin` registry |
| `release-go-image.yml` | ko build of `ghcr.io/iuclabs/<repo>:<tag>`, then moves `latest` |

The goreleaser release workflows log in to Vault at
`https://vault.service.consul:8200` via `jwt/github/iuclabs`. They expect the
caller's `.goreleaser.yml` to keep `release: {use_existing_draft: true, mode:
keep-existing}`, so `make patch|minor|major` drafts the release and the tag
push fills and publishes it.

## Inputs

### go-ci.yml

Permissions: `contents: read`.

| Input | Type | Default | Description |
|---|---|---|---|
| `working_directory` | string | `.` | Directory holding go.mod; pre-commit always runs at the repo root |
| `go_vet` | boolean | `true` | Run `go vet ./...` in `lint` |
| `golangci_lint_version` | string | `""` | golangci-lint version; empty skips it |
| `golangci_lint_only_new_issues` | boolean | `false` | Report only issues introduced by the change |
| `test_args` | string | `-count=1 ./...` | Arguments to `go test` |
| `cover_min` | number | `0` | Minimum total coverage percent; 0 disables the gate |
| `cover_exclude` | string | `""` | Extended regex of coverage profile lines left out of the gate |
| `strip` | boolean | `true` | Build with `-trimpath -ldflags="-s -w"` |

### release-vault-plugin.yml

Permissions: `contents: write`, `id-token: write`, `packages: write`. Output:
`artifacts` (goreleaser `dist/artifacts.json`). The image comes from
`dockers_v2` in the caller's `.goreleaser.yml` (`FROM scratch`, binary only).

| Input | Type | Default | Description |
|---|---|---|---|
| `vault_role` | string | `release` | Role on the `jwt/github/iuclabs` mount |
| `snapshot` | boolean | `false` | `goreleaser release --snapshot --clean --skip=publish,sign`; no Vault, no ghcr login |
| `runs_on` | string | `iuclabs-release` | Runner label |
| `working_directory` | string | `.` | Directory holding `.goreleaser.yml` |

### release-terraform-provider.yml

Permissions: `contents: write`, `id-token: write`. Output: `artifacts`. The
upload goes to `terraform-unkin/files/<namespace>/<type>/`, with `<type>` taken
from the repo name `terraform-provider-<type>`.

| Input | Type | Default | Description |
|---|---|---|---|
| `namespace` | string | `unkin` | Provider namespace in the registry |
| `vault_role` | string | `release` | Role on the `jwt/github/iuclabs` mount; needs `identity/oidc/token/artifactapi-writer` |
| `snapshot` | boolean | `false` | Snapshot build; no Vault, signing, publishing or upload |
| `runs_on` | string | `iuclabs-release` | Runner label |
| `working_directory` | string | `.` | Directory holding `.goreleaser.yml` |

### release-go-image.yml

Permissions: `contents: read`, `packages: write`.

| Input | Type | Default | Description |
|---|---|---|---|
| `main` | string | `./cmd/<repo>` | Main package for ko |
| `push` | boolean | `true` | Push and move `latest`; `false` only builds |
| `runs_on` | string | `iuclabs-release` | Runner label |
| `working_directory` | string | `.` | Directory holding go.mod and `.ko.yaml` |

It builds the Go image only. Repos with extra images (artifactapi's UI) keep
that build and its `latest` tag in their own job with `needs:` on this one.

No secrets are passed: the workflows use the caller's `GITHUB_TOKEN` and Vault
OIDC.

## Example

```yaml
# .github/workflows/release.yml
name: release

on:
  push:
    tags: ["v*"]

permissions: {}

jobs:
  release:
    uses: IUCLabs/workflows/.github/workflows/release-vault-plugin.yml@v0.1.0
    permissions:
      contents: write
      id-token: write
      packages: write
```

```yaml
# .github/workflows/ci.yml, dry-running the release config on every PR
jobs:
  go-ci:
    uses: IUCLabs/workflows/.github/workflows/go-ci.yml@v0.1.0
    permissions:
      contents: read
  release-snapshot:
    uses: IUCLabs/workflows/.github/workflows/release-vault-plugin.yml@v0.1.0
    permissions:
      contents: write
      id-token: write
      packages: write
    with:
      snapshot: true
      runs_on: ubuntu-latest
```

Jobs run through a reusable workflow report as `<caller job> / <job>` (for
example `go-ci / lint`), so update required status checks when switching.

## Development

`make pre-commit`, `make lint` (actionlint and zizmor) and `make test` (runs
`act` when installed). CI self-tests every workflow against `testdata/` with
snapshot and no-push modes. Releases: `make patch|minor|major`.
