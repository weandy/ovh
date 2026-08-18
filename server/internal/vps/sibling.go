package vps

import (
	"sort"
	"strings"
)

const (
	ContinentEurope       = "europe"
	ContinentNorthAmerica = "north_america"
	ContinentAsiaOceania  = "asia_oceania"
)

type LocationGroup struct {
	Canonical   string
	InvoiceName string
	Siblings    []CatalogPlan
}

func CanonicalPlanCode(planCode string) string {
	s := strings.TrimSpace(planCode)
	s = strings.TrimSuffix(s, "-eu")
	s = strings.TrimSuffix(s, "-ca")
	return s
}

func LocationSuffix(planCode string) string {
	s := strings.TrimSpace(planCode)
	switch {
	case strings.HasSuffix(s, "-eu"):
		return "eu"
	case strings.HasSuffix(s, "-ca"):
		return "ca"
	default:
		return ""
	}
}

func GroupLocationVariants(plans []CatalogPlan) []LocationGroup {
	idx := map[string]int{}
	var out []LocationGroup
	for _, p := range plans {
		fam := ClassifyPlan(p.PlanCode)
		if fam != Family2027 && fam != Family2027LZ {
			continue
		}
		can := CanonicalPlanCode(p.PlanCode)
		i, ok := idx[can]
		if !ok {
			out = append(out, LocationGroup{Canonical: can, InvoiceName: p.InvoiceName})
			i = len(out) - 1
			idx[can] = i
		}
		if out[i].InvoiceName == "" {
			out[i].InvoiceName = p.InvoiceName
		}
		out[i].Siblings = append(out[i].Siblings, p)
	}
	return out
}

func FilterStorefrontDCs(catalogNames []string, homeRules []DatacenterStock, accountSub string) []string {
	allowed := map[string]bool{}
	for _, d := range homeRules {
		if d.Name != "" {
			allowed[strings.ToUpper(d.Name)] = true
		}
		if d.Code != "" {
			allowed[strings.ToUpper(d.Code)] = true
		}
	}
	var out []string
	seen := map[string]bool{}
	useRules := len(homeRules) > 0
	for _, n := range catalogNames {
		k := strings.ToUpper(n)
		if n == "" || seen[k] {
			continue
		}
		if useRules && !allowed[k] {
			continue
		}
		if !StorefrontAllowsDC(accountSub, n) {
			continue
		}
		seen[k] = true
		out = append(out, n)
	}
	if useRules {
		for _, d := range homeRules {
			if d.Name == "" || seen[strings.ToUpper(d.Name)] {
				continue
			}
			if !StorefrontAllowsDC(accountSub, d.Name) {
				continue
			}
			seen[strings.ToUpper(d.Name)] = true
			out = append(out, d.Name)
		}
	}
	sort.Strings(out)
	return out
}

func StorefrontAllowsDC(accountSub, dcName string) bool {
	if RegionOfSubsidiary(accountSub) == RegionUS && ContinentOfDC(dcName, "") == ContinentAsiaOceania {
		return false
	}
	return true
}

func CatalogNamesFromGroup(g LocationGroup) []string {
	seen := map[string]bool{}
	var names []string
	for _, p := range g.Siblings {
		for _, n := range CatalogDatacenterNames(p) {
			k := strings.ToUpper(n)
			if n == "" || seen[k] {
				continue
			}
			seen[k] = true
			names = append(names, n)
		}
	}
	sort.Strings(names)
	return names
}

func AssignOrderPlanCode(dcName string, siblings []CatalogPlan) string {
	want := strings.ToUpper(dcName)
	var fallback string
	for _, p := range siblings {
		if fallback == "" || LocationSuffix(p.PlanCode) == "" {
			fallback = p.PlanCode
		}
		for _, n := range CatalogDatacenterNames(p) {
			if strings.ToUpper(n) == want {
				return p.PlanCode
			}
		}
	}
	// 外区 API 补进来的机房：按大洲落到对应 SKU
	cont := ContinentOfDC(dcName, "")
	prefer := ""
	switch cont {
	case ContinentEurope:
		prefer = "eu"
	case ContinentAsiaOceania:
		prefer = "ca"
	case ContinentNorthAmerica:
		if strings.EqualFold(dcName, "BHS") {
			prefer = "ca"
		}
	}
	if prefer != "" {
		for _, p := range siblings {
			if LocationSuffix(p.PlanCode) == prefer {
				return p.PlanCode
			}
		}
	}
	return fallback
}

func ContinentOfDC(name, code string) string {
	u := strings.ToUpper(name + " " + code)
	switch {
	case strings.Contains(u, "SGP"), strings.Contains(u, "SYD"), strings.Contains(u, "YNM"),
		strings.Contains(u, "AP-"), strings.Contains(u, "ASIA"), strings.Contains(u, "OCEANIA"),
		strings.Contains(u, "MUM"):
		return ContinentAsiaOceania
	case strings.Contains(u, "US-"), strings.Contains(u, "BHS"), strings.Contains(u, "CA-EAST"),
		strings.Contains(u, "CANADA"), strings.Contains(u, "HIL"), strings.Contains(u, "VIN"):
		return ContinentNorthAmerica
	default:
		return ContinentEurope
	}
}

func OutsideUnitedStates(name, code string) bool {
	u := strings.ToUpper(name + " " + code)
	return !strings.Contains(u, "US-") && !strings.Contains(u, "US EAST") && !strings.Contains(u, "US WEST")
}

func CartRegion(orderPlanCode, dcName string) string {
	switch LocationSuffix(orderPlanCode) {
	case "eu":
		return "europe"
	case "ca":
		return "canada"
	}
	if strings.HasPrefix(strings.ToUpper(dcName), "US-") {
		return "united_states"
	}
	if strings.EqualFold(dcName, "BHS") {
		return "canada"
	}
	return "europe"
}

// ExtraRuleQueries 不再跨区拉别人货架上的机房。
// 美国店铺能卖的位置以美国 API 自己的 sibling rule 为准（欧洲 + 美国 + BHS，没有亚洲）。
func ExtraRuleQueries(canonical, accountSub string) [][2]string {
	_ = canonical
	_ = accountSub
	return nil
}
