#!/bin/bash
# vLLM 环境检查脚本

echo "=========================================="
echo "vLLM 部署环境检查"
echo "=========================================="
echo ""

# 检查Python
echo "1. 检查Python..."
if command -v python3 &> /dev/null; then
    PYTHON_VERSION=$(python3 --version)
    echo "   ✓ Python已安装: $PYTHON_VERSION"
else
    echo "   ✗ Python未安装，请先安装Python 3.8+"
fi
echo ""

# 检查Docker
echo "2. 检查Docker..."
if command -v docker &> /dev/null; then
    DOCKER_VERSION=$(docker --version)
    echo "   ✓ Docker已安装: $DOCKER_VERSION"
    
    # 检查Docker是否运行
    if docker info &> /dev/null; then
        echo "   ✓ Docker服务正在运行"
    else
        echo "   ✗ Docker服务未运行，请启动Docker"
    fi
else
    echo "   ✗ Docker未安装（可选，但推荐使用Docker方式）"
fi
echo ""

# 检查GPU（如果有NVIDIA GPU）
echo "3. 检查GPU..."
if command -v nvidia-smi &> /dev/null; then
    echo "   ✓ 检测到NVIDIA GPU:"
    nvidia-smi --query-gpu=name,memory.total --format=csv,noheader | sed 's/^/     /'
else
    echo "   ⚠ 未检测到NVIDIA GPU，将使用CPU模式（速度较慢）"
fi
echo ""

# 检查端口8000是否被占用
echo "4. 检查端口8000..."
if lsof -Pi :8000 -sTCP:LISTEN -t >/dev/null 2>&1 || netstat -an 2>/dev/null | grep -q ":8000.*LISTEN"; then
    echo "   ⚠ 端口8000已被占用，vLLM可能需要使用其他端口"
else
    echo "   ✓ 端口8000可用"
fi
echo ""

# 检查网络连接（测试HuggingFace）
echo "5. 检查网络连接..."
if curl -s --head https://huggingface.co | head -n 1 | grep -q "200 OK"; then
    echo "   ✓ 可以访问HuggingFace（下载模型需要）"
else
    echo "   ⚠ 无法访问HuggingFace，可能需要使用代理或镜像"
fi
echo ""

echo "=========================================="
echo "检查完成！"
echo "=========================================="
echo ""
echo "如果所有检查都通过，可以开始部署vLLM了！"
echo "请参考: forge/docs/vllm_deployment_guide.md"

