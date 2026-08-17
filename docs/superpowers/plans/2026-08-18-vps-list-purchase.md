# VPS 列表抢购页 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 新增 `/vps` 抢购列表页（侧栏挂在「抢购」下），风格对齐 `/servers`，展示 2027 / Local Zone 实时库存并入队 VPS cart。

**Architecture:** 后端 `GET /api/vps-stock` 并发拉各 plan 的 rule 接口；前端新路由 `web/src/routes/vps.tsx` 复用 catalog + stock。入队走已有 `POST /api/queue`（`productKind=vps`）。Linux/Windows 在本页叫系统轨，不叫监控。

**Tech Stack:** Go/Gin、现有 `internal/vps` 纯函数、React/TanStack、Vite 文件路由。

**Spec:** `docs/superpowers/specs/2026-08-18-vps-list-purchase-design.md`

---

## File map

| 文件 | 职责 |
|---|---|
| `server/internal/vps/stock.go` | 聚合 rule 结果、可买判定、月费 |
| `server/internal/vps/stock_test.go` | 夹具测试 |
| `server/internal/handlers/vps_catalog.go` | `GetVPSStock` |
| `server/main.go` | 注册 `GET /api/vps-stock` |
| `server/internal/handlers/queue.go` | VPS 入队校验 Zone / vpsSpec |
| `web/src/routes/vps.tsx` | 列表页 |
| `web/src/hooks/use-vps-stock.ts` | stock query |
| `web/src/hooks/use-queue.ts` | 支持 VPS 入队字段 |
| `Sidebar.tsx` / `TopBar.tsx` / `CommandPalette.tsx` | 导航 |
| `README.md` / `VERSION` | 文档与版本 0.2.0 |

---

### Task 1: BuildPlanStock + PlanHasBuyableStock + MonthlyPrice

**Files:** Create `server/internal/vps/stock.go`, `stock_test.go`；Modify testdata catalog US subset 给 model2 加一条 month pricing。

测试：

- US LZ fixture：SEA linux 有货 → `PlanHasBuyableStock(false, dcs)==true`（仅 Linux 型号）
- ATL 全无货单独看该 DC `TrackAvailable` false
- 仅 Windows 有货且 `supportsWindows=false` → 整 plan 不可买
- 仅 Windows 有货且 `supportsWindows=true` → 可买
- `BuildPlanStock` 字段映射 name/code/linux/windows/headline
- `MonthlyPrice`：price `1000000000` → `10.00`

实现后 `go test ./internal/vps -count=1`。

### Task 2: GET /api/vps-stock

`GetVPSStock`：LoadPlans → 过滤 2027/LZ → 并发 FetchRuleStock → 返回 `{subsidiary, currency, plans:[{planCode, invoiceName, supportsWindows, isLocalZone, monthlyPrice, osImages, stockError, datacenters}]}`。

catalog 月费写进每条 plan。rule 失败则 `stockError` 非空、datacenters 空。

`main.go`：`api.GET("/vps-stock", handlers.GetVPSStock(state))`

### Task 3: 入队校验

`AddQueueItem`：`productKind=="vps"` 时必须有 `vpsSpec.subsidiary`、`datacenterName`、`osTrack`。账户 `Zone` 必须等于 `vpsSpec.subsidiary`（忽略大小写）。Windows 轨且无 name 时 400。

### Task 4: 前端 `/vps`

新页面抄 `/servers` 布局：子公司、搜索、仅可用、卡片灯、详情弹窗（系统轨、镜像、备份、有货机房多选、账户、数量、重试、入队、加入补货监控）。

`useCreateQueueItem` 增加 `productKind`、`vpsByDc`（code→name）、`osTrack`、`osImage`、`backupPlan`、`subsidiary`。

导航：抢购组插入 VPS 列表；`G P` 跳 `/vps`。

### Task 5: 文档、编译、发布

VERSION `0.2.0`。README 补 `/vps` 说明。`npx vite build` + 双平台 `-tags ui` 二进制，打 GitHub Release `v0.2.0`，推 `main`。
