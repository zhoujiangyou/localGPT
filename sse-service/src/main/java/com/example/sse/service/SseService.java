package com.example.sse.service;

import com.example.sse.model.PushData;
import com.fasterxml.jackson.databind.ObjectMapper;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Service;
import org.springframework.web.servlet.mvc.method.annotation.SseEmitter;

import java.io.IOException;
import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.atomic.AtomicInteger;

/**
 * SSE推送服务
 */
@Slf4j
@Service
public class SseService {
    
    /**
     * 存储所有的SSE连接
     * Key: clientId, Value: SseEmitter
     */
    private final Map<String, SseEmitter> sseEmitters = new ConcurrentHashMap<>();
    
    /**
     * 客户端计数器
     */
    private final AtomicInteger clientCounter = new AtomicInteger(0);
    
    /**
     * JSON转换器
     */
    private final ObjectMapper objectMapper = new ObjectMapper();
    
    /**
     * SSE超时时间（毫秒）
     */
    private static final long SSE_TIMEOUT = 30 * 60 * 1000; // 30分钟
    
    /**
     * 创建SSE连接
     * 
     * @param clientId 客户端ID，如果为空则自动生成
     * @return SseEmitter
     */
    public SseEmitter createConnection(String clientId) {
        // 如果没有提供clientId，自动生成
        if (clientId == null || clientId.isEmpty()) {
            clientId = "client_" + clientCounter.incrementAndGet() + "_" + System.currentTimeMillis();
        }
        
        // 创建SseEmitter，设置超时时间
        SseEmitter emitter = new SseEmitter(SSE_TIMEOUT);
        
        final String finalClientId = clientId;
        
        // 设置完成回调
        emitter.onCompletion(() -> {
            log.info("SSE连接完成: clientId={}", finalClientId);
            sseEmitters.remove(finalClientId);
        });
        
        // 设置超时回调
        emitter.onTimeout(() -> {
            log.warn("SSE连接超时: clientId={}", finalClientId);
            sseEmitters.remove(finalClientId);
        });
        
        // 设置错误回调
        emitter.onError((ex) -> {
            log.error("SSE连接错误: clientId={}, error={}", finalClientId, ex.getMessage());
            sseEmitters.remove(finalClientId);
        });
        
        // 保存连接
        sseEmitters.put(finalClientId, emitter);
        
        log.info("新的SSE连接建立: clientId={}, 当前连接数={}", finalClientId, sseEmitters.size());
        
        // 发送连接成功消息
        try {
            emitter.send(SseEmitter.event()
                    .id(finalClientId)
                    .name("connected")
                    .data("连接成功，clientId: " + finalClientId));
        } catch (IOException e) {
            log.error("发送连接消息失败: {}", e.getMessage());
        }
        
        return emitter;
    }
    
    /**
     * 推送数据给指定客户端
     * 
     * @param clientId 客户端ID
     * @param data 数据
     * @return 是否成功
     */
    public boolean pushToClient(String clientId, PushData data) {
        SseEmitter emitter = sseEmitters.get(clientId);
        if (emitter == null) {
            log.warn("客户端不存在: clientId={}", clientId);
            return false;
        }
        
        try {
            String jsonData = objectMapper.writeValueAsString(data);
            emitter.send(SseEmitter.event()
                    .id(data.getId())
                    .name(data.getType())
                    .data(jsonData));
            
            log.info("推送数据到客户端: clientId={}, type={}", clientId, data.getType());
            return true;
        } catch (IOException e) {
            log.error("推送数据失败: clientId={}, error={}", clientId, e.getMessage());
            // 移除失败的连接
            sseEmitters.remove(clientId);
            return false;
        }
    }
    
    /**
     * 广播数据给所有客户端
     * 
     * @param data 数据
     * @return 成功推送的客户端数量
     */
    public int broadcast(PushData data) {
        int successCount = 0;
        int failCount = 0;
        
        for (Map.Entry<String, SseEmitter> entry : sseEmitters.entrySet()) {
            String clientId = entry.getKey();
            SseEmitter emitter = entry.getValue();
            
            try {
                String jsonData = objectMapper.writeValueAsString(data);
                emitter.send(SseEmitter.event()
                        .id(data.getId())
                        .name(data.getType())
                        .data(jsonData));
                successCount++;
            } catch (IOException e) {
                log.error("广播数据失败: clientId={}, error={}", clientId, e.getMessage());
                // 移除失败的连接
                sseEmitters.remove(clientId);
                failCount++;
            }
        }
        
        log.info("广播数据完成: type={}, 成功={}, 失败={}, 总连接数={}", 
                data.getType(), successCount, failCount, sseEmitters.size());
        
        return successCount;
    }
    
    /**
     * 关闭指定客户端连接
     * 
     * @param clientId 客户端ID
     */
    public void closeConnection(String clientId) {
        SseEmitter emitter = sseEmitters.remove(clientId);
        if (emitter != null) {
            emitter.complete();
            log.info("关闭SSE连接: clientId={}", clientId);
        }
    }
    
    /**
     * 获取当前连接数
     * 
     * @return 连接数
     */
    public int getConnectionCount() {
        return sseEmitters.size();
    }
    
    /**
     * 获取所有客户端ID
     * 
     * @return 客户端ID集合
     */
    public java.util.Set<String> getAllClientIds() {
        return sseEmitters.keySet();
    }
    
    /**
     * 发送心跳给所有客户端
     */
    public void sendHeartbeat() {
        PushData heartbeat = PushData.builder()
                .id("heartbeat_" + System.currentTimeMillis())
                .type("heartbeat")
                .message("heartbeat")
                .timestamp(java.time.LocalDateTime.now())
                .build();
        
        broadcast(heartbeat);
    }
}
