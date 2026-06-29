# 幻兽帕鲁服务器管理面板 (Palworld Server Manager)

一个完整的幻兽帕鲁 (Palworld) 专用服务器 Web 管理面板，支持服务器监控、配置编辑、启停控制、自动/手动更新、定时重启等功能。

## 一键部署

**仅支持 Debian 系发行版**（Ubuntu、Debian、Mint 等）：

```bash
curl -fsSL https://raw.githubusercontent.com/TTTTSR/PalServeManager/main/installpalserve.sh | sudo bash
```

部署完成后启动：

```bash
palserve start -d
```

访问 `http://<服务器IP>:8080` 进入管理面板。

## 非 Debian 系发行版

需要手动安装 SteamCMD 后拉取脚本和程序：

```bash
# 1. 根据你的发行版安装 SteamCMD（参见 https://developer.valvesoftware.com/wiki/SteamCMD）
#    RHEL/CentOS/Fedora: yum install steamcmd
#    Arch: AUR makepkg 或 Valve 官方 tar.gz

# 2. 拉取管理脚本和二进制
sudo mkdir -p /opt/palworld-manager/logs
sudo curl -fsSL https://raw.githubusercontent.com/TTTTSR/PalServeManager/main/palservemanage.sh -o /opt/palworld-manager/palservemanage.sh
sudo curl -fsSL https://raw.githubusercontent.com/TTTTSR/PalServeManager/main/palworld-manager-linux -o /opt/palworld-manager/palworld-manager-linux
sudo chmod +x /opt/palworld-manager/palservemanage.sh /opt/palworld-manager/palworld-manager-linux
sudo ln -sf /opt/palworld-manager/palservemanage.sh /usr/local/bin/palserve

# 3. 创建 steam 用户并授予权限
sudo useradd -m steam
sudo chown -R steam:steam /opt/palworld-manager

# 4. 启动
sudo -u steam palserve start
```

## 功能特性

- 实时监控仪表盘 — 系统资源（CPU/内存/磁盘）和服务器进程监控
- 服务器控制 — 启动、停止、重启服务器，通过 Palworld 官方 REST API
- 配置编辑器 — 可视化编辑 PalWorldSettings.ini，支持所有参数
- 更新管理 — 通过 SteamCMD 检查更新、手动更新、验证文件
- 定时任务 — 定时自动重启、自动检查并安装更新
- 日志查看器 — 实时查看服务器日志，支持过滤和自动刷新

## 技术架构

```
palworldserve/
├── backend/                    # Go 后端 API 服务
│   ├── main.go                # 入口，路由，定时任务
│   ├── config/
│   │   └── config.go          # 应用配置管理
│   ├── handlers/
│   │   ├── server.go          # 服务器控制 API
│   │   ├── config.go          # 配置管理 API
│   │   ├── update.go          # 更新管理 API
│   │   ├── monitor.go         # 监控统计 API
│   │   ├── logs.go            # 日志 API
│   │   ├── schedule.go        # 定时任务 API
│   │   └── helpers.go         # 工具函数
│   ├── services/
│   │   ├── process.go         # PalServer 进程管理 + 状态机
│   │   ├── statemachine.go    # 状态转换表 + 后台轮询
│   │   ├── steamcmd.go        # SteamCMD 封装
│   │   ├── restapi.go         # REST API 客户端
│   │   ├── ini.go             # PalWorldSettings.ini 解析器
│   │   ├── monitor.go         # 系统/服务器监控
│   │   ├── logger.go          # 日志系统
│   │   ├── wshub.go           # WebSocket 推送
│   │   └── process_signal*.go # 进程信号处理
│   ├── middleware/
│   │   ├── auth.go            # 认证 & CORS 中间件
│   │   └── logging.go         # 请求日志中间件
│   └── frontend-dist/
│       └── index.html         # 内嵌前端单文件
├── palserve                   # 一键部署脚本
├── palservemanage.sh          # 服务管理脚本
├── installpalserve.sh         # Debian 系安装脚本
└── palworld-manager-linux     # 预编译 Linux 二进制
```

前端使用原生 HTML/CSS/JS 编写为单文件，通过 Go embed 直接嵌入二进制，无需 Node.js 或任何前端构建工具。

## 安装细则

### 配置 Palworld 服务器

编辑生成的配置文件：

```bash
# 首次运行 PalServer 会自动生成配置文件，或手动复制默认配置
cp /opt/palworld/DefaultPalWorldSettings.ini /opt/palworld/Pal/Saved/Config/LinuxServer/PalWorldSettings.ini
```

### Systemd 服务（可选）

```bash
sudo tee /etc/systemd/system/palworld-manager.service << EOF
[Unit]
Description=Palworld Server Manager
After=network.target

[Service]
Type=simple
User=steam
WorkingDirectory=/opt/palworld-manager
ExecStart=/opt/palworld-manager/palservemanage.sh start
Restart=on-failure
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable --now palworld-manager
```

## API 接口

所有 API 无需认证（类似 frp 面板风格）。

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/server/status` | 获取服务器状态 |
| POST | `/api/server/start` | 启动服务器 |
| POST | `/api/server/stop` | 停止服务器 |
| POST | `/api/server/restart` | 重启服务器 |
| POST | `/api/server/command` | 发送 REST API 命令 |
| GET | `/api/server/players` | 在线玩家列表 |
| GET | `/api/server/info` | 服务器信息 |
| POST | `/api/server/install` | 安装 Palworld 服务器 |
| GET | `/api/config` | 获取配置 |
| PUT | `/api/config` | 更新配置 |
| POST | `/api/config/reset` | 重置配置 |
| GET | `/api/config/defaults` | 默认配置 |
| GET | `/api/update/check` | 检查更新 |
| POST | `/api/update/run` | 执行更新 |
| GET | `/api/update/status` | 更新状态 |
| GET | `/api/monitor/stats` | 综合监控 |
| GET | `/api/monitor/system` | 系统资源 |
| GET | `/api/monitor/server` | 服务器资源 |
| GET | `/api/logs` | 服务器日志 |
| GET | `/api/schedule` | 定时任务配置 |
| PUT | `/api/schedule` | 更新定时任务 |
| GET | `/api/health` | 健康检查 |
| WS | `/api/ws` | WebSocket 状态推送 |

## 注意事项

1. 内存泄漏 — 幻兽帕鲁服务器存在内存泄漏问题，建议配置定时自动重启
2. RCON 已弃用 — Palworld 官方已弃用 RCON，本面板使用 REST API (端口 8212)
3. 更新前需停止服务器 — SteamCMD 更新需要服务器处于已停止状态
4. 防火墙端口 — 确保开放 8211 (UDP)、8212 (TCP, REST API) 和管理面板端口

## 许可

MIT License
