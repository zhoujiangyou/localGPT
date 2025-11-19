# 🚀 快速启动指南

## 1️⃣ 启动服务（3分钟）

### 方式1: Maven启动（推荐）

```bash
cd sse-service
mvn spring-boot:run
```

### 方式2: 打包运行

```bash
cd sse-service
mvn clean package
java -jar target/sse-service-1.0.0.jar
```

启动成功后会看到：

```
🚀 SSE推送服务已启动
📡 SSE连接地址: http://localhost:8080/api/sse/subscribe
📥 数据推送接口: http://localhost:8080/api/data/push
📊 测试页面: http://localhost:8080/test.html
```

## 2️⃣ 测试连接（1分钟）

### 方法1: 浏览器测试（最简单）

1. 打开浏览器
2. 访问: http://localhost:8080/test.html
3. 点击"连接SSE"按钮
4. 点击"发送测试数据"查看效果

### 方法2: 命令行测试

```bash
# 推送一条消息
curl -X POST http://localhost:8080/api/data/push \
  -H "Content-Type: application/json" \
  -d '{
    "type": "notification",
    "message": "Hello SSE!"
  }'

# 查看连接数
curl http://localhost:8080/api/sse/stats
```

### 方法3: 运行测试脚本

```bash
cd sse-service
./test-push.sh
```

## 3️⃣ 客户端接入（5分钟）

### JavaScript客户端（最常用）

```html
<script>
    // 1. 建立SSE连接
    const eventSource = new EventSource('http://localhost:8080/api/sse/subscribe');
    
    // 2. 监听连接成功
    eventSource.addEventListener('connected', (e) => {
        console.log('连接成功, clientId:', e.lastEventId);
    });
    
    // 3. 接收消息
    eventSource.onmessage = (e) => {
        const data = JSON.parse(e.data);
        console.log('收到消息:', data);
    };
    
    // 4. 监听特定类型消息
    eventSource.addEventListener('notification', (e) => {
        const data = JSON.parse(e.data);
        alert(data.message);
    });
    
    // 5. 错误处理
    eventSource.onerror = (e) => {
        console.error('连接错误');
    };
</script>
```

### Java客户端

```bash
cd sse-service
mvn test-compile
java -cp target/test-classes:target/classes com.example.sse.client.JavaSseClient
```

## 4️⃣ 推送数据（2分钟）

### 从其他服务推送数据

```bash
# 广播给所有客户端
curl -X POST http://localhost:8080/api/data/push \
  -H "Content-Type: application/json" \
  -d '{
    "type": "notification",
    "message": "系统维护通知",
    "data": {
      "startTime": "2024-01-01 02:00:00",
      "duration": "2小时"
    }
  }'

# 推送给指定客户端
curl -X POST http://localhost:8080/api/data/push/client_001 \
  -H "Content-Type: application/json" \
  -d '{
    "type": "alert",
    "message": "您有新订单"
  }'
```

### Java代码推送

```java
import org.springframework.http.HttpEntity;
import org.springframework.http.HttpHeaders;
import org.springframework.http.MediaType;
import org.springframework.web.client.RestTemplate;

public class DataPusher {
    public void pushData() {
        RestTemplate restTemplate = new RestTemplate();
        
        Map<String, Object> data = new HashMap<>();
        data.put("type", "notification");
        data.put("message", "测试消息");
        
        HttpHeaders headers = new HttpHeaders();
        headers.setContentType(MediaType.APPLICATION_JSON);
        
        HttpEntity<Map<String, Object>> request = 
            new HttpEntity<>(data, headers);
        
        String result = restTemplate.postForObject(
            "http://localhost:8080/api/data/push",
            request,
            String.class
        );
        
        System.out.println("推送结果: " + result);
    }
}
```

## 5️⃣ 常用场景

### 场景1: 实时通知

```javascript
// 客户端代码
eventSource.addEventListener('notification', (e) => {
    const data = JSON.parse(e.data);
    showNotification(data.message);
});
```

```bash
# 服务端推送
curl -X POST http://localhost:8080/api/data/push \
  -H "Content-Type: application/json" \
  -d '{
    "type": "notification",
    "message": "您收到一条新消息",
    "data": {"from": "张三", "content": "你好"}
  }'
```

### 场景2: 实时数据监控

```javascript
// 客户端代码
eventSource.addEventListener('data', (e) => {
    const data = JSON.parse(e.data);
    updateChart(data.data);
});
```

```bash
# 服务端定时推送监控数据
curl -X POST http://localhost:8080/api/data/push \
  -H "Content-Type: application/json" \
  -d '{
    "type": "data",
    "message": "系统状态",
    "data": {
      "cpu": 45.2,
      "memory": 67.8,
      "disk": 82.1
    }
  }'
```

### 场景3: 系统告警

```javascript
// 客户端代码
eventSource.addEventListener('alert', (e) => {
    const data = JSON.parse(e.data);
    showAlert(data.message, 'error');
});
```

```bash
# 服务端推送告警
curl -X POST http://localhost:8080/api/data/push \
  -H "Content-Type: application/json" \
  -d '{
    "type": "alert",
    "message": "服务器CPU使用率超过90%",
    "data": {"cpu": 95.8, "server": "server-01"},
    "priority": 3
  }'
```

## 6️⃣ 监控和管理

### 查看连接状态

```bash
# 查看当前连接数和客户端列表
curl http://localhost:8080/api/sse/stats | jq '.'

# 实时监控连接数
watch -n 1 'curl -s http://localhost:8080/api/sse/stats | jq ".data.connectionCount"'
```

### 断开指定客户端

```bash
curl -X DELETE http://localhost:8080/api/sse/disconnect/client_001
```

### 发送心跳测试

```bash
curl -X POST http://localhost:8080/api/sse/heartbeat
```

## 🎯 核心API速查

| 接口 | 方法 | 说明 |
|------|------|------|
| `/api/sse/subscribe` | GET | 客户端订阅SSE |
| `/api/data/push` | POST | 推送数据（广播） |
| `/api/data/push/{clientId}` | POST | 推送给指定客户端 |
| `/api/data/push/batch` | POST | 批量推送 |
| `/api/sse/stats` | GET | 获取连接统计 |
| `/api/sse/disconnect/{clientId}` | DELETE | 断开客户端 |
| `/api/sse/heartbeat` | POST | 发送心跳 |

## 🔧 故障排除

### 问题1: 服务启动失败

```bash
# 检查端口占用
lsof -i :8080

# 或更换端口
mvn spring-boot:run -Dspring-boot.run.arguments=--server.port=8081
```

### 问题2: 连接立即断开

- 检查防火墙设置
- 确保客户端正确处理SSE事件
- 查看服务端日志

### 问题3: 推送消息客户端收不到

```bash
# 1. 检查客户端是否连接
curl http://localhost:8080/api/sse/stats

# 2. 查看服务端日志
tail -f logs/spring.log

# 3. 检查浏览器控制台
# F12 -> Console -> 查看错误信息
```

## 📚 下一步

- 📖 查看完整文档: [README.md](README.md)
- 🧪 运行测试脚本: `./test-push.sh`
- 🌐 在浏览器打开测试页面: http://localhost:8080/test.html
- 💻 查看Java客户端示例: `src/test/java/com/example/sse/client/JavaSseClient.java`

## 💡 提示

1. **生产环境建议**:
   - 添加认证机制
   - 配置HTTPS
   - 使用负载均衡
   - 引入消息队列

2. **性能优化**:
   - 调整线程池大小
   - 使用Redis存储连接
   - 实现集群部署

3. **监控告警**:
   - 集成Prometheus
   - 配置日志采集
   - 设置连接数告警

---

**准备好了吗？立即启动服务试试吧！**

```bash
cd sse-service && mvn spring-boot:run
```
