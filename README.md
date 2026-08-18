# jsonlqc

JSONL 数据质检命令行工具（纯 Go 标准库实现，零第三方依赖）。

读取命令行参数指定的 `.jsonl` 文件，逐行统计并输出总行数、空行数、非空行数，
并对每一非空行用标准库 `encoding/json` 真实解析，统计合法 JSON 行数与非法（坏）行数，
输出坏行的行号与具体错误原因；可选 `--schema` 做字段级结构校验。

## 选题说明

选择「JSONL 数据质检工具」作为本次 AI 开发考核的选题，主要基于以下几点考量：

- **贴近真实场景**：JSONL（每行一个 JSON 对象）是 LLM 训练数据（SFT / RLHF / DPO 数据集）、
  日志采集、数据管道交换的行业事实标准。训练数据在进入模型前必须先过质量关，
  一个能快速定位「哪一行、为什么坏」的质检工具具备直接落地的实用价值，而非纯演示代码。
- **能集中考察 Go 语言核心能力**：只用标准库即可完成完整功能——
  `flag` 命令行解析、`bufio` 流式读取、`encoding/json` 解析与错误处理、
  `os`/`io` 文件操作、`sort` 确定性输出，覆盖了 Go 工程中最常用的标准库面。
- **能考察工程化与边界处理能力**：题目天然包含大量边界情况——空文件、空行/空白行、
  末尾无换行符、单行 2MB 以上的超长行、非法 JSON 不中断扫描、schema 字段类型判断
  （`integer` 与 `number` 的 float64 兼容等），能充分验证对健壮性的把控。
- **便于形成完整的交付物闭环**：从源码、单元测试、测试数据，到 Makefile
  （build / test / lint）、Dockerfile（多阶段构建 + scratch 空镜像）、README 文档，
  一套完整、可复现、可移植的工程实践，能全面体现「编码之外」的交付素养。

## 功能特性

- 统计总行数、空行数（含仅空白字符的行）、非空行数
- 逐行真实解析 JSON，统计合法行数与非法行数；单行失败不中断整个扫描
- 坏行报告输出行号 + 具体错误原因（如 `unexpected end of JSON input`）
- `--max-errors` 控制最多报告多少条坏行详情
- `--schema` 字段级结构校验：必填字段、字段类型（`string`/`integer`/`number`/`boolean`/`array`/`object`/`null`）
- `bufio.Reader` 流式读取：无单行长度限制、不整文件读入内存，超长行照常处理
- `-q`/`--quiet` 静默模式，仅输出统计数字，便于脚本消费
- 纯标准库实现，静态编译，可打包进 `scratch` 空镜像

## 安装

### 方式一：go install（推荐）

```bash
go install github.com/sunding009/jsonlqc@latest
```

安装后二进制位于 `$GOPATH/bin/jsonlqc`（请确保该目录已加入 `PATH`）。

### 方式二：源码构建

```bash
git clone git@github.com:sunding009/jsonlqc.git
cd jsonlqc
go build -o jsonlqc .     # 或 make build
./jsonlqc --help
```

### 方式三：Docker 镜像

```bash
docker build -t jsonlqc .   # 或 make docker-build
docker run --rm -v "$PWD:/data" jsonlqc /data/testdata/sample.jsonl
```

镜像采用多阶段构建，运行阶段基于 `scratch` 空镜像，仅包含静态二进制，体积最小、
攻击面最小。由于工具只读取本地文件、不做网络请求，无需额外打包 CA 证书等运行时文件。

## 使用示例

### 基本用法

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

### 静默模式（便于脚本消费）

```bash
$ jsonlqc -q testdata/sample.jsonl
6 2 4 3 1
```

数字顺序固定为：`总行数 空行数 非空行数 合法行数 非法行数`。

### 限制坏行报告数量

```bash
$ jsonlqc --max-errors 1 testdata/invalid.jsonl
文件:       testdata/invalid.jsonl
总行数:     3
空行数:     0
非空行数:   3
合法 JSON:  0
非法 JSON:  3
坏行详情（显示前 1 条，共 3 条）:
  第 1 行: invalid character 'o' in literal null (expecting 'u')
  ... 其余 2 条未显示
```

### schema 结构校验

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

schema 文件本身是一个 JSON，`required` 列出必填字段，`properties` 声明字段类型
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

校验规则：每行顶层必须是 JSON 对象；缺必填字段、字段类型不符、顶层非对象均计为非法行，
并在坏行详情中说明具体原因。`number` 类型兼容整数（`integer` 满足 `number` 约束）。

## 参数文档

位置参数（必填）：

| 参数 | 说明 |
| --- | --- |
| `<文件.jsonl>` | 待质检的 JSONL 文件路径 |

可选参数：

| 参数 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `-q` / `--quiet` | 布尔 | `false` | 静默模式，仅输出 5 个统计数字（顺序：总行数 空行数 非空行数 合法行数 非法行数） |
| `--max-errors <N>` | 整数 | `0` | 最多报告多少条坏行详情；`0` 或负数表示不限，报告全部 |
| `--schema <文件>` | 字符串 | 空 | 指定 schema 文件（JSON），对每行做必填字段与字段类型校验 |
| `-h` / `--help` | — | — | 打印帮助信息并退出（退出码 `0`） |

## 退出码

| 退出码 | 含义 |
| --- | --- |
| `0` | 成功（含 `-h`/`--help` 打印帮助） |
| `1` | 文件打开/读取失败，或 schema 文件加载/解析失败 |
| `2` | 命令行参数错误（缺少文件参数或参数数量不对） |

## 开发

### Makefile

```bash
make build      # 编译二进制（-trimpath -s -w 缩小体积）
make test       # 运行单元测试（-race 数据竞争检测）
make lint       # 代码质量检查（gofmt 格式 + go vet）
make vet        # 仅运行 go vet
make fmt        # 格式化所有 Go 源文件
make docker-build  # 构建 Docker 镜像（多阶段 + scratch）
make clean      # 清理构建产物
make help       # 列出所有目标
```

### 运行测试

```bash
go test ./...        # 或 make test
```

测试覆盖：正常文件、空文件、含坏行的文件、超长行（2MB）、末尾无换行符、schema 校验
（缺字段 / 类型错误 / 顶层非对象 / 全类型边界）、`--max-errors` 截断逻辑等。

测试数据位于 `testdata/` 目录：

| 文件 | 说明 |
| --- | --- |
| `sample.jsonl` | 混合内容：合法 JSON、空行、空白行、非法 JSON |
| `valid.jsonl` | 全部为合法 JSON |
| `invalid.jsonl` | 全部为非法 JSON |
| `blank.jsonl` | 全部为空行 / 空白行 |
| `empty.jsonl` | 空文件 |
| `schema.json` | schema 校验示例（required + properties 全类型） |
| `schema-violations.jsonl` | schema 违规数据（缺必填、类型不符、非对象） |

## 项目结构

```text
jsonlqc/
├── main.go          # 程序入口与核心逻辑（Inspect / schema 校验 / 坏行报告）
├── main_test.go     # 单元测试（表驱动，覆盖边界与 schema 全类型）
├── go.mod           # 模块定义（module github.com/sunding009/jsonlqc）
├── testdata/        # 测试数据
├── Dockerfile       # 多阶段构建 + scratch 基础镜像
├── Makefile         # build / test / lint 等目标
├── .dockerignore    # Docker 构建上下文忽略项
└── README.md        # 本文档
```
