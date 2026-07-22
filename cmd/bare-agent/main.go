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

const systemPrompt = "当用户给出明确的符号名或文本时，先使用 search_text 定位，不要先遍历目录。" +
	"没有依赖关系的工具调用应在同一轮发出。" +
	"需要调用工具时，只返回工具调用，不要输出计划、过程说明或过渡语。" +
	"获得足够信息后再回答用户；回答结论先行、务必简洁，只包含与问题直接相关的内容。"

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
