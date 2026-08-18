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
	"io"
	"math"
	"os"
	"sort"
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
	// --schema 指定 schema 文件，校验每行结构。
	schemaPath := flag.String("schema", "", "指定 schema 文件（JSON）：required 列必填字段，properties 声明字段类型")
	flag.Parse()

	args := flag.Args()
	if len(args) != 1 {
		usage()
		os.Exit(2)
	}

	path := args[0]

	var schema *Schema
	if *schemaPath != "" {
		var err error
		schema, err = LoadSchema(*schemaPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "错误: %v\n", err)
			os.Exit(1)
		}
	}

	stats, err := Inspect(path, schema)
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
	if schema != nil {
		fmt.Printf("Schema:     %s\n", *schemaPath)
	}
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

// ---- schema 校验 ----

// Property 描述 schema 中某个字段的类型约束。
type Property struct {
	Type string `json:"type"`
}

// Schema 描述 JSONL 数据行的结构约束：required 列出必填字段，properties 声明字段类型。
type Schema struct {
	Required   []string            `json:"required"`
	Properties map[string]Property `json:"properties"`
}

// allowedTypes 是 schema 支持的字段类型集合。
var allowedTypes = map[string]bool{
	"string": true, "integer": true, "number": true, "boolean": true,
	"array": true, "object": true, "null": true,
}

// LoadSchema 从 path 读取并解析 schema 文件（JSON 格式），并校验其字段类型合法。
func LoadSchema(path string) (*Schema, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s Schema
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("解析 schema 文件 %s 失败: %w", path, err)
	}
	if err := s.validate(); err != nil {
		return nil, fmt.Errorf("schema 文件 %s 非法: %w", path, err)
	}
	return &s, nil
}

// validate 校验 schema 中每个 property 的类型是否在支持范围内。
func (s *Schema) validate() error {
	for name, prop := range s.Properties {
		if prop.Type != "" && !allowedTypes[prop.Type] {
			return fmt.Errorf("字段 %q 的类型 %q 不受支持（支持：string/integer/number/boolean/array/object/null）", name, prop.Type)
		}
	}
	return nil
}

// check 校验解析后的 JSON 值是否符合 schema，返回违规原因（空串表示通过）。
func (s *Schema) check(v any) string {
	obj, ok := v.(map[string]any)
	if !ok {
		return fmt.Sprintf("顶层必须是 JSON 对象，实际是 %s", jsonType(v))
	}

	for _, name := range s.Required {
		if _, ok := obj[name]; !ok {
			return fmt.Sprintf("缺少必填字段 %q", name)
		}
	}

	for _, name := range sortedPropertyNames(s.Properties) {
		prop := s.Properties[name]
		val, ok := obj[name]
		if !ok {
			continue // 字段未出现则跳过（是否必填已在上一步校验）
		}
		actual := jsonType(val)
		if prop.Type != "" && !typeMatches(actual, prop.Type) {
			return fmt.Sprintf("字段 %q 类型应为 %s，实际是 %s", name, prop.Type, actual)
		}
	}

	return ""
}

// jsonType 返回一个已解析 JSON 值的类型名（string/integer/number/boolean/array/object/null）。
func jsonType(v any) string {
	switch x := v.(type) {
	case nil:
		return "null"
	case bool:
		return "boolean"
	case string:
		return "string"
	case float64:
		// JSON 数字统一解析为 float64；无小数部分即视为 integer。
		if x == math.Trunc(x) {
			return "integer"
		}
		return "number"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	default:
		return "unknown"
	}
}

// typeMatches 判断实际类型是否满足期望类型（number 兼容 integer）。
func typeMatches(actual, want string) bool {
	if actual == want {
		return true
	}
	return want == "number" && actual == "integer"
}

// sortedPropertyNames 返回 schema 属性名按字典序排序的切片，保证错误报告顺序稳定。
func sortedPropertyNames(props map[string]Property) []string {
	names := make([]string, 0, len(props))
	for name := range props {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Inspect 打开并流式读取 path 指向的文件，返回统计结果。
// 使用 bufio.Reader 逐行读取：无单行长度限制，也不会把整个文件读入内存；
// 超长行同样正常计数并校验，随后继续处理后续行。若 schema 非 nil，则对每行做结构校验。
// 若文件不存在或读取失败则返回错误。
func Inspect(path string, schema *Schema) (Stats, error) {
	var stats Stats

	f, err := os.Open(path)
	if err != nil {
		return Stats{}, err
	}
	defer f.Close()

	r := bufio.NewReader(f)

	for {
		line, err := r.ReadString('\n')

		// 只有读到实际内容（含空行本身的换行符）才计为一行；
		// 文件末尾不带换行符时，最后一次返回剩余内容 + io.EOF。
		if len(line) > 0 {
			stats.TotalLines++
			line = trimLineEnd(line)

			if strings.TrimSpace(line) == "" {
				stats.EmptyLines++
			} else {
				stats.NonEmptyLines++

				// 逐行尝试用 encoding/json 解析；单行解析失败不中断整个扫描，仅记为坏行。
				var v any
				if uerr := json.Unmarshal([]byte(line), &v); uerr != nil {
					stats.InvalidLines++
					stats.BadLines = append(stats.BadLines, BadLine{
						Line: stats.TotalLines,
						Err:  uerr.Error(),
					})
				} else if schema != nil {
					// 语法合法时，若指定了 schema，进一步做结构校验。
					if reason := schema.check(v); reason != "" {
						stats.InvalidLines++
						stats.BadLines = append(stats.BadLines, BadLine{
							Line: stats.TotalLines,
							Err:  reason,
						})
					} else {
						stats.ValidLines++
					}
				} else {
					stats.ValidLines++
				}
			}
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return Stats{}, err
		}
	}

	return stats, nil
}

// trimLineEnd 去掉行尾的换行符（兼容 "\n" 与 "\r\n"）。
func trimLineEnd(s string) string {
	s = strings.TrimSuffix(s, "\n")
	s = strings.TrimSuffix(s, "\r")
	return s
}
