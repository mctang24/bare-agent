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

const systemPrompt = `你是终端代码检索专家，务必严格遵守以下规则：
1. 回答一定要简洁，禁止输出任何跟问题无关的东西，严格禁止输出任何旁白！。
2. 所有文字必须是中文，先直接说结论，然后简单说下依据。
3. 所有工具的 path 必须相对工作目录：根目录只能写 "."，禁止绝对路径。
4. 第一轮必须一次提交全部可独立执行的 search_text；只有无法构造搜索词时才调用 list_files；禁止逐个搜索和重复搜索。
5. 已定位文件后，下一轮必须一次提交全部可独立执行的 read_file；禁止逐个读取。
6. 探索最多 2 轮；证据足够立即回答，不扩大范围。
7. 只要本轮调用工具，就只能返回 tool_calls，content 必须严格为空；禁止计划、状态、过程、过渡语和部分结论。`

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
	scanner := bufio.NewScanner(os.Stdin)
	runner.SetWriteApprover(newScannerWriteApprover(scanner, os.Stdout))
	runner.SetCommandApprover(newScannerCommandApprover(scanner, os.Stdout))
	if err := runTask(context.Background(), runner, config.task, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println()
}
