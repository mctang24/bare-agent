package main

import (
	"bare-agent/internal/agent"
	"bare-agent/internal/deepseek"
	"bare-agent/internal/trace"
	"bufio"
	"context"
	"fmt"
	"os"
)

const systemPrompt = "代码检索时，有明确符号或文本就优先使用 search_text，无法构造搜索词时才使用 list_files。" +
	"参数已知且互不依赖的工具调用必须同轮发出，存在依赖时才串行。" +
	"每次工具调用都必须获得回答所需的新证据；除非上次结果不完整或文件已变化，不要重复相同搜索或读取。" +
	"证据足够后立即回答，不要继续扩大检索范围。" +
	"任何包含 tool_calls 的 assistant 消息，其 content 必须为空，禁止输出计划、状态、过渡语或解释。" +
	"最终回答先用一句话给出结论，再用必要的简短要点覆盖用户明确要求；每项信息只说一次，禁止表格、代码块、流程图、背景复述和探索过程。"

func main() {
	config, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	client, err := deepseek.NewClient(os.Getenv("DEEPSEEK_API_KEY"), "", "")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	runner, err := agent.NewAgent(config.root, client, systemPrompt)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if config.tracePath != "" {
		if err := runner.EnableTrace(trace.Writer{Path: config.tracePath}); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	if config.task == "" {
		if err := runInteractive(context.Background(), runner, os.Stdin, os.Stdout, os.Stderr); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	runner.SetWriteApprover(newScannerWriteApprover(bufio.NewScanner(os.Stdin), os.Stdout))
	if err := runTask(context.Background(), runner, config.task, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println()
}
