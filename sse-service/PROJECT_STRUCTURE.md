# 📁 项目结构说明

## 完整目录结构

```
sse-service/
├── pom.xml                                 # Maven配置文件
├── README.md                               # 项目文档
├── QUICK_START.md                          # 快速启动指南
├── PROJECT_STRUCTURE.md                    # 本文件
├── test-push.sh                            # 测试脚本
│
├── src/
│   ├── main/
│   │   ├── java/
│   │   │   └── com/example/sse/
│   │   │       ├── SseServiceApplication.java       # 主程序入口
│   │   │       │
│   │   │       ├── controller/                      # 控制器层
│   │   │       │   ├── SseController.java          # SSE连接控制器
│   │   │       │   └── DataPushController.java     # 数据推送控制器
│   │   │       │
│   │   │       ├── service/                        # 服务层
│   │   │       │   └── SseService.java             # SSE核心服务
│   │   │       │
│   │   │       ├── model/                          # 数据模型
│   │   │       │   ├── PushData.java               # 推送数据模型
│   │   │       │   └── ApiResponse.java            # API响应模型
│   │   │       │
│   │   │       └── config/                         # 配置类
│   │   │           ├── CorsConfig.java             # 跨域配置
│   │   │           └── ScheduleConfig.java         # 定时任务配置
│   │   │
│   │   └── resources/
│   │       ├── application.yml                     # 应用配置
│   │       └── static/
│   │           └── test.html                       # 测试页面
│   │
│   └── test/
│       └── java/
│           └── com/example/sse/
│               └── client/
│                   └── JavaSseClient.java          # Java客户端示例
│
└── target/                                         # 编译输出目录（自动生成）
```

## 核心文件说明

### 1. 主程序入口

**SseServiceApplication.java**
- Spring Boot应用主类
- 启动服务
- 显示服务信息

### 2. 控制器层 (Controller)

#### SseController.java
负责SSE连接管理：
- `GET /api/sse/subscribe` - 客户端订阅SSE
- `DELETE /api/sse/disconnect/{clientId}` - 断开连接
- `GET /api/sse/stats` - 获取统计信息
- `POST /api/sse/heartbeat` - 发送心跳

#### DataPushController.java
负责接收和转发数据：
- `POST /api/data/push` - 推送数据（广播）
- `POST /api/data/push/{clientId}` - 推送给指定客户端
- `POST /api/data/push/batch` - 批量推送

### 3. 服务层 (Service)

#### SseService.java
SSE核心服务，包含：

**连接管理**:
- `createConnection()` - 创建SSE连接
- `closeConnection()` - 关闭连接
- `getConnectionCount()` - 获取连接数
- `getAllClientIds()` - 获取所有客户端ID

**数据推送**:
- `pushToClient()` - 推送给指定客户端
- `broadcast()` - 广播给所有客户端
- `sendHeartbeat()` - 发送心跳

**核心数据结构**:
```java
// 存储所有SSE连接
private final Map<String, SseEmitter> sseEmitters = new ConcurrentHashMap<>();
```

### 4. 数据模型 (Model)

#### PushData.java
推送数据模型：
```java
{
    id: "消息ID",
    type: "消息类型",
    message: "消息内容",
    data: {扩展数据},
    timestamp: "时间戳",
    priority: 优先级,
    targetClientId: "目标客户端ID"
}
```

#### ApiResponse.java
统一API响应格式：
```java
{
    code: 状态码,
    message: "消息",
    data: 数据,
    timestamp: 时间戳
}
```

### 5. 配置类 (Config)

#### CorsConfig.java
跨域配置：
- 允许所有域名访问
- 支持所有HTTP方法
- 允许携带Cookie

#### ScheduleConfig.java
定时任务配置：
- 每30秒发送心跳
- 保持SSE连接活跃

### 6. 配置文件

#### application.yml
应用配置：
- 服务端口: 8080
- 超时时间: 30分钟
- 线程池配置
- 日志级别

### 7. 前端文件

#### test.html
测试页面功能：
- SSE连接管理
- 消息实时显示
- 统计信息展示
- 测试数据发送

### 8. 客户端示例

#### JavaSseClient.java
Java SSE客户端示例：
- 连接SSE服务器
- 接收并解析SSE事件
- 错误处理和重连

### 9. 测试脚本

#### test-push.sh
自动化测试脚本：
- 推送各种类型消息
- 批量推送测试
- 查看连接统计

## 数据流程

### 1. SSE连接建立流程

```
客户端 → SseController.subscribe()
    ↓
SseService.createConnection()
    ↓
创建 SseEmitter 并存储
    ↓
发送连接成功消息
    ↓
返回 SseEmitter 给客户端
```

### 2. 数据推送流程

```
远程服务/API → DataPushController.pushData()
    ↓
解析 PushData
    ↓
判断单播还是广播
    ↓
SseService.pushToClient() 或 broadcast()
    ↓
遍历 sseEmitters Map
    ↓
SseEmitter.send() → 推送给客户端
```

### 3. 心跳流程

```
ScheduleConfig (定时30秒)
    ↓
SseService.sendHeartbeat()
    ↓
创建心跳消息
    ↓
broadcast() 广播给所有客户端
```

## 关键技术点

### 1. SseEmitter

Spring提供的SSE实现类：
```java
SseEmitter emitter = new SseEmitter(timeout);

// 发送消息
emitter.send(SseEmitter.event()
    .id("消息ID")
    .name("事件类型")
    .data("数据"));

// 完成连接
emitter.complete();
```

### 2. ConcurrentHashMap

线程安全的Map，存储客户端连接：
```java
private final Map<String, SseEmitter> sseEmitters = new ConcurrentHashMap<>();
```

### 3. 回调处理

```java
emitter.onCompletion(() -> {
    // 连接完成时清理
});

emitter.onTimeout(() -> {
    // 超时时处理
});

emitter.onError((ex) -> {
    // 错误时处理
});
```

### 4. 跨域支持

```java
@Bean
public CorsFilter corsFilter() {
    // 允许所有域名、所有方法、所有请求头
}
```

## 扩展点

### 1. 添加认证

在 `SseController` 中：
```java
@GetMapping("/subscribe")
public SseEmitter subscribe(@RequestHeader("Authorization") String token) {
    // 验证token
    validateToken(token);
    return sseService.createConnection(extractClientId(token));
}
```

### 2. 消息持久化

在 `DataPushController` 中：
```java
@PostMapping("/push")
public ApiResponse pushData(@RequestBody PushData pushData) {
    // 保存到数据库
    messageRepository.save(pushData);
    // 推送
    sseService.broadcast(pushData);
}
```

### 3. 集群支持

使用Redis存储连接信息：
```java
// 连接时
redisTemplate.opsForHash().put("sse:clients", clientId, instanceId);

// 推送时
String instanceId = redisTemplate.opsForHash().get("sse:clients", clientId);
if (instanceId.equals(currentInstanceId)) {
    // 本实例处理
} else {
    // 通过消息队列转发到其他实例
}
```

### 4. 消息队列集成

```java
@KafkaListener(topics = "sse-push")
public void handleMessage(PushData pushData) {
    sseService.broadcast(pushData);
}
```

## 日志说明

### 日志级别

- `INFO`: 正常业务日志
- `DEBUG`: 详细调试信息
- `WARN`: 警告信息
- `ERROR`: 错误信息

### 重要日志

```
新的SSE连接建立: clientId={}, 当前连接数={}
推送数据到客户端: clientId={}, type={}
广播数据完成: type={}, 成功={}, 失败={}
SSE连接超时: clientId={}
SSE连接错误: clientId={}, error={}
```

## 性能指标

### 默认配置

- 最大连接数: 200 (受线程池限制)
- 单连接超时: 30分钟
- 心跳间隔: 30秒
- 消息推送延迟: < 100ms

### 优化建议

1. **增加线程池**:
```yaml
server:
  tomcat:
    threads:
      max: 500
```

2. **使用异步处理**:
```java
@Async
public void broadcast(PushData data) {
    // 异步广播
}
```

3. **批量推送优化**:
```java
// 使用并行流
sseEmitters.entrySet().parallelStream()
    .forEach(entry -> pushToClient(entry.getKey(), data));
```

## 监控建议

### 1. 连接数监控

```bash
curl http://localhost:8080/api/sse/stats
```

### 2. JVM监控

使用 JMX 或 Spring Boot Actuator

### 3. 日志监控

使用 ELK 或其他日志分析工具

---

**更多信息请参考**:
- [README.md](README.md) - 完整使用文档
- [QUICK_START.md](QUICK_START.md) - 快速入门
