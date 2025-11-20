package com.example.sse.model;

import com.fasterxml.jackson.annotation.JsonFormat;
import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.time.LocalDateTime;
import java.util.Map;

/**
 * 推送数据模型
 */
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class PushData {
    
    /**
     * 数据ID
     */
    private String id;
    
    /**
     * 消息类型
     */
    private String type;
    
    /**
     * 消息内容
     */
    private String message;
    
    /**
     * 扩展数据
     */
    private Map<String, Object> data;
    
    /**
     * 时间戳
     */
    @JsonFormat(pattern = "yyyy-MM-dd HH:mm:ss")
    private LocalDateTime timestamp;
    
    /**
     * 优先级
     */
    private Integer priority;
    
    /**
     * 目标客户端ID（为空则广播给所有客户端）
     */
    private String targetClientId;
}
