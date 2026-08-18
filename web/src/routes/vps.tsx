import { createFileRoute } from "@tanstack/react-router";
import {
  Cloud,
  RefreshCw,
  Search,
  Filter,
  ShoppingCart,
  MapPin,
  Bell,
  Loader2,
} from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { toast } from "sonner";
import { PageHeader } from "@/components/common/PageHeader";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Chip } from "@/components/common/Chip";
import { StatusDot } from "@/components/common/StatusDot";
import { Skeleton } from "@/components/common/Skeleton";
import { EmptyState } from "@/components/common/EmptyState";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { AccountSelect } from "@/components/common/AccountSelect";
import { useDefaultAccount, useAccounts } from "@/hooks/use-accounts";
import { useCreateQueueItem } from "@/hooks/use-queue";
import { useCreateVPSMonitorSubscription } from "@/hooks/use-vps-monitor";
import {
  useVPSStock,
  planHasBuyableStock,
  shortDcLabel,
  trackAvailable,
  type VpsStockPlan,
  type VpsStockDC,
} from "@/hooks/use-vps-stock";
import { regionOfSubsidiary } from "@/lib/ovh-regions";

export const Route = createFileRoute("/vps")({
  component: VpsListPage,
});

function VpsListPage() {
  const defaultAcc = useDefaultAccount();
  const accounts = useAccounts();
  const [accountId, setAccountId] = useState("");

  useEffect(() => {
    if (!accountId && defaultAcc) setAccountId(defaultAcc.id);
  }, [defaultAcc?.id, accountId]);

  const account = (accounts.data || []).find((a) => a.id === accountId);
  const accountRegion = account ? regionOfSubsidiary(account.zone) : "US";
  const stockQ = useVPSStock(accountRegion, account?.zone);
  const [search, setSearch] = useState("");
  const [onlyAvailable, setOnlyAvailable] = useState(false);
  const [detailCode, setDetailCode] = useState<string | null>(null);

  const list = stockQ.data?.plans || [];
  const filtered = useMemo(() => {
    const s = search.trim().toLowerCase();
    let out = list;
    if (s) {
      out = out.filter((p) =>
        `${p.planCode} ${p.invoiceName} ${p.isLocalZone ? "local zone lz" : "2027"}`.toLowerCase().includes(s)
      );
    }
    if (onlyAvailable) out = out.filter(planHasBuyableStock);
    return out;
  }, [list, search, onlyAvailable]);

  const detail = detailCode ? list.find((p) => p.planCode === detailCode) || null : null;

  return (
    <div className="space-y-6">
      <PageHeader
        icon={Cloud}
        title="VPS 列表"
        description="跟官网一样按账户店铺拉全球机房。美国账户买欧洲/加拿大机房走 -eu/-ca SKU，购物车仍用该账户"
        action={
          <Button
            variant="outline"
            onClick={() => stockQ.refetch()}
            disabled={stockQ.isFetching}
          >
            <RefreshCw className={`w-4 h-4 ${stockQ.isFetching ? "animate-spin" : ""}`} />
            刷新
          </Button>
        }
      />

      <Card>
        <CardContent className="p-4 flex flex-col sm:flex-row sm:items-center gap-3">
          <div className="relative flex-1 min-w-0">
            <Search className="absolute left-3.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-muted-foreground pointer-events-none" />
            <Input
              placeholder="搜索 planCode / 型号 / Local Zone..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="pl-9 rounded-full"
            />
          </div>
          <Button
            variant={onlyAvailable ? "default" : "outline"}
            size="sm"
            className="rounded-full"
            onClick={() => setOnlyAvailable((v) => !v)}
          >
            <Filter className="w-3.5 h-3.5" />
            仅显示可用
          </Button>
          <div className="w-full sm:w-[220px]">
            <AccountSelect value={accountId} onChange={setAccountId} />
          </div>
          <span className="text-[12px] text-muted-foreground whitespace-nowrap">
            {stockQ.isPending ? "加载中..." : `共 ${filtered.length} 款`}
          </span>
        </CardContent>
      </Card>

      {stockQ.isPending ? (
        <div className="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-4">
          {Array.from({ length: 6 }).map((_, i) => (
            <Skeleton key={i} className="h-[240px] rounded-2xl" />
          ))}
        </div>
      ) : stockQ.isError ? (
        <Card>
          <EmptyState
            icon={Cloud}
            title="无法加载 VPS 目录"
            description="检查本机能否访问 OVH API，或换一个子公司再试"
          />
        </Card>
      ) : filtered.length === 0 ? (
        <Card>
          <EmptyState
            icon={Cloud}
            title="未找到 VPS"
            description={list.length === 0 ? "该区没有 2027 / Local Zone 型号" : "没有匹配的搜索结果"}
          />
        </Card>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-4">
          {filtered.map((plan) => (
            <VpsCard key={plan.planCode} plan={plan} onView={() => setDetailCode(plan.planCode)} />
          ))}
        </div>
      )}

      <Dialog open={!!detail} onOpenChange={(v) => !v && setDetailCode(null)}>
        <DialogContent className="w-[95vw] sm:w-full sm:max-w-3xl max-h-[90vh] overflow-hidden flex flex-col">
          {detail ? (
            <VpsDetail
              plan={detail}
              accountId={accountId}
              onAccountId={setAccountId}
              onClose={() => setDetailCode(null)}
            />
          ) : null}
        </DialogContent>
      </Dialog>
    </div>
  );
}

function formatMoney(v: number, currency: string): string {
  const sym =
    currency === "EUR" ? "€" : currency === "USD" ? "$" : currency === "GBP" ? "£" : currency === "CAD" ? "CA$" : `${currency} `;
  return `${sym}${v.toFixed(2)}`;
}

function VpsCard({ plan, onView }: { plan: VpsStockPlan; onView: () => void }) {
  const buyable = planHasBuyableStock(plan);
  const total = plan.datacenters.length;
  const ok = plan.datacenters.filter(
    (dc) => trackAvailable(dc, "linux") || (plan.supportsWindows && trackAvailable(dc, "windows"))
  ).length;

  return (
    <Card className="overflow-hidden transition-colors hover:bg-secondary/30">
      <CardContent className="p-5 flex flex-col gap-4">
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0 flex-1">
            <h3 className="text-[15px] font-semibold truncate">{plan.invoiceName || plan.planCode}</h3>
            <p className="font-mono text-[12px] text-muted-foreground truncate mt-0.5">{plan.planCode}</p>
          </div>
          {plan.stockError ? (
            <Chip tone="warning">库存拉取失败</Chip>
          ) : buyable ? (
            <Chip tone="success">
              <StatusDot tone="success" pulse size="xs" />
              {ok}/{total || 0} 可用
            </Chip>
          ) : (
            <Chip tone="danger">
              <StatusDot tone="danger" size="xs" />
              暂时缺货
            </Chip>
          )}
        </div>

        <div className="flex gap-1.5 flex-wrap">
          {plan.isLocalZone ? <Chip tone="info">Local Zone</Chip> : <Chip tone="default">2027</Chip>}
          {!plan.supportsWindows && <Chip tone="default">仅 Linux</Chip>}
        </div>

        <div className="text-xl font-bold tabular-nums">
          {plan.monthlyPrice != null ? (
            <>
              {formatMoney(plan.monthlyPrice, plan.currency)}
              <span className="text-[11px] font-normal text-muted-foreground ml-1">/月 · 目录价</span>
            </>
          ) : (
            <span className="text-sm font-normal text-muted-foreground">价格见购物车</span>
          )}
        </div>

        <div className="flex flex-wrap gap-1.5">
          {plan.datacenters.map((dc) => {
            const linuxOk = trackAvailable(dc, "linux");
            const winOk = plan.supportsWindows && trackAvailable(dc, "windows");
            const any = linuxOk || winOk;
            return (
              <div
                key={dc.code || dc.name}
                className="inline-flex items-center gap-1 rounded-full border border-border px-2 py-0.5"
                title={`${dc.name} Linux=${dc.linux} Windows=${dc.windows}`}
              >
                <StatusDot tone={any ? "success" : "danger"} size="xs" pulse={any} />
                <span className="font-mono text-[10px] font-semibold">{shortDcLabel(dc)}</span>
                {plan.supportsWindows && (
                  <span className="text-[9px] text-muted-foreground">
                    <span className={linuxOk ? "text-emerald-600" : "text-rose-500"}>L</span>
                    <span className="mx-0.5">/</span>
                    <span className={winOk ? "text-emerald-600" : "text-rose-500"}>W</span>
                  </span>
                )}
              </div>
            );
          })}
        </div>

        <Button size="sm" className="w-full" onClick={onView}>
          <ShoppingCart className="w-3.5 h-3.5" />
          查看 / 抢购
        </Button>
      </CardContent>
    </Card>
  );
}

function linuxImages(images: string[] | undefined): string[] {
  return (images || []).filter((n) => !n.toLowerCase().startsWith("windows"));
}
function windowsImages(images: string[] | undefined): string[] {
  return (images || []).filter((n) => n.toLowerCase().startsWith("windows"));
}
function defaultLinuxImage(images: string[] | undefined): string {
  const list = linuxImages(images);
  return list.find((n) => n === "Ubuntu 24.04") || list.find((n) => n === "Debian 12") || list[0] || "";
}
function defaultWindowsImage(images: string[] | undefined): string {
  const list = windowsImages(images);
  return list.find((n) => n.includes("2022")) || list[list.length - 1] || "";
}

const CONTINENT_LABEL: Record<string, string> = {
  europe: "Europe",
  north_america: "North America",
  asia_oceania: "Asia / Oceania",
};

function VpsDetail({
  plan,
  accountId,
  onAccountId,
  onClose,
}: {
  plan: VpsStockPlan;
  accountId: string;
  onAccountId: (id: string) => void;
  onClose: () => void;
}) {
  const create = useCreateQueueItem();
  const addMon = useCreateVPSMonitorSubscription();
  const accounts = useAccounts();
  const [osTrack, setOsTrack] = useState<"linux" | "windows">("linux");
  const [osImage, setOsImage] = useState("");
  const [backupPlan, setBackupPlan] = useState("1");
  const [selectedDCs, setSelectedDCs] = useState<string[]>([]);
  const [quantity, setQuantity] = useState("1");
  const [retryInterval, setRetryInterval] = useState("30");

  useEffect(() => {
    if (!plan.supportsWindows && osTrack === "windows") setOsTrack("linux");
  }, [plan.supportsWindows, osTrack]);

  useEffect(() => {
    setOsImage(osTrack === "windows" ? defaultWindowsImage(plan.osImages) : defaultLinuxImage(plan.osImages));
    setSelectedDCs([]);
  }, [osTrack, plan.planCode, plan.osImages]);

  const account = (accounts.data || []).find((a) => a.id === accountId);
  const zoneOk = !!account;

  const images = osTrack === "windows" ? windowsImages(plan.osImages) : linuxImages(plan.osImages);
  const instock = plan.datacenters.filter((dc) => trackAvailable(dc, osTrack));
  const qty = Math.max(1, Math.min(20, Number(quantity) || 1));
  const totalTasks = selectedDCs.length * qty;

  const toggleDC = (code: string) =>
    setSelectedDCs((prev) => (prev.includes(code) ? prev.filter((c) => c !== code) : [...prev, code]));

  const dcNames: Record<string, string> = {};
  for (const dc of plan.datacenters) dcNames[dc.code] = dc.name;

  return (
    <>
      <DialogHeader>
        <div className="flex items-start justify-between gap-3 pr-6">
          <div className="min-w-0">
            <DialogTitle className="text-xl truncate">{plan.invoiceName}</DialogTitle>
            <DialogDescription className="font-mono truncate mt-0.5">{plan.planCode}</DialogDescription>
          </div>
          {planHasBuyableStock(plan) ? (
            <Chip tone="success">
              <StatusDot tone="success" pulse size="xs" />
              当前有可买机房
            </Chip>
          ) : (
            <Chip tone="danger">
              <StatusDot tone="danger" size="xs" />
              暂时缺货
            </Chip>
          )}
        </div>
      </DialogHeader>

      <div className="overflow-y-auto -mx-6 px-6 space-y-5 flex-1">
        <div className="flex gap-1.5 flex-wrap text-[12px]">
          {plan.isLocalZone ? <Chip tone="info">Local Zone</Chip> : <Chip tone="default">2027 常规</Chip>}
          {!plan.supportsWindows && <Chip tone="default">仅 Linux</Chip>}
          {account && <Chip tone="default">账户 {account.zone}</Chip>}
        </div>

        {plan.monthlyPrice != null && (
          <div className="border border-border rounded-2xl p-4 bg-secondary/30">
            <div className="text-[11px] text-muted-foreground">目录月费 · 下单以购物车为准</div>
            <div className="text-2xl font-bold tabular-nums mt-0.5">
              {formatMoney(plan.monthlyPrice, plan.currency)}
            </div>
          </div>
        )}

        <div>
          <h3 className="text-[13px] font-semibold mb-2">系统</h3>
          <div className="grid grid-cols-2 gap-2">
            <button
              type="button"
              onClick={() => setOsTrack("linux")}
              className={
                "border rounded-xl px-3 py-2 text-left text-sm " +
                (osTrack === "linux" ? "border-foreground bg-foreground text-background" : "border-border")
              }
            >
              Linux
            </button>
            <button
              type="button"
              disabled={!plan.supportsWindows}
              onClick={() => setOsTrack("windows")}
              className={
                "border rounded-xl px-3 py-2 text-left text-sm " +
                (!plan.supportsWindows
                  ? "opacity-50 cursor-not-allowed border-border"
                  : osTrack === "windows"
                    ? "border-foreground bg-foreground text-background"
                    : "border-border")
              }
            >
              Windows
              {!plan.supportsWindows && (
                <div className="text-[10px] opacity-80 mt-0.5">此型号不提供</div>
              )}
            </button>
          </div>
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <div>
            <label className="block text-[11px] text-muted-foreground mb-1">镜像</label>
            <Select value={osImage} onValueChange={setOsImage}>
              <SelectTrigger>
                <SelectValue placeholder="选择镜像" />
              </SelectTrigger>
              <SelectContent>
                {images.map((img) => (
                  <SelectItem key={img} value={img}>
                    {img}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div>
            <label className="block text-[11px] text-muted-foreground mb-1">备份</label>
            <Select value={backupPlan} onValueChange={setBackupPlan}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="1">1 天</SelectItem>
                <SelectItem value="7">7 天</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>

        <div>
          <div className="flex items-center justify-between mb-2.5 gap-2 flex-wrap">
            <h3 className="text-[13px] font-semibold flex items-center gap-1.5">
              <MapPin className="w-3.5 h-3.5 text-muted-foreground" />
              数据中心 · {osTrack === "linux" ? "Linux" : "Windows"} 有货 {instock.length} / {plan.datacenters.length}
            </h3>
            <Button
              variant="outline"
              size="sm"
              className="h-7 text-[11px]"
              onClick={() =>
                setSelectedDCs(
                  selectedDCs.length === instock.length ? [] : instock.map((d) => d.code)
                )
              }
            >
              {selectedDCs.length > 0 ? "清空" : "选可用"}
            </Button>
          </div>
          {(["europe", "north_america", "asia_oceania"] as const).map((cont) => {
            const dcs = plan.datacenters.filter((d) => (d.continent || "europe") === cont);
            if (dcs.length === 0) return null;
            return (
              <div key={cont} className="mb-3 last:mb-0">
                <div className="text-[11px] text-muted-foreground mb-1.5">{CONTINENT_LABEL[cont]}</div>
                <div className="grid grid-cols-2 sm:grid-cols-3 gap-2">
                  {dcs.map((dc) => (
                    <DcPick key={dc.code} dc={dc} track={osTrack} selected={selectedDCs.includes(dc.code)} onToggle={() => toggleDC(dc.code)} />
                  ))}
                </div>
              </div>
            );
          })}
        </div>

        <div className="border-t border-border pt-4 space-y-3">
          <h3 className="text-[13px] font-semibold flex items-center gap-1.5">
            <ShoppingCart className="w-3.5 h-3.5 text-muted-foreground" />
            抢购参数
          </h3>
          <div>
            <label className="block text-[11px] text-muted-foreground mb-1">OVH 账户 *</label>
            <AccountSelect value={accountId} onChange={onAccountId} />
            {account?.zone === "US" && selectedDCs.some((c) => plan.datacenters.find((d) => d.code === c)?.outsideUnitedStates) && (
              <p className="text-[11px] text-amber-700 dark:text-amber-300 mt-1">
                美国官网说明：美国以外机房交付后不能再改配或升级。
              </p>
            )}
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-[11px] text-muted-foreground mb-1">每个机房数量</label>
              <Input type="number" min={1} max={20} value={quantity} onChange={(e) => setQuantity(e.target.value)} />
            </div>
            <div>
              <label className="block text-[11px] text-muted-foreground mb-1">重试间隔（秒）</label>
              <Input type="number" min={10} value={retryInterval} onChange={(e) => setRetryInterval(e.target.value)} />
            </div>
          </div>
        </div>
      </div>

      <DialogFooter className="border-t border-border pt-4 -mx-6 px-6">
        <div className="mr-auto text-[12px] text-muted-foreground">
          {selectedDCs.length > 0 ? `将创建 ${totalTasks} 个任务（${selectedDCs.length} 机房 × ${qty}）` : "请选有货机房"}
        </div>
        <Button variant="outline" onClick={onClose} disabled={create.isPending}>
          关闭
        </Button>
        <Button
          variant="outline"
          disabled={addMon.isPending || create.isPending}
          onClick={() =>
            addMon.mutate({
              planCode: plan.planCode,
              ovhSubsidiary: account?.zone || "US",
              datacenters: selectedDCs.length > 0 ? selectedDCs : instock.map((d) => d.code),
              monitorLinux: osTrack === "linux",
              monitorWindows: osTrack === "windows",
              notifyAvailable: true,
              notifyUnavailable: false,
            })
          }
        >
          <Bell className="w-4 h-4" />
          加入补货监控
        </Button>
        <Button
          disabled={selectedDCs.length === 0 || create.isPending || !accountId || !zoneOk || !osImage}
          onClick={async () => {
            if (!zoneOk) {
              toast.error("请选择 OVH 账户");
              return;
            }
            const orderPlanByDc: Record<string, string> = {};
            for (const dc of plan.datacenters) {
              if (dc.orderPlanCode) orderPlanByDc[dc.code] = dc.orderPlanCode;
            }
            const result = await create.mutateAsync({
              account_id: accountId,
              planCode: plan.planCode,
              datacenters: selectedDCs,
              quantity: qty,
              retryInterval: Number(retryInterval) || 30,
              productKind: "vps",
              subsidiary: account.zone,
              orderPlanByDc,
              osTrack,
              osImage,
              backupPlan,
              dcNames,
            });
            if (result.success > 0) {
              toast.success(`已创建 ${result.success}/${result.total} 个 VPS 任务`);
              onClose();
            }
            if (result.failed > 0) toast.error(`${result.failed} 个任务创建失败`);
          }}
        >
          {create.isPending ? (
            <>
              <Loader2 className="w-4 h-4 animate-spin" />
              创建中…
            </>
          ) : (
            <>
              <ShoppingCart className="w-4 h-4" />
              {selectedDCs.length > 0 ? `创建 ${totalTasks} 个任务` : "创建抢购任务"}
            </>
          )}
        </Button>
      </DialogFooter>
    </>
  );
}

function DcPick({
  dc,
  track,
  selected,
  onToggle,
}: {
  dc: VpsStockDC;
  track: "linux" | "windows";
  selected: boolean;
  onToggle: () => void;
}) {
  const ok = trackAvailable(dc, track);
  return (
    <button
      type="button"
      disabled={!ok}
      onClick={onToggle}
      className={
        "text-left border rounded-xl px-3 py-2 flex items-center justify-between transition-colors " +
        (!ok
          ? "opacity-40 cursor-not-allowed border-border"
          : selected
            ? "border-foreground bg-foreground text-background"
            : "border-border hover:bg-secondary/50")
      }
    >
      <div className="min-w-0">
        <div className="text-[12px] font-bold font-mono">{shortDcLabel(dc)}</div>
        <div className={"text-[10px] truncate " + (selected && ok ? "text-background/70" : "text-muted-foreground")}>
          {dc.name}
        </div>
      </div>
      <StatusDot tone={ok ? "success" : "danger"} size="sm" pulse={ok && !selected} />
    </button>
  );
}
