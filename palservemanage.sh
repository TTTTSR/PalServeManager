#!/bin/bash
# palservemanage — 服务管理入口，所有操作以 steam 用户执行
set -e

SERVICE_USER="steam"

# 若非 steam 用户，切换后重新执行
if [ "$(whoami)" != "$SERVICE_USER" ]; then
    exec sudo -u "$SERVICE_USER" env PATH="/usr/local/bin:/usr/bin:/bin" bash "$0" "$@"
fi

INSTALL_DIR="/opt/palworld-manager"
BINARY="${INSTALL_DIR}/palworld-manager-linux"
PID_FILE="${INSTALL_DIR}/palserve.pid"
LOG_DIR="${INSTALL_DIR}/logs"
MAIN_LOG="${LOG_DIR}/manager.log"
BINARY_URL="https://raw.githubusercontent.com/TTTTSR/PalServeManager/main/palworld-manager-linux"

do_start() {
    if [ -f "$PID_FILE" ] && kill -0 "$(cat "$PID_FILE")" 2>/dev/null; then
        echo "管理面板已在运行 (PID: $(cat "$PID_FILE"))"
        exit 0
    fi
    rm -f "$PID_FILE"
    mkdir -p "$LOG_DIR"
    nohup "$BINARY" >> "$MAIN_LOG" 2>&1 &
    local pid=$!
    echo $pid > "$PID_FILE"
    sleep 1
    if kill -0 "$pid" 2>/dev/null; then
        echo "管理面板已启动 (PID: $pid)"
    else
        echo "启动失败，查看日志: $MAIN_LOG"
        rm -f "$PID_FILE"
        exit 1
    fi
}

do_stop() {
    if [ ! -f "$PID_FILE" ]; then
        echo "管理面板未在运行"
        exit 0
    fi
    local pid=$(cat "$PID_FILE")
    if ! kill -0 "$pid" 2>/dev/null; then
        echo "管理面板未在运行 (PID 文件过期)"
        rm -f "$PID_FILE"
        exit 0
    fi
    kill "$pid" 2>/dev/null || true
    local waited=0
    while kill -0 "$pid" 2>/dev/null && [ $waited -lt 10 ]; do
        sleep 1; waited=$((waited + 1))
    done
    if kill -0 "$pid" 2>/dev/null; then
        kill -9 "$pid" 2>/dev/null || true
        echo "管理面板已强制停止 (PID: $pid)"
    else
        echo "管理面板已停止 (PID: $pid)"
    fi
    rm -f "$PID_FILE"
}

do_restart() {
    do_stop
    sleep 1
    do_start
}

do_update() {
    echo "正在下载更新..."
    curl -fsSL "$BINARY_URL" -o "${BINARY}.tmp" || {
        echo "下载失败"
        exit 1
    }
    chmod +x "${BINARY}.tmp"
    mv "${BINARY}.tmp" "$BINARY"
    echo "更新完成"
}

case "${1:-}" in
    start)   do_start ;;
    stop)    do_stop ;;
    restart) do_restart ;;
    update)  do_update ;;
    *)       echo "用法: $0 {start|stop|restart|update}"; exit 1 ;;
esac
