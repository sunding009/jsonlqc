// jsonlqc 是一个 JSONL 数据质检命令行工具。
// 它读取指定 .jsonl 文件，统计总行数、空行数、非空行数，
// 并对每一非空行逐行尝试用 encoding/json 解析，统计合法 JSON 行数与非法（坏）行数，
// 非法行不会中断整个扫描，最后输出统计结果。
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
)

// BadLine 描述一条非法 JSON 行。
type BadLine struct {
	Line int    // 行号（1 起始）
	Err  string // 解析错误原因
}

// Stats 保存一次质检的统计结果。
type Stats struct {
	TotalLines    int       // 总行数
	EmptyLines    int       // 空行数（空白字符组成的行也视为空行）
	NonEmptyLines int       // 非空行数
	ValidLines    int       // 合法 JSON 的行数（仅统计非空行）
	InvalidLines  int       // 非法 JSON 的行数（仅统计非空行）
	BadLines      []BadLine // 非法行详情（行号 + 错误原因）
}

// usage 打印命令行用法。
func usage() {
	fmt.Fprintf(os.Stderr, "用法: %s [选项] <文件.jsonl>\n\n", os.Args[0])
	fmt.Fprintln(os.Stderr, "统计指定 .jsonl 文件的总行数、空行数、非空行数，并校验每行 JSON 是否合法。")
	fmt.Fprintln(os.Stderr, "\n选项:")
	flag.PrintDefaults()
}

func main() {
	flag.Usage = usage

	// 支持 -q/--quiet 静默模式：仅输出统计数字，便于脚本消费。
	quiet := flag.Bool("q", false, "静默模式，仅输出统计数字（顺序：总行数 空行数 非空行数 合法行数 非法行数）")
	flag.BoolVar(quiet, "quiet", false, "同 -q")
	// --max-errors 控制最多报告多少条坏行详情（0 表示不限）。
	maxErrors := flag.Int("max-errors", 0, "最多报告多少条坏行详情（0 表示不限，报告全部）")
	flag.Parse()

	args := flag.Args()
	if len(args) != 1 {
		usage()
		os.Exit(2)
	}

	path := args[0]
	stats, err := Inspect(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}

	if *quiet {
		fmt.Printf("%d %d %d %d %d\n",
			stats.TotalLines, stats.EmptyLines, stats.NonEmptyLines,
			stats.ValidLines, stats.InvalidLines)
		return
	}

	fmt.Printf("文件:       %s\n", path)
	fmt.Printf("总行数:     %d\n", stats.TotalLines)
	fmt.Printf("空行数:     %d\n", stats.EmptyLines)
	fmt.Printf("非空行数:   %d\n", stats.NonEmptyLines)
	fmt.Printf("合法 JSON:  %d\n", stats.ValidLines)
	fmt.Printf("非法 JSON:  %d\n", stats.InvalidLines)
	reportBadLines(stats, *maxErrors)
}

// limitBadLines 返回需要显示的坏行切片，受 maxErrors 限制（0 或负数表示不限）。
func limitBadLines(lines []BadLine, maxErrors int) []BadLine {
	if maxErrors <= 0 || maxErrors > len(lines) {
		maxErrors = len(lines)
	}
	return lines[:maxErrors]
}

// reportBadLines 打印非法行详情（行号 + 错误原因），受 maxErrors 限制（0 表示不限制）。
func reportBadLines(stats Stats, maxErrors int) {
	if len(stats.BadLines) == 0 {
		return
	}

	shown := limitBadLines(stats.BadLines, maxErrors)
	if len(shown) < len(stats.BadLines) {
		fmt.Printf("坏行详情（显示前 %d 条，共 %d 条）:\n", len(shown), len(stats.BadLines))
	} else {
		fmt.Println("坏行详情:")
	}
	for _, b := range shown {
		fmt.Printf("  第 %d 行: %s\n", b.Line, b.Err)
	}
	if len(shown) < len(stats.BadLines) {
		fmt.Printf("  ... 其余 %d 条未显示\n", len(stats.BadLines)-len(shown))
	}
}

// Inspect 打开并扫描 path 指向的文件，返回统计结果。
// 若文件不存在或读取失败则返回错误。
func Inspect(path string) (Stats, error) {
	var stats Stats

	f, err := os.Open(path)
	if err != nil {
		return Stats{}, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	// 默认缓冲 64KB，扩大到 1MB 以容纳超长的 JSON 行。
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		stats.TotalLines++
		line := scanner.Text()

		if strings.TrimSpace(line) == "" {
			stats.EmptyLines++
			continue
		}

		stats.NonEmptyLines++

		// 逐行尝试用 encoding/json 解析；单行解析失败不中断整个扫描，仅记为坏行。
		var v any
		if err := json.Unmarshal([]byte(line), &v); err != nil {
			stats.InvalidLines++
			stats.BadLines = append(stats.BadLines, BadLine{
				Line: stats.TotalLines,
				Err:  err.Error(),
			})
		} else {
			stats.ValidLines++
		}
	}

	if err := scanner.Err(); err != nil {
		return Stats{}, err
	}

	return stats, nil
}
