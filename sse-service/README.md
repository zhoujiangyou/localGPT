# SSE推送服务

## 📖 项目简介

这是一个基于Spring Boot的SSE（Server-Sent Events）推送服务，可以接收远程推送的数据，并通过SSE方式实时下发给已连接的客户端。

### 核心功能

- ✅ 接收远程HTTP推送的数据
- ✅ 通过SSE实时推送给客户端
- ✅ 支持单播（指定客户端）和广播（所有客户端）
- ✅ 自动心跳保持连接
- ✅ 连接管理和状态监控
- ✅ 跨域支持（CORS）

## 🚀 快速开始

### 环境要求

- JDK 17+
- Maven 3.6+

### 启动服务

```bash
cd sse-service
mvn spring-boot:run
```

服务启动后：
- 服务地址: http://localhost:8080
- SSE订阅地址: http://localhost:8080/api/sse/subscribe
- 数据推送接口: http://localhost:8080/api/data/push
- 测试页面: http://localhost:8080/test.html

## 📡 API接口

### 1. SSE订阅接口（客户端使用）

**接口**: `GET /api/sse/subscribe`

**参数**:
- `clientId` (可选): 自定义客户端ID

**示例**:
```javascript
// JavaScript客户端
const eventSource = new EventSource('/api/sse/subscribe?clientId=client_001');

eventSource.addEventListener('connected', (e) => {
    console.log('连接成功:', e.data);
});

eventSource.onmessage = (e) => {
    console.log('收到消息:', e.data);
};
```

### 2. 数据推送接口（远程服务使用）

**接口**: `POST /api/data/push`

**请求体**:
```json
{
    "id": "msg_001",
    "type": "notification",
    "message": "这是一条通知消息",
    "data": {
        "key1": "value1",
        "key2": 123
    },
    "timestamp": "2024-01-01 12:00:00",
    "priority": 1,
    "targetClientId": null
}
```

**字段说明**:
- `id`: 消息ID（可选，系统会自动生成）
- `type`: 消息类型（如: notification、alert、data等）
- `message`: 消息内容
- `data`: 扩展数据（可选）
- `timestamp`: 时间戳（可选）
- `priority`: 优先级（可选）
- `targetClientId`: 目标客户端ID（可选，为空则广播给所有客户端）

**响应**:
```json
{
    "code": 200,
    "message": "数据已广播",
    "data": "成功推送到 3 个客户端",
    "timestamp": 1704096000000
}
```

### 3. 推送给指定客户端

**接口**: `POST /api/data/push/{clientId}`

**示例**:
```bash
curl -X POST http://localhost:8080/api/data/push/client_001 \
  -H "Content-Type: application/json" \
  -d '{
    "type": "notification",
    "message": "这是发给指定客户端的消息"
  }'
```

### 4. 批量推送

**接口**: `POST /api/data/push/batch`

**请求体**:
```json
[
    {
        "type": "notification",
        "message": "消息1"
    },
    {
        "type": "alert",
        "message": "消息2"
    }
]
```

### 5. 获取连接统计

**接口**: `GET /api/sse/stats`

**响应**:
```json
{
    "code": 200,
    "message": "success",
    "data": {
        "connectionCount": 5,
        "clientIds": ["client_001", "client_002", "..."],
        "timestamp": 1704096000000
    }
}
```

### 6. 断开指定客户端

**接口**: `DELETE /api/sse/disconnect/{clientId}`

## 💻 客户端示例

### JavaScript/HTML客户端

```html
<!DOCTYPE html>
<html>
<body>
    <script>
        const eventSource = new EventSource('/api/sse/subscribe');
        
        // 连接成功
        eventSource.addEventListener('connected', (e) => {
            console.log('连接成功, clientId:', e.lastEventId);
        });
        
        // 接收消息
        eventSource.onmessage = (e) => {
            const data = JSON.parse(e.data);
            console.log('收到消息:', data);
        };
        
        // 接收特定类型的消息
        eventSource.addEventListener('notification', (e) => {
            const data = JSON.parse(e.data);
            console.log('通知消息:', data.message);
        });
        
        eventSource.addEventListener('alert', (e) => {
            const data = JSON.parse(e.data);
            alert('告警: ' + data.message);
        });
        
        // 错误处理
        eventSource.onerror = (e) => {
            console.error('连接错误:', e);
        };
        
        // 页面关闭时断开连接
        window.addEventListener('beforeunload', () => {
            eventSource.close();
        });
    </script>
</body>
</html>
```

### Java客户端

```java
package com.example.client;

import java.io.BufferedReader;
import java.io.InputStreamReader;
import java.net.HttpURLConnection;
import java.net.URL;

public class SseClient {
    public void connect(String serverUrl) {
        try {
            URL url = new URL(serverUrl + "/api/sse/subscribe");
            HttpURLConnection conn = (HttpURLConnection) url.openConnection();
            conn.setRequestProperty("Accept", "text/event-stream");
            
            BufferedReader reader = new BufferedReader(
                new InputStreamReader(conn.getInputStream()));
            
            String line;
            while ((line = reader.readLine()) != null) {
                if (line.startsWith("data:")) {
                    String data = line.substring(5);
                    System.out.println("收到数据: " + data);
                }
            }
        } catch (Exception e) {
            e.printStackTrace();
        }
    }
}
```

### Python客户端

```python
import sseclient
import requests

def connect_sse(server_url):
    url = f"{server_url}/api/sse/subscribe"
    response = requests.get(url, stream=True)
    client = sseclient.SSEClient(response)
    
    for event in client.events():
        print(f"事件类型: {event.event}")
        print(f"数据: {event.data}")
        print(f"ID: {event.id}")
```

## 🔧 配置说明

### application.yml

```yaml
server:
  port: 8080                    # 服务端口

spring:
  mvc:
    async:
      request-timeout: 1800000  # 30分钟超时

logging:
  level:
    com.example.sse: DEBUG      # 日志级别
```

### SSE超时配置

在 `SseService.java` 中：

```java
private static final long SSE_TIMEOUT = 30 * 60 * 1000; // 30分钟
```

## 📊 架构说明

### 核心组件

1. **SseService**: SSE连接管理服务
   - 维护所有客户端连接
   - 提供单播和广播功能
   - 管理连接生命周期

2. **SseController**: SSE订阅控制器
   - 处理客户端订阅请求
   - 提供连接管理接口

3. **DataPushController**: 数据推送控制器
   - 接收远程推送的数据
   - 转发数据到SSE客户端

4. **ScheduleConfig**: 定时任务配置
   - 定时发送心跳（30秒一次）

### 数据流程

```
远程服务 → POST /api/data/push → DataPushController 
    ↓
SseService.broadcast() 或 pushToClient()
    ↓
SseEmitter → 客户端浏览器/应用
```

## 🧪 测试

### 1. 使用测试页面

1. 启动服务
2. 打开浏览器访问: http://localhost:8080/test.html
3. 点击"连接SSE"
4. 点击"发送测试数据"查看效果

### 2. 使用curl测试

**推送数据**:
```bash
curl -X POST http://localhost:8080/api/data/push \
  -H "Content-Type: application/json" \
  -d '{
    "type": "notification",
    "message": "测试消息",
    "data": {"test": "value"}
  }'
```

**查看连接数**:
```bash
curl http://localhost:8080/api/sse/stats
```

### 3. 使用Java客户端测试

```bash
cd sse-service
mvn test-compile
java -cp target/test-classes:target/classes com.example.sse.client.JavaSseClient
```

## 🔒 安全建议

### 生产环境配置

1. **添加认证**:
```java
@GetMapping("/subscribe")
public SseEmitter subscribe(@RequestHeader("Authorization") String token) {
    // 验证token
    if (!validateToken(token)) {
        throw new UnauthorizedException();
    }
    return sseService.createConnection(extractClientId(token));
}
```

2. **限流**:
```java
@RateLimiter(name = "sse")
@GetMapping("/subscribe")
public SseEmitter subscribe() {
    // ...
}
```

3. **HTTPS**:
```yaml
server:
  ssl:
    enabled: true
    key-store: classpath:keystore.p12
    key-store-password: password
```

## 📈 性能优化

### 1. 连接池配置

```yaml
server:
  tomcat:
    threads:
      max: 200
      min-spare: 10
```

### 2. 消息队列

对于高并发场景，考虑引入消息队列：

```
远程服务 → Kafka/RabbitMQ → SSE服务 → 客户端
```

### 3. 负载均衡

使用Redis存储连接信息，实现多实例部署：

```java
// 将连接信息存储到Redis
redisTemplate.opsForValue().set("client:" + clientId, instanceId);
```

## 🐛 常见问题

### Q1: 连接经常断开？

**A**: 增加超时时间或添加心跳机制。已内置30秒心跳。

### Q2: 推送消息失败？

**A**: 检查客户端是否在线：
```bash
curl http://localhost:8080/api/sse/stats
```

### Q3: 如何处理大量客户端？

**A**: 
- 使用集群部署
- 引入消息队列
- 使用WebSocket替代SSE（双向通信场景）

## 📚 相关资源

- [SSE规范](https://html.spec.whatwg.org/multipage/server-sent-events.html)
- [Spring Boot SSE文档](https://docs.spring.io/spring-framework/docs/current/reference/html/web.html#mvc-ann-async-sse)
- [EventSource API](https://developer.mozilla.org/zh-CN/docs/Web/API/EventSource)

## 📝 TODO

- [ ] 添加Redis支持实现集群部署
- [ ] 添加消息持久化
- [ ] 添加消息确认机制
- [ ] 添加消息重试机制
- [ ] 添加监控和告警
- [ ] 添加WebSocket支持

## 📄 License

MIT License
