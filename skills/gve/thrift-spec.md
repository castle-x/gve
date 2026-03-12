# GVE Thrift IDL 规范

## 目录

1. [文件位置与命名](#1-文件位置与命名)
2. [namespace 声明](#2-namespace-声明)
3. [struct 定义](#3-struct-定义)
4. [支持的类型](#4-支持的类型)
5. [service 定义](#5-service-定义)
6. [不支持的特性](#6-不支持的特性)
7. [完整示例](#7-完整示例)
8. [最佳实践](#8-最佳实践)

---

GVE 使用 [Apache Thrift IDL](https://thrift.apache.org/docs/idl) 作为 API 契约的 schema 语言，但 **不使用 Thrift 二进制协议**。`gve api generate` 从 `.thrift` 文件中提取 struct 和 service 定义，生成：

- Go struct（纯值类型 + JSON tag，无 Thrift 运行时依赖）
- Go HTTP 客户端（基于 `net/http`）
- TypeScript fetch 客户端

本文档说明在 GVE 项目中编写 `.thrift` 文件的完整规范。

---

## 1. 文件位置与命名

```
api/{project}/{resource}/v{N}/{resource}.thrift
```

- `{project}` — 项目标识（如 `ai-worker`、`ai-console`）
- `{resource}` — 业务资源名（如 `task`、`user`），小写中划线分隔
- `v{N}` — 大版本号（`v1`、`v2`），破坏性变更才升版本
- 文件名必须与 `{resource}` 一致，如 `task.thrift`

示例：

```
api/
└── my-app/
    ├── task/
    │   └── v1/
    │       └── task.thrift
    └── user/
        └── v1/
            └── user.thrift
```

---

## 2. namespace 声明

**必须**声明 `go` namespace，用作生成 Go 代码的 package 名：

```thrift
namespace go task
```

可选声明 `js` namespace（当前不影响生成，留作扩展）：

```thrift
namespace js task
```

namespace 命名规则：
- 小写，单个单词（如 `task`、`user`、`auth`）
- 不要用点号分隔（如 ~~`my.app.task`~~），Go package 名不支持
- 允许下划线（如 `user_profile`），会被原样保留为 Go package 名

---

## 3. Struct 定义

### 基本语法

```thrift
struct User {
  1: required i64 id,
  2: required string name,
  3: required string email,
  4: optional string avatar,
  5: required i64 created_at,
}
```

生成的 Go 代码：

```go
type User struct {
    ID        int64  `json:"id"`
    Name      string `json:"name"`
    Email     string `json:"email"`
    Avatar    string `json:"avatar,omitempty"`
    CreatedAt int64  `json:"created_at"`
}
```

### 字段编号

每个字段必须有唯一的正整数编号（`1:`、`2:`、`3:` ...）。编号在 GVE 中不影响序列化（走 JSON），但作为 Thrift IDL 语法要求必须声明。

### required vs optional

| 修饰符 | Go JSON tag | 语义 |
|--------|------------|------|
| `required` | `json:"field_name"` | 必填字段 |
| `optional` | `json:"field_name,omitempty"` | 可选字段（零值时 JSON 省略） |
| （不写） | `json:"field_name"` | 等同 required |

### 字段命名

- Thrift 中使用 **snake_case**（如 `created_at`、`page_size`）
- 生成的 Go 字段自动转换为 **PascalCase**（如 `CreatedAt`、`PageSize`）
- JSON tag 保留原始 snake_case 命名

常见缩写的特殊处理：

| Thrift 字段名 | Go 字段名 | 说明 |
|--------------|-----------|------|
| `id` | `ID` | |
| `user_id` | `UserID` | |
| `url` | `URL` | |
| `api_key` | `APIKey` | |
| `http_status` | `HTTPStatus` | |
| `cpu_usage` | `CPUUsage` | |
| `uuid` | `UUID` | |

完整缩写列表：`id`, `url`, `http`, `https`, `api`, `ip`, `uri`, `uid`, `uuid`, `sql`, `ssh`, `tcp`, `udp`, `cpu`, `gpu`

---

## 4. 支持的类型

### 基础类型

| Thrift 类型 | Go 类型 | TypeScript 类型 | 说明 |
|------------|---------|----------------|------|
| `bool` | `bool` | `boolean` | |
| `byte` / `i8` | `int8` | `number` | |
| `i16` | `int16` | `number` | |
| `i32` | `int32` | `number` | |
| `i64` | `int64` | `number` | 时间戳、ID 推荐用 i64 |
| `double` | `float64` | `number` | |
| `string` | `string` | `string` | |
| `binary` | `[]byte` | `string` | |

### 容器类型

| Thrift 类型 | Go 类型 | 示例 |
|------------|---------|------|
| `list<T>` | `[]T` | `list<string>` → `[]string` |
| `set<T>` | `[]T` | `set<i64>` → `[]int64`（Go 无 set，用 slice） |
| `map<K,V>` | `map[K]V` | `map<string,i32>` → `map[string]int32` |

容器可嵌套：

```thrift
struct Dashboard {
  1: required map<string, list<i64>> tag_ids,
}
```

生成：`TagIDs map[string][]int64`

### Struct 引用

Struct 之间可以互相引用（同文件内）：

```thrift
struct User {
  1: required i64 id,
  2: required string name,
}

struct GetUserResponse {
  1: required User user,
}

struct ListUsersResponse {
  1: required list<User> users,
  2: required i64 total,
}
```

生成的 Go 代码中，struct 引用为**值类型**（非指针）：

```go
type GetUserResponse struct {
    User User `json:"user"`
}

type ListUsersResponse struct {
    Users []User `json:"users"`
    Total int64  `json:"total"`
}
```

---

## 5. Service 定义

Service 定义 API 方法签名，每个 `.thrift` 文件应有**恰好一个** service：

```thrift
service TaskService {
  GetTaskResponse GetTask(1: GetTaskRequest req),
  ListTasksResponse ListTasks(1: ListTasksRequest req),
  void DeleteTask(1: DeleteTaskRequest req),
}
```

### 命名约定

- Service 名使用 PascalCase，以 `Service` 结尾（如 `TaskService`、`UserService`）
- 方法名使用 PascalCase（如 `GetTask`、`ListTasks`、`CreateTask`）
- 每个方法**最多一个参数**（推荐使用 Request struct 包装）

### 方法签名模式

**推荐：Request/Response struct 模式**

```thrift
struct CreateTaskRequest {
  1: required string name,
  2: optional string description,
}

struct CreateTaskResponse {
  1: required i64 id,
}

service TaskService {
  CreateTaskResponse CreateTask(1: CreateTaskRequest req),
}
```

**允许：简单类型参数**

```thrift
service TaskService {
  string Echo(1: string msg),     // 简单参数
  void Ping(),                     // 无参数无返回
  i64 GetCount(1: i64 category),   // 基础类型参数
}
```

### 生成的客户端

Go（HTTP 客户端）：

```go
// 类型名: {ServiceName}HTTPClient
type TaskServiceHTTPClient struct { ... }

func NewTaskServiceHTTPClient(baseURL string, httpClient *http.Client) *TaskServiceHTTPClient

// 每个方法生成对应函数，POST 到 baseURL/{MethodName}
func (c *TaskServiceHTTPClient) CreateTask(ctx context.Context, reqBody any) (json.RawMessage, error)
func (c *TaskServiceHTTPClient) Ping(ctx context.Context, reqBody any) (json.RawMessage, error)
```

TypeScript（fetch 客户端）：

```typescript
/** Options for customizing client behavior. */
interface ClientOptions {
  fetch?: typeof globalThis.fetch
  baseHeaders?: Record<string, string>
  onError?: (error: Error, method: string) => void
}

export class TaskServiceClient {
  private readonly baseUrl: string
  private readonly options: ClientOptions

  constructor(baseUrl: string, options?: ClientOptions) {
    this.baseUrl = baseUrl
    this.options = options ?? {}
  }

  async CreateTask(reqBody: unknown): Promise<unknown> { ... }
  async Ping(reqBody: unknown): Promise<void> { ... }
}
```

### ClientOptions

生成的 TypeScript 客户端支持可选的 `ClientOptions` 参数，用于注入横切关注点：

- **`fetch`**: 替换默认的 `globalThis.fetch`，用于注入认证、日志、重试等
- **`baseHeaders`**: 每个请求附加的公共 header（与默认 `Content-Type` 合并，可覆盖）
- **`onError`**: 请求失败时的回调（在 throw 之前调用），可用于统一错误上报或 401 登出

所有字段均为可选，不传时行为与旧版完全一致。

用法示例：

```typescript
// 注入认证
const authFetch: typeof fetch = (input, init) => {
  const headers = new Headers(init?.headers)
  headers.set('Authorization', `Bearer ${getToken()}`)
  return fetch(input, { ...init, headers })
}

const client = new TaskServiceClient('/api/tasks/v1', {
  fetch: authFetch,
  baseHeaders: { 'X-Request-Source': 'web-app' },
  onError: (error, method) => {
    console.error(`[${method}]`, error)
  },
})
```

---

## 6. 不支持的特性

以下 Thrift 特性在 GVE 中**不使用且不生成代码**：

| 特性 | 原因 |
|------|------|
| `include` | 不支持跨文件引用，每个 `.thrift` 文件自包含 |
| `enum` | 暂不支持，用 `string` + 注释约定枚举值 |
| `typedef` | 暂不支持，直接使用基础类型 |
| `union` | 暂不支持 |
| `exception` | GVE 用 HTTP 状态码 + JSON 错误体，不用 Thrift exception |
| `const` | 不生成代码 |
| `oneway` | 无意义（HTTP 是请求-响应模型） |
| 多 service | 每个文件只取第一个 service |

### enum 的替代方案

```thrift
// 用 string 字段 + 注释说明取值范围
struct Task {
  1: required i64 id,
  2: required string name,
  3: required string status,   // "pending" | "running" | "completed" | "failed"
}
```

---

## 7. 完整示例

```thrift
namespace go task

// ========== 数据模型 ==========

struct Task {
  1: required i64 id,
  2: required string name,
  3: optional string description,
  4: required string status,         // "pending" | "running" | "completed"
  5: required i64 created_at,
  6: required i64 updated_at,
  7: optional map<string, string> metadata,
}

// ========== 请求/响应 ==========

struct CreateTaskRequest {
  1: required string name,
  2: optional string description,
  3: optional map<string, string> metadata,
}

struct CreateTaskResponse {
  1: required Task task,
}

struct GetTaskRequest {
  1: required i64 id,
}

struct GetTaskResponse {
  1: required Task task,
}

struct ListTasksRequest {
  1: optional i32 page,
  2: optional i32 page_size,
  3: optional string status,
}

struct ListTasksResponse {
  1: required list<Task> tasks,
  2: required i64 total,
}

struct DeleteTaskRequest {
  1: required i64 id,
}

// ========== 服务接口 ==========

service TaskService {
  CreateTaskResponse CreateTask(1: CreateTaskRequest req),
  GetTaskResponse GetTask(1: GetTaskRequest req),
  ListTasksResponse ListTasks(1: ListTasksRequest req),
  void DeleteTask(1: DeleteTaskRequest req),
}
```

执行 `gve api generate` 后生成的文件：

```
internal/api/{project}/task/v1/
├── task.go       # Task, CreateTaskRequest, CreateTaskResponse 等 struct
└── client.go     # TaskServiceHTTPClient（Go HTTP 客户端）

site/src/api/{project}/task/v1/
└── client.ts     # TaskServiceClient（TypeScript fetch 客户端）
```

---

## 8. 最佳实践

1. **一个 resource 一个 thrift 文件** — 不要把多个不相关的业务资源塞进一个文件
2. **Request/Response 包装** — 即使只有一个字段，也用 struct 包装，方便后续扩展
3. **时间用 i64** — Unix 毫秒时间戳，前后端统一处理
4. **ID 用 i64** — 保持一致性
5. **snake_case 字段名** — 与 JSON 惯例一致，Go 端自动转 PascalCase
6. **写注释说明枚举值** — `string` 类型字段如果有固定取值，用注释标注
7. **避免深层嵌套** — 保持 struct 结构扁平，最多两层嵌套
