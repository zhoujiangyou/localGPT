package com.example.sse.controller;

import com.example.sse.model.ApiResponse;
import com.example.sse.service.SseService;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.http.MediaType;
import org.springframework.web.bind.annotation.*;
import org.springframework.web.servlet.mvc.method.annotation.SseEmitter;

import java.util.HashMap;
import java.util.Map;

/**
 * SSE连接控制器
 */
@Slf4j
@RestController
@RequestMapping("/api/sse")
@RequiredArgsConstructor
public class SseController {
    
    private final SseService sseService;
    
    /**
     * 客户端订阅SSE
     * 
     * @param clientId 客户端ID（可选）
     * @return SseEmitter
     */
    @GetMapping(value = "/subscribe", produces = MediaType.TEXT_EVENT_STREAM_VALUE)
    public SseEmitter subscribe(@RequestParam(required = false) String clientId) {
        log.info("收到SSE订阅请求: clientId={}", clientId);
        return sseService.createConnection(clientId);
    }
    
    /**
     * 关闭指定客户端连接
     * 
     * @param clientId 客户端ID
     * @return 响应
     */
    @DeleteMapping("/disconnect/{clientId}")
    public ApiResponse<String> disconnect(@PathVariable String clientId) {
        log.info("关闭SSE连接请求: clientId={}", clientId);
        sseService.closeConnection(clientId);
        return ApiResponse.success("连接已关闭");
    }
    
    /**
     * 获取连接统计信息
     * 
     * @return 统计信息
     */
    @GetMapping("/stats")
    public ApiResponse<Map<String, Object>> getStats() {
        Map<String, Object> stats = new HashMap<>();
        stats.put("connectionCount", sseService.getConnectionCount());
        stats.put("clientIds", sseService.getAllClientIds());
        stats.put("timestamp", System.currentTimeMillis());
        
        return ApiResponse.success(stats);
    }
    
    /**
     * 发送心跳（测试用）
     * 
     * @return 响应
     */
    @PostMapping("/heartbeat")
    public ApiResponse<String> sendHeartbeat() {
        sseService.sendHeartbeat();
        return ApiResponse.success("心跳已发送");
    }
}
