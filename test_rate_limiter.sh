#!/bin/bash

# 限流功能测试脚本
# 使用方法: ./test_rate_limiter.sh [API_URL] [请求数量]

API_URL="${1:-http://localhost:8080/api/biz/v1/user/test}"
REQUEST_COUNT="${2:-100}"

echo "=========================================="
echo "限流功能测试"
echo "=========================================="
echo "目标 URL: $API_URL"
echo "请求数量: $REQUEST_COUNT"
echo "=========================================="
echo ""

# 计数器
SUCCESS_COUNT=0
RATE_LIMITED_COUNT=0
ERROR_COUNT=0

echo "开始发送请求..."
echo ""

for i in $(seq 1 $REQUEST_COUNT); do
    # 发送请求并获取 HTTP 状态码
    HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$API_URL")
    
    # 根据状态码分类统计
    if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "201" ]; then
        SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
        echo -ne "\r请求 $i/$REQUEST_COUNT - 成功: $SUCCESS_COUNT | 限流: $RATE_LIMITED_COUNT | 错误: $ERROR_COUNT"
    elif [ "$HTTP_CODE" = "429" ]; then
        RATE_LIMITED_COUNT=$((RATE_LIMITED_COUNT + 1))
        echo -ne "\r请求 $i/$REQUEST_COUNT - 成功: $SUCCESS_COUNT | 限流: $RATE_LIMITED_COUNT | 错误: $ERROR_COUNT"
    else
        ERROR_COUNT=$((ERROR_COUNT + 1))
        echo -ne "\r请求 $i/$REQUEST_COUNT - 成功: $SUCCESS_COUNT | 限流: $RATE_LIMITED_COUNT | 错误: $ERROR_COUNT (状态码: $HTTP_CODE)"
    fi
done

echo ""
echo ""
echo "=========================================="
echo "测试结果汇总"
echo "=========================================="
echo "总请求数: $REQUEST_COUNT"
echo "成功: $SUCCESS_COUNT ($(awk "BEGIN {printf \"%.2f\", ($SUCCESS_COUNT/$REQUEST_COUNT)*100}")%)"
echo "被限流: $RATE_LIMITED_COUNT ($(awk "BEGIN {printf \"%.2f\", ($RATE_LIMITED_COUNT/$REQUEST_COUNT)*100}")%)"
echo "错误: $ERROR_COUNT ($(awk "BEGIN {printf \"%.2f\", ($ERROR_COUNT/$REQUEST_COUNT)*100}")%)"
echo "=========================================="
echo ""

# 判断限流是否生效
if [ $RATE_LIMITED_COUNT -gt 0 ]; then
    echo "✅ 限流功能正常工作！"
else
    echo "⚠️  未触发限流，可能原因："
    echo "   1. 限流阈值设置过高"
    echo "   2. 限流功能未启用"
    echo "   3. 请求数量不足以触发限流"
fi
echo ""


