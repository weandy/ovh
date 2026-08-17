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
  region?: string;
  subsidiary: string;
  families: VpsCatalogFamily[];
}

export function useVPSCatalog(region: string, accountZone?: string) {
  return useQuery({
    queryKey: qk.vpsCatalog(region, accountZone),
    queryFn: async () =>
      (
        await api.get<VpsCatalogResponse>("/vps-catalog", {
          params: { region, accountZone: accountZone || undefined },
        })
      ).data,
    enabled: !!region,
    staleTime: 5 * 60_000,
  });
}
