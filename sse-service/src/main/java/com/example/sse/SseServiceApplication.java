package com.example.sse;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.scheduling.annotation.EnableAsync;

/**
 * SSE推送服务主应用
 */
@SpringBootApplication
@EnableAsync
public class SseServiceApplication {

    public static void main(String[] args) {
        SpringApplication.run(SseServiceApplication.class, args);
        System.out.println("\n=================================================");
        System.out.println("🚀 SSE推送服务已启动");
        System.out.println("📡 SSE连接地址: http://localhost:8080/api/sse/subscribe");
        System.out.println("📥 数据推送接口: http://localhost:8080/api/data/push");
        System.out.println("📊 测试页面: http://localhost:8080/test.html");
        System.out.println("=================================================\n");
    }
}
