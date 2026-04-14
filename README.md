# AssembleFlow

Public distribution repository for:

- `assemble-flow`
- agent skill packs for Codex / Claude
- install scripts
- examples
- release workflows

This repository is the public delivery surface for developers and agents to use hosted BatchJob skills.

## Current MVP

The first public CLI is HTTP-based and focuses on:

- `assemble-flow doctor`
- `assemble-flow input-asset upload <file>`
- `assemble-flow model list --step-type image-generate`
- `assemble-flow model get <model-id>`
- `assemble-flow template list`
- `assemble-flow template schema <template-id>`
- `assemble-flow template download <template-id>`
- `assemble-flow template submit-file <template-id> <xlsx-path>`
- `assemble-flow template validate-file <template-id> <xlsx-path>`
- `assemble-flow template backfill-results <run-id> <xlsx-path>`
- `assemble-flow run submit <template-id> -f rows.jsonl`
- `assemble-flow run watch <run-id>`
- `assemble-flow artifact list <run-id>`
- `assemble-flow artifact download <run-id>`

Authentication is environment-variable based:

```bash
export BATCHJOB_SERVER="https://batchjob-test.shengsuanyun.com/batch"
export BATCHJOB_TOKEN="your-token"
```

## Install From GitHub Release

```bash
curl -fsSL https://raw.githubusercontent.com/SSYCloud/AssembleFlow/main/install.sh | bash
```

By default the installer downloads the latest release, installs `assemble-flow` into `~/.local/bin`, and installs the Codex skill into `~/.codex/skills/assemble-flow/SKILL.md`.

On macOS/Linux, if `brew` is available, the installer prefers Homebrew for the CLI and still installs the matching skill pack. Use `--no-brew` if you want the release binary path instead.

Useful flags:

```bash
curl -fsSL https://raw.githubusercontent.com/SSYCloud/AssembleFlow/main/install.sh | bash -s -- --agent claude
curl -fsSL https://raw.githubusercontent.com/SSYCloud/AssembleFlow/main/install.sh | bash -s -- --no-brew
curl -fsSL https://raw.githubusercontent.com/SSYCloud/AssembleFlow/main/install.sh | bash -s -- --version v0.1.0
```

### Windows (PowerShell)

```powershell
irm https://raw.githubusercontent.com/SSYCloud/AssembleFlow/main/install.ps1 | iex
```

Useful flags:

```powershell
& ([scriptblock]::Create((irm https://raw.githubusercontent.com/SSYCloud/AssembleFlow/main/install.ps1))) -Agent claude
& ([scriptblock]::Create((irm https://raw.githubusercontent.com/SSYCloud/AssembleFlow/main/install.ps1))) -Version v0.1.0
```

By default the PowerShell installer places `assemble-flow.exe` under:

```powershell
$HOME\AppData\Local\Programs\assemble-flow
```

If that directory is not already in `PATH`, add it before using `assemble-flow`.

## Uninstall

macOS / Linux:

```bash
curl -fsSL https://raw.githubusercontent.com/SSYCloud/AssembleFlow/main/uninstall.sh | bash
```

Windows PowerShell:

```powershell
irm https://raw.githubusercontent.com/SSYCloud/AssembleFlow/main/uninstall.ps1 | iex
```

The uninstall scripts remove the GitHub Release install and, when detected, also uninstall the Homebrew `assemble-flow` formula so old CLI versions do not stay on your `PATH`.

Useful flags:

```bash
curl -fsSL https://raw.githubusercontent.com/SSYCloud/AssembleFlow/main/uninstall.sh | bash -s -- --agent claude
curl -fsSL https://raw.githubusercontent.com/SSYCloud/AssembleFlow/main/uninstall.sh | bash -s -- --cli-only
curl -fsSL https://raw.githubusercontent.com/SSYCloud/AssembleFlow/main/uninstall.sh | bash -s -- --skill-only
```

```powershell
& ([scriptblock]::Create((irm https://raw.githubusercontent.com/SSYCloud/AssembleFlow/main/uninstall.ps1))) -Agent claude
& ([scriptblock]::Create((irm https://raw.githubusercontent.com/SSYCloud/AssembleFlow/main/uninstall.ps1))) -CliOnly
& ([scriptblock]::Create((irm https://raw.githubusercontent.com/SSYCloud/AssembleFlow/main/uninstall.ps1))) -SkillOnly
```

## Install With Homebrew

```bash
brew install ssycloud/tap/assemble-flow
```

Or:

```bash
brew tap ssycloud/tap
brew install assemble-flow
```

Direct Homebrew commands install the CLI only. If you also want the Codex or Claude skill pack, use the installer above so the skill is installed too.

## Local Build

```bash
cd cli
GOWORK=off go build ./cmd/assemble-flow
```

## Quick Start

```bash
export BATCHJOB_SERVER="https://batchjob-test.shengsuanyun.com/batch"
export BATCHJOB_TOKEN="your-token"

./cli/assemble-flow doctor
./cli/assemble-flow input-asset upload ./local-input.txt
./cli/assemble-flow model list --step-type image-generate
./cli/assemble-flow model get google/gemini-2.5-flash-image
./cli/assemble-flow template list
./cli/assemble-flow template schema text-image-v1
./cli/assemble-flow template download text-image-v1 --output-file ./text-image-v1.xlsx
./cli/assemble-flow template validate-file text-image-v1 ./filled-text-image-v1.xlsx
./cli/assemble-flow template submit-file text-image-v1 ./filled-text-image-v1.xlsx
./cli/assemble-flow run watch <run-id>
./cli/assemble-flow template backfill-results <run-id> ./filled-text-image-v1.xlsx
./cli/assemble-flow artifact list <run-id>
./cli/assemble-flow artifact download <run-id> --output-dir ./downloads
```

If `template list` returns `no templates`, the target environment likely has not imported official template seed data yet.

## Model Discovery

Use model discovery when you need to understand which executable models are currently
available for one step type:

```bash
./cli/assemble-flow model list --step-type text-generate
./cli/assemble-flow model list --step-type image-generate --provider vertex
./cli/assemble-flow model get google/gemini-2.5-flash-image
```

`model list` is step-type scoped on purpose. Common values are:

- `text-generate`
- `image-generate`
- `video-generate`

## Input Asset Upload

When the agent should not inline a large local file into its own context, upload the
raw file first and keep the returned `input_asset_id` for later structured-input
assembly:

```bash
./cli/assemble-flow input-asset upload ./runtime/codex-exec.mjs
./cli/assemble-flow input-asset upload ./diagram.png --content-type image/png
```

This command currently covers Phase 1 only:

- upload one local file
- get back `input_asset_id`
- reuse the asset ID later when structured-input references are supported

It does not yet submit a run by itself.

## Excel Template Workflow

For official templates, the default workflow is Excel:

```bash
./cli/assemble-flow template download text-image-v1 --output-file ./text-image-v1.xlsx
./cli/assemble-flow template validate-file text-image-v1 ./filled-text-image-v1.xlsx
./cli/assemble-flow template submit-file text-image-v1 ./filled-text-image-v1.xlsx
./cli/assemble-flow run watch <run-id>
./cli/assemble-flow template backfill-results <run-id> ./filled-text-image-v1.xlsx
```

`template submit-file` uploads the filled workbook and directly creates a run.
`template backfill-results` keeps `__batchjob_meta` intact, fetches run artifacts, and by default writes result columns back into the same workbook file. Use `--output-file` only when you explicitly want a separate workbook copy.

## Input File Format

`run submit` is a non-default path for advanced or programmatic usage. It accepts:

- JSONL with one flat object per line
- JSON array of flat objects

The field names must match the template schema. Starter files live under `examples/`.
