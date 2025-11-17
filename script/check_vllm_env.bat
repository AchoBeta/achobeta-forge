@echo off
REM vLLM 环境检查脚本 (Windows)

echo ==========================================
echo vLLM 部署环境检查
echo ==========================================
echo.

REM 检查Python
echo 1. 检查Python...
python --version >nul 2>&1
if %errorlevel% == 0 (
    for /f "tokens=*" %%i in ('python --version') do echo    ✓ Python已安装: %%i
) else (
    echo    ✗ Python未安装，请先安装Python 3.8+
)
echo.

REM 检查Docker
echo 2. 检查Docker...
docker --version >nul 2>&1
if %errorlevel% == 0 (
    for /f "tokens=*" %%i in ('docker --version') do echo    ✓ Docker已安装: %%i
    
    REM 检查Docker是否运行
    docker info >nul 2>&1
    if %errorlevel% == 0 (
        echo    ✓ Docker服务正在运行
    ) else (
        echo    ✗ Docker服务未运行，请启动Docker Desktop
    )
) else (
    echo    ✗ Docker未安装（可选，但推荐使用Docker方式）
)
echo.

REM 检查GPU（如果有NVIDIA GPU）
echo 3. 检查GPU...
nvidia-smi >nul 2>&1
if %errorlevel% == 0 (
    echo    ✓ 检测到NVIDIA GPU:
    nvidia-smi --query-gpu=name,memory.total --format=csv,noheader
) else (
    echo    ⚠ 未检测到NVIDIA GPU，将使用CPU模式（速度较慢）
)
echo.

REM 检查端口8000是否被占用
echo 4. 检查端口8000...
netstat -an | findstr ":8000" >nul 2>&1
if %errorlevel% == 0 (
    echo    ⚠ 端口8000已被占用，vLLM可能需要使用其他端口
) else (
    echo    ✓ 端口8000可用
)
echo.

echo ==========================================
echo 检查完成！
echo ==========================================
echo.
echo 如果所有检查都通过，可以开始部署vLLM了！
echo 请参考: forge\docs\vllm_deployment_guide.md
pause

