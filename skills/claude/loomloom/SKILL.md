---
name: loomloom
description: Use loomloom when the user mentions LoomLoom, batchjob, batchflow, 批处理, 批量处理, 模板提交, Excel 模板批量执行, run submit, or artifact/result backfill workflows.
---

# loomloom

Use this skill when the user is referring to our LoomLoom-hosted batch-processing workflow through `loomloom`, including when they still use older or looser names such as `batchjob`, `batchflow`, “批处理”, “批量处理”, “批量跑一下”, “提交 Excel 模板”, “回填结果”, or “下载批量产物” instead of naming LoomLoom explicitly.

## When To Use

- The user needs batch-oriented generation workflows.
- The user says `LoomLoom`, `batchjob`, `batchflow`, “批处理”, “批量处理”, “批量任务”, “批量生成”, “批量跑模板”, or similar wording that implies our LoomLoom workflow.
- The user wants to submit or validate an official Excel template, backfill results into Excel, watch a batch run, or download batch artifacts.
- The user is a developer or uses an agent that can call a CLI.
- The task is better modeled as a template-driven batch run than as an in-chat one-off answer.

## First Steps

0. If the user asks for an internal-test CLI build, install an explicit release channel instead of default stable:
   `curl -fsSL https://raw.githubusercontent.com/SSYCloud/loomloom/main/install.sh | bash -s -- --channel beta --no-brew`
1. Ensure:
   `LOOMLOOM_SERVER`
   `LOOMLOOM_TOKEN`
2. Run:
   `loomloom doctor`
3. Upload reusable raw input files when the local file is large and should not be pasted into agent context:
   `loomloom input-asset upload <file>`
4. Discover templates:
   `loomloom template list`
5. Inspect schema:
   `loomloom template schema <template-id>`
6. For agent-authored custom templates, use TemplateSpec JSON:
   `loomloom template-spec models <text-generate|image-generate|video-generate>`
   `loomloom template-spec check <spec.json>`
   `loomloom template-spec create <spec.json>`
   `loomloom template-spec download-workbook <template-id> <version-id>`
   `loomloom template-spec validate-workbook <template-id> <version-id> <xlsx-path>`
   `loomloom template-spec submit-workbook <template-id> <version-id> <xlsx-path>`
7. Default to the official Excel workflow when the user does not ask for custom template authoring:
   `loomloom template download <template-id>`
   `loomloom template validate-file <template-id> <xlsx-path>`
   `loomloom template submit-file <template-id> <xlsx-path>`
   `loomloom run result-workbook <run-id>`
   Use `loomloom template backfill-results <run-id> <xlsx-path>` only for the legacy local backfill flow.
8. Use JSON/JSONL only when the user explicitly wants a programmatic non-Excel path:
   `loomloom run submit <template-id> -f rows.jsonl`
9. Watch the run:
   `loomloom run watch <run-id>`
10. Inspect or download aligned run results:
   `loomloom run result-rows <run-id>`
   `loomloom run result-workbook <run-id>`
11. Download outputs when ready:
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

Use that result to quickly separate:

- local setup issues
- outdated CLI version issues
- server-side failures

Do this before concluding that the template, model, or run itself is wrong.

## Current MVP Scope

The first public CLI release covers:

- environment verification
- raw input asset upload
- executable model discovery
- template discovery
- official template Excel download / validation / submission / server result workbook
- server-side result rows and result workbook downloads
- TemplateSpec JSON custom template creation / workbook download / workbook validation / workbook submission
- executable model discovery for TemplateSpec authoring
- official template row submission
- run watching
- artifact listing and download

## Large Local Files

When the user wants to process local code files, large text files, or local images,
prefer uploading the raw file first instead of pasting the whole content into agent
context:

1. `loomloom input-asset upload <file>`
2. keep the returned `input_asset_id`
3. continue assembling the final structured input in smaller steps

Current scope is upload only. Reusing `input_asset_id` inside structured inputs will
land in a later phase.

## Default Behavior

Unless the user explicitly asks for a JSON or JSONL workflow, default to the official
Excel template workflow.

When the user asks to create or customize a workflow/template, prefer the
TemplateSpec JSON workflow. Treat `TemplateSpec JSON` as the source of truth and
the downloaded workbook as a derived artifact. Do not promise old workbooks remain
compatible after the template version changes; download a fresh workbook instead.

TemplateSpec authoring guardrails:

- Use `text-generate`, `image-generate`, or `video-generate` execution units unless the user has a documented custom unit.
- Use lowerCamel TemplateSpec keys from OpenAPI, such as `meta.name`, `steps[].stepId`, and `defaultModelRef.modelKey`.
- Before choosing `defaultModelRef.modelKey`, run `loomloom template-spec models <execution-unit>` and use one of the returned `model_id` values.
- Only expose a model column when the step has `allowModelOverride=true` and a field binding to `paramKey=model`.
- Do not bind `provider` or `mode`; these routing controls are not exposed through templates.

When using:

- `loomloom template submit-file <template-id> <xlsx-path>`
- `loomloom template-spec submit-workbook <template-id> <version-id> <xlsx-path>`
- `loomloom run result-workbook <run-id>`

assume the submitted workbook and the server-side run input snapshot are the source of truth.
Prefer `run result-workbook` after a run completes because the server joins the original
input snapshot with the produced artifacts. Use `template backfill-results` only when the
user explicitly needs the older local Excel backfill behavior.

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
