# 网络搜索工具使用说明

## 概述

本项目已为AI模型添加了网络搜索工具（`web_search`），使AI能够在需要时进行联网搜索，获取实时信息和网络资料。

## 功能特性

- ✅ **自动工具调用**: AI模型会在需要查询实时信息或网络资料时自动调用搜索工具
- ✅ **多搜索引擎支持**: 支持 DuckDuckGo（免费）、SerpAPI 等搜索服务
- ✅ **智能结果格式化**: 返回标题、链接和摘要的结构化搜索结果
- ✅ **可配置搜索数量**: 支持设置返回结果的数量（1-10条）

## 配置说明

### 1. 配置文件设置

在 `conf/config.yaml` 中添加或修改搜索服务配置：

```yaml
search:       # 搜索服务配置
  provider: "duckduckgo"               # 搜索服务提供商
  api_key: ""                          # API密钥（可选）
```

### 2. 支持的搜索提供商

#### DuckDuckGo (推荐，免费)
- **配置**: `provider: "duckduckgo"`
- **API Key**: 不需要
- **优点**: 免费、无需注册、尊重隐私
- **限制**: 结果数量可能受限

#### SerpAPI (高级功能)
- **配置**: `provider: "serpapi"`
- **API Key**: 需要（在 https://serpapi.com 注册获取）
- **优点**: 更丰富的搜索结果、更稳定
- **限制**: 免费账户有请求次数限制

配置示例：
```yaml
search:
  provider: "serpapi"
  api_key: "your_serpapi_key_here"
```

## 使用方式

### AI自动调用

AI模型会在以下场景自动调用搜索工具：

1. **查询实时信息**
   - 用户: "最新的人工智能新闻是什么？"
   - AI: 自动调用 `web_search(query="人工智能最新新闻")`

2. **获取网络资料**
   - 用户: "Go语言怎么使用channels？"
   - AI: 自动调用 `web_search(query="Go语言 channels 教程")`

3. **查找特定信息**
   - 用户: "帮我找一下DuckDuckGo API的文档"
   - AI: 自动调用 `web_search(query="DuckDuckGo API documentation")`

### 工具参数

```go
type WebSearchParams struct {
    Query      string // 搜索关键词（必填）
    MaxResults int    // 返回的最大结果数量，默认5条，最多10条（可选）
}
```

### 返回格式

搜索工具返回的结果格式：

```
搜索「关键词」找到 N 条结果：

1. **结果标题**
   链接: https://example.com
   摘要: 这是结果的摘要信息...

2. **另一个结果标题**
   链接: https://another-example.com
   摘要: 这是另一个结果的摘要...

...
```

## 实现细节

### 文件结构

```
infra/eino/
├── search_service.go      # 搜索服务实现
├── tool.go                # 工具定义（包含WebSearch函数）
└── aichat_service.go      # AI客户端（注册搜索工具）

infra/configs/
└── configs.go             # 配置读取（添加SearchConfig）

conf/
├── config.yaml            # 实际配置文件
└── config.yaml.template   # 配置模板
```

### 核心代码

#### 1. 搜索服务 (`search_service.go`)
```go
type SearchService struct {
    Provider string
    APIKey   string
    client   *http.Client
}

func (s *SearchService) Search(ctx context.Context, query string, maxResults int) ([]SearchResult, error)
```

#### 2. 工具定义 (`tool.go`)
```go
func (a *AiChatClient) WebSearch(ctx context.Context, params *WebSearchParams) (string, error)
func (a *AiChatClient) CreateWebSearchTool() tool.InvokableTool
```

#### 3. 工具注册 (`aichat_service.go`)
```go
// 在NewAiChatClient函数中
webSearchTool := aiChatClient.CreateWebSearchTool()
// 添加到工具列表
Tools: []tool.BaseTool{
    updateMindMapTool,
    webSearchTool,  // 新增
}
```

## 测试验证

### 1. 启动应用
```bash
go run cmd/main.go
```

### 2. 测试搜索功能

通过AI对话接口发送消息：

```json
{
  "message": "帮我搜索一下Go语言的最新版本"
}
```

AI会自动调用搜索工具并返回结果。

### 3. 查看日志

搜索工具会输出以下日志：
```
[INFO] 开始网络搜索: query=Go语言最新版本, maxResults=5
[INFO] 网络搜索完成，返回 5 条结果
```

## 故障排查

### 问题1: 搜索无结果
**原因**: DuckDuckGo API可能对某些查询返回空结果
**解决**: 系统会自动降级到HTML搜索方式，或考虑切换到SerpAPI

### 问题2: 请求超时
**原因**: 网络问题或搜索服务响应慢
**解决**: 
- 检查网络连接
- 增加超时时间（在 `search_service.go` 中修改 `Timeout` 值）

### 问题3: API Key无效（SerpAPI）
**原因**: API密钥配置错误或已过期
**解决**: 
- 检查 `config.yaml` 中的 `api_key` 配置
- 访问 https://serpapi.com 验证API密钥

### 问题4: AI不调用搜索工具
**原因**: 查询可能不需要联网搜索
**解决**: 
- 明确要求AI搜索："帮我搜索..."
- 询问实时信息："最新的...是什么？"

## 扩展功能

### 添加新的搜索提供商

在 `search_service.go` 中添加新的搜索方法：

```go
func (s *SearchService) searchYourProvider(ctx context.Context, query string, maxResults int) ([]SearchResult, error) {
    // 实现你的搜索逻辑
}

// 在Search方法中添加case分支
case "your_provider":
    return s.searchYourProvider(ctx, query, maxResults)
```

### 自定义搜索结果处理

可以在 `tool.go` 的 `WebSearch` 方法中自定义结果格式化逻辑。

## 性能优化建议

1. **缓存搜索结果**: 对相同查询在短时间内返回缓存结果
2. **并发控制**: 限制同时进行的搜索请求数量
3. **结果过滤**: 根据业务需求过滤不相关的搜索结果
4. **超时控制**: 设置合理的请求超时时间

## 安全注意事项

1. **API密钥安全**: 不要将API密钥提交到版本控制系统
2. **请求频率限制**: 注意搜索服务的频率限制，避免被封禁
3. **用户输入过滤**: 对搜索关键词进行适当的过滤和清理
4. **结果验证**: 对搜索结果进行基本的安全检查

## 更新日志

### v1.0.0 (2025-11-25)
- ✨ 初始版本
- ✅ 支持 DuckDuckGo 免费搜索
- ✅ 支持 SerpAPI 高级搜索
- ✅ 自动工具调用集成
- ✅ 可配置的结果数量

## 贡献指南

欢迎提交问题和改进建议！

## 许可证

与主项目保持一致

