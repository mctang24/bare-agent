# Bare Agent

A minimal coding agent built from scratch in Go to make the agent loop, tool calling, and context management easy to understand.

[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go)](https://go.dev/)
[![DeepSeek](https://img.shields.io/badge/LLM-DeepSeek-4D6BFE)](https://www.deepseek.com/)
![Coding Agent](https://img.shields.io/badge/AI-Coding_Agent-8A2BE2)

[中文](README.md)

The runtime follows a complete agent loop: the LLM requests a tool, the agent executes it and returns the result, and the LLM continues with the updated context until it produces a final answer.

DeepSeek is the default provider. To use another LLM, implement the project's `Model` interface; the agent loop remains unchanged.

## Core features

- Three read-only tools for listing files, reading files, and searching code.
- Multi-step tool calling until the LLM produces a final answer.
- In-memory conversation history, with `/new` to start a fresh context.
- Bounded API retries and structured error results.
- Optional JSONL tracing for sessions, runs, model calls, and tool executions.

## Usage

Requires Go 1.26, `rg`, and a DeepSeek API key.

```bash
export DEEPSEEK_API_KEY="your-api-key"
go run ./cmd/bare-agent -root .
```

Interactive mode:

```text
> Find GenerateResponse and explain what it does
> /new
> /exit
```

You can also run a one-off task:

```bash
go run ./cmd/bare-agent -root . "Explain the entry point of this project"
```

To record an execution trace, provide a trace file:

```bash
go run ./cmd/bare-agent -root . -trace trace.jsonl "Explain the entry point of this project"
```

This version is intentionally read-only. It does not edit files, run arbitrary commands, or persist conversations across processes.
