# LoomLoom README

# LoomLoom

> **批量内容生成平台** —— 用自然语言驱动 AI，批量完成文案、图片、视频生成任务。
> 由胜算云出品 · [github.com/SSYCloud/loomloom](http://github.com/SSYCloud/loomloom)

[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/SSYCloud/loomloom/1-overview)

---

## ✨ 它能做什么？

LoomLoom 是一个面向 AI 驱动批量内容生产的平台工具。你不需要写代码，只需用自然语言告诉 AI 你想做什么，它会自动完成从模板下载、数据填写、任务提交到结果下载的全流程。

**典型场景：**

- 📄 **批量文案写作** —— 100 条产品描述、改写、摘要、问答，文件批量修改，一次搞定
- 🖼️ **批量图片生成** —— 电商配图、社媒插画、概念图，按行批出
- 🎬 **批量视频制作** —— 短剧分镜、广告素材，文字到视频全自动

---

## 🤖 支持的 AI 平台

安装 LoomLoom 后，以下平台将获得操作批处理任务的「技能包」：

| 平台 | 支持状态 |
| --- | --- |
| **Codex**（OpenAI） | ✅ 支持 |
| **Claude Code**（Anthropic） | ✅ 支持 |
| **OpenClaw**（胜算云） | ✅ 支持 |

---

## ⚡ 快速开始

### 方式一：超简模式（推荐）

把以下这段话发给 Codex / Claude / OpenClaw，将 `your-token` 替换为你的真实 Token，Agent 会自动完成安装与配置：

```
请你去这个 GitHub 项目安装 LoomLoom：https://github.com/SSYCloud/loomloom
我的服务器地址是 https://batchjob-test.shengsuanyun.com/batch，Token 是 your-token
安装好之后帮我跑一下 doctor 检查是否正常
```

### 方式二：自己安装

**macOS / Linux：**

```bash
# 默认安装（Codex 技能包）
curl -fsSL https://raw.githubusercontent.com/SSYCloud/loomloom/main/install.sh | bash

# Claude 用户
curl -fsSL https://raw.githubusercontent.com/SSYCloud/loomloom/main/install.sh | bash -s -- --agent claude

# OpenClaw 用户
curl -fsSL https://raw.githubusercontent.com/SSYCloud/loomloom/main/install.sh | bash -s -- --agent openclaw

# 指定版本
curl -fsSL https://raw.githubusercontent.com/SSYCloud/loomloom/main/install.sh | bash -s -- --version v0.1.0
```

> 如果系统有 Homebrew，安装脚本会优先使用 Homebrew 安装 CLI；可加 `--no-brew` 改用二进制包。

**Windows（PowerShell）：**

```powershell
# 默认安装
irm https://raw.githubusercontent.com/SSYCloud/loomloom/main/install.ps1 | iex

# Claude 用户
& ([scriptblock]::Create((irm https://raw.githubusercontent.com/SSYCloud/loomloom/main/install.ps1))) -Agent claude

# OpenClaw 用户
& ([scriptblock]::Create((irm https://raw.githubusercontent.com/SSYCloud/loomloom/main/install.ps1))) -Agent openclaw
```

> Windows 默认安装路径：`$HOME\AppData\Local\Programs\loomloom`，若未自动加入 PATH，请手动添加。

**Homebrew（仅 CLI）：**

```bash
brew install ssycloud/tap/loomloom
```

> 注意：Homebrew 仅安装 CLI，不含技能包。如需技能包请使用上方安装脚本。

---

## 🔑 配置 Token

```bash
export LOOMLOOM_SERVER="https://batchjob-test.shengsuanyun.com/batch"
export LOOMLOOM_TOKEN="your-token"
```

> 建议写入 `~/.zshrc` 或 `~/.bashrc`，避免每次重新设置。CLI 仍兼容旧的 `BATCHJOB_SERVER` / `BATCHJOB_TOKEN`，但新配置建议统一使用 `LOOMLOOM_*`。Token 请在胜算云官网申请（[https://console.shengsuanyun.com/user/keys](https://console.shengsuanyun.com/user/keys)）。

---

## ✅ 验证安装

```bash
loomloom doctor
```

看到绿色提示即表示环境配置正常，可以开始使用 🎉

---

## 📦 三种任务模板

| 模板 ID | 适合场景 | 输出类型 | 底层步骤 |
| --- | --- | --- | --- |
| `text-v1` | 文案写作、改写、摘要、问答、代码审查 | 📄 文本 / 文件 | 1 步：文本生成 |
| `text-image-v1` | 插画、概念图、社媒配图 | 🖼️ 图片 | 2 步：提示词整理 → 出图 |
| `text-image-video-v1` | 短剧、广告视频批量制作 | 🖼️ 图片 + 🎬 视频 | 3 步：描述 → 出图 → 出视频 |

## 🔄 标准工作流（Excel 模板）

```bash
# 1. 下载模板
loomloom template download text-image-v1 --output-file ./task.xlsx

# 2. 填写 Excel 后，校验格式
loomloom template validate-file text-image-v1 ./task.xlsx

# 3. 提交任务
loomloom template submit-file text-image-v1 ./task.xlsx

# 4. 监控进度（获取 run-id 后执行）
loomloom run watch <run-id>

# 5. 结果回写到 Excel
loomloom template backfill-results <run-id> ./task.xlsx

# 6. 下载生成文件
loomloom artifact download <run-id> --output-dir ./downloads
```

---

## 📝 模板字段说明

### 📄 文本模板 `text-v1`

| 字段 | 是否必填 | 说明 |
| --- | --- | --- |
| 文本提示词 | ⭐ 必填 | 主任务描述，如"把这段介绍改写为中文，80-120字" |
| 写作要求 | 选填 | 补充风格或格式要求 |
| 参考文本 | 选填 | 短内容直接填入；长内容先用 `input-asset upload` 上传，填入返回的 `input_asset_id` |

### 🖼️ 生图模板 `text-image-v1`

| 字段 | 是否必填 | 说明 |
| --- | --- | --- |
| 图片提示词 | ⭐ 必填 | 描述想要的图片内容 |
| 风格要求 | 选填 | 如"水彩风格"、"写实风格" |
| 图片比例 | ⭐ 必填 | `1:1` / `4:5` / `16:9` / `9:16` |

### 🎬 生视频模板 `text-image-video-v1`

| 字段 | 是否必填 | 说明 |
| --- | --- | --- |
| 画面描述 | ⭐ 必填 | 描述视频画面内容 |
| 视觉风格要求 | 选填 | 如"电影色调"、"动漫风格" |
| 参考图片URL | 选填 | 仅支持 1 张公网可访问图片（http/https 开头） |
| 图片比例 | ⭐ 必填 | `1:1` / `4:5` / `16:9` / `9:16` |
| 视频比例 | ⭐ 必填 | `16:9` / `9:16` |
| 视频时长（秒） | ⭐ 必填 | `4` / `6` / `8` |
| 是否生成声音 | ⭐ 必填 | `false` / `true` |

---

## 📤 大文件上传（Input Asset）

当参考文本较大，不适合直接填入 Excel 时，先上传文件获取 ID：

```bash
loomloom input-asset upload ./my-reference.txt
loomloom input-asset upload ./diagram.png --content-type image/png
```

返回的 `input_asset_id` 填入模板"参考文本"字段即可。

---

## 📊 任务进度查看

所有任务运行状态可在线查看：

**👉 [https://batchjob-test.shengsuanyun.com/home](https://batchjob-test.shengsuanyun.com/home)**

| 状态 | 含义 |
| --- | --- |
| ⚪ 排队中 | 任务已提交，等待调度 |
| 🔵 执行中 | 任务运行中，请耐心等待 |
| 🟢 已结束 | 全部行执行完成，可下载结果 |
| 🟡 部分失败 | 任务完成但部分行出错，成功行仍可下载 |
| 🔴 失败 | 任务整体失败，请检查输入数据 |
| ⏰ 已过期 | 结果已过期，需重新提交 |

---

## 🗂️ 完整 CLI 命令速查

| 命令 | 说明 |
| --- | --- |
| `loomloom doctor` | 检查环境配置是否正常 |
| `loomloom template list` | 查看可用模板列表 |
| `loomloom template schema <id>` | 查看模板字段说明 |
| `loomloom template download <id>` | 下载 Excel 模板 |
| `loomloom template validate-file <id> <xlsx>` | 校验 Excel 填写格式 |
| `loomloom template submit-file <id> <xlsx>` | 提交任务 |
| `loomloom template backfill-results <run-id> <xlsx>` | 结果回写到 Excel |
| `loomloom run submit <id> -f rows.jsonl` | 高级：JSONL 方式提交 |
| `loomloom run watch <run-id>` | 监控任务进度 |
| `loomloom artifact list <run-id>` | 查看生成文件列表 |
| `loomloom artifact download <run-id>` | 下载所有生成文件 |
| `loomloom input-asset upload <file>` | 上传大文件获取 ID |

---

## 🗑️ 卸载

**macOS / Linux：**

```bash
# 全部卸载
curl -fsSL https://raw.githubusercontent.com/SSYCloud/loomloom/main/uninstall.sh | bash

# 仅卸载 CLI
curl -fsSL https://raw.githubusercontent.com/SSYCloud/loomloom/main/uninstall.sh | bash -s -- --cli-only

# 仅卸载技能包
curl -fsSL https://raw.githubusercontent.com/SSYCloud/loomloom/main/uninstall.sh | bash -s -- --skill-only
```

**Windows：**

```powershell
irm https://raw.githubusercontent.com/SSYCloud/loomloom/main/uninstall.ps1 | iex
```

---

## 🙋 常见问题

**Q：不会用终端怎么办？**

Mac 用户按 `Command + 空格` 搜索「终端」；Windows 用户按 `Win 键` 搜索「PowerShell」，打开后粘贴命令回车即可，无需懂代码。

**Q：Token 去哪里获取？**

登录胜算云官网申请，获得 Token 字符串后替换命令里的 `your-token`。

**Q：`template list` 显示 no templates 怎么办？**

`no templates` 是旧版本里的历史提示，这个提示现在已经去除了。
如果 `loomloom template list` 没有返回任何模板，通常表示当前账号或环境下暂无可见模板，联系胜算云管理员确认模板发布状态或权限配置即可。

**Q：可以不用 AI，手动操作吗？**

完全可以。所有功能均可通过 CLI 命令手动执行，参见上方命令速查表。

---

## 🔗 相关链接

- 📦 GitHub 项目：[github.com/SSYCloud/loomloom](http://github.com/SSYCloud/loomloom)
- 📊 任务进度看板：[batchjob-test.shengsuanyun.com/home](http://batchjob-test.shengsuanyun.com/home)
- 🏢 胜算云官网：[shengsuanyun.com](http://shengsuanyun.com)
