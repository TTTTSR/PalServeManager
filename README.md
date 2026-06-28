# 🦊 幻兽帕鲁服务器管理面板 (Palworld Server Manager)

一个完整的幻兽帕鲁 (Palworld) 专用服务器 Web 管理面板，支持服务器监控、配置编辑、启停控制、自动/手动更新、定时重启等功能。

## ✨ 功能特性

- **📊 实时监控仪表盘** — 系统资源（CPU/内存/磁盘）和服务器进程监控，在线玩家统计
- **🎮 服务器控制** — 启动、停止、重启服务器，使用 Palworld 官方 REST API (RCON 已弃用)
- **⚙️ 配置编辑器** — 可视化编辑 `PalWorldSettings.ini`，支持所有游戏参数，按类别分组
- **🔄 更新管理** — 通过 SteamCMD 检查更新、手动更新、验证文件完整性
- **⏰ 定时任务** — Cron 表达式定时自动重启、自动检查并安装更新
- **📋 日志查看器** — 实时查看服务器日志，支持过滤和自动刷新
- **🔐 安全认证** — 基于 Token 的管理面板登录认证

## 🏗️ 技术架构

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
│   │   ├── process.go         # PalServer 进程管理
│   │   ├── steamcmd.go        # SteamCMD 封装
│   │   ├── restapi.go         # REST API 客户端 (官方替代 RCON)
│   │   ├── ini.go             # PalWorldSettings.ini 解析器
│   │   ├── monitor.go         # 系统/服务器监控
│   │   ├── process_signal.go  # Unix 进程信号
│   │   └── process_signal_windows.go
│   └── middleware/
│       └── auth.go            # 认证 & CORS 中间件
│   ├── frontend-dist/
│   │   └── index.html         # 内嵌前端单文件 (~48KB 原生 JS)
└── README.md
```

> **前端**：使用原生 HTML/CSS/JS 编写为单文件，通过 Go `embed` 直接嵌入二进制。无需 Node.js、npm 或任何前端构建工具。管理面板和 API 绑定在同一端口。

## 🚀 快速开始

### 前置要求

- **Linux 服务器**（推荐 Ubuntu 20.04+ / Debian 11+）
- **Go 1.21+**（如需从源码编译）
- **SteamCMD** 已安装
- **幻兽帕鲁服务器** 已通过 SteamCMD 安装

### 1. 安装 SteamCMD 和 Palworld 服务器

```bash
# 安装 SteamCMD (Ubuntu/Debian)
sudo add-apt-repository multiverse
sudo dpkg --add-architecture i386
sudo apt update
sudo apt install steamcmd

# 安装 Palworld 服务器
steamcmd +force_install_dir /opt/palworld +login anonymous +app_update 2394010 validate +quit

# 首次运行以生成配置文件
cd /opt/palworld
./PalServer.sh -port=8211 -players=32
# 等待几秒后 Ctrl+C 停止
```

### 2. 配置 Palworld 服务器

编辑生成的配置文件：
```bash
# 复制默认配置
cp /opt/palworld/DefaultPalWorldSettings.ini /opt/palworld/Pal/Saved/Config/LinuxServer/PalWorldSettings.ini

# 编辑配置，至少设置以下项：
# ServerName, AdminPassword, RESTAPIEnabled=True, RESTAPIPort=8212
```

### 3. 部署管理面板

#### 方式一：从源码编译

```bash
# 克隆项目
cd /opt
git clone <your-repo> palworld-manager
cd palworld-manager/backend

# 构建（前端已嵌入，无需额外构建步骤）
go mod tidy
go build -o palworld-manager .

# 首次运行（自动生成 config.json）
./palworld-manager
# Ctrl+C 停止，然后编辑 config.json
```

#### 方式二：下载预编译二进制

```bash
# 下载对应平台的二进制文件
# 将 palworld-manager 和 frontend-dist/ 放在同一目录
chmod +x palworld-manager
./palworld-manager
```

### 4. 配置 config.json

首次运行后，编辑生成的 `config.json`：

```json
{
  "host": "0.0.0.0",
  "port": 8080,
  "palServerDir": "/opt/palworld",
  "palServerBinary": "PalServer.sh",
  "steamCmdPath": "/usr/games/steamcmd",
  "palServerAppId": "2394010",
  "restApiHost": "127.0.0.1",
  "restApiPort": 8212,
  "restApiPassword": "你的Admin密码",
  "serverPort": 8211,
  "maxPlayers": 32,
  "extraArgs": "-useperfthreads -NoAsyncLoadingThread -UseMultithreadForDS",
  "publicIp": "",
  "usePublicLobby": false,
  "adminToken": "你的管理面板登录密码",
  "autoRestartEnabled": false,
  "autoRestartCron": "0 */6 * * *",
  "autoUpdateEnabled": false,
  "autoUpdateCron": "0 4 * * *"
}
```

### 5. 启动管理面板

```bash
./palworld-manager
# 访问 http://你的服务器IP:8080
```

### 6. 配置 Systemd 服务（推荐）

```bash
sudo tee /etc/systemd/system/palworld-manager.service << EOF
[Unit]
Description=Palworld Server Manager
After=network.target

[Service]
Type=simple
User=steam
WorkingDirectory=/opt/palworld-manager
ExecStart=/opt/palworld-manager/palworld-manager
Restart=on-failure
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable palworld-manager
sudo systemctl start palworld-manager
```

## 📖 使用指南

### 登录

1. 访问 `http://服务器IP:8080`
2. 输入 `config.json` 中设置的 `adminToken`

### 仪表盘

- 查看服务器状态（PID、运行时间、CPU/内存使用）
- 查看在线玩家数量
- 查看系统资源使用（CPU、内存、磁盘）

### 服务器控制

- **启动** — 使用预配置参数启动 PalServer
- **停止** — 通过 REST API 发送关机通知，然后优雅关闭（SIGTERM），30秒后强制杀死
- **重启** — 保存存档→广播通知→停止→启动
- **REST API 命令控制台** — 执行 Info/Metrics/Players/Announce/Save/Shutdown/Kick/Ban/Unban 等管理操作

### 配置编辑

- 按类别分组显示所有 `PalWorldSettings.ini` 参数
- 每个设置都有中文标签和说明
- 修改后高亮显示变更
- 保存需确认，修改后需重启服务器生效

### 更新管理

- **检查更新** — 对比本地 Build ID 和 Steam 最新 Build ID
- **更新服务器** — 运行 `steamcmd +app_update`
- **更新并验证** — 运行 `steamcmd +app_update validate`
- ⚠️ 更新前需手动停止服务器

### 定时任务

- **自动重启** — 设置 Cron 表达式定时重启服务器
  - 建议：`0 */6 * * *`（每 6 小时）
  - 重启流程：REST API Save → Announce 通知 → 停止 → 启动
- **自动更新** — 设置 Cron 表达式定时检查并安装更新
  - 建议：`0 4 * * *`（每天凌晨 4 点）
  - 更新流程：检查更新 → 如有更新则 Save → 停止 → SteamCMD 更新 → 启动

### 日志查看

- 显示最近的服务器日志
- 支持按行数调整（50-1000 行）
- 支持关键词过滤
- 5 秒自动刷新（可关闭）
- 自动高亮 Error/Warning 行

## 🔧 API 接口

所有 API 需要 `Authorization: Bearer <token>` 头。

### 服务器控制
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/server/status` | 获取服务器状态 |
| POST | `/api/server/start` | 启动服务器 |
| POST | `/api/server/stop?message=&delay=` | 停止服务器 |
| POST | `/api/server/restart` | 重启服务器 |
| POST | `/api/server/command` | 发送 REST API 命令 `{"action":"info",...}` |
| GET | `/api/server/players` | 获取在线玩家列表 |
| GET | `/api/server/info` | 获取服务器信息 |

### 配置
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/config` | 获取当前配置 |
| PUT | `/api/config` | 更新配置 `{"Key":"Value",...}` |
| POST | `/api/config/reset` | 重置为默认值 |
| GET | `/api/config/defaults` | 获取默认配置 |

### 更新
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/update/check` | 检查更新 |
| POST | `/api/update/run?validate=true` | 执行更新 |
| GET | `/api/update/status` | 获取更新状态 |

### 监控
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/monitor/stats` | 综合统计 |
| GET | `/api/monitor/system` | 系统资源 |
| GET | `/api/monitor/server` | 服务器资源 |

### 其他
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/logs?lines=100` | 获取日志 |
| GET | `/api/schedule` | 获取定时任务配置 |
| PUT | `/api/schedule` | 更新定时任务配置 |
| POST | `/api/login` | 登录 `{"token":"..."}` |
| GET | `/api/health` | 健康检查 |

## ⚠️ 注意事项

1. **内存泄漏**：幻兽帕鲁服务器存在内存泄漏问题，建议配置每 4-6 小时自动重启
2. **WorldOption.sav**：此文件会覆盖 `PalWorldSettings.ini` 中的部分世界设置，修改基地相关配置后可能需要删除此文件
3. **RCON 已弃用**：Palworld 官方已弃用 RCON，本面板使用 REST API (端口 8212)。请在 `PalWorldSettings.ini` 中设置 `RESTAPIEnabled=True`
4. **更新前停止服务器**：SteamCMD 更新需要服务器停止后才能执行
5. **端口**：确保防火墙开放 8211 (UDP)、27015 (UDP)、8212 (TCP, REST API) 和管理面板端口
6. **WorldOption.sav 覆盖**：`BaseCampMaxNum` 和 `BaseCampMaxNumInGuild` 等设置由 `WorldOption.sav` 控制，修改 INI 后可能需要删除该文件

## 📝 参考文档

- [Palworld 官方服务器文档](https://docs.palworldgame.com/getting-started/deploy-dedicated-server/)
- [Palworld 服务器参数配置](https://docs.palworldgame.com/0.6.8/settings-and-operation/arguments/)
- [PalWorldSettings.ini 配置参考](https://docs.palworldgame.com/0.4.15/settings-and-operation/configuration/)
- [Valve SteamCMD 文档](https://developer.valvesoftware.com/wiki/SteamCMD)
- [Palworld REST API 文档](https://docs.palworldgame.com/category/rest-api/)
- [Palworld REST API 参考](https://xgamingserver.com/docs/palworld/rest-api)

## 📄 许可

MIT License
