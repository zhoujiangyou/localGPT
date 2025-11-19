package com.example.sse.controller;

import com.example.sse.model.ApiResponse;
import com.example.sse.model.PushData;
import com.example.sse.service.SseService;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.validation.annotation.Validated;
import org.springframework.web.bind.annotation.*;

import java.time.LocalDateTime;
import java.util.UUID;

/**
 * 数据推送控制器 - 接收远程推送的数据
 */
@Slf4j
@RestController
@RequestMapping("/api/data")
@RequiredArgsConstructor
public class DataPushController {
    
    private final SseService sseService;
    
    /**
     * 接收远程推送的数据，并通过SSE下发给客户端
     * 
     * @param pushData 推送数据
     * @return 响应
     */
    @PostMapping("/push")
    public ApiResponse<String> pushData(@RequestBody @Validated PushData pushData) {
        log.info("收到数据推送请求: type={}, message={}", pushData.getType(), pushData.getMessage());
        
        // 设置ID和时间戳（如果没有）
        if (pushData.getId() == null || pushData.getId().isEmpty()) {
            pushData.setId(UUID.randomUUID().toString());
        }
        if (pushData.getTimestamp() == null) {
            pushData.setTimestamp(LocalDateTime.now());
        }
        
        // 判断是单播还是广播
        if (pushData.getTargetClientId() != null && !pushData.getTargetClientId().isEmpty()) {
            // 单播 - 推送给指定客户端
            boolean success = sseService.pushToClient(pushData.getTargetClientId(), pushData);
            if (success) {
                return ApiResponse.success("数据已推送到客户端: " + pushData.getTargetClientId());
            } else {
                return ApiResponse.error("推送失败，客户端不存在或连接已断开");
            }
        } else {
            // 广播 - 推送给所有客户端
            int count = sseService.broadcast(pushData);
            return ApiResponse.success("数据已广播", "成功推送到 " + count + " 个客户端");
        }
    }
    
    /**
     * 批量推送数据
     * 
     * @param pushDataList 推送数据列表
     * @return 响应
     */
    @PostMapping("/push/batch")
    public ApiResponse<String> pushBatchData(@RequestBody java.util.List<PushData> pushDataList) {
        log.info("收到批量数据推送请求: count={}", pushDataList.size());
        
        int successCount = 0;
        for (PushData pushData : pushDataList) {
            // 设置ID和时间戳（如果没有）
            if (pushData.getId() == null || pushData.getId().isEmpty()) {
                pushData.setId(UUID.randomUUID().toString());
            }
            if (pushData.getTimestamp() == null) {
                pushData.setTimestamp(LocalDateTime.now());
            }
            
            // 广播数据
            int count = sseService.broadcast(pushData);
            if (count > 0) {
                successCount++;
            }
        }
        
        return ApiResponse.success("批量推送完成", 
                String.format("成功推送 %d/%d 条数据", successCount, pushDataList.size()));
    }
    
    /**
     * 推送给指定客户端
     * 
     * @param clientId 客户端ID
     * @param pushData 推送数据
     * @return 响应
     */
    @PostMapping("/push/{clientId}")
    public ApiResponse<String> pushToClient(
            @PathVariable String clientId,
            @RequestBody @Validated PushData pushData) {
        
        log.info("收到指定客户端推送请求: clientId={}, type={}", clientId, pushData.getType());
        
        // 设置ID和时间戳（如果没有）
        if (pushData.getId() == null || pushData.getId().isEmpty()) {
            pushData.setId(UUID.randomUUID().toString());
        }
        if (pushData.getTimestamp() == null) {
            pushData.setTimestamp(LocalDateTime.now());
        }
        
        boolean success = sseService.pushToClient(clientId, pushData);
        if (success) {
            return ApiResponse.success("数据已推送到客户端: " + clientId);
        } else {
            return ApiResponse.error("推送失败，客户端不存在或连接已断开");
        }
    }
}
