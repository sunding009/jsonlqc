# jsonlqc

JSONL 数据质检命令行工具（Go 标准库实现，无第三方依赖）。

读取命令行参数指定的 `.jsonl` 文件，统计并输出：

- 总行数
- 空行数（空白字符组成的行也视为空行）
- 非空行数
- 合法 JSON 行数
- 非法 JSON 行数及其行号

校验逻辑：对每一非空行逐行尝试用标准库 `encoding/json` 的 `json.Unmarshal` 解析；
单行解析失败不会中断整个扫描，仅记为坏行（非法行），最终输出坏行计数与行号。

## 安装

```bash
go install github.com/sunding009/jsonlqc@latest
```

或本地构建：

```bash
go build -o jsonlqc .
```

## 用法

```bash
jsonlqc <文件.jsonl>
```

示例：

```bash
$ jsonlqc testdata/sample.jsonl
文件:       testdata/sample.jsonl
总行数:     6
空行数:     2
非空行数:   4
合法 JSON:  3
非法 JSON:  1
非法行号:   [6]
```

静默模式（仅输出统计数字，便于脚本消费）：

```bash
$ jsonlqc -q testdata/sample.jsonl
6 2 4 3 1
```

数字顺序为：`总行数 空行数 非空行数 合法行数 非法行数`。

## 退出码

- `0`：成功
- `1`：文件打开或读取失败
- `2`：命令行参数错误

## 运行测试

```bash
go test ./...
```

测试数据位于 `testdata/` 目录：

| 文件 | 说明 |
| --- | --- |
| `sample.jsonl` | 混合内容：合法 JSON、空行、空白行、非法 JSON |
| `valid.jsonl` | 全部为合法 JSON |
| `invalid.jsonl` | 全部为非法 JSON |
| `blank.jsonl` | 全部为空行 / 空白行 |
| `empty.jsonl` | 空文件 |
