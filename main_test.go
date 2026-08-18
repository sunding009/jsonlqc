package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestInspect(t *testing.T) {
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
				InvalidRows:   []int{6},
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
				InvalidRows:   nil,
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
				InvalidRows:   nil,
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
				InvalidRows:   []int{1, 2, 3},
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
		InvalidRows:   []int{3},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got = %+v\nwant = %+v", got, want)
	}
}
