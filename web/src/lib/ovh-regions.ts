/** OVH 三个 API 区。子公司挂在区下面，货架和下单不能跨区。 */
export type OvhRegion = "US" | "IE" | "CA";

export const OVH_REGIONS: { code: OvhRegion; label: string; hint: string }[] = [
  { code: "US", label: "US 美国", hint: "api.us.ovhcloud.com · 子公司 US" },
  { code: "IE", label: "IE 欧洲", hint: "eu.api.ovh.com · FR / IE / DE / GB …" },
  { code: "CA", label: "CA 加拿大及亚太", hint: "ca.api.ovh.com · CA / SG / AU / ASIA …" },
];

const CA_SUBS = new Set(["CA", "QC", "ASIA", "SG", "AU", "IN", "MA", "TN", "SN", "WS"]);

export function regionOfSubsidiary(subsidiary?: string | null): OvhRegion {
  const s = (subsidiary || "").trim().toUpperCase();
  if (!s || s === "US") return "US";
  if (CA_SUBS.has(s)) return "CA";
  return "IE";
}

export function sameRegion(a?: string | null, b?: string | null): boolean {
  if (!a?.trim() || !b?.trim()) return false;
  return regionOfSubsidiary(a) === regionOfSubsidiary(b);
}

export function catalogSubsidiary(region: OvhRegion, accountZone?: string | null): string {
  const zone = (accountZone || "").trim().toUpperCase();
  if (zone && regionOfSubsidiary(zone) === region) return zone;
  return region;
}

export function regionLabel(region: OvhRegion): string {
  return OVH_REGIONS.find((r) => r.code === region)?.label || region;
}
