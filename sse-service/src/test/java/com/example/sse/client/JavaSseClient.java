package com.example.sse.client;

import java.io.BufferedReader;
import java.io.InputStreamReader;
import java.net.HttpURLConnection;
import java.net.URL;
import java.nio.charset.StandardCharsets;

/**
 * Java SSE客户端示例
 */
public class JavaSseClient {
    
    private volatile boolean running = true;
    
    public void connect(String serverUrl, String clientId) {
        try {
            String url = serverUrl + "/api/sse/subscribe";
            if (clientId != null && !clientId.isEmpty()) {
                url += "?clientId=" + clientId;
            }
            
            System.out.println("连接到SSE服务器: " + url);
            
            HttpURLConnection connection = (HttpURLConnection) new URL(url).openConnection();
            connection.setRequestMethod("GET");
            connection.setRequestProperty("Accept", "text/event-stream");
            connection.setRequestProperty("Cache-Control", "no-cache");
            connection.setDoInput(true);
            
            int responseCode = connection.getResponseCode();
            if (responseCode == 200) {
                System.out.println("✅ SSE连接成功");
                
                BufferedReader reader = new BufferedReader(
                    new InputStreamReader(connection.getInputStream(), StandardCharsets.UTF_8));
                
                String line;
                StringBuilder eventData = new StringBuilder();
                String eventType = null;
                String eventId = null;
                
                while (running && (line = reader.readLine()) != null) {
                    if (line.isEmpty()) {
                        // 空行表示一个事件结束
                        if (eventData.length() > 0) {
                            handleEvent(eventType, eventId, eventData.toString());
                            eventData.setLength(0);
                            eventType = null;
                            eventId = null;
                        }
                    } else if (line.startsWith("event:")) {
                        eventType = line.substring(6).trim();
                    } else if (line.startsWith("id:")) {
                        eventId = line.substring(3).trim();
                    } else if (line.startsWith("data:")) {
                        if (eventData.length() > 0) {
                            eventData.append("\n");
                        }
                        eventData.append(line.substring(5).trim());
                    } else if (line.startsWith(":")) {
                        // 注释行，忽略
                    }
                }
                
                reader.close();
            } else {
                System.err.println("❌ 连接失败，响应码: " + responseCode);
            }
            
            connection.disconnect();
            
        } catch (Exception e) {
            System.err.println("❌ SSE连接异常: " + e.getMessage());
            e.printStackTrace();
        }
    }
    
    private void handleEvent(String type, String id, String data) {
        String eventType = type != null ? type : "message";
        System.out.println("\n========================================");
        System.out.println("📨 收到事件");
        System.out.println("类型: " + eventType);
        if (id != null) {
            System.out.println("ID: " + id);
        }
        System.out.println("数据: " + data);
        System.out.println("时间: " + java.time.LocalDateTime.now());
        System.out.println("========================================");
    }
    
    public void disconnect() {
        running = false;
        System.out.println("⏹️  断开SSE连接");
    }
    
    public static void main(String[] args) {
        String serverUrl = "http://localhost:8080";
        String clientId = "java_client_" + System.currentTimeMillis();
        
        // 支持命令行参数
        if (args.length > 0) {
            serverUrl = args[0];
        }
        if (args.length > 1) {
            clientId = args[1];
        }
        
        System.out.println("=".repeat(50));
        System.out.println("Java SSE客户端");
        System.out.println("服务器: " + serverUrl);
        System.out.println("客户端ID: " + clientId);
        System.out.println("=".repeat(50));
        
        JavaSseClient client = new JavaSseClient();
        
        // 添加关闭钩子
        Runtime.getRuntime().addShutdownHook(new Thread(() -> {
            System.out.println("\n程序退出，断开连接...");
            client.disconnect();
        }));
        
        // 连接SSE服务器
        client.connect(serverUrl, clientId);
    }
}
