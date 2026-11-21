# workflows

Shared reusable GitHub Actions workflows for IUCLabs repos. Every third-party
action is pinned by commit SHA; Dependabot bumps them weekly.

| Workflow | What it does |
|---|---|
| `pre-commit.yml` | `uvx pre-commit run --all-files`, or [prek](https://github.com/j178/prek) |
| `actions.yml` | `lint` (actionlint, with shellcheck on `run:` blocks) and `audit` (zizmor) |
| `go-ci.yml` | `lint` (go vet, optional golangci-lint), `test` (optional coverage gate) and `build` jobs |
| `terraform.yml` | `fmt`, `tflint`, `validate` and a lock-free terragrunt `plan` |
| `terraform-apply.yml` | terragrunt `apply` on the default branch, queued per repo |
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
| `terraform` | `terraform.yml` | `terraform / fmt`, `terraform / tflint`, `terraform / validate`, `terraform / plan` |
| `apply` | `terraform-apply.yml` | `apply / apply` |

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

### terraform.yml and terraform-apply.yml

Permissions: `contents: read`, `id-token: write`. Each job installs pinned
tofu, terragrunt and tflint (sha256-checked), exports `env`, and runs
`terragrunt run --all` per `environments_dir` line. `apply` runs only on
`apply_ref` and queues on `terraform-apply-<repo>` without cancelling.

| Input | Type | Default | Description |
|---|---|---|---|
| `runs_on` | string | `iuclabs-infra` | Runner label |
| `working_directory` | string | `.` | Directory holding the terraform/terragrunt tree |
| `environments_dir` | string | `.` | Newline-separated terragrunt dirs, relative to `working_directory` |
| `parallelism` | number | `4` plan, `2` apply | terragrunt `--parallelism` |
| `init_args` | string | `-upgrade` | Arguments after `--` for `init` |
| `plan_args` / `apply_args` | string | `-lock=false` / `""` | Arguments after `--` for `plan` / `apply` |
| `apply_ref` | string | default branch | Apply only: the ref allowed to apply |
| `tofu_version`, `terragrunt_version`, `tflint_version` (+ `_sha256`) | string | `1.10.3`, `1.0.3`, `0.62.0` | Tool versions and linux_amd64 checksums |
| `ca_file` | string | `/etc/vault-ca/ca.crt` | Appended to the system bundle, exported as `CA_BUNDLE`, `VAULT_CACERT`, `SSL_CERT_FILE`; empty skips |
| `vault_addr` | string | `https://vault.service.consul:8200` | Exported as `VAULT_ADDR`; also the JWT audience |
| `vault_path` | string | `jwt/github/iuclabs` | JWT auth mount |
| `vault_role` | string | `""` | Plan/apply Vault role; empty skips the login |
| `vault_secrets` | string | `""` | `hashicorp/vault-action` `secrets` mapping |
| `reachability_urls` | string | `""` | Plan only: newline-separated URLs that must answer before planning |
| `env` | string | `""` | Newline-separated `KEY=value` exported before terragrunt; keys match `^[A-Z_][A-Z0-9_]*$` or `TF_VAR_<name>`, and `GITHUB_*`, `RUNNER_*`, `ACTIONS_*`, `PATH`, `LD_*`, `BASH_ENV`, `NODE_OPTIONS` are rejected |

Secret `tf_env` takes the same `KEY=value` lines, masked; with `secrets:
inherit` a repo or org secret named `TF_ENV` fills it. Prefer Vault.

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
# terraform repo: .github/workflows/plan.yml (pull_request) and apply.yml (push to main)
jobs:
  terraform:
    uses: IUCLabs/workflows/.github/workflows/terraform.yml@v0.1.0
    permissions:
      contents: read
      id-token: write
    with:
      environments_dir: environments
      vault_role: terraform_example_plan
      vault_secrets: |
        consul_root/au/syd1/creds/terraform-example-plan token | CONSUL_HTTP_TOKEN
  apply:
    uses: IUCLabs/workflows/.github/workflows/terraform-apply.yml@v0.1.0
    permissions:
      contents: read
      id-token: write
    with:
      environments_dir: environments
      vault_role: terraform_example
      vault_secrets: |
        consul_root/au/syd1/creds/terraform-example token | CONSUL_HTTP_TOKEN
```

## Development

`make pre-commit`, `make lint` (actionlint and zizmor) and `make test` (runs
`act` when installed). `self-test.yml` runs every workflow against `testdata/`
with snapshot and no-push modes, and checks actionlint rejects
`testdata/actions/bad/`. Its own required checks are `pre-commit` and `lint`. Releases: `make patch|minor|major`.
