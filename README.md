# workflows

Shared reusable GitHub Actions workflows for IUCLabs repos. Every third-party
action is pinned by commit SHA; Dependabot bumps them weekly.

| Workflow | What it does |
|---|---|
| `pre-commit.yml` | `uvx pre-commit run --all-files`, or [prek](https://github.com/j178/prek) |
| `actions.yml` | `lint` (actionlint, with shellcheck on `run:` blocks) and `audit` (zizmor) |
| `go-ci.yml` | `lint` (go vet, optional golangci-lint), `test` (optional coverage gate) and `build` jobs |
| `terraform.yml` | `fmt`, `tflint`, `validate`, a lock-free terragrunt `plan`, and `apply` on push to the default branch, queued per repo |
| `release-vault-plugin.yml` | goreleaser release: RPMs, raw binary, checksums signed via the Vault gpg engine, and the `ghcr.io/iuclabs/<repo>:<tag>` plugin image |
| `release-terraform-provider.yml` | goreleaser release, then PUTs the linux_amd64 zip to the artifactapi `terraform-unkin` registry |
| `release-go-image.yml` | ko build of `ghcr.io/iuclabs/<repo>:<tag>`, then moves `latest` |

The goreleaser release workflows log in to Vault at
`https://vault.service.consul:8200` via `jwt/github/iuclabs`. They expect the
caller's `.goreleaser.yml` to keep `release: {use_existing_draft: true, mode:
keep-existing}`, so `make patch|minor|major` drafts the release and the tag
push fills and publishes it.

Every workflow takes a `runs_on` string input (runner label) used by all of its
jobs: `ubuntu-latest` for CI workflows, `iuclabs-release` for release ones and
`iuclabs-infra` for terraform ones.

## Caller job ids

Checks from a reusable workflow render as `<caller job id> / <callee job id>`.
Use these caller job ids so every repo reports the same names; those names are
what branch protection requires.

| Caller job id | Uses | Checks rendered |
|---|---|---|
| `ci` | `pre-commit.yml` | `ci / pre-commit` |
| `actions` | `actions.yml` | `actions / lint`, `actions / audit` |
| `go` | `go-ci.yml` | `go / lint`, `go / test`, `go / build` |
| `terraform` | `terraform.yml` | `terraform / fmt`, `terraform / tflint`, `terraform / validate`, `terraform / plan`, `terraform / apply` |

## Inputs

### pre-commit.yml

Permissions: `contents: read`.

| Input | Type | Default | Description |
|---|---|---|---|
| `tool` | string | `pre-commit` | `pre-commit` or `prek`, run via `uvx`; prek treats `working_directory` as its project root |
| `runs_on` | string | `ubuntu-latest` | Runner label |
| `working_directory` | string | `.` | Directory holding `.pre-commit-config.yaml`; hooks still see every file in the repo |
| `python_version` | string | `""` | Python for uv to run pre-commit with; empty uses uv's default |
| `go_version_file` | string | `""` | go.mod or `.go-version` path from the repo root, for hooks that need Go; empty skips setup-go |

### actions.yml

Permissions: `contents: read`. actionlint v1.7.12 and zizmor 1.32.0 are pinned;
`lint` fails if shellcheck is not on the runner. zizmor reports as annotations,
so no `security-events: write`.

| Input | Type | Default | Description |
|---|---|---|---|
| `runs_on` | string | `ubuntu-latest` | Runner label |
| `working_directory` | string | `.` | Directory holding the `.github/` to check |
| `min_severity` | string | `low` | Lowest zizmor severity that fails: `unknown`, `informational`, `low`, `medium`, `high` |
| `zizmor_config` | string | `""` | zizmor config path relative to `working_directory`; empty uses zizmor's discovery |

### go-ci.yml

Permissions: `contents: read`, `pull-requests: read` (lint only-new-issues).

| Input | Type | Default | Description |
|---|---|---|---|
| `runs_on` | string | `ubuntu-latest` | Runner label |
| `working_directory` | string | `.` | Directory holding go.mod |
| `go_vet` | boolean | `true` | Run `go vet ./...` in `lint` |
| `golangci_lint_version` | string | `""` | golangci-lint version; empty skips it |
| `golangci_lint_only_new_issues` | boolean | `false` | Report only issues introduced by the change |
| `test_args` | string | `-count=1 ./...` | Arguments to `go test` |
| `cover_min` | number | `0` | Minimum total coverage percent; 0 disables the gate |
| `cover_exclude` | string | `""` | Extended regex of coverage profile lines left out of the gate |
| `strip` | boolean | `true` | Build with `-trimpath -ldflags="-s -w"` |

### terraform.yml

Permissions: `contents: read`, `id-token: write`. Output: `applied` (`'true'`
when the apply step ran). Each job installs pinned tofu, terragrunt and tflint
(sha256-checked), exports `env` (after the Vault login), and runs `terragrunt
run --all` per `environments_dir` line.

`apply` runs only when `github.event_name == apply_event` and the ref is
`apply_ref` (default branch when empty); elsewhere it reports skipped. It
needs `validate` and applies without a prior plan. On
`terraform-apply-<repo>`, running applies are never cancelled; at most one is
queued, and a newer queued run replaces an older one. On a run that applies, `fmt`,
`tflint`, `validate` and `plan` are skipped unless `checks_on_apply` is set.

| Input | Type | Default | Description |
|---|---|---|---|
| `runs_on` | string | `iuclabs-infra` | Runner label |
| `working_directory` | string | `.` | Directory holding the terraform/terragrunt tree |
| `environments_dir` | string | `.` | Newline-separated terragrunt dirs, relative to `working_directory` |
| `plan_parallelism` / `apply_parallelism` | number | `4` / `2` | terragrunt `--parallelism` for plan / apply |
| `init_args` | string | `-upgrade` | Arguments after `--` for `init` |
| `plan_args` / `apply_args` | string | `-lock=false` / `""` | Arguments after `--` for `plan` / `apply` |
| `apply_event` | string | `push` | `github.event_name` that applies |
| `apply_ref` | string | default branch | The ref allowed to apply |
| `apply_concurrency_group` | string | `terraform-apply-<repo>` | Concurrency group that serialises applies |
| `checks_on_apply` | boolean | `false` | Also run `fmt`, `tflint`, `validate`, `plan` on a run that applies |
| `download_base` | string | `https://github.com` | Base for `<owner>/<repo>/releases/download/<tag>/<file>` tool downloads; `https://`, no trailing slash |
| `tofu_version`, `terragrunt_version`, `tflint_version` (+ `_sha256`) | string | `1.10.3`, `1.0.3`, `0.62.0` | Tool versions and linux_amd64 checksums |
| `ca_file` | string | `/etc/vault-ca/ca.crt` | Appended to the system bundle, used for the tool downloads, exported as `CA_BUNDLE`, `VAULT_CACERT`, `SSL_CERT_FILE`; empty skips |
| `vault_addr` | string | `https://vault.service.consul:8200` | Exported as `VAULT_ADDR`; also the JWT audience |
| `vault_path` | string | `jwt/github/iuclabs` | JWT auth mount |
| `plan_vault_role` / `apply_vault_role` | string | `""` | Plan / apply Vault role; empty skips the login |
| `plan_vault_secrets` / `apply_vault_secrets` | string | `""` | Plan / apply `hashicorp/vault-action` `secrets` mapping |
| `plan_reachability_urls` | string | `""` | Plan only: newline-separated literal URLs (not expanded, so write the full URL, e.g. `https://vault.service.consul:8200/v1/sys/health`) that must answer with any HTTP status before planning |
| `env` | string | `""` | Newline-separated `KEY=value` exported before terragrunt; keys match `^[A-Z_][A-Z0-9_]*$` or `TF_VAR_<name>`; each line needs `=`, else the run fails |

Secret `tf_env` takes the same `KEY=value` lines, masked, and reaches only
`plan` and `apply`; with `secrets: inherit` a repo or org secret named `TF_ENV`
fills it. Prefer Vault.

### release-vault-plugin.yml

Permissions: `contents: write`, `id-token: write`, `packages: write`. Outputs:
`artifacts` (goreleaser `dist/artifacts.json`), `image` (`<ref>:<tag>@<digest>`)
and `image_configs` (JSON array of each platform image's config).
goreleaser runs with `--skip=docker --draft`; crane then pushes
`ghcr.io/iuclabs/<project_name>:<tag>` and `latest` without a Docker daemon
(`FROM scratch`, each linux binary at `/<binary>` as the entrypoint, OCI
source/version/revision labels, one index entry per arch) and only then
publishes the draft. A caller's `dockers_v2` block and Dockerfile are unused.
The image ships without an SBOM attestation.

| Input | Type | Default | Description |
|---|---|---|---|
| `vault_role` | string | `release` | Role on the `jwt/github/iuclabs` mount |
| `snapshot` | boolean | `false` | `goreleaser release --snapshot --clean --skip=publish,sign,docker`; no Vault, image pushed to a job-local registry |
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
  ci:
    uses: IUCLabs/workflows/.github/workflows/pre-commit.yml@v0.1.0
    permissions:
      contents: read
    with:
      go_version_file: go.mod
  actions:
    uses: IUCLabs/workflows/.github/workflows/actions.yml@v0.1.0
    permissions:
      contents: read
  go:
    uses: IUCLabs/workflows/.github/workflows/go-ci.yml@v0.1.0
    permissions:
      contents: read
      pull-requests: read
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

Switching a repo to these workflows renames its checks, so update its required
status checks to the names above.

```yaml
# terraform repo: .github/workflows/terraform.yml
name: terraform

on:
  pull_request:
  push:
    branches: [main]

permissions: {}

concurrency:
  group: terraform-${{ github.ref }}
  cancel-in-progress: ${{ github.event_name == 'pull_request' }}

jobs:
  terraform:
    uses: IUCLabs/workflows/.github/workflows/terraform.yml@v0.1.0
    permissions:
      contents: read
      id-token: write
    with:
      environments_dir: environments
      # artifactapi's GitHub proxy; reachable from internal runners only, not ubuntu-latest
      download_base: https://artifactapi.k8s.syd1.au.unkin.net/api/v1/remote/github
      plan_reachability_urls: https://vault.service.consul:8200/v1/sys/health
      plan_vault_role: terraform_example_plan
      plan_vault_secrets: |
        consul_root/au/syd1/creds/terraform-example-plan token | CONSUL_HTTP_TOKEN
      apply_vault_role: terraform_example
      apply_vault_secrets: |
        consul_root/au/syd1/creds/terraform-example token | CONSUL_HTTP_TOKEN
      # init_args: ""  # terraform-authentik inits without -upgrade
```

## Development

`make pre-commit`, `make lint` (actionlint and zizmor) and `make test` (runs
`act` when installed). `self-test.yml` runs every workflow against `testdata/`
with snapshot and no-push modes, and checks actionlint rejects
`testdata/actions/bad/`. Releases: `make patch|minor|major`.

This repo's own required checks are `pre-commit` and `lint`.

actionlint and zizmor versions are pinned in `actions.yml`, `self-test.yml`,
`Makefile` and `.pre-commit-config.yaml`; bump them together.
