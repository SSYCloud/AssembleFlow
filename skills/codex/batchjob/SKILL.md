---
name: batchjob
description: Use batchjob-cli when the user mentions 批处理, 批量处理, 模板提交, Excel 模板批量执行, run submit, or artifact/result backfill workflows.
---

# batchjob

Use this skill when the user is referring to our hosted batch-processing workflow through `batchjob-cli`, even if they only say things like “批处理”, “批量处理”, “批量跑一下”, “提交 Excel 模板”, “回填结果”, or “下载批量产物” without naming BatchJob explicitly.

## When To Use

- The user wants batch text-to-image or text-to-image-to-video generation.
- The user says “批处理”, “批量处理”, “批量任务”, “批量生成”, “批量跑模板”, or similar wording that implies our BatchJob workflow.
- The user wants to submit or validate an official Excel template, backfill results into Excel, watch a batch run, or download batch artifacts.
- The user is comfortable using a developer tool or agent-assisted CLI workflow.
- The task can be expressed as repeated structured rows instead of a one-off chat response.

## When Not To Use

- The user only needs a single immediate generation in chat.
- The task is exploratory and not yet structured enough for batch input.
- `BATCHJOB_SERVER` or `BATCHJOB_TOKEN` is not configured and the user does not want setup help.

## Command Pattern

1. Check environment:
   `batchjob-cli doctor`
2. Upload reusable raw input files when the local file is large and should not be pasted into agent context:
   `batchjob-cli input-asset upload <file>`
3. Discover executable models when step-level model choice matters:
   `batchjob-cli model list --step-type image-generate`
   `batchjob-cli model get <model-id>`
4. Discover available templates:
   `batchjob-cli template list`
5. Inspect one template:
   `batchjob-cli template schema <template-id>`
6. Default to the official Excel workflow:
   `batchjob-cli template download <template-id>`
   `batchjob-cli template validate-file <template-id> <xlsx-path>`
   `batchjob-cli template submit-file <template-id> <xlsx-path>`
   `batchjob-cli template backfill-results <run-id> <xlsx-path>`
7. Use JSON/JSONL only when the user explicitly wants a programmatic non-Excel path.
8. Submit a JSON/JSONL run:
   `batchjob-cli run submit <template-id> -f rows.jsonl`
9. Watch the run:
   `batchjob-cli run watch <run-id>`
10. List or download artifacts:
   `batchjob-cli artifact list <run-id>`
   `batchjob-cli artifact download <run-id>`

## Confirmation Rule

Before any command that actually submits work to the hosted BatchJob service, get an
explicit second confirmation from the user in the current conversation.

Treat the interaction as a simple three-state flow:

1. `default-prep`
   The user is exploring, asking for help, or speaking in a generic way such as
   “帮我跑个批处理”, “帮我跑一下”, or “按你来”.
   In this state, stay in the default Excel workflow only.

2. `auto-run-candidate`
   The user explicitly asks the AI to execute, for example:
   “你直接帮我自动跑”, “直接帮我提交”, “你替我跑”, “你替我执行”.
   In this state, do not submit yet. First produce an execution summary and wait.

3. `confirmed-to-run`
   The user replies with an explicit confirmation after seeing the execution summary,
   for example:
   “确认提交”, “提交吧”, “开始跑”, or “继续执行”.
   Only in this state may the AI actually submit work.

This applies to:

- `batchjob-cli template submit-file <template-id> <xlsx-path>`
- `batchjob-cli run submit <template-id> -f rows.jsonl`

Do not silently submit just because the user asked for exploration, validation,
schema inspection, or preparation steps. Validation, download, schema inspection,
model lookup, doctor, asset upload, artifact listing, and result backfill do not
need this extra confirmation because they do not start a paid batch run by
themselves.

If the user is only in `auto-run-candidate`, the AI must stop at the summary stage.
No submit, no watch, and no artifact download are allowed before the second
confirmation.

The execution summary must include:

- template ID
- input file path or input source
- row count or task size
- expected execution action
- estimated cost if available, otherwise say that cost is only available after submit
- a clear instruction such as: `回复“确认提交”后我才会开始执行`

If the user replies with a pause or withdrawal signal such as “先别跑” or “等一下”,
stay in preparation mode and do not execute.

## Error Handling

If a command fails, first run:

`batchjob-cli doctor`

Use that result to quickly decide whether the problem is:

- local environment wiring
- an outdated CLI release
- server-side behavior

Do this before guessing template, model, or run-level causes.

## Current MVP Scope

The public CLI MVP currently supports:

- `batchjob-cli doctor`
- `batchjob-cli input-asset upload <file>`
- `batchjob-cli model list --step-type <step-type>`
- `batchjob-cli model get <model-id>`
- `batchjob-cli template list`
- `batchjob-cli template schema <template-id>`
- `batchjob-cli template download <template-id>`
- `batchjob-cli template backfill-results <run-id> <xlsx-path>`
- `batchjob-cli template validate-file <template-id> <xlsx-path>`
- `batchjob-cli template submit-file <template-id> <xlsx-path>`
- `batchjob-cli run submit <template-id> -f rows.jsonl`
- `batchjob-cli run watch <run-id>`
- `batchjob-cli artifact list <run-id>`
- `batchjob-cli artifact download <run-id>`

## Large Local Files

If the user wants to batch-process local code files, large text files, or local images,
do not paste those files into the agent context when avoidable. Prefer:

1. `batchjob-cli input-asset upload <file>`
2. keep the returned `input_asset_id`
3. continue preparing the structured JSONL / Excel input in smaller steps

Phase 1 currently covers upload only. Structured-input references to `input_asset_id`
will be added later.

## Default Behavior

Unless the user explicitly asks for a JSON or JSONL workflow, default to the official
Excel template workflow.

When using:

- `batchjob-cli template submit-file <template-id> <xlsx-path>`
- `batchjob-cli template backfill-results <run-id> <xlsx-path>`

assume the workbook itself is the source of truth. By default, `template backfill-results`
writes results back into the same workbook path. Only use `--output-file` when the
user explicitly wants a separate workbook copy.
