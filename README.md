# OVH 控制台

OVH 独立服务器 / Eco / **VPS 2027（含 Local Zone）** 的抢购、监控与管理控制台。

实时检测 OVH 各数据中心库存。独立服务器按机房与硬件 addon 自动下单；VPS 2027 常规与 Local Zone（planCode 带 `.LZ`）按 Linux / Windows 库存轨监控，有货后走 `/order/cart/{id}/vps` 下单。已购机器支持重启、重装、IPMI、合同期等生命周期操作。支持**多 OVH 账户**，抢购 / 监控按账户隔离。

> 本仓库 fork 自 [gokele/ovh](https://github.com/gokele/ovh)，灵感来自 [coolci/OVH-BUY](https://github.com/coolci/OVH-BUY)。
> 许可证：**AGPL-3.0**。对外以网络服务形式提供时，须开放对应源码。

当前 VPS 在售线（以 OVH catalog 为准，[vps-in-stock.ovh](https://vps-in-stock.ovh/) 作对照）：

| 系列 | planCode | 说明 |
|---|---|---|
| VPS-1 2027 | `vps-2027-model1` | 仅 Linux |
| VPS-2 2027 | `vps-2027-model2` | Linux + Windows |
| VPS-2 LZ 2027 | `vps-2027-model2.LZ` | Local Zone，仅 Linux |
| VPS-3 2027 | `vps-2027-model3` | Linux + Windows |
| VPS-4 2027 | `vps-2027-model4` | Linux + Windows |

---

## 部署教程

下面按「能在一台干净机器上从零跑起来」写。生产推荐**方式 A（单二进制）**；改代码用方式 B。

### 0. 你需要准备什么

| 项目 | 要求 | 说明 |
|---|---|---|
| 操作系统 | Windows 10/11、Linux（x86_64）或 macOS | 本文以 Windows 和 Debian/Ubuntu 为例 |
| Git | 任意近期版本 | 拉代码 |
| Go | **1.22+**（`go.mod` 声明 1.25） | 首次 `go build` 若提示升级 toolchain，按提示执行或安装新版 Go |
| Node.js | **20 LTS 或 22 LTS** | 只在构建前端时需要；跑起来的单二进制不再依赖 Node |
| npm | 随 Node 安装 | 不要用太老的 16 |
| 出网 | 能访问 `api.us.ovhcloud.com` / `eu.api.ovh.com` / `ca.api.ovh.com` | 库存与下单都走官方 API |
| OVH API 密钥 | Application Key + Secret + Consumer Key | 见第 2 节 |
| Telegram Bot（可选） | Bot Token + Chat ID | VPS 监控启动要求通知可用 |

本机无需安装 SQLite。二进制自带驱动：有 C 编译器时用 `mattn/go-sqlite3`，`CGO_ENABLED=0` 时用纯 Go 的 `modernc.org/sqlite`。

### 1. 拉取代码

```powershell
# Windows PowerShell
git clone https://github.com/weandy/ovh.git
cd ovh
git checkout feat/vps-2027-localzone-monitor-purchase
```

```bash
# Linux / macOS
git clone https://github.com/weandy/ovh.git
cd ovh
git checkout feat/vps-2027-localzone-monitor-purchase
```

`feat/vps-2027-localzone-monitor-purchase` 是带 VPS 2027 / Local Zone 监控与 `/vps` 下单的分支。若已合并进 `main`，可直接用默认分支。

目录约定：

```
ovh/
├── server/     # Go 后端，默认端口 19998
├── web/        # Vite 前端，开发端口 19997
└── README.md   # 本文件
```

### 2. 申请 OVH API 凭据

1. 打开对应区域的密钥页（子公司决定区域，必须和以后订阅的 Zone 一致）：
   - 美国：https://api.us.ovhcloud.com/createApp/
   - 欧洲：https://eu.api.ovh.com/createApp/
   - 加拿大 / 亚太：https://ca.api.ovh.com/createApp/
2. 创建应用，得到 **Application Key**、**Application Secret**。
3. 再为该应用申请 **Consumer Key**，权限至少覆盖：
   - `GET /me`
   - `GET/POST /order/cart*`（下单）
   - `GET /vps*`、`GET /dedicated/server*`（库存与已购管理）
   - 若只要监控不下单，可以只给 GET。
4. 美国账户填 Zone = `US`；爱尔兰 / 多数欧洲站填 `IE` 或实际子公司代码（`FR` `DE` `GB` …）。
5. **US 账户不能用来下欧洲 Local Zone**（例如 `EU-WEST-LZ-AMS`）。订阅子公司必须等于账户 Zone。

凭据**不要写进 git**。启动后在网页里录入，落在本地 SQLite。

### 3. 写配置文件

在 `server/` 下复制样板：

```powershell
cd server
copy .env.example .env
```

```bash
cd server
cp .env.example .env
```

编辑 `server/.env`（进程**当前工作目录**下的 `.env` 才会被读到；生产请在放二进制的目录再放一份）：

```bash
# 前端登录口令，必须改。生成：openssl rand -base64 32
API_SECRET_KEY=请改成足够长的随机串

PORT=19998
LISTEN_HOST=
ENABLE_API_KEY_AUTH=true
GIN_MODE=release
DEBUG=false
```

| 变量 | 默认 | 含义 |
|---|---|---|
| `API_SECRET_KEY` | `123456` | 浏览器 AuthGate 要填的访问密码。不改等于裸奔 |
| `PORT` | `19998` | HTTP 端口 |
| `LISTEN_HOST` | 空 | 空 = 所有网卡（IPv4+IPv6）。只本机访问写 `127.0.0.1`；反代后面也可写 `127.0.0.1` |
| `ENABLE_API_KEY_AUTH` | `true` | `false` 时关闭 `/api` 鉴权，**仅本机调试** |
| `GIN_MODE` | `release` | `debug` 会打更啰嗦的路由日志 |
| `DEBUG` | `false` | `true` 打开 debug 日志 |
| `DATA_DIR` | `data` | SQLite 与运行时目录，相对**启动时的工作目录** |
| `CACHE_DIR` | `$DATA_DIR/cache` | 缓存 |
| `LOGS_DIR` | `$DATA_DIR/logs` | 日志 `app.log.json` |

Telegram **不必**写进 `.env`。进系统后到「设置」填 Bot Token 和 Chat ID。VPS 补货订阅在通知未配好时会拒绝创建。

### 4. 方式 A：单二进制（推荐生产）

思路：先把前端打进 `server/web/`，再用 `-tags ui` 把该目录嵌进 Go 二进制。部署机只需这一个文件 + `.env`。

#### 4.1 Windows

需要已安装 Go 与 Node，并确保 `go`、`npm` 在 PATH 里。

```powershell
cd C:\path\to\ovh\web
npm install

# 推荐直接走 Vite（会生成 TanStack 路由树）。若 npm run build 因 tsc 找不到 routeTree.gen.ts 失败，用下面这行
npx vite build

cd ..\server
# 可选：没有 gcc 时强制纯 Go SQLite
$env:CGO_ENABLED = "0"
go build -tags ui -trimpath -ldflags "-s -w" -o ovh-server.exe

# 和工作目录放在一起再启动，这样 .env 和 data/ 都在旁边
copy .env.example .env   # 若还没有
# 编辑 .env 改 API_SECRET_KEY
.\ovh-server.exe
```

浏览器打开 http://localhost:19998 。

没有 gcc 时务必 `CGO_ENABLED=0`，否则会去编 `mattn/go-sqlite3` 并报缺 C 编译器。

#### 4.2 Linux（Debian / Ubuntu）

```bash
sudo apt-get update
sudo apt-get install -y git ca-certificates

# Go：https://go.dev/dl/ 安装 1.22+，或用发行版包后再靠 toolchain 自动拉
# Node 20：
curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
sudo apt-get install -y nodejs

cd /opt
sudo git clone https://github.com/weandy/ovh.git
sudo chown -R "$USER:$USER" /opt/ovh
cd /opt/ovh
git checkout feat/vps-2027-localzone-monitor-purchase

cd web
npm install
npx vite build

cd ../server
cp -n .env.example .env
# 编辑 .env，至少改 API_SECRET_KEY
export CGO_ENABLED=0
go build -tags ui -trimpath -ldflags "-s -w" -o ovh-server

# 建议用独立目录跑，避免在源码树里堆 data/
mkdir -p /opt/ovh-run
cp ovh-server /opt/ovh-run/
cp .env /opt/ovh-run/
cd /opt/ovh-run
./ovh-server
```

浏览器打开 `http://服务器IP:19998`。云厂商安全组 / 本机防火墙放行 `19998/tcp`。

#### 4.3 交叉编译（在 Windows 上打 Linux 包）

```powershell
cd C:\path\to\ovh\web
npx vite build

cd ..\server
$env:CGO_ENABLED = "0"
$env:GOOS = "linux"
$env:GOARCH = "amd64"
go build -tags ui -trimpath -ldflags "-s -w" -o ovh-server
```

把 `ovh-server` 和 `.env` 拷到 Linux，在同一目录执行 `chmod +x ovh-server && ./ovh-server`。

#### 4.4 用 systemd 常驻（Linux）

`/etc/systemd/system/ovh-console.service`：

```ini
[Unit]
Description=OVH console (VPS 2027 monitor)
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=ovh
Group=ovh
WorkingDirectory=/opt/ovh-run
ExecStart=/opt/ovh-run/ovh-server
Restart=on-failure
RestartSec=5
# 环境也可全部写进 WorkingDirectory 下的 .env
NoNewPrivileges=true
PrivateTmp=true

[Install]
WantedBy=multi-user.target
```

```bash
sudo useradd --system --home /opt/ovh-run --shell /usr/sbin/nologin ovh
sudo chown -R ovh:ovh /opt/ovh-run
sudo systemctl daemon-reload
sudo systemctl enable --now ovh-console
sudo systemctl status ovh-console
journalctl -u ovh-console -f
```

#### 4.5 Windows 开机自启（简要）

任选其一：

- 任务计划程序：登录时启动 `ovh-server.exe`，起始于放 `.env` 的目录。
- [NSSM](https://nssm.cc/) 注册成服务，同样把「启动目录」设成 exe 所在目录。

不要双击运行后关掉那个控制台窗口——关窗口进程就停。

### 5. 方式 B：开发（前后端分开）

两个终端：

```powershell
# 终端 1
cd ovh\server
copy .env.example .env
go run .

# 终端 2
cd ovh\web
npm install
npm run dev
```

打开 **http://localhost:19997**（Vite）。`/api/*` 会反代到 `19998`。不要只开 19998，开发态二进制默认**不含**前端。

### 6. 可选：反向代理

只本机用可以跳过。要 HTTPS 或走 80/443，用 Caddy 或 Nginx 反代到 `127.0.0.1:19998`，并把 `LISTEN_HOST=127.0.0.1`。

Caddy 示例：

```
ovh.example.com {
    reverse_proxy 127.0.0.1:19998
}
```

Nginx 关键段：

```nginx
location / {
    proxy_pass http://127.0.0.1:19998;
    proxy_http_version 1.1;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
}
```

Telegram Webhook 若走公网，把 Bot 的 webhook 指到 `https://你的域名/api/telegram/webhook`（该路径不校验 API Key）。

### 7. 首次打开系统

1. 浏览器访问生产地址（`:19998`）或开发地址（`:19997`）。
2. **AuthGate**：输入 `API_SECRET_KEY`。对不上会一直拦在登录层。
3. **OvhCredsGate**：添加第一个 OVH 账户。
   - 名称：随便起（如 `美国主号`）
   - 子公司 / Zone：美国选 `US`，欧洲常见 `IE`
   - 填入 Application Key / Secret / Consumer Key
   - Endpoint 会按 Zone 自动变成 `ovh-us` / `ovh-eu` / `ovh-ca`
4. 后端会立刻请求 OVH `/me`。失败不入库，检查密钥区域是否和 Zone 一致。
5. 到「设置」配置 Telegram（VPS 补货必做）：
   - 找 [@BotFather](https://t.me/BotFather) 建 Bot，拿到 Token
   - 把 Bot 拉进你的群或先私聊一次；Chat ID 可用 `@userinfobot` 或类似工具查
   - 保存后应能发出测试消息

### 8. 配一条 VPS 2027 / Local Zone 监控

1. 左侧进入 **VPS 补货**。
2. 添加订阅：
   - 子公司选 `US` 或 `IE`（必须和将要下单的账户 Zone 相同）
   - 系列选「VPS 2027 常规」或「VPS 2027 Local Zone」
   - Local Zone 目前主要是 `vps-2027-model2.LZ`，Windows 会禁用
   - 机房可多选；不选 = 该子公司下该型号全部机房
   - 勾选 Linux / Windows（有货条件看对应 `linuxStatus` / `windowsStatus`）
   - 要自动下单：勾选、选账户、数量 1–20
3. 点启动监控。间隔下限 60 秒。
4. 补货时 Telegram 会推送；若开了自动下单，「抢购队列」里会出现带 **VPS** chip 的任务。
5. 下单默认**不自动扣款**。去 OVH 控制台支付生成的订单，逾期作废。

美国 Local Zone 机房只出现在 `US` 订阅里；欧洲 Local Zone（如阿姆斯特丹 `EU-WEST-LZ-AMS`）只出现在 `IE`/`FR` 等欧洲子公司里。

### 9. 升级

```bash
cd /opt/ovh
git fetch
git pull
cd web && npm install && npx vite build
cd ../server
export CGO_ENABLED=0
go build -tags ui -trimpath -ldflags "-s -w" -o ovh-server
cp ovh-server /opt/ovh-run/ovh-server
sudo systemctl restart ovh-console
```

SQLite 启动时会跑 `IF NOT EXISTS` 和缺列迁移，一般不用手工改库。升级前仍建议备份：

```bash
cp /opt/ovh-run/data/sniper.db /opt/ovh-run/data/sniper.db.bak-$(date +%Y%m%d)
```

Windows 把 `data\sniper.db` 拷走即可。库里有 OVH 密钥，备份文件当机密保管。

### 10. 卸载 / 换机

停进程后删除运行目录。敏感数据都在：

- `data/sniper.db` — 账户密钥、队列、订阅、历史
- `data/logs/app.log.json` — 运行日志（可能含 planCode / 机房）
- `.env` — 访问密码

换机：拷贝这三个（至少库和 `.env`），用同一架构的二进制启动即可。

### 11. 常见问题

| 现象 | 处理 |
|---|---|
| 打开页面一直要密码 | `.env` 的 `API_SECRET_KEY` 和输入不一致；改过后要重启进程 |
| `go build` 报 gcc / cgo | `set CGO_ENABLED=0`（PowerShell: `$env:CGO_ENABLED="0"`） |
| `npm run build` 报找不到 `routeTree.gen.ts` | 改用 `npx vite build`（Vite 插件会生成路由树） |
| 单二进制打开是纯 API / 没有页面 | 漏了 `-tags ui`，或没先 `npx vite build`，`server/web/index.html` 不存在 |
| 浏览器打不开 localhost | Windows 上 `localhost` 可能走 IPv6。保持 `LISTEN_HOST` 为空，或试 `http://127.0.0.1:19998` |
| 添加 OVH 账户失败 | 密钥区域和 Zone 不一致（US 密钥配了 IE），或 Consumer Key 权限不够 |
| 加不了 VPS 订阅 | 先配好 Telegram；自动下单必须选账户，且账户 Zone = 订阅子公司 |
| 监控到有货但队列失败 | 看「日志」和「抢购历史」。缺 `vps_datacenter` / storage addon 会 fail-fast 清购物车 |
| 下了单但机器没出来 | 本程序默认不代扣。去 OVH 订单页付款 |
| 端口被占 | 改 `PORT`，或关掉占用 19998 的旧进程 |
| 公网裸奔 | 必须改掉默认 `123456`，建议 `LISTEN_HOST=127.0.0.1` + 反代 + HTTPS |

健康检查（无需密钥）：

```bash
curl http://127.0.0.1:19998/health
```

---

## 技术栈

| 层 | 技术 |
|---|---|
| 前端 | Vite 5 + React 18 + TypeScript + TanStack Router + TanStack Query + shadcn-ui + Tailwind |
| 后端 | Go + Gin + 官方 [go-ovh](https://github.com/ovh/go-ovh) |
| 持久化 | SQLite（cgo / 纯 Go 双驱动，build tag 自动选） |
| 通知 | Telegram Bot |
| 部署 | `//go:embed` 单二进制，或开发时前后端分开 |

## 项目结构

```
.
├── server/   # Go 后端 (默认 :19998)
│   ├── main.go
│   ├── webembed_ui.go    # -tags ui 时 embed server/web
│   ├── webembed_noui.go  # 默认，纯 API
│   └── internal/
│       ├── purchase/     # Eco + VPS cart 下单
│       ├── vps/          # VPS catalog / 库存轨 / 监控循环
│       ├── monitor/      # 独立服务器补货
│       └── ...
└── web/      # 前端 (dev :19997)
```

后端模块说明见 [server/README.md](server/README.md)。VPS 设计见 [docs/superpowers/specs/2026-08-18-vps-2027-monitor-purchase-design.md](docs/superpowers/specs/2026-08-18-vps-2027-monitor-purchase-design.md)。

## 配置与鉴权（摘要）

- OVH 凭据走网页，不进 `.env`。
- 除 `/health`、`/api/health`、`/api/version*`、`/api/telegram/webhook` 外，`/api/*` 都要请求头 `X-API-Key`。
- 浏览器把密钥放 localStorage。

## 主要功能

### 多账户

设置页管理多个 Zone / 密钥。队列、历史、自动下单按 `account_id` 隔离。删账户会清关联队列 / 历史，监控订阅只清空自动下单账户、订阅本身保留。

### 独立服务器抢购

卡片库存灯、addon 分组选择、队列重试、fail-fast（选了 NVMe 绝不会静默落到 HDD）。下单走 `/order/cart/{id}/eco`。

### VPS 2027 / Local Zone

- 目录接口 `GET /api/vps-catalog?ovhSubsidiary=US` 动态出型号，不写死 2025。
- 库存用公开接口 `/v1/vps/order/rule/datacenter`，按 `linuxStatus` / `windowsStatus` 判定。
- 自动下单走 `/order/cart/{id}/vps`，补齐 os、磁盘、1 天备份；`autoPayWithPreferredPaymentMethod=false`。

### 已购管理

独立服务器与 VPS 控制页：开关机、重装（EU `reinstall` / US `rebuild`）、快照、DDoS、合同期等。

## 持久化

运行目录下 `data/sniper.db`。升级会自动加列（如 `product_kind`、`vps_spec`、VPS 自动下单字段）。日志在 `data/logs/app.log.json`。

## 端口

| 服务 | 端口 |
|---|---|
| 生产单二进制 / Go 后端 | **19998** |
| Vite 开发服务器 | 19997 |
| Telegram webhook | `/api/telegram/webhook` |
