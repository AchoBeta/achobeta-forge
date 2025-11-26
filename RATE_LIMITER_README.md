# 限流功能使用说明

## 概述

本项目已实现基于 **Redis 的分布式全局限流**功能，使用滑动窗口算法保证在分布式环境下的限流准确性。

## 功能特点

- ✅ **基于 Redis 实现**：支持分布式部署，多实例共享限流状态
- ✅ **滑动窗口算法**：比固定窗口更精确，避免窗口边界突刺问题
- ✅ **全局限流**：限制整个系统的请求速率
- ✅ **按 IP 限流**：支持针对不同 IP 地址进行独立限流（可选）
- ✅ **自动降级**：Redis 不可用时自动跳过限流，不影响服务
- ✅ **可配置**：通过配置文件灵活控制限流参数

## 目录结构

```
interface/middleware/rate_limiter.go    # 限流中间件实现
infra/cache/redis_driver.go             # Redis 客户端封装（已添加获取方法）
infra/configs/configs.go                # 配置结构（已添加限流配置）
conf/config.yaml.template               # 配置模板（已添加限流配置项）
interface/router/router.go              # 路由注册（已应用限流中间件）
pkg/response/code_der.go                # 错误码定义（已添加限流错误码）
```

## 配置说明

### 1. 配置文件（conf/config.yaml）

在配置文件中添加以下配置：

```yaml
rate_limit:   # 限流配置
  enable: true                         # 是否启用限流
  limit: 1000                          # 时间窗口内允许的最大请求数
  window_seconds: 60                   # 时间窗口（秒），默认 60 秒
```

**配置参数说明：**

- `enable`: 
  - `true` - 启用限流
  - `false` - 禁用限流（开发环境可以设置为 false）
  
- `limit`: 时间窗口内允许的最大请求数
  - 示例：`1000` 表示 60 秒内最多 1000 个请求
  - 根据实际业务情况调整
  
- `window_seconds`: 时间窗口大小（秒）
  - 示例：`60` 表示 60 秒的时间窗口
  - 常用值：`60`（1分钟）、`300`（5分钟）

### 2. 常见配置场景

#### 开发环境（不限流）
```yaml
rate_limit:
  enable: false
  limit: 10000
  window_seconds: 60
```

#### 测试环境（宽松限流）
```yaml
rate_limit:
  enable: true
  limit: 5000      # 每分钟 5000 个请求
  window_seconds: 60
```

#### 生产环境（严格限流）
```yaml
rate_limit:
  enable: true
  limit: 1000      # 每分钟 1000 个请求
  window_seconds: 60
```

#### 高频场景（每秒限流）
```yaml
rate_limit:
  enable: true
  limit: 100       # 每秒 100 个请求
  window_seconds: 1
```

## 使用方式

### 1. 全局限流（已自动启用）

限流中间件已在路由注册时全局应用：

```go
// interface/router/router.go
func register() (router *gin.Engine) {
    gin.SetMode(gin.DebugMode)
    r := gin.Default()
    r.Use(middleware.CorsMiddleware())
    r.Use(middleware.RateLimiter())  // 全局限流中间件
    r.RouterGroup = *r.Group("/api/biz/v1", middleware.AddTracer())
    // ...
}
```

### 2. 按 IP 限流（可选）

如果需要针对不同 IP 独立限流，可以使用 `RateLimiterByIP()` 中间件：

```go
// 方式 1: 全局应用
r.Use(middleware.RateLimiterByIP())

// 方式 2: 针对特定路由组
apiGroup := r.Group("/api")
apiGroup.Use(middleware.RateLimiterByIP())
{
    apiGroup.GET("/data", handler.GetData)
}
```

### 3. 针对特定路由限流

```go
// 只对特定路由组应用限流
sensitiveGroup := r.Group("/api/sensitive")
sensitiveGroup.Use(middleware.RateLimiter())
{
    sensitiveGroup.POST("/important", handler.ImportantAction)
}
```

## 工作原理

### 滑动窗口算法

本实现使用 Redis Sorted Set 实现滑动窗口算法：

1. **记录请求时间戳**：每个请求的时间戳作为 score 存入 Sorted Set
2. **清理过期数据**：移除窗口之外的旧请求记录
3. **统计当前请求数**：计算窗口内的请求总数
4. **判断是否限流**：如果请求数超过限制，则拒绝请求

```
时间线：  |-------- 窗口 (60s) --------|
请求：    .....********************...
          清理    统计并判断          当前时间
```

### Redis 数据结构

```
Key: rate_limit:global
Type: Sorted Set
Member: 时间戳（纳秒）
Score: 时间戳（纳秒）
Expire: 窗口时间的 2 倍
```

## 响应格式

### 正常请求

请求通过时，正常返回业务数据。

### 限流响应

当请求被限流时，返回 HTTP 429 状态码：

```json
{
  "code": 429,
  "message": "请求过于频繁，请稍后再试",
  "data": {},
  "meta": {
    "traceId": "xxx",
    "requestTime": "2025-11-24T10:00:00Z"
  }
}
```

## 监控和日志

### 日志示例

**启用限流：**
```
[INFO] 限流中间件已启用: 1000 请求/60 秒
```

**请求被限流：**
```
[WARN] 请求被限流: GET /api/biz/v1/data
[WARN] IP 192.168.1.100 请求被限流: POST /api/biz/v1/action
```

**Redis 错误：**
```
[WARN] Redis 客户端未初始化，跳过限流
[ERROR] 限流检查失败: connection refused
```

## 性能优化

### 1. Redis Pipeline

使用 Redis Pipeline 批量执行命令，减少网络往返：

```go
pipe := client.Pipeline()
pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart))
pipe.ZCard(ctx, key)
pipe.ZAdd(ctx, key, &redis.Z{Score: float64(now), Member: fmt.Sprintf("%d", now)})
pipe.Expire(ctx, key, window*2)
cmds, err := pipe.Exec(ctx)
```

### 2. 自动过期

设置 Redis Key 的过期时间为窗口时间的 2 倍，自动清理过期数据，避免内存泄漏。

### 3. 错误降级

当 Redis 不可用时，自动跳过限流，确保服务可用性：

```go
if redisClient == nil {
    zlog.Warnf("Redis 客户端未初始化，跳过限流")
    c.Next()
    return
}
```

## 常见问题

### Q1: 如何临时禁用限流？

在配置文件中设置 `enable: false`：

```yaml
rate_limit:
  enable: false
```

### Q2: 如何调整限流阈值？

修改配置文件中的 `limit` 和 `window_seconds` 参数：

```yaml
rate_limit:
  enable: true
  limit: 2000        # 调整为 2000
  window_seconds: 60
```

然后重启服务或等待配置热更新生效。

### Q3: Redis 不可用时会怎样？

限流中间件会自动降级，跳过限流检查，不影响服务正常运行。同时会记录警告日志。

### Q4: 如何区分不同环境的限流策略？

为不同环境准备不同的配置文件：

- `config.dev.yaml` - 开发环境（禁用或宽松限流）
- `config.test.yaml` - 测试环境（中等限流）
- `config.prod.yaml` - 生产环境（严格限流）

### Q5: 限流是否支持白名单？

当前版本不支持白名单。如需白名单功能，可以在中间件中添加 IP 白名单判断逻辑。

### Q6: 如何监控限流状态？

可以通过以下方式监控：

1. **日志**：查看 `[WARN]` 级别的限流日志
2. **Redis**：监控 Redis 中 `rate_limit:*` 键的访问情况
3. **APM**：通过 CozeLoop 链路追踪查看 HTTP 429 响应

## 扩展功能

### 1. 添加按用户限流

可以基于用户 ID 实现限流：

```go
func RateLimiterByUser() gin.HandlerFunc {
    // ... 配置初始化 ...
    
    return func(c *gin.Context) {
        userID := c.GetString("user_id") // 从 JWT 中获取
        if userID == "" {
            c.Next()
            return
        }
        
        key := fmt.Sprintf("rate_limit:user:%s", userID)
        allowed, err := checkRateLimit(ctx, redisClient, key, limit, window)
        // ... 后续处理 ...
    }
}
```

### 2. 添加按路由限流

为不同的 API 端点设置不同的限流策略：

```go
func RateLimiterByRoute(limit int, window time.Duration) gin.HandlerFunc {
    return func(c *gin.Context) {
        route := c.FullPath()
        key := fmt.Sprintf("rate_limit:route:%s", route)
        allowed, err := checkRateLimit(ctx, redisClient, key, limit, window)
        // ... 后续处理 ...
    }
}

// 使用
r.GET("/api/expensive", middleware.RateLimiterByRoute(10, time.Minute), handler.ExpensiveOperation)
```

## 测试建议

### 1. 单元测试

```bash
# 测试限流逻辑
go test -v ./interface/middleware -run TestRateLimiter
```

### 2. 压力测试

使用 `ab` 或 `wrk` 进行压力测试：

```bash
# 使用 ab
ab -n 2000 -c 10 http://localhost:8080/api/biz/v1/test

# 使用 wrk
wrk -t10 -c100 -d30s http://localhost:8080/api/biz/v1/test
```

观察响应中是否有 HTTP 429 状态码。

## 总结

本限流功能提供了：

✅ 分布式环境下的全局限流  
✅ 精确的滑动窗口算法  
✅ 灵活的配置和扩展性  
✅ 完善的错误处理和降级机制  
✅ 详细的日志记录  

根据你的实际业务需求调整配置参数，享受稳定可靠的限流保护！


