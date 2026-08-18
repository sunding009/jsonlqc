// jsonlqc 是一个 JSONL 数据质检命令行工具。
// 它读取指定 .jsonl 文件，统计总行数、空行数、非空行数，
// 并对每一非空行做 JSON 合法性校验，最后输出统计结果。
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
)

// Stats 保存一次质检的统计结果。
type Stats struct {
	TotalLines    int   // 总行数
	EmptyLines    int   // 空行数（空白字符组成的行也视为空行）
	NonEmptyLines int   // 非空行数
	ValidLines    int   // 合法 JSON 的行数（仅统计非空行）
	InvalidLines  int   // 非法 JSON 的行数（仅统计非空行）
	InvalidRows   []int // 非法 JSON 行所在的行号（1 起始）
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
	if len(stats.InvalidRows) > 0 {
		fmt.Printf("非法行号:   %v\n", stats.InvalidRows)
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
		if json.Valid([]byte(line)) {
			stats.ValidLines++
		} else {
			stats.InvalidLines++
			stats.InvalidRows = append(stats.InvalidRows, stats.TotalLines)
		}
	}

	if err := scanner.Err(); err != nil {
		return Stats{}, err
	}

	return stats, nil
}
