import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { qk } from "@/lib/query";

export interface VpsCatalogDatacenter {
  name: string;
  code: string;
}

export interface VpsCatalogPlan {
  planCode: string;
  invoiceName: string;
  supportsWindows: boolean;
  isLocalZone: boolean;
  datacenters: VpsCatalogDatacenter[];
  osImages?: string[];
}

export interface VpsCatalogFamily {
  id: string;
  label: string;
  plans: VpsCatalogPlan[];
}

export interface VpsCatalogResponse {
  subsidiary: string;
  families: VpsCatalogFamily[];
}

export function useVPSCatalog(subsidiary: string) {
  return useQuery({
    queryKey: qk.vpsCatalog(subsidiary),
    queryFn: async () =>
      (await api.get<VpsCatalogResponse>("/vps-catalog", { params: { ovhSubsidiary: subsidiary } })).data,
    enabled: !!subsidiary,
    staleTime: 5 * 60_000,
  });
}
