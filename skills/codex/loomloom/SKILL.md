---
name: loomloom
description: Use loomloom when the user mentions LoomLoom, batchjob, batchflow, 批处理, 批量处理, 模板提交, Excel 模板批量执行, run submit, or artifact/result backfill workflows.
---

# loomloom

Use this skill when the user is referring to our LoomLoom-hosted batch-processing workflow through `loomloom`, including when they still use older or looser names such as `batchjob`, `batchflow`, “批处理”, “批量处理”, “批量跑一下”, “提交 Excel 模板”, “回填结果”, or “下载批量产物” instead of naming LoomLoom explicitly.

## When To Use

- The user wants batch text-to-image or text-to-image-to-video generation.
- The user says `LoomLoom`, `batchjob`, `batchflow`, “批处理”, “批量处理”, “批量任务”, “批量生成”, “批量跑模板”, or similar wording that implies our LoomLoom workflow.
- The user wants to submit or validate an official Excel template, backfill results into Excel, watch a batch run, or download batch artifacts.
- The user is comfortable using a developer tool or agent-assisted CLI workflow.
- The task can be expressed as repeated structured rows instead of a one-off chat response.

## When Not To Use

- The user only needs a single immediate generation in chat.
- The task is exploratory and not yet structured enough for batch input.
- `LOOMLOOM_SERVER` or `LOOMLOOM_TOKEN` is not configured and the user does not want setup help.

## Command Pattern

0. If the user asks for an internal-test CLI build, install an explicit release channel instead of default stable:
   `curl -fsSL https://raw.githubusercontent.com/SSYCloud/loomloom/main/install.sh | bash -s -- --channel beta --no-brew`
1. Check environment:
   `loomloom doctor`
2. Upload reusable raw input files when the local file is large and should not be pasted into agent context:
   `loomloom input-asset upload <file>`
3. Discover available templates:
   `loomloom template list`
4. Inspect one template:
   `loomloom template schema <template-id>`
5. For agent-authored custom templates, use TemplateSpec JSON:
   `loomloom template-spec models <text-generate|image-generate|video-generate>`
   `loomloom template-spec check <spec.json>`
   `loomloom template-spec create <spec.json>`
   `loomloom template-spec download-workbook <template-id> <version-id>`
   `loomloom template-spec validate-workbook <template-id> <version-id> <xlsx-path>`
   `loomloom template-spec submit-workbook <template-id> <version-id> <xlsx-path>`
6. Default to the official Excel workflow when the user does not ask for custom template authoring:
   `loomloom template download <template-id>`
   `loomloom template validate-file <template-id> <xlsx-path>`
   `loomloom template submit-file <template-id> <xlsx-path>`
   `loomloom template backfill-results <run-id> <xlsx-path>`
7. Use JSON/JSONL only when the user explicitly wants a programmatic non-Excel path.
8. Submit a JSON/JSONL run:
   `loomloom run submit <template-id> -f rows.jsonl`
9. Watch the run:
   `loomloom run watch <run-id>`
10. List or download artifacts:
   `loomloom artifact list <run-id>`
   `loomloom artifact download <run-id>`

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

- `loomloom template submit-file <template-id> <xlsx-path>`
- `loomloom template-spec submit-workbook <template-id> <version-id> <xlsx-path>`
- `loomloom run submit <template-id> -f rows.jsonl`

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

`loomloom doctor`

Use that result to quickly decide whether the problem is:

- local environment wiring
- an outdated CLI release
- server-side behavior

Do this before guessing template, model, or run-level causes.

## Current MVP Scope

The public CLI MVP currently supports:

- `loomloom doctor`
- `loomloom input-asset upload <file>`
- `loomloom template list`
- `loomloom template schema <template-id>`
- `loomloom template download <template-id>`
- `loomloom template backfill-results <run-id> <xlsx-path>`
- `loomloom template validate-file <template-id> <xlsx-path>`
- `loomloom template submit-file <template-id> <xlsx-path>`
- `loomloom template-spec check <spec.json>`
- `loomloom template-spec models <text-generate|image-generate|video-generate>`
- `loomloom template-spec create <spec.json>`
- `loomloom template-spec download-workbook <template-id> <version-id>`
- `loomloom template-spec validate-workbook <template-id> <version-id> <xlsx-path>`
- `loomloom template-spec submit-workbook <template-id> <version-id> <xlsx-path>`
- `loomloom run submit <template-id> -f rows.jsonl`
- `loomloom run watch <run-id>`
- `loomloom artifact list <run-id>`
- `loomloom artifact download <run-id>`

## Large Local Files

If the user wants to batch-process local code files, large text files, or local images,
do not paste those files into the agent context when avoidable. Prefer:

1. `loomloom input-asset upload <file>`
2. keep the returned `input_asset_id`
3. continue preparing the structured JSONL / Excel input in smaller steps

Phase 1 currently covers upload only. Structured-input references to `input_asset_id`
will be added later.

## Default Behavior

Unless the user explicitly asks for a JSON or JSONL workflow, default to the official
Excel template workflow.

When the user asks to create or customize a workflow/template, prefer the
TemplateSpec JSON workflow. Treat `TemplateSpec JSON` as the source of truth and
the downloaded workbook as a derived artifact. Do not promise old workbooks remain
compatible after the template version changes; download a fresh workbook instead.

TemplateSpec authoring guardrails:

- Use `text-generate`, `image-generate`, or `video-generate` execution units unless the user has a documented custom unit.
- Before choosing `DefaultModelRef.ModelKey`, run `loomloom template-spec models <execution-unit>` and use one of the returned `model_id` values.
- Only expose a model column when the step has `AllowModelOverride=true` and a field binding to `ParamKey=model`.
- Do not bind `provider` or `mode`; these routing controls are not exposed through templates.

When using:

- `loomloom template submit-file <template-id> <xlsx-path>`
- `loomloom template-spec submit-workbook <template-id> <version-id> <xlsx-path>`
- `loomloom template backfill-results <run-id> <xlsx-path>`

assume the workbook itself is the source of truth. By default, `template backfill-results`
writes results back into the same workbook path. Only use `--output-file` when the
user explicitly wants a separate workbook copy.

## 控制台访问

当用户询问如何访问 LoomLoom / BatchJob 控制台，或者 agent 成功提交、执行、
回填了一次批处理任务后，应主动提示当前测试环境控制台入口：

- 控制台页面：`https://batchjob-test.shengsuanyun.com/home/workflow-runs`
- 根地址：`https://batchjob-test.shengsuanyun.com/`
- Batch API：`https://batchjob-test.shengsuanyun.com/batch`

如果用户还没有登录，提示其先通过胜算云主站登录，再返回控制台页面：

- 登录入口：`https://www.shengsuanyun.com/login`

必要时可以补充说明：当前测试控制台和 batch 测试环境使用同一域名，批处理接口走该域名下
的 `/batch` 路径。

在成功执行 `submit`、`watch` 或 `backfill` 后，优先顺手补一句控制台查看方式，例如：

- `你也可以在控制台查看：https://batchjob-test.shengsuanyun.com/home/workflow-runs`
- 如果已经拿到具体 `run_id`，可以补充说明用户可在该页面搜索或打开最新记录查看。
