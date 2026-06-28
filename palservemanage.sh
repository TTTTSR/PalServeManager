#!/bin/bash
# palservemanage — 服务管理入口，仅供 palserve 内部调用
set -e

INSTALL_DIR="/opt/palworld-manager"
BINARY="${INSTALL_DIR}/palworld-manager-linux"
PID_FILE="${INSTALL_DIR}/palserve.pid"
LOG_DIR="${INSTALL_DIR}/logs"
MAIN_LOG="${LOG_DIR}/manager.log"
BINARY_URL="https://raw.githubusercontent.com/TTTTSR/PalServeManager/main/palworld-manager-linux"

do_start() {
    if [ -f "$PID_FILE" ]; then
        if kill -0 "$(cat "$PID_FILE")" 2>/dev/null; then
            exit 0
        fi
        rm -f "$PID_FILE"
    fi
    mkdir -p "$LOG_DIR"
    nohup "$BINARY" >> "$MAIN_LOG" 2>&1 &
    echo $! > "$PID_FILE"
}

do_stop() {
    if [ -f "$PID_FILE" ]; then
        local pid=$(cat "$PID_FILE")
        kill "$pid" 2>/dev/null || true
        local waited=0
        while kill -0 "$pid" 2>/dev/null && [ $waited -lt 10 ]; do
            sleep 1; waited=$((waited + 1))
        done
        kill -9 "$pid" 2>/dev/null || true
        rm -f "$PID_FILE"
    fi
}

do_restart() {
    do_stop
    sleep 1
    do_start
}

do_update() {
    curl -fsSL "$BINARY_URL" -o "${BINARY}.tmp"
    chmod +x "${BINARY}.tmp"
    mv "${BINARY}.tmp" "$BINARY"
}

case "${1:-}" in
    start)   do_start ;;
    stop)    do_stop ;;
    restart) do_restart ;;
    update)  do_update ;;
esac
