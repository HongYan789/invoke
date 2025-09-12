@echo off
title Dubbo Invoke Web UI
echo ============================================================
echo 🚀 Dubbo Invoke Web UI 启动中...
echo ============================================================
echo.

REM 检查dubbo-invoke.exe是否存在
if not exist "dubbo-invoke.exe" (
    echo ❌ 错误: 未找到 dubbo-invoke.exe 文件
    echo 请确保此批处理文件与 dubbo-invoke.exe 在同一目录下
    echo.
    pause
    exit /b 1
)

REM 启动Web UI
echo 📡 正在启动Web服务...
echo 💡 服务启动后将自动打开浏览器
echo ⚠️  请勿关闭此窗口以保持服务运行
echo.
start "" dubbo-invoke.exe web

REM 等待几秒让用户看到启动信息
timeout /t 5 /nobreak >nul

echo ============================================================
echo ✅ Dubbo Invoke Web UI 已启动
echo 🌐 浏览器应该已自动打开
echo 📱 如果未自动打开，请手动访问: http://localhost:8080
echo ============================================================
echo.
echo 按任意键关闭此窗口...
pause >nul