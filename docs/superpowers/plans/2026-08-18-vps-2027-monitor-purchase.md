# VPS 2027 监控与抢购 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 `weandy/ovh`（fork gokele v0.0.8）上让 VPS 2027 常规与 VPS 2027 Local Zone（`.LZ`）可订阅、按 Linux/Windows 轨监控，并从无货翻有货时走 `/order/cart/{id}/vps` 自动下单。

**Architecture:** 新增 `server/internal/vps` 纯函数（分类 / 库存 / 默认 addon），catalog 走公开 `/order/catalog/public/vps`，库存走 `/vps/order/rule/datacenter`。队列项增加 `productKind=vps` + `VpsSpec`，处理器分流到 `purchase.PurchaseVPS`。前端 VPS 补货页改为目录驱动。Eco 路径不改行为。

**Tech Stack:** Go 1.25 + Gin + SQLite + go-ovh；Vite / React / TanStack；`go test` 用 JSON 夹具，不打真 checkout。

**Spec:** `docs/superpowers/specs/2026-08-18-vps-2027-monitor-purchase-design.md`

**工作目录:** `C:\Users\123\Desktop\AI\OVH\ovh`，分支 `feat/vps-2027-localzone-monitor-purchase`。

**LZ:** Local Zone。当前在售只跟踪 5 个 planCode：`vps-2027-model1`、`vps-2027-model2`、`vps-2027-model2.LZ`、`vps-2027-model3`、`vps-2027-model4`。分类规则仍用正则，以便以后出现 `vps-2027-model1.LZ` 自动进 LZ 组。

---

## File map

| 文件 | 职责 |
|---|---|
| `server/internal/vps/classify.go` | planCode → 2027 / 2027-lz / other / drop |
| `server/internal/vps/availability.go` | rule JSON、OS 轨有货判定、lastStatus key、是否自动下单 |
| `server/internal/vps/defaults.go` | 默认镜像、addon、region、infrastructure |
| `server/internal/vps/catalog.go` | 拉 catalog、裁剪影子 SKU、按子公司分组、缓存 |
| `server/internal/vps/vps.go` | 现有 rule HTTP + 监控循环；改为按 OS 轨 |
| `server/internal/vps/*_test.go` + `testdata/` | 夹具与纯函数测试 |
| `server/internal/types/types.go` | `VpsOrderSpec`、订阅/队列新字段 |
| `server/internal/db/schema.sql` + `db.go` + `queue.go` + `vps.go` + `history.go` | 迁移与 CRUD |
| `server/internal/purchase/enqueue.go` | 共享入队 + 队列内指纹去重 |
| `server/internal/purchase/purchase_vps.go` | VPS cart |
| `server/internal/purchase/queue_processor.go` | `vps` 走 `PurchaseVPS` |
| `server/internal/handlers/vps_catalog.go` | `GET /api/vps-catalog` |
| `server/internal/handlers/vps_monitor.go` | 订阅校验（跨子公司 / Windows / 自动下单） |
| `server/internal/handlers/queue.go` + `quick_order.go` | 接收 `productKind` / `vpsSpec` |
| `server/main.go` | 注册 catalog 路由 |
| `web/src/routes/vps-monitor.tsx` | 目录驱动表单 |
| `web/src/hooks/use-vps-monitor.ts` | 新字段 |
| `web/src/hooks/use-vps-catalog.ts` | catalog query |
| `web/src/routes/queue.tsx` + `use-queue.ts` | VPS chip |

---

### Task 1: ClassifyPlan

**Files:**
- Create: `server/internal/vps/classify.go`
- Test: `server/internal/vps/classify_test.go`

- [ ] **Step 1: Write the failing test**

```go
package vps

import "testing"

func TestClassifyPlan(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"vps-2027-model1", Family2027},
		{"vps-2027-model2", Family2027},
		{"vps-2027-model4", Family2027},
		{"vps-2027-model2.LZ", Family2027LZ},
		{"vps-2027-model1.LZ", Family2027LZ},
		{"vps-2025-model1", FamilyOther},
		{"vps-2025-model1.LZ", FamilyOther},
		{"vps-2027-model2-eu", FamilyDrop},
		{"vps-2027-model2-ca", FamilyDrop},
		{"vps-2027-model2-degressivity12", FamilyDrop},
		{"vps-comfort-4-8-80-vps-2025-model2-10percent", FamilyDrop},
	}
	for _, tc := range cases {
		if got := ClassifyPlan(tc.in); got != tc.want {
			t.Fatalf("ClassifyPlan(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd server && go test ./internal/vps -run TestClassifyPlan -count=1`
Expected: FAIL `undefined: ClassifyPlan` / `Family2027`

- [ ] **Step 3: Write minimal implementation**

```go
package vps

import "regexp"

const (
	FamilyDrop   = ""
	Family2027   = "vps-2027"
	Family2027LZ = "vps-2027-lz"
	FamilyOther  = "other"
)

var (
	reShadow = regexp.MustCompile(`-(eu|ca)$|degressivity|percent`)
	re2027   = regexp.MustCompile(`^vps-2027-model[0-9]+$`)
	re2027LZ = regexp.MustCompile(`^vps-2027-model[0-9]+\.LZ$`)
)

func ClassifyPlan(planCode string) string {
	if reShadow.MatchString(planCode) {
		return FamilyDrop
	}
	if re2027LZ.MatchString(planCode) {
		return Family2027LZ
	}
	if re2027.MatchString(planCode) {
		return Family2027
	}
	if planCode == "" {
		return FamilyDrop
	}
	return FamilyOther
}
```

注意：`reShadow` 必须在 2027 正则之前，否则 `vps-2027-model2-eu` 不会被丢掉。`10percent` 含 `percent`。

- [ ] **Step 4: Run test to verify it passes**

Run: `cd server && go test ./internal/vps -run TestClassifyPlan -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add server/internal/vps/classify.go server/internal/vps/classify_test.go
git commit -m "feat(vps): classify 2027 regular and Local Zone planCodes"
```

---

### Task 2: 库存轨与自动下单判定

**Files:**
- Create: `server/internal/vps/availability.go`
- Test: `server/internal/vps/availability_test.go`
- Create: `server/internal/vps/testdata/rule-us-model2-lz.json`
- Create: `server/internal/vps/testdata/rule-ie-model2-lz.json`

- [ ] **Step 1: Write fixtures + failing tests**

`testdata/rule-us-model2-lz.json`（摘自 2026-08-18 US rule 实测，至少含 SEA available/linux available/windows oos，以及 ATL 全 oos）：

```json
{
  "datacenters": [
    {
      "datacenter": "US-WEST-LZ-SEA",
      "code": "us-west-lz-sea",
      "status": "available",
      "linuxStatus": "available",
      "windowsStatus": "out-of-stock",
      "daysBeforeDelivery": 0
    },
    {
      "datacenter": "US-EAST-LZ-ATL",
      "code": "us-east-lz-atl",
      "status": "out-of-stock",
      "linuxStatus": "out-of-stock",
      "windowsStatus": "out-of-stock",
      "daysBeforeDelivery": 0
    }
  ]
}
```

`testdata/rule-ie-model2-lz.json`：`EU-WEST-LZ-AMS` linux available / windows oos；`EU-WEST-LZ-BRU` 全 oos。

```go
func TestTrackAvailableUsesOSStatusNotHeadline(t *testing.T) {
	raw, _ := os.ReadFile("testdata/rule-us-model2-lz.json")
	dcs, err := ParseDatacenters(raw)
	if err != nil {
		t.fatal(err)
	}
	sea := FindDC(dcs, "us-west-lz-sea")
	if !TrackAvailable(sea, "linux") {
		t.Fatal("SEA linux should be available")
	}
	if TrackAvailable(sea, "windows") {
		t.Fatal("SEA windows should be out of stock")
	}
}

func TestStatusKeyAndShouldAutoOrder(t *testing.T) {
	if StatusKey("us-west-lz-sea", "linux") != "us-west-lz-sea|linux" {
		t.Fatal(StatusKey("us-west-lz-sea", "linux"))
	}
	if ShouldAutoOrder(false, true, true, true, "acc") {
		t.Fatal("first seen must not auto-order")
	}
	if !ShouldAutoOrder(true, true, true, true, "acc") {
		t.Fatal("flip out-of-stock -> available should auto-order")
	}
	if ShouldAutoOrder(true, true, true, true, "") {
		t.Fatal("no account, no order")
	}
	if IsUnavailable("out-of-stock-preorder-allowed") != true {
		t.Fatal("preorder is unavailable")
	}
}

func TestLegacyLastStatusIsFirstSeen(t *testing.T) {
	// 升级前 lastStatus 只有 "us-west-lz-sea"="available"
	// HasTrackStatus 为 false → 当首次，不下单
	last := map[string]string{"us-west-lz-sea": "available"}
	if HasTrackStatus(last, "us-west-lz-sea", "linux") {
		t.Fatal("legacy headline key is not a track key")
	}
}
```

- [ ] **Step 2: Run tests, expect FAIL** (`undefined: ParseDatacenters` 等)

Run: `cd server && go test ./internal/vps -run "TestTrack|TestStatus|TestLegacy" -count=1`

- [ ] **Step 3: Implement**

```go
package vps

import "encoding/json"

type DatacenterStock struct {
	Name          string `json:"datacenter"`
	Code          string `json:"code"`
	Status        string `json:"status"`
	LinuxStatus   string `json:"linuxStatus"`
	WindowsStatus string `json:"windowsStatus"`
	Days          int    `json:"daysBeforeDelivery"`
}

func ParseDatacenters(raw []byte) ([]DatacenterStock, error) {
	var wrap struct {
		Datacenters []DatacenterStock `json:"datacenters"`
	}
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return nil, err
	}
	return wrap.Datacenters, nil
}

func FindDC(dcs []DatacenterStock, code string) DatacenterStock {
	for _, d := range dcs {
		if d.Code == code {
			return d
		}
	}
	return DatacenterStock{}
}

func IsUnavailable(status string) bool {
	return status == "out-of-stock" || status == "out-of-stock-preorder-allowed"
}

func TrackStatus(dc DatacenterStock, track string) string {
	if track == "windows" {
		return dc.WindowsStatus
	}
	return dc.LinuxStatus
}

func TrackAvailable(dc DatacenterStock, track string) bool {
	st := TrackStatus(dc, track)
	if st == "" {
		return false
	}
	return !IsUnavailable(st)
}

func StatusKey(code, track string) string { return code + "|" + track }

func HasTrackStatus(last map[string]string, code, track string) bool {
	if last == nil {
		return false
	}
	_, ok := last[StatusKey(code, track)]
	return ok
}

func ShouldAutoOrder(hadPrev, wasUnavail, nowAvail, autoOrder bool, accountID string) bool {
	return hadPrev && wasUnavail && nowAvail && autoOrder && accountID != ""
}
```

- [ ] **Step 4: `go test ./internal/vps -count=1` PASS**

- [ ] **Step 5: Commit** `feat(vps): parse rule stock and gate auto-order on OS track flips`

---

### Task 3: 默认镜像 / addon / region

**Files:**
- Create: `server/internal/vps/defaults.go`
- Test: `server/internal/vps/defaults_test.go`
- Create: `server/internal/vps/testdata/catalog-us-2027-subset.json`
- Create: `server/internal/vps/testdata/catalog-ie-2027-subset.json`

夹具只需 `plans[]` 里这几条的 `planCode / invoiceName / configurations / addonFamilies`（从 2026-08-18 catalog 裁剪）：

- US: `vps-2027-model1`（无 windows addon）、`vps-2027-model2`、`vps-2027-model2.LZ`、影子 `vps-2027-model2-eu`
- IE: `vps-2027-model2`（含 `infrastructure`）、`vps-2027-model2.LZ`

- [ ] **Step 1: Failing tests**

```go
func TestDefaultAddonsLocalZoneLinux(t *testing.T) {
	plan := mustPlan(t, "testdata/catalog-us-2027-subset.json", "vps-2027-model2.LZ")
	got, err := DefaultAddons(plan, "linux", "1")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"option-linux", "option-storage-remote-2027", "option-auto-backup-2027-1-model2.LZ"}
	if !sameSet(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestDefaultOSImage(t *testing.T) {
	plan := mustPlan(t, "testdata/catalog-us-2027-subset.json", "vps-2027-model2")
	if DefaultOSImage(plan, "linux") != "Ubuntu 24.04" {
		t.Fatal(DefaultOSImage(plan, "linux"))
	}
	if DefaultOSImage(plan, "windows") != "Windows Server 2022 Standard (Desktop)" {
		t.Fatal(DefaultOSImage(plan, "windows"))
	}
}

func TestRegionFor(t *testing.T) {
	if RegionFor("BHS") != "canada" {
		t.Fatal(RegionFor("BHS"))
	}
	if RegionFor("GRA") != "europe" {
		t.Fatal(RegionFor("GRA"))
	}
	if RegionFor("US-EAST-VA") != "united_states" {
		t.Fatal(RegionFor("US-EAST-VA"))
	}
	if RegionFor("US-WEST-LZ-SEA") != "united_states" {
		t.Fatal(RegionFor("US-WEST-LZ-SEA"))
	}
	if RegionFor("EU-WEST-LZ-AMS") != "europe" {
		t.Fatal(RegionFor("EU-WEST-LZ-AMS"))
	}
}

func TestWindowsRejectedWhenNoAddon(t *testing.T) {
	plan := mustPlan(t, "testdata/catalog-us-2027-subset.json", "vps-2027-model1")
	if SupportsWindows(plan) {
		t.Fatal("model1 has no windows addon")
	}
	if _, err := DefaultAddons(plan, "windows", "1"); err == nil {
		t.Fatal("expected error")
	}
}

func TestDatacenterNameFromRule(t *testing.T) {
	raw, _ := os.ReadFile("testdata/rule-us-model2-lz.json")
	dcs, _ := ParseDatacenters(raw)
	sea := FindDC(dcs, "us-west-lz-sea")
	if sea.Name != "US-WEST-LZ-SEA" {
		t.Fatal(sea.Name)
	}
}
```

- [ ] **Step 2: FAIL**

- [ ] **Step 3: Implement `defaults.go`**

`CatalogPlan` 结构只取需要的字段。`DefaultAddons`：

- `os` family：linux → `option-linux`；windows → 列表里第一个含 `windows` 的 addon，没有则 error
- `storage` family：唯一项（或第一项）
- `automatedBackup`：`backupPlan=="7"` 时选含 `-7-` 的，否则选含 `-1-` 的，再没有用第一项

`RegionFor(name)`：`strings.HasPrefix(name, "US-")` → `united_states`；`name=="BHS"` → `canada`；其余 → `europe`。

`DefaultOSImage`：linux 优先 Ubuntu 24.04 → Debian 12 → 第一个非 Windows；windows 优先 2022 Desktop → 最后一个 Windows*。

`DefaultInfrastructure` 恒返回 `"production"`。

- [ ] **Step 4: PASS** `go test ./internal/vps -count=1`

- [ ] **Step 5: Commit** `feat(vps): default OS image, addons, and region for 2027 carts`

---

### Task 4: 类型与 SQLite 迁移

**Files:**
- Modify: `server/internal/types/types.go`
- Modify: `server/internal/db/schema.sql`
- Modify: `server/internal/db/db.go`
- Modify: `server/internal/db/vps.go`
- Modify: `server/internal/db/queue.go`
- Modify: `server/internal/db/history.go`

- [ ] **Step 1: 扩展类型**

`VPSSubscription` 增加：

```go
AutoOrder          bool   `json:"autoOrder,omitempty"`
Quantity           int    `json:"quantity,omitempty"`
OSImage            string `json:"osImage,omitempty"`
BackupPlan         string `json:"backupPlan,omitempty"`
```

（`AutoOrderAccountID` 已有）

`QueueItem` 增加：

```go
ProductKind string         `json:"productKind,omitempty"`
VpsSpec     *VpsOrderSpec  `json:"vpsSpec,omitempty"`
```

```go
type VpsOrderSpec struct {
	Subsidiary     string `json:"subsidiary"`
	DatacenterName string `json:"datacenterName"`
	DatacenterCode string `json:"datacenterCode"`
	OSTrack        string `json:"osTrack"`
	OSImage        string `json:"osImage"`
	BackupPlan     string `json:"backupPlan"`
	Infrastructure string `json:"infrastructure,omitempty"`
}
```

`PurchaseHistoryEntry` 同样加 `ProductKind` + `VpsSpec`。

- [ ] **Step 2: schema + migrate**

`vps_subscriptions` 在 schema.sql 补列（新库）：

```sql
auto_order INTEGER NOT NULL DEFAULT 0,
quantity INTEGER NOT NULL DEFAULT 1,
os_image TEXT NOT NULL DEFAULT '',
backup_plan TEXT NOT NULL DEFAULT '1'
```

`queue` / `history`：

```sql
product_kind TEXT NOT NULL DEFAULT 'eco',
vps_spec TEXT NOT NULL DEFAULT ''
```

`db.migrate` 对旧库 `addColumnIfMissing` 上述列。`vps_subscriptions.auto_order_account_id` 已有迁移。

- [ ] **Step 3: 更新 row 映射**

`vps.go` SELECT/INSERT 带上新列。`queue.go` / `history.go` 的 `Replace*` SQL 加上 `product_kind, vps_spec`；空 `ProductKind` 读出来当成 `"eco"`。

- [ ] **Step 4: `cd server && go test ./...` 编译通过**

- [ ] **Step 5: Commit** `feat(db): persist VPS auto-order fields and queue productKind`

---

### Task 5: `GET /api/vps-catalog`

**Files:**
- Create: `server/internal/vps/catalog.go`
- Create: `server/internal/vps/catalog_test.go`
- Create: `server/internal/handlers/vps_catalog.go`
- Modify: `server/main.go`
- Modify: `server/internal/db/schema.sql` + `db.go`（`vps_catalogs` 表）

响应结构按 spec §9。`BuildFamilies(plans, ruleDCs)`：

- `ClassifyPlan == drop` 跳过
- 三组顺序固定：`vps-2027`（label `VPS 2027 常规`）、`vps-2027-lz`（`VPS 2027 Local Zone`）、`other`（`其他`）
- `supportsWindows = SupportsWindows(plan)`
- `isLocalZone = family==vps-2027-lz`
- datacenters：catalog `vps_datacenter` 名称；code 用 rule map（name→code），没有则暂空字符串（前端可展示名，下单前再补）

缓存：`vps_catalogs(subsidiary PRIMARY KEY, data TEXT, updated_at INTEGER)`，TTL 2h。rule 不缓存。

`GET /api/vps-catalog?ovhSubsidiary=US` 缺省 `IE`。

- [ ] **Step 1: `TestBuildFamiliesDropsShadowAndSplitsLZ`** 用 subset 夹具
- [ ] **Step 2: FAIL**
- [ ] **Step 3: 实现 BuildFamilies + handler**
- [ ] **Step 4: PASS**
- [ ] **Step 5: Commit** `feat(vps): serve catalog families for 2027 and Local Zone`

---

### Task 6: 前端订阅表单

**Files:**
- Create: `web/src/hooks/use-vps-catalog.ts`
- Modify: `web/src/hooks/use-vps-monitor.ts`
- Modify: `web/src/lib/query.ts`
- Modify: `web/src/routes/vps-monitor.tsx`

删掉 `VPS_MODELS` 常量。`AddVPSDialog`：

1. 选子公司（沿用现有 `SUBSIDIARIES`，补全 `OVH_SUBSIDIARIES` 里缺的也行，至少保留 US/IE/FR/DE/GB/CA）
2. 拉 `GET /api/vps-catalog?ovhSubsidiary=`
3. 系列 Select：常规 / Local Zone / 其他（其他折叠或放最后）
4. 型号 Select 来自该 family
5. 机房多选 checkbox，空=全部
6. `supportsWindows===false` 时 Windows 复选禁用并强制 false
7. 自动下单：账户必选、数量 1–20
8. 可选镜像（过滤 Windows*）、备份 1/7 天

`useCreateVPSMonitorSubscription` payload 增加 `autoOrder, quantity, osImage, backupPlan`。

列表行：Local Zone 用 chip「LZ」；显示 `invoiceName` 而不是把 `vps-2027-model2.LZ` 翻成 VPS-2。

- [ ] 手动：`npm run build` 在 `web/` 过编译
- [ ] Commit `feat(web): catalog-driven VPS 2027 and Local Zone subscriptions`

---

### Task 7: 监控循环按 OS 轨

**Files:**
- Modify: `server/internal/vps/vps.go`
- Modify: `server/internal/handlers/vps_monitor.go`

循环改动（保持现有 1s 间距、间隔下限 60s、TG 失效自停）：

对每个 DC、每条启用轨（linux / windows）：

```
key := StatusKey(code, track)
had := HasTrackStatus(sub.LastStatus, code, track)
cur := TrackStatus(dc, track)
avail := TrackAvailable(dc, track)
if !had {
  // 写 lastStatus[key]=cur；进 initial 列表；不下单
} else {
  old := sub.LastStatus[key]
  if IsUnavailable(old) && avail {
    // history + notify + 若 ShouldAutoOrder 则收集 orderTarget
  } else if !IsUnavailable(old) && IsUnavailable(cur) {
    // history + unavailable notify
  }
}
sub.LastStatus[key] = cur
```

创建订阅：

- `autoOrder` 且 `autoOrderAccountId==""` → 400
- 账户 `zone` 必须 `== ovhSubsidiary`（大小写不敏感）
- `monitorWindows && !supportsWindows(plan)` → 400「该型号不提供 Windows（Local Zone / VPS-1 2027）」
- `quantity` 默认 1，夹在 1–20
- `backupPlan` 只能 `1` 或 `7`

本任务**先不入队**，只把 orderTarget 打日志 `would auto-order`，入队放到 Task 10，方便单独验证通知。

- [ ] Commit `feat(vps): monitor Linux and Windows tracks separately`

---

### Task 8: `purchase.Enqueue` 与队列分流

**Files:**
- Create: `server/internal/purchase/enqueue.go`
- Create: `server/internal/purchase/enqueue_test.go`
- Modify: `server/internal/handlers/queue.go`
- Modify: `server/internal/handlers/quick_order.go`
- Modify: `server/internal/purchase/queue_processor.go`

```go
func QueueFingerprint(item types.QueueItem) string {
	kind := item.ProductKind
	if kind == "" {
		kind = "eco"
	}
	track := ""
	if item.VpsSpec != nil {
		track = item.VpsSpec.OSTrack
	}
	return kind + "|" + item.AccountID + "|" + item.PlanCode + "|" + item.Datacenter + "|" + track
}

func HasActiveDuplicate(queue []types.QueueItem, item types.QueueItem) bool {
	fp := QueueFingerprint(item)
	for _, q := range queue {
		if q.Status == "running" || q.Status == "pending" {
			if QueueFingerprint(q) == fp {
				return true
			}
		}
	}
	return false
}

func Enqueue(state *app.State, item types.QueueItem) (types.QueueItem, error) {
	if item.ID == "" {
		item.ID = uuid.NewString()
	}
	if item.ProductKind == "" {
		item.ProductKind = "eco"
	}
	if item.CreatedAt == "" {
		item.CreatedAt = types.NowISO()
	}
	item.UpdatedAt = types.NowISO()
	if item.Status == "" {
		item.Status = "running"
	}
	state.QueueMu.Lock()
	defer state.QueueMu.Unlock()
	if HasActiveDuplicate(state.Queue, item) {
		return item, ErrDuplicate
	}
	state.Queue = append(state.Queue, item)
	_ = state.SaveQueue()
	return item, nil
}
```

`handlers.AddQueueItem` / quick-order 改为调 `Enqueue`。`productKind` 缺省 eco。

`queue_processor.processSingle`：

```go
success := false
if snapshot.ProductKind == "vps" {
	success = PurchaseVPS(state, &snapshot)
} else {
	success = PurchaseServer(state, &snapshot)
}
```

Task 8 可先让 `PurchaseVPS` 返回 false 并打「not implemented」，Task 9 再填。测试只覆盖 fingerprint / duplicate。

- [ ] Commit `feat(purchase): enqueue with VPS fingerprint and queue dispatch`

---

### Task 9: `PurchaseVPS`

**Files:**
- Create: `server/internal/purchase/purchase_vps.go`
- Create: `server/internal/purchase/purchase_vps_test.go`

把 OVH 调用收成接口：

```go
type OVHCart interface {
	Get(url string, res interface{}) error
	Post(url string, req, res interface{}) error
	Delete(url string, res interface{}) error
}
```

`PurchaseVPS` 步骤按 spec §5.3。`VpsSpec` 为空或 `DatacenterName` 为空 → 立即失败，不创建 cart。

假 client 测试：

1. 记录调用顺序，断言出现 `/order/cart/{id}/vps` 且**不出现** `/eco`
2. checkout body `autoPayWithPreferredPaymentMethod==false`
3. 若 `vps/options` 缺少 `option-storage-remote-2027`（LZ），不调用 checkout，并 `Delete` cart

失败时复用现有 `recordFailure`。成功写 history，带 `productKind=vps`。

选 duration/pricingMode：`GET /order/cart/{id}/vps` 里找 `planCode` 匹配且 `duration=="P1M"` 的第一条，没有则第一条。

configuration：

- 永远设 `vps_datacenter=VpsSpec.DatacenterName`、`vps_os=VpsSpec.OSImage`
- `region=RegionFor(DatacenterName)`
- required 里有 `infrastructure` 才设（默认 `production`）

addon：用 catalog 缓存或 `GET /order/cart/{id}/vps/options` 的 planCode 列表，按 `DefaultAddons` 的名字匹配后 POST。

- [ ] Commit `feat(purchase): checkout VPS 2027 carts with mandatory addons`

---

### Task 10: 监控翻转入队 + 队列 UI

**Files:**
- Modify: `server/internal/vps/vps.go`（把 would-order 换成 `purchase.Enqueue`）
- Modify: `web/src/hooks/use-queue.ts`
- Modify: `web/src/routes/queue.tsx`

每个 orderTarget × quantity 入一条：

```go
item := types.QueueItem{
	AccountID:   sub.AutoOrderAccountID,
	PlanCode:    sub.PlanCode,
	Datacenter:  dc.Code,
	ProductKind: "vps",
	QuickOrder:  true,
	RetryInterval: 30,
	VpsSpec: &types.VpsOrderSpec{
		Subsidiary:     ovhSub,
		DatacenterName: dc.Name,
		DatacenterCode: dc.Code,
		OSTrack:        track,
		OSImage:        osImage, // 订阅 osImage 或 DefaultOSImage
		BackupPlan:     backup,  // 默认 1
		Infrastructure: "production",
	},
}
```

`Enqueue` 返回 duplicate 时日志跳过，不报错。Telegram 补货文案带：invoiceName、subsidiary、机房名、code、OS 轨、是否已入队。

队列页：`productKind==="vps"` 显示 chip「VPS」，旁注 `vpsSpec.osTrack`。

- [ ] `cd server && go test ./internal/vps ./internal/purchase -count=1`
- [ ] `cd web && npm run build`
- [ ] Commit `feat(vps): auto-enqueue Local Zone and 2027 restocks`

---

## Spec coverage

| Spec | Task |
|---|---|
| 5 个在售 plan + LZ=Local Zone 分类 | 1, 5, 6 |
| linuxStatus/windowsStatus | 2, 7 |
| 首次/升级旧 key 不下单 | 2, 7 |
| catalog 驱动 + 丢掉影子 SKU | 5, 6 |
| US/IE 机房集合随子公司 | 5, 7 |
| 跨子公司 400 | 7 |
| LZ / VPS-1 拒 Windows | 3, 7 |
| `/vps` cart + 必选 addon + 不自动扣款 | 9 |
| 入队去重、监控不 HTTP 自调 | 8, 10 |
| Eco 不动 | 8 默认 productKind=eco |
| vps_catalogs 缓存 | 5 |
| 队列 VPS chip | 10 |

## 验证命令

```bash
cd server
go test ./internal/vps ./internal/purchase -count=1

cd ../web
npm run build
```

不跑真实 OVH checkout。rule/catalog 可用公开 GET 手工抽查。
