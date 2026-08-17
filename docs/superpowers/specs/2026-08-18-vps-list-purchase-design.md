# VPS 抢购列表（`/vps`）

日期：2026-08-18  
仓库：https://github.com/weandy/ovh  
状态：待确认后写实现计划  
前序：`docs/superpowers/specs/2026-08-18-vps-2027-monitor-purchase-design.md`（监控 + `/vps` cart 已落地）

## 1. 要做什么

侧栏「抢购」下增加二级页 **VPS 列表**，地址 **`/vps`**，观感对齐现有 `/servers`（`https://ovh.667786.xyz/servers`）：工具条 + 卡片网格 + 详情里选机房/系统再入队。

本页是「现在货架上有什么、哪个机房能买、要点哪几台进队列」。  
`/vps-monitor` 仍是「没货时盯着，翻有货再通知 / 自动下单」。两页共用同一条 VPS cart，不另开购买通道。

第一期只铺当前在售 5 个 planCode（catalog 动态发现，规则与前序 spec 相同）：

| 系列 | planCode | Windows 产品 |
|---|---|---|
| VPS-1 2027 | `vps-2027-model1` | 无 |
| VPS-2 2027 | `vps-2027-model2` | 有 |
| VPS-2 Local Zone | `vps-2027-model2.LZ` | 无 |
| VPS-3 2027 | `vps-2027-model3` | 有 |
| VPS-4 2027 | `vps-2027-model4` | 有 |

---

## 2. 「监控系统」到底是什么（先把概念拆开）

用户感觉「买 VPS 好像不分监控系统」，又隐约记得 Local Zone 不一样。这里其实混了三件完全不同的事。

### 2.1 不是 OVH 的机器监控

已购 VPS 上曾经有 `/vps/{name}/monitoring`、`/statistics`。OVH 已废弃，本仓库 VPS 控制页也不再展示。那是买完之后看 CPU/内存的，**和抢购无关**。

### 2.2 也不是「要不要装监控软件」

订阅表单上的「Linux / Windows」**不是**选监控探针。它是选 **看哪一条库存轨**。

公开库存接口每条机房同时给三个字段：

```json
{
  "datacenter": "US-WEST-LZ-SEA",
  "code": "us-west-lz-sea",
  "status": "available",
  "linuxStatus": "available",
  "windowsStatus": "out-of-stock"
}
```

| 字段 | 含义 |
|---|---|
| `status` | 总灯。经常跟着「至少有一种系统有货」走，**不能当购买条件** |
| `linuxStatus` | 这个机房 **Linux 镜像** 能不能卖 |
| `windowsStatus` | 这个机房 **Windows 镜像** 能不能卖 |

同一机房可以 Linux 有货、Windows 没货（西雅图 LZ 实测就是这样）；也可以反过来（例如部分欧洲常规机房 Linux 无、Windows 有）。

`/vps-monitor` 勾「盯 Linux」= 只把 `linuxStatus` 从无货变有货当成补货。这是**等货用的过滤器**。

### 2.3 真正下单时分的是「系统轨」，不是「监控」

购物车要交两样和系统有关的东西：

1. 配置项 `vps_os`：具体镜像名，如 `Ubuntu 24.04` 或 `Windows Server 2022 Standard (Desktop)`。
2. 必选 addon `os`：`option-linux` 或 `option-windows-2027-modelN`。

对应关系必须自洽：

- 买 Linux → 库存看 `linuxStatus`，addon 用 `option-linux`，镜像从非 Windows 列表里选。
- 买 Windows → 库存看 `windowsStatus`，addon 用 `option-windows-2027-modelN`，镜像从 Windows 列表里选。
- catalog 里没有 Windows addon 的型号，**根本没有 Windows 这种商品**，不是「暂时没货」。

所以：购买**分系统轨，不分「监控系统」**。监控页上的 Linux/Windows 只是把同一条库存轨拿来盯补货。列表页上用户直接选「我要 Linux 还是 Windows」，不必再出现「监控 Linux」这种措辞。

### 2.4 Local Zone 为什么显得「没这个讲究」

Local Zone（`.LZ`）是另一条产品线，不是常规 2027 换个机房。

| | 2027 常规 | 2027 Local Zone |
|---|---|---|
| planCode | `vps-2027-model2` | `vps-2027-model2.LZ` |
| 机房 | 大区 DC（`US-EAST-VA` / `GRA` / `RBX` …） | 城市边缘（`US-WEST-LZ-SEA` / `EU-WEST-LZ-AMS` …） |
| 子公司切开 | US 账户只看见美区大区 DC | US 只看见美国 LZ；IE/FR 只看见欧洲 LZ |
| OS addon | model2/3/4 有 Windows | **只有 `option-linux`** |
| 盘 | 本地盘 `option-storage-local-2027-modelN` | 远程盘 `option-storage-remote-2027` |
| `windowsStatus` | 有意义 | 恒为无货，且不能买，应显示「不适用」而不是「缺货可抢」 |

你记得「LZ 不太一样」是对的：LZ **没有 Windows 可买**，所以界面上不该再让人选 Windows，也不该因为 Windows 全红就显示整卡缺货（Linux 有货仍算可买）。

VPS-1 2027 同样没有 Windows addon，处理与 LZ 相同。

### 2.5 两页怎么分工

```
/vps            现在能买吗？ → 选系统轨 + 机房 → 入队立刻抢
/vps-monitor    现在没货，翻有货叫我 / 自动入队
/queue          真正跑 cart 的地方（productKind=vps）
```

列表页入队和监控翻转入队走同一个 `purchase.Enqueue` + `PurchaseVPS`。监控页的 Linux/Windows 勾选**不出现在列表页**。

---

## 3. 方案选择

**A. 把 VPS 塞进现有 `/servers` 网格**  
Eco 和 VPS 的库存、价格、配置项完全不是一套。会把 `/servers` 搞乱。否决。

**B. 独立路由 `/vps`，布局抄 `/servers`（采用）**  
导航挂在「抢购」下。卡片 / 工具条 / 详情弹窗对位服务器列表，数据走 VPS catalog + rule。

**C. 只做一张大表格、没有弹窗**  
5 个型号也能用，但和 `/servers` 的「点卡片再选配」不一致，用户要的正是那个风格。否决。

---

## 4. 导航与信息架构

「抢购」组改为：

| 路径 | 文案 |
|---|---|
| `/servers` | 服务器列表 |
| `/vps` | VPS 列表 |
| `/queue` | 抢购队列 |

「监控」组不动：`/monitor` 服务器监控，`/vps-monitor` VPS 补货。

同步改：`Sidebar.tsx`、`MobileMenu.tsx`、`TopBar.tsx` 的 `PAGE_META`、命令面板（若有）。面包屑：`抢购 / VPS 列表`。

---

## 5. 页面结构（对齐 `/servers`）

### 5.1 页头

- 标题：VPS 列表
- 说明：按子公司看 2027 常规与 Local Zone 的实时库存；下单走账户所在子公司
- 按钮：刷新（同时重拉 catalog + 全部 plan 的 rule）

### 5.2 工具条

与服务器列表同一套控件，语义换成 VPS：

- 搜索：planCode / 发票名 / 系列（常规、Local Zone）
- **仅显示可用**：至少有一个机房在**可购买的系统轨**上有货（见 §6）
- **子公司**下拉：复用 `OVH_SUBSIDIARIES`。默认跟当前默认账户 Zone；用户手改则写入 localStorage（key：`ovh_vps_list_subsidiary`，与服务器列表的价格地区 key 分开）
- 计数：`共 N 款`

切换子公司必须整页重拉 catalog 和库存。US 与 IE 的 LZ 机房集合不同，不能复用上一份数据。

### 5.3 卡片网格

`grid-cols-1 sm:grid-cols-2 xl:grid-cols-3`，卡片结构：

1. 左上：`invoiceName`（如 `VPS-2 LZ 2027`），副行等宽字体 planCode
2. 右上：状态 chip — 有任一可买轨有货则绿「x/y 可用」，否则红「暂时缺货」
3. 系列 chip：`2027` 或 `Local Zone`
4. 无 Windows 产品时加一条 muted chip：`仅 Linux`
5. 月费（catalog 月付 default，见 §7）
6. 机房灯带：该子公司该型号 rule 返回的机房，**不要**用独立服务器那 12 个固定 DC
7. 每个机房一颗灯 + 短码；有 Windows 产品的型号，灯旁用极小字标 `L` / `W` 两态（绿有 / 红无）。仅 Linux 型号只显示一颗总灯（等于 Linux）
8. 底栏按钮：**查看 / 抢购**（打开详情）

「其他」分组（2025、老 Value 等）默认不进网格。工具条可加「显示其他」开关，默认关。第一期实现可以只做 2027 + LZ，开关留接口。

### 5.4 详情弹窗（对位服务器列表的抢购对话框）

打开某一型号后：

- 标题：invoiceName + planCode
- 简要：系列、是否仅 Linux、当前子公司
- **系统轨**：Linux / Windows 二选一。无 Windows addon 时 Windows 禁用并注明「此型号不提供 Windows」
- **镜像**：`vps_os` 下拉，按系统轨过滤
- **备份**：1 天 / 7 天
- **机房多选**：只列出当前系统轨有货的机房（红灯不可选）。提供「全选有货机房」
- **抢购参数**：下单账户、数量（1–20）、重试间隔（秒，默认 30）
- 账户 Zone 必须等于页面子公司，否则按钮禁用并提示去换账户或换子公司
- 主按钮：`创建 N 个任务` → `productKind=vps` 入队（每个机房 × 数量一条，与 `/servers` 相同）
- 次按钮：`加入补货监控` → 调现有 `POST /api/vps-monitor/subscriptions`，带上当前型号、子公司、所选机房、对应系统轨（Linux 轨则 `monitorLinux=true`，Windows 轨则 `monitorWindows=true`）

不在弹窗里做 cPanel / Plesk / 额外盘。必选 storage 由后端按型号自动补。

---

## 6. 库存怎么算

后端新增（或扩展现有 catalog 接口）一次返回列表页所需的聚合数据，避免前端对 5 个型号各打一轮再拼。

建议：

```
GET /api/vps-catalog?ovhSubsidiary=US          // 已有：型号、机房名、是否支持 Windows
GET /api/vps-stock?ovhSubsidiary=US            // 新增：每个 planCode 的 rule 结果
```

`/api/vps-stock`：

- 只拉 `ClassifyPlan` 为 `vps-2027` 或 `vps-2027-lz` 的 plan（第一期）
- 并发请求 rule 接口，单 plan 超时 10s，失败的 plan 标 `stockError`，前端显示「库存暂不可用」而不是假缺货
- 不缓存 rule（和前序 spec 一致）。前端 staleTime 可 30–60 秒，刷新按钮强制重拉
- 响应：

```json
{
  "subsidiary": "US",
  "plans": [
    {
      "planCode": "vps-2027-model2.LZ",
      "datacenters": [
        {
          "name": "US-WEST-LZ-SEA",
          "code": "us-west-lz-sea",
          "headline": "available",
          "linux": "available",
          "windows": "out-of-stock",
          "daysBeforeDelivery": 0
        }
      ]
    }
  ]
}
```

判定函数沿用已有 `IsUnavailable` / `TrackAvailable`：

- 无货：`out-of-stock`、`out-of-stock-preorder-allowed`
- 卡片「可用」：存在机房满足  
  `TrackAvailable(dc, "linux") || (supportsWindows && TrackAvailable(dc, "windows"))`
- 入队前再按用户选的系统轨过滤机房；选了 Windows 但该 DC `windows` 无货 → 前端不让选，后端 `Enqueue` 前也可再拒一次

**禁止**用总字段 `headline/status` 决定能不能买。

---

## 7. 价格

列表页用 catalog 里 plan 的月付 `default` 价展示（微单位 ÷ 1e8），币种跟子公司走（US=USD，IE=EUR）。不在点开前为每个型号开 cart。

详情里若以后要含税精确价，再复用账户走一遍 VPS cart summary；第一期卡片价够用。文案标明「目录月费，下单以购物车为准」。

---

## 8. 入队与下单

点击创建任务：

```
每个选中机房 × quantity → purchase.Enqueue({
  productKind: "vps",
  planCode,
  datacenter: dc.code,
  accountId,
  retryInterval,
  vpsSpec: {
    subsidiary,
    datacenterName: dc.name,
    datacenterCode: dc.code,
    osTrack: "linux" | "windows",
    osImage,
    backupPlan: "1" | "7",
    infrastructure: "production"
  }
})
```

队列处理器已有的 `PurchaseVPS` 不变。指纹去重仍是 `vps|account|plan|dcCode|osTrack`。

前端可走现有 `POST /api/queue`（已支持 `productKind` + `vpsSpec`），不必再造 quick-order。

---

## 9. 和 `/vps-monitor` 的关系

| 动作 | 列表页 `/vps` | 补货页 `/vps-monitor` |
|---|---|---|
| 看现在谁有货 | 主用途 | 只在订阅卡片的 lastStatus 里顺带看 |
| 立刻抢当前有货机房 | 主用途 | 无 |
| 没货等到有 | 「加入补货监控」跳过去 | 主用途 |
| Linux/Windows 勾选 | 不出现这两个词；改称系统轨 | 保留，表示盯哪条库存 |

不把监控循环改成给列表页推送。列表页自己拉 `/api/vps-stock`。

---

## 10. 前端文件

| 文件 | 职责 |
|---|---|
| `web/src/routes/vps.tsx` | 新页面（布局抄 `servers.tsx`，数据换 VPS） |
| `web/src/hooks/use-vps-stock.ts` | 拉 `/api/vps-stock` |
| `web/src/hooks/use-vps-catalog.ts` | 已有，复用 |
| `web/src/hooks/use-queue.ts` | 入队时带上 `productKind` / `vpsSpec`（若还缺就补） |
| `Sidebar.tsx` / `MobileMenu.tsx` / `TopBar.tsx` | 加上 `/vps` |

不要去改 `servers.tsx` 的 Eco 逻辑。

---

## 11. 后端文件

| 文件 | 职责 |
|---|---|
| `server/internal/handlers/vps_catalog.go` | 已有 catalog；新增 `GetVPSStock` 或同文件加 handler |
| `server/internal/vps/*.go` | 复用 `LoadPlans` / `FetchRuleStock` / `ClassifyPlan` / `TrackAvailable` |
| `server/main.go` | `GET /api/vps-stock` |
| `handlers/queue.go` | 已能收 VPS 任务；校验 `vpsSpec` 完整、账户 Zone == subsidiary |

---

## 12. 错误与边界

- catalog 失败：整页 EmptyState，提示检查出网 / 子公司
- 单个 plan 的 rule 失败：该卡显示「库存拉取失败」，不影响其他卡
- 无 Windows 型号选 Windows：前后端都拒
- 账户 Zone ≠ 页面子公司：不能创建任务（避免再出现「US 账户下欧洲 LZ」）
- 选中机房在点下单瞬间变无货：入队仍成功，队列里 `PurchaseVPS` 会失败并按间隔重试（与 Eco 队列相同）
- 不在列表页做价格校验 cart。VPS 没有 Eco 那种「FQN 在 vin 询价 400」的 addon 矩阵；有货轨 + 必选 addon 即可。若以后发现 US 也有假有货，再加可选询价，不进第一期

---

## 13. 明确不做什么

- 不把 VPS 混进 `/servers`
- 不重做已购 VPS 控制
- 不恢复 OVH 已废弃的 VPS 性能监控图
- 第一期不做 2025 / Value 主推（「显示其他」默认关）
- 不做自动扣款
- 不爬 vps-in-stock.ovh

---

## 14. 验收

1. 侧栏抢购下能进 `/vps`，移动端菜单同样有。
2. 子公司切到 `US`：看到 2027 四档 + `vps-2027-model2.LZ`，LZ 机房是 SEA/NYC 这类美国城市，没有 GRA。
3. 切到 `IE`：LZ 机房变成 AMS/PRG 等欧洲城市。
4. LZ 与 VPS-1 卡片标注仅 Linux，详情里 Windows 不可选。
5. VPS-2 常规卡片能看出某机房 Linux 绿、Windows 红；选 Windows 时该机房不能勾。
6. 选 Linux + 有货机房 + 匹配 Zone 的账户，能在队列里看到 `VPS · linux` 任务。
7. 「加入补货监控」后 `/vps-monitor` 出现对应订阅。
8. Eco `/servers` 行为与现在一致。

---

## 15. 实现顺序（确认 spec 后再拆 plan）

1. `GET /api/vps-stock` + 纯函数测试（聚合 2027/LZ）
2. `/vps` 页面骨架：导航、工具条、卡片灯
3. 详情弹窗入队
4. 「加入补货监控」
5. 账户 Zone 校验与仅 Linux 型号的交互打磨
