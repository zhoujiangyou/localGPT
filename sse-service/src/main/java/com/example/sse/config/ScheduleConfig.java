package com.example.sse.config;

import com.example.sse.service.SseService;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.context.annotation.Configuration;
import org.springframework.scheduling.annotation.EnableScheduling;
import org.springframework.scheduling.annotation.Scheduled;

/**
 * 定时任务配置
 */
@Slf4j
@Configuration
@EnableScheduling
@RequiredArgsConstructor
public class ScheduleConfig {
    
    private final SseService sseService;
    
    /**
     * 定时发送心跳（每30秒）
     */
    @Scheduled(fixedRate = 30000)
    public void sendHeartbeat() {
        int count = sseService.getConnectionCount();
        if (count > 0) {
            log.debug("发送心跳，当前连接数: {}", count);
            sseService.sendHeartbeat();
        }
    }
}
