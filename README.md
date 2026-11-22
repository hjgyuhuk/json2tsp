# json2tsp

[中文](#中文) | [English](#english)

---

## 中文

`json2tsp` 是一个用 Go 编写的命令行工具，读取 JSON 文件或标准输入（`stdin`），输出对应的 TypeSpec 定义。

### 特性

- 支持从 JSON 文件、标准输入或 `-` 读取数据
- 保留 JSON 对象的原始键顺序
- 默认将嵌套对象提取为独立命名 `model`（同形复用、异形按父级前缀去重），支持 `--inline` 切换为内联对象
- 对象数组自动合并字段，缺失字段自动标记为可选（`?`）
- 非法 TypeSpec 标识符与保留关键字自动使用反引号（`` `...` ``）包裹

### 类型映射

| JSON | TypeSpec | 说明 |
| --- | --- | --- |
| 整数 | `int64` | 与浮点数混合时拓宽为 `float64` |
| 浮点数 / 科学计数法 | `float64` | 超过 `int64` 范围的整数也映射为 `float64` |
| 布尔值 | `boolean` | |
| `null` | `null` | 联合类型中自动置后，如 `string \| null` |
| `YYYY-MM-DD` | `plainDate` | 与普通字符串混合时拓宽为 `string` |
| `HH:MM:SS(.fff)` | `plainTime` | 与普通字符串混合时拓宽为 `string` |
| RFC 3339 (`Z`) | `utcDateTime` | 与 `offsetDateTime` 混合时拓宽为 `offsetDateTime` |
| RFC 3339 (`±HH:MM`) | `offsetDateTime` | 与普通字符串混合时拓宽为 `string` |
| 其他字符串 | `string` | |
| 空数组 `[]` | `unknown[]` | |
| 对象数组 | `Item[]` | 合并元素字段，缺失字段标记 `?` |

### 构建

```bash
# 本地构建
go build -o json2tsp .

# 交叉编译（darwin/arm64 与 linux/amd64，输出到 build/）
bash build.sh
```

### 使用方式

```bash
json2tsp [file] [flags]
```

#### 参数

| 参数 | 短选项 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `--name` | `-n` | `Root` | 根 `model` 或 `alias` 名称 |
| `--inline` | | `false` | 内联嵌套对象，不提取独立 `model` |

#### 示例

```bash
# 读取文件
json2tsp order.json --name Order

# 从 stdin 读取
cat order.json | json2tsp -n Order

# 内联嵌套对象
json2tsp --inline order.json
```

输入 JSON：

```json
{
  "id": 1,
  "birth_date": "2026-11-22",
  "alarm_time": "18:32:22",
  "created_at": "2026-11-22T18:32:22Z",
  "updated_at": "2026-11-22T18:32:22+08:00",
  "items": [
    { "name": "book", "price": 12.5 },
    { "name": "pen", "price": 3, "note": "blue" }
  ]
}
```

输出 TypeSpec（`json2tsp --name Order`）：

```tsp
model Order {
  id: int64;
  birth_date: plainDate;
  alarm_time: plainTime;
  created_at: utcDateTime;
  updated_at: offsetDateTime;
  items: Item[];
}

model Item {
  name: string;
  price: float64;
  note?: string;
}
```

### 开发与测试

```bash
go vet ./...
go test -race ./...
```

---

## English

`json2tsp` is a Go CLI tool that reads JSON from a file or `stdin` and outputs corresponding TypeSpec definitions.

### Features

- Reads JSON from a file path, `stdin`, or `-`
- Preserves original JSON object key order
- Extracts nested objects into named `model` declarations by default (reusing identical shapes and disambiguating name collisions with parent prefixes), with `--inline` to emit anonymous inline objects
- Merges fields across object arrays and marks missing fields as optional (`?`)
- Escapes invalid TypeSpec identifiers and reserved keywords with backticks (`` `...` ``)

### Type Mapping

| JSON | TypeSpec | Notes |
| --- | --- | --- |
| Integer | `int64` | Widens to `float64` when mixed with floats |
| Float / Scientific | `float64` | Integers overflowing `int64` also map to `float64` |
| Boolean | `boolean` | |
| `null` | `null` | Placed last in unions, e.g. `string \| null` |
| `YYYY-MM-DD` | `plainDate` | Widens to `string` when mixed with plain strings |
| `HH:MM:SS(.fff)` | `plainTime` | Widens to `string` when mixed with plain strings |
| RFC 3339 (`Z`) | `utcDateTime` | Widens to `offsetDateTime` when mixed with `offsetDateTime` |
| RFC 3339 (`±HH:MM`) | `offsetDateTime` | Widens to `string` when mixed with plain strings |
| Other strings | `string` | |
| Empty array `[]` | `unknown[]` | |
| Object array | `Item[]` | Merges element fields; missing fields become optional (`?`) |

### Build

```bash
# Local build
go build -o json2tsp .

# Cross-compile (darwin/arm64 and linux/amd64 into build/)
bash build.sh
```

### Usage

```bash
json2tsp [file] [flags]
```

#### Flags

| Flag | Short | Default | Description |
| --- | --- | --- | --- |
| `--name` | `-n` | `Root` | Root `model` or `alias` name |
| `--inline` | | `false` | Inline nested objects instead of extracting named models |

#### Examples

```bash
# Read from a file
json2tsp order.json --name Order

# Read from stdin
cat order.json | json2tsp -n Order

# Inline nested objects
json2tsp --inline order.json
```

Input JSON:

```json
{
  "id": 1,
  "birth_date": "2026-11-22",
  "alarm_time": "18:32:22",
  "created_at": "2026-11-22T18:32:22Z",
  "updated_at": "2026-11-22T18:32:22+08:00",
  "items": [
    { "name": "book", "price": 12.5 },
    { "name": "pen", "price": 3, "note": "blue" }
  ]
}
```

Output TypeSpec (`json2tsp --name Order`):

```tsp
model Order {
  id: int64;
  birth_date: plainDate;
  alarm_time: plainTime;
  created_at: utcDateTime;
  updated_at: offsetDateTime;
  items: Item[];
}

model Item {
  name: string;
  price: float64;
  note?: string;
}
```

### Development & Testing

```bash
go vet ./...
go test -race ./...
```
