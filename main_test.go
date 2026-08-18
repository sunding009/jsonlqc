package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestInspect(t *testing.T) {
	// 注意：Err 字段为 encoding/json 在该 Go 版本下返回的错误文案。
	tests := []struct {
		name string
		file string // testdata 下的文件名
		want Stats
	}{
		{
			name: "混合内容",
			file: "sample.jsonl",
			want: Stats{
				TotalLines:    6,
				EmptyLines:    2, // 1 个真正空行 + 1 个空白字符行
				NonEmptyLines: 4,
				ValidLines:    3,
				InvalidLines:  1,
				BadLines: []BadLine{
					{Line: 6, Err: "unexpected end of JSON input"},
				},
			},
		},
		{
			name: "全部为空行",
			file: "blank.jsonl",
			want: Stats{
				TotalLines:    4,
				EmptyLines:    4,
				NonEmptyLines: 0,
				ValidLines:    0,
				InvalidLines:  0,
				BadLines:      nil,
			},
		},
		{
			name: "空文件",
			file: "empty.jsonl",
			want: Stats{},
		},
		{
			name: "全部合法",
			file: "valid.jsonl",
			want: Stats{
				TotalLines:    3,
				EmptyLines:    0,
				NonEmptyLines: 3,
				ValidLines:    3,
				InvalidLines:  0,
				BadLines:      nil,
			},
		},
		{
			name: "全部非法",
			file: "invalid.jsonl",
			want: Stats{
				TotalLines:    3,
				EmptyLines:    0,
				NonEmptyLines: 3,
				ValidLines:    0,
				InvalidLines:  3,
				BadLines: []BadLine{
					{Line: 1, Err: "invalid character 'o' in literal null (expecting 'u')"},
					{Line: 2, Err: "invalid character 'b' after object key"},
					{Line: 3, Err: "unexpected end of JSON input"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join("testdata", tt.file)
			got, err := Inspect(path)
			if err != nil {
				t.Fatalf("Inspect(%q) 返回错误: %v", path, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Inspect(%q)\n got = %+v\nwant = %+v", path, got, tt.want)
			}
		})
	}
}

func TestInspectFileNotFound(t *testing.T) {
	if _, err := Inspect(filepath.Join("testdata", "不存在的文件.jsonl")); err == nil {
		t.Error("期望文件不存在时返回错误，实际返回 nil")
	}
}

func TestMain_Quiet(t *testing.T) {
	// 通过临时文件验证主流程对统计数字的处理。
	dir := t.TempDir()
	path := filepath.Join(dir, "t.jsonl")
	content := "{\"a\":1}\n\nnot-json\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("写入临时文件失败: %v", err)
	}

	got, err := Inspect(path)
	if err != nil {
		t.Fatalf("Inspect 失败: %v", err)
	}
	want := Stats{
		TotalLines:    3,
		EmptyLines:    1,
		NonEmptyLines: 2,
		ValidLines:    1,
		InvalidLines:  1,
		BadLines: []BadLine{
			{Line: 3, Err: "invalid character 'o' in literal null (expecting 'u')"},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got = %+v\nwant = %+v", got, want)
	}
}

func TestLimitBadLines(t *testing.T) {
	lines := []BadLine{
		{Line: 1, Err: "e1"},
		{Line: 2, Err: "e2"},
		{Line: 3, Err: "e3"},
	}

	tests := []struct {
		name      string
		maxErrors int
		wantLen   int
	}{
		{"零表示不限", 0, 3},
		{"负数表示不限", -1, 3},
		{"限制为 1 条", 1, 1},
		{"限制为 3 条", 3, 3},
		{"超过总数", 10, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := limitBadLines(lines, tt.maxErrors)
			if len(got) != tt.wantLen {
				t.Errorf("limitBadLines(..., %d) 长度 = %d，期望 %d", tt.maxErrors, len(got), tt.wantLen)
			}
			if !reflect.DeepEqual(got, lines[:tt.wantLen]) {
				t.Errorf("limitBadLines(..., %d) = %+v，期望 %+v", tt.maxErrors, got, lines[:tt.wantLen])
			}
		})
	}

	// 空切片
	if got := limitBadLines(nil, 0); got != nil {
		t.Errorf("limitBadLines(nil, 0) = %v，期望 nil", got)
	}
}

func TestInspectLongLine(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "long.jsonl")

	// 构造一个超过 1MB 的合法 JSON 行；旧的 bufio.Scanner（默认 64KB，最大 1MB）会触发 bufio.ErrTooLong 中断。
	payload := strings.Repeat("x", 2*1024*1024) // 2MB
	longLine := `{"id":1,"payload":"` + payload + `"}`

	content := "{\"a\":1}\n" + longLine + "\n{\"b\":2}\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("写入临时文件失败: %v", err)
	}

	got, err := Inspect(path)
	if err != nil {
		t.Fatalf("Inspect 失败（超长行不应中断扫描）: %v", err)
	}

	if got.TotalLines != 3 {
		t.Errorf("总行数 = %d，期望 3", got.TotalLines)
	}
	if got.ValidLines != 3 {
		t.Errorf("合法行数 = %d，期望 3（超长行应被正常校验且后续行继续处理）", got.ValidLines)
	}
	if got.InvalidLines != 0 {
		t.Errorf("非法行数 = %d，期望 0", got.InvalidLines)
	}
}

func TestInspectNoTrailingNewline(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "noeol.jsonl")

	// 最后一行不带换行符。
	content := "{\"a\":1}\n{\"b\":2}"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("写入临时文件失败: %v", err)
	}

	got, err := Inspect(path)
	if err != nil {
		t.Fatalf("Inspect 失败: %v", err)
	}

	if got.TotalLines != 2 {
		t.Errorf("总行数 = %d，期望 2", got.TotalLines)
	}
	if got.ValidLines != 2 {
		t.Errorf("合法行数 = %d，期望 2", got.ValidLines)
	}
}
