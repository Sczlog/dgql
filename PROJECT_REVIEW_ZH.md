# dgql 项目分析与改进建议

本文档基于当前仓库代码进行静态审查，并结合一次本地测试执行结果给出改进优先级建议。

## 1. 项目定位与当前能力

`dgql` 的核心定位是：通过 GraphQL introspection 自动构建 document，调用方只需传 operationName + variables 即可请求，降低手写 query/mutation 的成本。

当前已实现能力：

- 通过 introspection 拉取 schema，并在客户端内生成 query/mutation document 映射。
- 普通 `Query` / `Mutation` 调用。
- `UploadMutation`（multipart 方式）上传。

## 2. 架构观察（亮点）

- API 入口简洁：`Query` / `Mutation` / `UploadMutation` 包装在同一 client 中，调用体验直观。
- 使用 `gjson` 返回结果，便于下游快速读取嵌套字段。
- Schema 解析逻辑独立在 `schema.go`，网络请求与 introspection 在 `introspection.go`，职责边界较清晰。

## 3. 主要问题与风险

### 3.1 健壮性问题（高优先级）

1. **`ParseSchema` 对 `Mutation` 为空场景不安全**：如果目标 schema 只有 `Query` 没有 `Mutation`，`for _, mutation := range mutation.Fields` 会触发空指针。
2. **未知 operationName 不会显式报错**：`Query/Mutation` 从 map 取不到 document 时会发送空 query，错误定位不友好。
3. **`objectTypeMap` 为包级全局变量**：多实例/并发场景下可能互相污染，也不利于测试隔离。
4. **`parseOutputType` / `parseObjectOutput` 使用 panic**：库代码中 panic 会放大错误影响，建议返回 error。

### 3.2 协议与语义问题（中高优先级）

1. **上传请求 `operations` 字段通过字符串拼接构造 JSON**：若 query 中包含引号或换行，可能产生转义问题；建议改为 struct + `json.Marshal`。
2. **GraphQL 错误处理过于简化**：仅检查 `errors` 字段存在，不包含 HTTP status、extensions、path 等上下文。
3. **返回类型绑定 `gjson.Result`**：轻量但耦合高，若用户希望直接反序列化到结构体不够方便。

### 3.3 测试与工程化问题（高优先级）

1. **测试依赖外部服务**：`go test` 默认会失败（要求本地先启动 `example/server.go`），不适合 CI。
2. **缺少单元测试覆盖关键解析逻辑**：如类型递归、参数拼接、嵌套输出选择等。
3. **README 表达与工程现状不一致**：文档已提示“测试不足”，但缺少可执行的“如何运行测试”分层说明（单元测试/集成测试）。

### 3.4 兼容性与可扩展问题（中优先级）

1. `RetrieveType.toArgString()` 目前只处理 `NonNull`，未完整表达 `List` 嵌套（如 `[String!]!`）。
2. `ParseSchema` 仅处理 `Query` 和 `Mutation`，对 `Subscription`、`Union`、`Interface` 支持不完整。
3. introspection 结果每次 `NewClient` 都拉取，缺少缓存与失效策略。

## 4. 建议路线图（按投入产出比排序）

### P0（建议 1~2 周内）

- 将 `ParseSchema` 从 panic/空指针路径改为显式 error 返回。
- operationName 查不到时立即报错（例如 `operation xxx not found`）。
- 把 `objectTypeMap` 改为实例级局部 map（挂在解析上下文对象上）。
- 调整测试分层：
  - 纯单元测试（默认 `go test ./...` 可过）。
  - 集成测试加 build tag（如 `-tags=integration`）或环境变量开关。

### P1（建议 2~4 周内）

- 上传协议的 `operations/map` 构造改为强类型结构 + `json.Marshal`。
- 引入统一错误类型（包含 HTTP status、GraphQL errors、raw body）。
- 增加 `QueryInto(ctx, op, vars, out)` / `MutationInto(...)` 辅助 API，支持直接解码到结构体。

### P2（中期）

- 完善 type system 支持：list/non-null 递归表达、union/interface/subscription。
- introspection schema 缓存与增量刷新。
- 性能优化：大 schema 场景下预编译与 document 生成耗时基准测试。

## 5. 可落地的具体改动清单（开发任务粒度）

1. `ParseSchema() (*GraphqlClient, error)`：
   - 若 query/mutation 不存在则安全跳过或返回可配置错误。
   - 移除 panic，改为 `fmt.Errorf` 上抛。
2. `Query/Mutation`：
   - map miss 时直接返回错误。
3. `RawUpload`：
   - 用结构体组织 `operations` 字段并 `json.Marshal`。
4. 测试：
   - 为 `retrieveType`、`toArgString`、`parseOutputType` 增加表驱动单测。
   - 将依赖示例服务的测试放入 integration 套件。
5. 文档：
   - README 增加“快速测试”章节（单元 vs 集成）。

## 6. 结论

这个项目的思路正确、API 方向清晰，已经具备可用雏形。当前主要瓶颈不是功能点数量，而是**健壮性 + 测试工程化 + 错误可观测性**。建议优先修复 P0，完成后再推进更复杂的 GraphQL 特性支持。
