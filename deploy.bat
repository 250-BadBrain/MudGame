@echo off
chcp 65001 >nul
setlocal enabledelayedexpansion

REM ====== 配置区 ======
set "SSH_HOST=oracle"
set "REMOTE_ROOT=/home/ubuntu/MudGame"
set "SERVICE_NAME=mud"
set "EXE_NAME=mud"
REM ====================

set "LOCAL_ROOT=%~dp0"
set "LOCAL_DATA=%LOCAL_ROOT%data"
set "LOCAL_SERVER=%LOCAL_ROOT%MudGame-server"
set "LOCAL_EXE=%LOCAL_SERVER%\%EXE_NAME%"
set "LOCAL_RESOURCES=%LOCAL_SERVER%\resources"

set "REMOTE_SERVER=%REMOTE_ROOT%/MudGame-server"
set "REMOTE_EXE=%REMOTE_SERVER%/%EXE_NAME%"

echo.
echo ========== 检查本地文件 ==========

if not exist "%LOCAL_DATA%" (
    echo [ERROR] 找不到本地 data 文件夹：%LOCAL_DATA%
    exit /b 1
)

if not exist "%LOCAL_SERVER%" (
    echo [ERROR] 找不到本地 MudGame-server 文件夹：%LOCAL_SERVER%
    exit /b 1
)

if not exist "%LOCAL_EXE%" (
    echo [ERROR] 找不到本地可执行文件：%LOCAL_EXE%
    exit /b 1
)

if not exist "%LOCAL_RESOURCES%" (
    echo [ERROR] 找不到本地 resources 文件夹：%LOCAL_RESOURCES%
    exit /b 1
)

echo [OK] 本地文件检查通过


echo.
echo ========== 测试 SSH 连接 ==========

ssh %SSH_HOST% "echo SSH connected"

if errorlevel 1 (
    echo [ERROR] SSH 连接失败，请先确认 ssh %SSH_HOST% 能正常登录
    exit /b 1
)


echo.
echo ========== 停止远程服务 ==========

if "%SERVICE_NAME%"=="" (
    echo [INFO] 未设置 SERVICE_NAME，尝试停止旧进程
    ssh %SSH_HOST% "pkill -f '%REMOTE_EXE%' || true"
) else (
    echo [INFO] 停止 systemd 服务：%SERVICE_NAME%
    ssh %SSH_HOST% "sudo systemctl stop %SERVICE_NAME% || true"
)

if errorlevel 1 (
    echo [ERROR] 远程停止服务失败
    exit /b 1
)


echo.
echo ========== 删除远程旧文件 ==========

ssh %SSH_HOST% "rm -rf '%REMOTE_ROOT%/data' '%REMOTE_ROOT%/MudGame-server' && mkdir -p '%REMOTE_ROOT%' '%REMOTE_SERVER%'"

if errorlevel 1 (
    echo [ERROR] 删除远程旧文件失败
    exit /b 1
)


echo.
echo ========== 上传 data 文件夹 ==========

scp -r "%LOCAL_DATA%" "%SSH_HOST%:%REMOTE_ROOT%/"

if errorlevel 1 (
    echo [ERROR] 上传 data 失败
    exit /b 1
)


echo.
echo ========== 上传 mud 可执行文件 ==========

scp "%LOCAL_EXE%" "%SSH_HOST%:%REMOTE_SERVER%/%EXE_NAME%"

if errorlevel 1 (
    echo [ERROR] 上传 mud 可执行文件失败
    exit /b 1
)


echo.
echo ========== 上传 resources 文件夹 ==========

scp -r "%LOCAL_RESOURCES%" "%SSH_HOST%:%REMOTE_SERVER%/"

if errorlevel 1 (
    echo [ERROR] 上传 resources 失败
    exit /b 1
)


echo.
echo ========== 修改远程权限 ==========

ssh %SSH_HOST% "chmod +x '%REMOTE_EXE%'"

if errorlevel 1 (
    echo [ERROR] chmod 失败
    exit /b 1
)


echo.
echo ========== 启动远程服务 ==========

if "%SERVICE_NAME%"=="" (
    echo [INFO] 使用 nohup 启动
    ssh %SSH_HOST% "cd '%REMOTE_SERVER%' && nohup './%EXE_NAME%' > '%REMOTE_ROOT%/mud.log' 2>&1 < /dev/null & echo \$! > '%REMOTE_ROOT%/mud.pid'"
) else (
    echo [INFO] 启动 systemd 服务：%SERVICE_NAME%
    ssh %SSH_HOST% "sudo systemctl start %SERVICE_NAME% && sudo systemctl status %SERVICE_NAME% --no-pager -l"
)

if errorlevel 1 (
    echo [ERROR] 启动远程服务失败
    exit /b 1
)


echo.
echo ========== 部署完成 ==========
echo 远程目录：%REMOTE_ROOT%
echo 可执行文件：%REMOTE_EXE%

if "%SERVICE_NAME%"=="" (
    echo.
    echo 查看日志：
    echo ssh %SSH_HOST% "tail -f %REMOTE_ROOT%/mud.log"
) else (
    echo.
    echo 查看日志：
    echo ssh %SSH_HOST% "journalctl -u %SERVICE_NAME% -f"
)

echo.
pause