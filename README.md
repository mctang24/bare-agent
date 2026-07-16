# Bare Agent

一个用 Go 从零实现的极简 Coding Agent，用尽量少的代码跑通 Agent loop、tool calling 和 context management。

[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go)](https://go.dev/)
[![DeepSeek](https://img.shields.io/badge/LLM-DeepSeek-4D6BFE)](https://www.deepseek.com/)
![Coding Agent](https://img.shields.io/badge/AI-Coding_Agent-8A2BE2)

[English](README_EN.md)

项目完整呈现了 Agent 的工作流程：LLM 发起 tool calling，Agent 执行工具并返回结果，LLM 基于更新后的 context 继续推理，直到给出最终回答。

目前默认接入 DeepSeek；如需使用其他 LLM，只需实现项目定义的 `Model` 接口，Agent loop 无需改动。

## 核心功能

- 内置 `list_files`、`read_file` 和 `search_text` 三个只读工具。
- 支持多轮 tool calling，直到 LLM 给出最终回答。
- 支持进程内连续对话，使用 `/new` 清空 context。
- 提供有限 API 重试和结构化错误回传。
- 可选输出 JSONL trace，记录 session、run、模型调用和工具执行事件。

## 使用

需要 Go 1.26、`rg` 和 DeepSeek API Key。

```bash
export DEEPSEEK_API_KEY="your-api-key"
go run ./cmd/bare-agent -root .
```

交互模式：

```text
> 找到 GenerateResponse 并说明它的作用
> /new
> /exit
```

也可以执行一次性任务：

```bash
go run ./cmd/bare-agent -root . "分析这个项目的入口"
```

需要记录运行轨迹时，指定 trace 文件：

```bash
go run ./cmd/bare-agent -root . -trace trace.jsonl "分析这个项目的入口"
```

当前版本只保留 Agent Runtime 的核心能力：不修改文件，不执行任意命令，也不保存跨进程会话。
