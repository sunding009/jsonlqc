# jsonlqc

JSONL 数据质检命令行工具（Go 标准库实现，无第三方依赖）。

读取命令行参数指定的 `.jsonl` 文件，统计并输出：

- 总行数
- 空行数（空白字符组成的行也视为空行）
- 非空行数
- 合法 JSON 行数
- 非法 JSON 行数（坏行计数）及其行号与具体错误原因

校验逻辑：对每一非空行逐行尝试用标准库 `encoding/json` 的 `json.Unmarshal` 解析；
单行解析失败不会中断整个扫描，仅记为坏行（非法行），最终输出坏行计数，并对每个非法行输出行号与具体错误原因（如 `unexpected end of JSON input`）。

读取方式：使用 `bufio.Reader` 流式逐行读取（`ReadString('\n')`）——无单行长度限制，
也不会把整个文件读入内存；超长行同样正常计数并校验，随后继续处理后续行。

可选 schema 校验：通过 `--schema` 指定 schema 文件（JSON），对每行做结构校验——
`required` 列出必填字段，`properties` 声明字段类型；每行顶层必须是 JSON 对象，
缺必填字段或字段类型不符的行计为非法，报告中说明具体原因。

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
坏行详情:
  第 6 行: unexpected end of JSON input
```

`--max-errors` 控制最多报告多少条坏行详情（默认 `0` 表示不限，报告全部）：

```bash
$ jsonlqc --max-errors 1 testdata/invalid.jsonl
...
非法 JSON:  3
坏行详情（显示前 1 条，共 3 条）:
  第 1 行: invalid character 'o' in literal null (expecting 'u')
  ... 其余 2 条未显示
```

`--schema` 指定 schema 文件（JSON），对每行做结构校验：

```bash
$ jsonlqc --schema testdata/schema.json testdata/schema-violations.jsonl
文件:       testdata/schema-violations.jsonl
Schema:     testdata/schema.json
总行数:     5
空行数:     0
非空行数:   5
合法 JSON:  1
非法 JSON:  4
坏行详情:
  第 2 行: 缺少必填字段 "name"
  第 3 行: 字段 "id" 类型应为 integer，实际是 string
  第 4 行: 顶层必须是 JSON 对象，实际是 array
  第 5 行: 字段 "score" 类型应为 number，实际是 string
```

schema 文件格式：`required` 列出必填字段，`properties` 声明字段类型
（支持 `string` / `integer` / `number` / `boolean` / `array` / `object` / `null`）：

```json
{
  "required": ["id", "name"],
  "properties": {
    "id":   {"type": "integer"},
    "name": {"type": "string"}
  }
}
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
| `schema.json` | schema 校验示例（required + properties） |
| `schema-violations.jsonl` | schema 违规数据（缺必填、类型不符、非对象） |
