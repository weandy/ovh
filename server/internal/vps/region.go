package vps

import "strings"

// OVH 只有三个 API 区。子公司挂在区下面，货架和购物车不能跨区。
//
//	US → api.us.ovhcloud.com     子公司 US
//	IE → eu.api.ovh.com          欧洲子公司 FR/IE/DE/GB…
//	CA → ca.api.ovh.com          加拿大 / 亚太 CA/QC/ASIA/SG/AU/IN…
const (
	RegionUS = "US"
	RegionIE = "IE"
	RegionCA = "CA"
)

type ListQuery struct {
	Region            string
	CatalogSubsidiary string
	AccountCanBuy     bool
}

func RegionOfSubsidiary(subsidiary string) string {
	s := strings.ToUpper(strings.TrimSpace(subsidiary))
	if s == "" {
		return RegionUS
	}
	switch s {
	case "US":
		return RegionUS
	case "CA", "QC", "ASIA", "SG", "AU", "IN", "MA", "TN", "SN", "WS":
		return RegionCA
	default:
		return RegionIE
	}
}

func NormalizeRegion(raw string) string {
	s := strings.ToUpper(strings.TrimSpace(raw))
	switch s {
	case "", RegionUS, "OVH-US":
		return RegionUS
	case RegionCA, "OVH-CA":
		return RegionCA
	case RegionIE, "EU", "OVH-EU":
		return RegionIE
	default:
		return RegionOfSubsidiary(s)
	}
}

func CatalogSubsidiary(region, accountZone string) string {
	r := NormalizeRegion(region)
	def := r
	if r != RegionUS && r != RegionCA && r != RegionIE {
		def = RegionUS
	}
	zone := strings.ToUpper(strings.TrimSpace(accountZone))
	if zone != "" && RegionOfSubsidiary(zone) == r {
		return zone
	}
	return def
}

func SameRegion(a, b string) bool {
	if strings.TrimSpace(a) == "" || strings.TrimSpace(b) == "" {
		return false
	}
	return RegionOfSubsidiary(a) == RegionOfSubsidiary(b)
}

func CrossRegionError(accountZone, productSub string) string {
	return RegionOfSubsidiary(accountZone) + " 区账户不能购买 " + RegionOfSubsidiary(productSub) + " 区产品"
}

func ResolveListQuery(region, accountZone string) ListQuery {
	accZone := strings.ToUpper(strings.TrimSpace(accountZone))
	var accRegion string
	if accZone != "" {
		accRegion = RegionOfSubsidiary(accZone)
	}
	r := NormalizeRegion(region)
	if strings.TrimSpace(region) == "" && accRegion != "" {
		r = accRegion
	}
	return ListQuery{
		Region:            r,
		CatalogSubsidiary: CatalogSubsidiary(r, accZone),
		AccountCanBuy:     accRegion != "" && accRegion == r,
	}
}
