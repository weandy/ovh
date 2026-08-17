import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { qk } from "@/lib/query";

export interface VpsStockDC {
  name: string;
  code: string;
  headline: string;
  linux: string;
  windows: string;
  daysBeforeDelivery: number;
}

export interface VpsStockPlan {
  planCode: string;
  invoiceName: string;
  supportsWindows: boolean;
  isLocalZone: boolean;
  monthlyPrice?: number;
  currency: string;
  osImages?: string[];
  stockError?: string;
  datacenters: VpsStockDC[];
}

export interface VpsStockResponse {
  subsidiary: string;
  currency: string;
  plans: VpsStockPlan[];
}

export function isVpsUnavailable(status: string | undefined): boolean {
  return !status || status === "out-of-stock" || status === "out-of-stock-preorder-allowed";
}

export function trackAvailable(dc: VpsStockDC, track: "linux" | "windows"): boolean {
  return !isVpsUnavailable(track === "windows" ? dc.windows : dc.linux);
}

export function planHasBuyableStock(plan: VpsStockPlan): boolean {
  if (plan.stockError) return false;
  return plan.datacenters.some(
    (dc) => trackAvailable(dc, "linux") || (plan.supportsWindows && trackAvailable(dc, "windows"))
  );
}

export function shortDcLabel(dc: VpsStockDC): string {
  const parts = (dc.code || dc.name || "").split("-");
  return (parts[parts.length - 1] || dc.name || "?").toUpperCase();
}

export function useVPSStock(subsidiary: string) {
  return useQuery({
    queryKey: qk.vpsStock(subsidiary),
    queryFn: async () =>
      (await api.get<VpsStockResponse>("/vps-stock", { params: { ovhSubsidiary: subsidiary } })).data,
    enabled: !!subsidiary,
    staleTime: 30_000,
  });
}
