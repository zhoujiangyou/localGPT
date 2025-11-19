#!/bin/bash

# SSE推送服务测试脚本

SERVER_URL="http://localhost:8080"

echo "======================================"
echo "SSE推送服务测试脚本"
echo "======================================"
echo ""

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 1. 测试服务健康状态
echo -e "${YELLOW}1. 测试服务健康状态${NC}"
curl -s "${SERVER_URL}/api/sse/stats" | jq '.' || echo "服务未启动或jq未安装"
echo ""

# 2. 推送广播消息
echo -e "${YELLOW}2. 推送广播消息${NC}"
curl -X POST "${SERVER_URL}/api/data/push" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "notification",
    "message": "这是一条广播通知消息",
    "data": {
      "level": "info",
      "source": "test-script",
      "timestamp": "'$(date +%s)'"
    },
    "priority": 1
  }' | jq '.'
echo ""

# 3. 推送告警消息
echo -e "${YELLOW}3. 推送告警消息${NC}"
curl -X POST "${SERVER_URL}/api/data/push" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "alert",
    "message": "⚠️ 系统告警：CPU使用率超过90%",
    "data": {
      "cpu": 92.5,
      "memory": 85.2,
      "disk": 78.9
    },
    "priority": 3
  }' | jq '.'
echo ""

# 4. 批量推送
echo -e "${YELLOW}4. 批量推送消息${NC}"
curl -X POST "${SERVER_URL}/api/data/push/batch" \
  -H "Content-Type: application/json" \
  -d '[
    {
      "type": "data",
      "message": "数据更新1",
      "data": {"value": 100}
    },
    {
      "type": "data",
      "message": "数据更新2",
      "data": {"value": 200}
    },
    {
      "type": "data",
      "message": "数据更新3",
      "data": {"value": 300}
    }
  ]' | jq '.'
echo ""

# 5. 查看连接统计
echo -e "${YELLOW}5. 查看连接统计${NC}"
curl -s "${SERVER_URL}/api/sse/stats" | jq '.'
echo ""

# 6. 发送心跳
echo -e "${YELLOW}6. 发送心跳测试${NC}"
curl -X POST "${SERVER_URL}/api/sse/heartbeat" | jq '.'
echo ""

echo -e "${GREEN}======================================"
echo "测试完成"
echo "======================================${NC}"
echo ""
echo "提示："
echo "1. 在浏览器中打开 ${SERVER_URL}/test.html 查看实时效果"
echo "2. 使用 'watch -n 1 curl -s ${SERVER_URL}/api/sse/stats' 监控连接数"
