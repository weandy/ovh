# VPS 2027 全自动监控与抢购

日期：2026-08-18  
仓库：https://github.com/weandy/ovh（fork 自 [gokele/ovh](https://github.com/gokele/ovh) v0.0.8）  
上游：`upstream` = `https://github.com/gokele/ovh.git`  
状态：待确认后写实现计划

## 1. 背景

本仓库是 gokele/ovh 的 fork。原项目已经能抢 OVH Eco / 独立服务器，也能轮询 VPS 库存并推 Telegram。VPS 这条线停在 2025 写死型号，而且**不会下单**。

我们要在原有控制台上补齐：针对当前在售的 **VPS 2027 常规** 和 **VPS 2027 LZ**，做全自动监控 + 抢购。独立服务器 / Eco 路径保持不动。

在售系列以 [vps-in-stock.ovh](https://vps-in-stock.ovh/) 为产品线对照，不以它为运行时数据源。2026-08-17 该站只跟踪这 5 个 planCode：

| 系列 | planCode | 镜像 |
|---|---|---|
| VPS-1 2027 | `vps-2027-model1` | 仅 Linux（catalog 无 Windows addon） |
| VPS-2 2027 | `vps-2027-model2` | Linux + Windows |
| VPS-2 LZ 2027 | `vps-2027-model2.LZ` | 仅 Linux |
| VPS-3 2027 | `vps-2027-model3` | Linux + Windows |
| VPS-4 2027 | `vps-2027-model4` | Linux + Windows |

catalog 里还有 `vps-2025-model1.LZ` 和大量 2025 / Value / Essential SKU。它们不是当前主售系列，实现上走「其他」分组即可，不单独做第一优先级。

## 2. 目标

操作员用已有 OVH 多账户，订阅 2027 常规或 2027 LZ 的某个型号 + 子公司 + 机房 + Linux/Windows，系统按间隔拉取公开库存接口。某机房从无货翻到有货时：

1. 发 Telegram。
2. 若订阅开了自动下单并绑定了账户，按数量把 VPS 任务推进现有抢购队列。
3. 队列处理器走 **VPS cart**（`/order/cart/{id}/vps`），不是现有 Eco cart。
4. 结账成功写入抢购历史，失败清购物车并记录原因。

成功标准：

- 订阅页能从 catalog 选出 2027 常规 4 档 + 2027 LZ，不再写死 `vps-2025-model1~6`。
- 同一 `planCode` 在 US / IE 看到的机房集合不同，且与 OVH catalog / rule 接口一致。
- Linux 有货、Windows 无货时，只盯 Linux 的订阅会触发；只盯 Windows 的不会。
- 自动下单生成的队列项 `productKind=vps`，checkout 走 `/vps`，必选 addon（os / storage / automatedBackup）自动补齐。
- Eco 独立服务器监控、队列、下单行为与 fork 前一致。

非目标：

- 不爬 [vps-in-stock.ovh](https://vps-in-stock.ovh/)，不把它当库存源。
- 不改成多租户 SaaS，不做客户计费。
- 默认不自动扣款。`autoPayWithPreferredPaymentMethod` 保持 `false`，与现有 Eco 下单一致。订单生成后由账户在 OVH 支付，有效期按 OVH 规则。
- 不重写已购 VPS 控制中心。
- 第一期不做 cPanel / Plesk / 额外磁盘 / 快照选配；只下必选 addon 的默认值。

## 3. 现有代码缺口

对照 gokele v0.0.8：

| 位置 | 现状 | 问题 |
|---|---|---|
| `web/src/routes/vps-monitor.tsx` | 写死 6 个 `vps-2025-model*` | 2027 / LZ 选不了 |
| `server/internal/vps/vps.go` | 已调 `/v1/vps/order/rule/datacenter` | 只看总字段 `status`，忽略 `linuxStatus` / `windowsStatus` |
| `types.VPSSubscription` | 有 `AutoOrderAccountID` | 没有 `AutoOrder`、`Quantity`、OS 模板、备份档 |
| `server/internal/db/schema.sql` 的 `vps_subscriptions` | 无 `auto_order` / `quantity` | 前端勾了自动下单也落不下去 |
| `vps.MonitorLoop` | 只通知 | 从不入队 |
| `purchase.PurchaseServer` | 查 `/dedicated/server/datacenter/availabilities`，加 `/eco` | VPS 走这条会失败 |
| `web/src/lib/datacenters.ts` | 12 个独立服务器机房 | 没有 LZ，也没有 `eu-south-mil` / `eu-west-rbx` 这类 VPS 码 |

前端「有货自动下单」已经画了，后端是半截。改造补后端，不要再做一套平行队列。

## 4. 方案选择

考虑过三种做法：

**A. 只改下拉框**  
把 2025 换成 2027 五个 planCode，监控逻辑不动。最快，但 Windows/Linux 会误报，LZ 在 US/IE 机房集合会错，自动下单仍会走 Eco。否决。

**B. 目录驱动的 VPS 产品线 + 现有队列分流（采用）**  
用 `/order/catalog/public/vps` 发现型号，用 `/vps/order/rule/datacenter` 盯库存，队列项带 `productKind`，处理器按种类走 Eco 或 VPS cart。2027 / LZ 是第一优先级分组，以后新年款不用再改常量。

**C. 拆独立 VPS 抢购服务**  
丢掉 Eco / 服务器控制。范围太大，也浪费 fork 里已经能用的多账户、队列、Telegram、VPS 控制。否决。

采用 B。

## 5. OVH 事实（实现必须遵守）

以下全部用 2026-08-18 公开接口实测，不是猜测。

### 5.1 库存

```
GET {regionalHost}/v1/vps/order/rule/datacenter?ovhSubsidiary={SUB}&planCode={PLAN}
```

公开、无需签名。主机必须和子公司对齐：

| 子公司 | Host |
|---|---|
| `US` | `https://api.us.ovhcloud.com` |
| `CA` / `QC` / `ASIA` / `SG` / `AU` / `IN` | `https://ca.api.ovh.com` |
| 其他（`IE` `FR` `DE` `GB` …） | `https://eu.api.ovh.com` |

返回每条机房：

```json
{
  "datacenter": "US-WEST-LZ-SEA",
  "code": "us-west-lz-sea",
  "status": "available",
  "linuxStatus": "available",
  "windowsStatus": "out-of-stock",
  "daysBeforeDelivery": 0
}
```

判定规则：

- 无货：`out-of-stock` 或 `out-of-stock-preorder-allowed`。
- 有货：其余（`available` 等）。
- 订阅盯 Linux：看 `linuxStatus`。
- 订阅盯 Windows：看 `windowsStatus`。
- 两个都盯：任一有货即「该 OS 轨有货」，通知里写明是哪条轨。
- 总字段 `status` 只做展示，**不作为自动下单条件**。VPS-1 / LZ 的 Windows 永远无货，总状态经常跟着 Linux 走，用总状态会误下。

同一 `vps-2027-model2.LZ`：

- `ovhSubsidiary=US` 只返回 8 个美国 LZ。
- `ovhSubsidiary=IE` / `FR` 只返回 7 个欧洲 LZ。

订阅必须绑定一个子公司。禁止用 US 凭据去下 IE 看到的 `EU-WEST-LZ-AMS`。

### 5.2 目录

```
GET {regionalHost}/1.0/order/catalog/public/vps?ovhSubsidiary={SUB}
```

2027 常规与 LZ 的关键差异：

| | 2027 常规 | 2027 LZ |
|---|---|---|
| planCode | `vps-2027-model{1-4}` | 当前仅 `vps-2027-model2.LZ` |
| 磁盘 | 必选 `option-storage-local-2027-modelN` | 必选 `option-storage-remote-2027` |
| OS addon | model1 仅 `option-linux`；model2/3/4 另有 `option-windows-2027-modelN` | 仅 `option-linux` |
| 备份 | `option-auto-backup-2027-{1\|7}-modelN` | `option-auto-backup-2027-{1\|7}-model2.LZ` |
| 机房配置值 | `GRA` / `US-EAST-VA` 这种目录名 | `US-WEST-LZ-SEA` / `EU-WEST-LZ-AMS` |
| US 配置项 | `region`, `vps_datacenter`, `vps_os` | 同左 |
| IE 配置项 | 另加必填 `infrastructure`：`production` / `preproduction` | 同左 |

下单时 `vps_datacenter` 必须用目录名（rule 的 `datacenter` 字段），不能用 `code`。`GRA` 对，`eu-west-gra` 不对。

IE 的 `region` 枚举是 `canada` / `europe`。`BHS` 走 `canada`，欧洲和亚太机房走 `europe`。US 子公司只有 `united_states`。

### 5.3 下单

```
POST /order/cart                         { ovhSubsidiary }
POST /order/cart/{id}/assign
GET  /order/cart/{id}/vps                选 duration / pricingMode
POST /order/cart/{id}/vps                { planCode, duration, pricingMode, quantity: 1 }
GET  /order/cart/{id}/item/{itemId}/requiredConfiguration
POST /order/cart/{id}/item/{itemId}/configuration   按 required 逐项
GET  /order/cart/{id}/vps/options
POST /order/cart/{id}/vps/options        每个必选 exclusive family 一个 addon
POST /order/cart/{id}/checkout           { autoPayWithPreferredPaymentMethod: false, waiveRetractationPeriod: true }
失败则 DELETE /order/cart/{id}
```

默认 addon（用户第一期不选配时）：

- `os`：Linux 轨 → `option-linux`；Windows 轨 → `option-windows-2027-modelN`。计划没有 Windows addon 则拒绝 Windows 订阅。
- `storage`：该 family 唯一项（常规本地盘 / LZ 远程盘）。
- `automatedBackup`：`option-auto-backup-2027-1-*`（1 天）。没有 1 天档再退到列表第一项。
- 不选 cPanel / Plesk / additionalDisk / snapshot。

默认 `vps_os` 文本：

- Linux：优先 `Ubuntu 24.04`，没有则 `Debian 12`，再没有则 Linux 镜像列表第一项（排除 `Windows*`）。
- Windows：优先 `Windows Server 2022 Standard (Desktop)`，没有则 Windows 列表最后一项（通常最新）。

默认 `infrastructure`（仅 IE 等要求该 label 时）：`production`。

## 6. 架构

继续用现有单二进制：Go/Gin + SQLite + 官方 `go-ovh`，前端 Vite/React/TanStack。不新增进程。

新增 / 扩的边界：

```
web/src/routes/vps-monitor.tsx
        │  订阅 CRUD
        ▼
server/internal/handlers/vps_monitor.go
        │
        ▼
server/internal/vps/
        ├── catalog.go     拉 public VPS catalog，分类 2027 / 2027-LZ / 其他
        ├── availability.go  rule 接口 + Linux/Windows 判定
        ├── defaults.go     OS / addon / region / infrastructure 默认值
        └── loop.go         监控循环；翻转时调 vps.EnqueueOrders
                │
                ▼
         现有 queue 表（多一列 product_kind + vps_spec JSON）
                │
                ▼
server/internal/purchase/
        ├── purchase.go         Eco，保持原样
        └── purchase_vps.go     新文件，只负责 VPS cart
                │
                ▼
         现有 history / Telegram
```

入队抽到 `purchase.Enqueue(state, item)`，`POST /api/queue`、`quick-order` 和 VPS 监控共用。禁止再走「监控 goroutine HTTP POST 本地 127.0.0.1」这条 Eco 监控现在用的弯路；VPS 新代码不要复制这个反模式。Eco 监控暂不重构。

## 7. 数据模型

### 7.1 VPS 订阅

在 `types.VPSSubscription` 和 `vps_subscriptions` 表上补列（旧库 `addColumnIfMissing`）：

| 字段 | 类型 | 含义 |
|---|---|---|
| `auto_order` | INTEGER 0/1 | 是否自动下单 |
| `quantity` | INTEGER 默认 1 | 每个翻转的「机房 × OS 轨」下几单 |
| `auto_order_account_id` | TEXT | 已有，继续用 |
| `os_image` | TEXT | 具体 `vps_os` 文本；空则按轨套默认 |
| `backup_plan` | TEXT `1` / `7` | 备份天数，默认 `1` |

`monitorLinux` / `monitorWindows` 继续当「盯哪条库存轨」，不是「下单镜像名」。

`lastStatus` 的 key 从 `dcCode` 改成 `dcCode|linux` 和 `dcCode|windows`。老数据只有 `dcCode` 时，第一次检查当首次，不触发自动下单（只补状态 + 可选初始通知）。避免升级后把「本来就有货」误判成补货。

### 7.2 队列项

`types.QueueItem` 增加：

```go
ProductKind string `json:"productKind"` // "eco"（默认/空）或 "vps"
VpsSpec     *VpsOrderSpec `json:"vpsSpec,omitempty"`
```

```go
type VpsOrderSpec struct {
    Subsidiary     string `json:"subsidiary"`
    DatacenterName string `json:"datacenterName"` // GRA / US-WEST-LZ-SEA
    DatacenterCode string `json:"datacenterCode"` // eu-west-gra / us-west-lz-sea
    OSTrack        string `json:"osTrack"`        // linux | windows
    OSImage        string `json:"osImage"`
    BackupPlan     string `json:"backupPlan"`     // 1 | 7
    Infrastructure string `json:"infrastructure"` // production；US 可空
}
```

SQLite：`queue.product_kind TEXT NOT NULL DEFAULT 'eco'`，`queue.vps_spec TEXT`（JSON）。`history` 同样加这两列，方便事后区分。

`QueueItem.Datacenter` 对 VPS 存 **code**（小写，监控和去重用）。下单时用 `VpsSpec.DatacenterName`。

去重：`productKind + planCode + datacenterCode + osTrack + accountId` 在 running/pending 队列里已有则不再入。监控批量下单带 `skipDuplicateCheck` 时仍要挡「同一秒同一轨灌进几十条」以外的跨轮重复；`skipDuplicateCheck` 只跳过 2 分钟成功历史限制，不跳过队列内相同指纹。

## 8. 监控行为

间隔沿用现有下限 60 秒。每个订阅串行请求 rule 接口，订阅之间停 1 秒，避免打爆公开 API。

对每个订阅、每个机房、每条启用的 OS 轨：

1. 读当前 `linuxStatus` / `windowsStatus`。
2. 和 `lastStatus[code|track]` 比。
3. 无货 → 有货：记 history；若 `notifyAvailable` 则进本轮通知聚合；若 `autoOrder && autoOrderAccountId != ""` 则该轨入下单列表。
4. 有货 → 无货：记 history；若 `notifyUnavailable` 则通知。不撤已经进队列的任务。
5. 首次（该 key 不存在）：只写 lastStatus。`notifyAvailable` 时发「初始状态」汇总，**不下单**。

Windows 轨在 catalog 无 Windows addon 时：创建订阅若勾了 Windows，后端 400，提示该型号不卖 Windows。LZ 和 VPS-1 2027 走这条。

自动下单数量：`len(翻转的机房×轨) * quantity`。每个队列项 quantity 恒为 1（OVH cart 的 quantity 字段），多次入队。

Telegram 文案用 catalog 的 invoiceName（`VPS-2 LZ 2027`），不要只显示裸 planCode。补货通知必须带：子公司、机房名、code、OS 轨、是否已入队、账户名。

## 9. 前端

VPS 补货页改为目录驱动，不再维护 `VPS_MODELS` 常量。

新增 `GET /api/vps-catalog?ovhSubsidiary=US`，返回：

```json
{
  "subsidiary": "US",
  "families": [
    {
      "id": "vps-2027",
      "label": "VPS 2027 常规",
      "plans": [
        {
          "planCode": "vps-2027-model2",
          "invoiceName": "VPS-2 2027",
          "supportsWindows": true,
          "isLocalZone": false,
          "datacenters": [
            { "name": "US-EAST-VA", "code": "us-east-vin" }
          ]
        }
      ]
    },
    {
      "id": "vps-2027-lz",
      "label": "VPS 2027 LZ",
      "plans": []
    },
    {
      "id": "other",
      "label": "其他",
      "plans": []
    }
  ]
}
```

分类规则：

- `planCode` 匹配 `^vps-2027-model[0-9]+$` → 2027 常规。
- `planCode` 匹配 `^vps-2027-model[0-9]+\.LZ$` → 2027 LZ。
- 其余进其他。第一期 UI 默认展开前两组，其他折叠。
- 带 `-eu` / `-ca` / `degressivity` / `percent` 后缀的影子 SKU 直接丢掉，不下拉。

机房列表：catalog 的 `vps_datacenter` 名称 ∪ 最近一次 rule 返回的 `datacenter`/`code` 映射。rule 暂不可用时，名称仍展示，code 用 `strings.ToLower(name)` 仅作占位，下单前必须再打 rule 补 code；补不到就失败，不准拿占位 code 去 checkout。

添加订阅表单：

- 先选子公司，再选系列，再选型号。
- 机房多选，不要逗号文本框。空 = 该子公司下该型号全部机房。
- Windows 复选框在 `supportsWindows=false` 时禁用。
- 自动下单：账户必选、数量 1–20（上限 20，避免一次翻机房把账户打爆）。
- 可选：镜像下拉（来自 catalog `vps_os`，按当前 OS 轨过滤）、备份 1 天/7 天。

队列页给 `productKind=vps` 的行加「VPS」chip，并显示 OS 轨。

## 10. 后端 API

现有 `/api/vps-monitor/*` 保留。变更：

- `POST /api/vps-monitor/subscriptions` 接收并校验 `autoOrder` / `quantity` / `osImage` / `backupPlan`。`autoOrder=true` 时 `autoOrderAccountId` 必填，账户必须存在，且账户 `zone` 必须等于订阅 `ovhSubsidiary`。跨子公司下单直接 400。
- 新增 `GET /api/vps-catalog?ovhSubsidiary=`。
- `POST /api/queue` 与 `POST /api/queue/quick-order` 增加可选 `productKind`、`vpsSpec`。缺省仍是 eco。
- 队列处理器：`productKind=="vps"` 调 `purchase.PurchaseVPS`，否则原 `PurchaseServer`。

catalog 缓存：单独建 `vps_catalogs` 表（`subsidiary` 主键 + JSON + `updated_at`），TTL 2 小时。不复用 Eco 的 `catalogs` 表，避免两份目录互相覆盖。rule 接口不缓存（库存必须实时）。

## 11. 错误与限流

- VPS cart 任一步失败：记 history `failed`，`DELETE` 购物车，队列项按现有 retryInterval 重试。
- OVH `409` / `CartError` / `out of stock`：当本轮无货，不算账户配置错误。
- OVH `400` 缺 configuration / addon：fail-fast，停止该任务（`status=failed`），不要空转。
- 公开 rule 接口连续 3 次非 200：该订阅本轮跳过，不把 lastStatus 改成无货。
- 不在 VPS 监控里做低于 60 秒的全局间隔。多订阅时串行 + 1 秒间距。

## 12. 许可证与仓库

- 本 fork 继承上游 **AGPL-3.0**。对外提供网络服务须开放对应源码。
- GitHub：`weandy/ovh`，`upstream` 指向 `gokele/ovh`。功能提交只推 origin。
- 不把 OVH 凭据、consumer key、Telegram token 写入仓库。

## 13. 测试

原仓库几乎无测试。本次在 `server/internal/vps/` 加纯函数测试，用录好的 JSON 夹具，不打真 OVH、不 checkout。

夹具目录：`server/internal/vps/testdata/`

- `rule-us-model2-lz.json`
- `rule-ie-model2-lz.json`
- `catalog-us-2027-subset.json`（只含 2027 相关 plan，手工裁剪）
- `catalog-ie-2027-subset.json`

必须覆盖：

1. `ClassifyPlan("vps-2027-model2.LZ") == 2027-lz`；`vps-2027-model2` → 2027；`vps-2025-model1` → other；`vps-2027-model2-eu` → drop。
2. `TrackAvailable(dc, "linux")` 在 SEA linux=available / windows=oos 时为 true/false。
3. `DefaultAddons` 对 LZ 选出 `option-linux` + `option-storage-remote-2027` + `option-auto-backup-2027-1-model2.LZ`。
4. `DatacenterName` 映射：rule `code=us-west-lz-sea` → name `US-WEST-LZ-SEA`。
5. `RegionFor`：`BHS` → `canada`；`GRA` → `europe`；`US-EAST-VA` → `united_states`。
6. 首次 lastStatus 为空时 `ShouldAutoOrder=false`。
7. 无 Windows addon 的 plan 创建 Windows 订阅被拒。

`PurchaseVPS` 的 HTTP 用 fake `ovh.Client` 或把 cart 步骤抽成接口，测「缺 storage 则不 checkout」。CI 不跑真单。

## 14. 实现顺序

写完本 spec 并确认后，再拆 `docs/superpowers/plans/` 下的实现计划。建议批次：

1. 数据模型 + 分类 / 库存纯函数 + 测试。
2. `GET /api/vps-catalog` + 前端订阅表单换成 2027 / LZ。
3. 监控循环改用 OS 轨，打通 Telegram。
4. `PurchaseVPS` + 队列分流。
5. 监控翻转入队 + 去重 + 历史 chip。

每一批都应单独可验证：先能看 2027 库存，再能手动入 VPS 队列下单，最后才自动下单。

## 15. 明确锁定的产品决策

| 决策点 | 选择 |
|---|---|
| 底座 | fork gokele，不从 coolci 改 |
| 主售系列 | 2027 常规 + 2027 LZ；以 catalog 为准，vps-in-stock 只作对照 |
| 库存源 | OVH 官方 rule 接口 |
| 下单通道 | `/order/cart/{id}/vps` |
| 自动扣款 | 关 |
| 默认备份 | 1 天 |
| 默认 Linux 镜像 | Ubuntu 24.04 |
| 默认基础设施 | production |
| 跨子公司 | 禁止（账户 zone 必须等于订阅 subsidiary） |
| Eco 功能 | 保持 |
| 监控 HTTP 自调用 | VPS 新代码不采用 |
