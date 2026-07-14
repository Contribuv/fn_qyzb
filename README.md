# 千盈助播 (fn_qyzb)

Go + SQLite + WebSocket 房产直播助播提词系统。高性能原生二进制，多房间实时消息同步，支持桌面端提词器和 Web 端访问。

## 功能特性

### 核心功能
- **多房间实时助播提词**：支持多个助播室，主持人与助理实时消息同步
- **WebSocket 实时通信**：基于 gorilla/websocket 的低延迟消息推送
- **桌面提词器**：Windows 客户端，字体大小、透明度自由调节，窗口置顶
- **多用户体系**：管理员后台管理用户和房间
- **内置反向代理**：一键配置 HTTPS 公网访问，支持域名证书自动加载

### 管理后台
- 房间管理：创建、编辑、删除助播室
- 用户管理：用户增删改查，权限控制
- 系统设置：应用名称、版权信息配置
- API 管理：接口密钥配置

### 部署特性
- 飞牛 NAS 一键安装包（FPK 格式）
- 安装时设置管理员账号和监听端口
- 数据统一存放于 data/ 子目录
- 卸载可选保留/删除数据库
- 零外部依赖，Go 原生二进制
- 支持 x86_64 / ARM64 双架构

## 技术栈

| 层级 | 技术 |
|------|------|
| 后端 | Go 1.26 + Gin + GORM |
| 数据库 | SQLite (modernc.org/sqlite 纯 Go 驱动) |
| 实时通信 | gorilla/websocket |
| 前端 | 原生 HTML + CSS + JavaScript（服务端渲染） |
| 桌面端 | Python + PyQt5 |
| 反向代理 | Go 原生 net/http + TLS |
| 打包 | fnpack (FPK 格式) |

## 项目结构

```
fn_qyzb/
├── server-src/              # 服务端源码
│   ├── cmd/server/          # 主程序入口
│   ├── internal/
│   │   ├── handlers/        # HTTP 处理器
│   │   ├── middleware/      # 中间件
│   │   ├── models/          # 数据模型
│   │   ├── services/        # 业务逻辑层
│   │   ├── utils/           # 工具类（rproxy 管理、证书管理等）
│   │   └── websocket/       # WebSocket 核心逻辑
│   ├── rproxy/              # 反向代理独立程序
│   ├── templates/           # HTML 模板
│   ├── static/              # 静态资源
│   │   ├── css/
│   │   ├── js/
│   │   └── soft/            # 桌面端提词器 exe
│   └── assets.go            # 静态资源嵌入
├── desk_zb/                 # 桌面端提词器（独立项目）
│   └── main.py              # PyQt5 桌面客户端
├── package/                 # FPK 打包目录
│   ├── app/
│   │   ├── server/          # 服务端二进制
│   │   ├── rproxy/          # 反向代理二进制
│   │   └── ui/              # UI 资源
│   ├── cmd/                 # 安装/卸载脚本
│   ├── config/              # 配置文件
│   └── wizard/              # 安装向导
├── .github/workflows/       # GitHub Actions CI
└── manifest                 # 飞牛应用清单
```

## 快速开始

### 飞牛 NAS 安装
1. 下载 `fn_qyzb.fpk` 安装包
2. 在飞牛管理后台上传并安装
3. 按照向导设置管理员账号和端口
4. 访问 `http://NAS_IP:3008/admin` 进入管理后台

### 本地开发

#### 服务端
```bash
cd server-src
go run cmd/server/main.go
```

#### 桌面提词器
```bash
cd desk_zb
pip install -r requirements.txt
python main.py
```

### 编译打包

#### 服务端二进制
```bash
# Linux amd64
GOOS=linux GOARCH=amd64 go build -o fn_qyzb-server-amd64 ./cmd/server/

# Linux arm64
GOOS=linux GOARCH=arm64 go build -o fn_qyzb-server-arm64 ./cmd/server/
```

#### 反向代理
```bash
cd server-src/rproxy
GOOS=linux GOARCH=amd64 go build -o qyzb-rproxy-amd64 .
GOOS=linux GOARCH=arm64 go build -o qyzb-rproxy-arm64 .
```

#### FPK 安装包
```bash
fnpack build -d package
```

## 主要模块说明

### WebSocket 消息系统
- `hub.go`：连接集线器，管理所有客户端连接
- `client.go`：客户端连接封装，处理消息收发
- 支持消息广播、在线用户统计、房间隔离

### 反向代理 (rproxy)
- 独立 Go 程序，通过 HTTP API 控制启停
- 支持 TLS 证书热重载
- 实时日志输出
- 自动状态恢复（重启后探测已有进程）

### 桌面提词器
- 半透明置顶窗口，直播不遮挡
- 字号 12-60 可调
- 透明度 30%-100% 可调
- WebSocket 实时接收消息
- 支持多房间切换

## API 接口

### 公共接口
- `GET /api/public/settings` - 获取公共设置
- `GET /api/rooms` - 获取房间列表

### 用户接口
- `POST /room/login` - 房间登录
- `POST /room/register` - 用户注册
- `GET /api/user/info` - 获取用户信息
- `GET /api/room/:id/latest-messages` - 获取历史消息
- `GET /ws/chat` - WebSocket 连接

### 管理后台
- `POST /admin/login` - 管理员登录
- `GET /admin/rooms` - 房间管理
- `GET /admin/users` - 用户管理
- `GET /admin/settings` - 系统设置

### 网关（反向代理）
- `GET /gateway/` - 网关配置页面
- `GET /gateway/api/status` - 获取反代状态
- `POST /gateway/api/start` - 启动反向代理
- `POST /gateway/api/stop` - 停止反向代理
- `GET /gateway/api/logs` - 获取实时日志
- `GET /gateway/api/certs` - 获取可用证书

## 配置说明

### 环境变量
- `APP_PORT`：服务监听端口（默认 3008）
- `DATA_DIR`：数据目录
- `GATEWAY_PREFIX`：网关路径前缀
- `UPLOAD_DIR`：上传文件目录

### 反向代理配置
存储于 `data/rproxy-config.json`，包含域名、端口、证书路径、API 端口等信息。

## 支持平台

- Linux amd64 (x86_64)
- Linux arm64 (aarch64)
- Windows（桌面提词器）

## 许可证

MIT License

## 维护者

- 豪子 - [GitHub](https://github.com/Contribuv)

## 版本日志


### v1.0.5 (2026-07-14)
- 新增：网关页面访问限制，仅允许统一网关路径访问
- 修复：TCP 端口直连时前缀传递问题（通过 X-Gateway-Prefix 头）
- 修复：manifest 描述去除技术术语，改为更易懂的介绍

### v1.0.4 (2026-07-14)
- 新增：网关页面采用 macOS 风格 UI（侧边栏导航、亮色/暗色主题切换）
- 新增：反向代理配置、运行状态、实时日志单独 Tab 展示
- 新增：运行状态显示作者信息（联系反馈 微信：CQGGTF）
- 新增：证书获取规则优化（排除 fnOS 内置和 fnos.net 动态域名证书）
- 修复：HTTPS 默认端口改为 5558
- 修复：证书 API 返回格式与参考项目一致（包含 sans、expired、expires 等字段）
- 修复：微信 ID 点击可复制功能

### v1.0.3 (2026-07-09)
- 修复：同一端口同时支持 HTTP 和 HTTPS，HTTP 自动 301 跳转到 HTTPS
- 修复：并发请求下网关前缀互相覆盖导致 CSS/JS 路径错误（goroutine ID 隔离）
- 修复：反向代理 gzip 压缩数据可能损坏（复制独立副本）
- 修复：移动端 CSS/JS 偶发加载失败（增加 Cache-Control 头）
- 修复：反代重启后不会自动恢复（配置持久化 + 自动重启）

### v1.0.2 (2026-07-09)
- 修复：网关 404（socket 路径不一致，ARM 架构）
- 修复：登录按钮变形（admin.css 全局覆盖 .btn 样式）
- 修复：网关页面限制内网访问（去除 RequireLocalNetwork）
- 修复：安装向导密码不生效（initDefaultAdmin 每次启动对比更新）
- 修复：卸载向导"删除数据"不执行（uninstall_callback 补充删除逻辑）
- 修复：端口直连/反代访问强制 301 到 /app/fn_qyzb（区分 TCP/Unix Socket 来源）
- 新增：网关页面显示内网地址+端口超链接
- 新增：Session Secret 持久化（避免重启后 Cookie 失效）

### v1.0.1 (2026-07-08)
- 修复：统一网关 404（重定向路径缺少前缀）
- 修复：模板路径硬编码导致网关模式下 CSS/JS/链接 404
- 修复：模板嵌套 {{.ID}} 语法错误导致 500
- 修复：头像/WebSocket 路径在网关模式下 404
- 新增：GitHub Actions 工作流自动编译二进制

### v1.0.0 (2026-07-07)
- Go 重构版本，原生二进制高性能
- SQLite 本地数据库，零外部依赖
- 内置反向代理，支持公网域名 HTTPS 访问
- 支持 x86_64 / ARM64 双架构
- 飞牛 NAS FPK 安装包
