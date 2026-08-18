package vps

import "strings"

type StockDC struct {
	Name                string `json:"name"`
	Code                string `json:"code"`
	Headline            string `json:"headline"`
	Linux               string `json:"linux"`
	Windows             string `json:"windows"`
	Days                int    `json:"daysBeforeDelivery"`
	OrderPlanCode       string `json:"orderPlanCode,omitempty"`
	Continent           string `json:"continent,omitempty"`
	OutsideUnitedStates bool   `json:"outsideUnitedStates,omitempty"`
}

type PlanStock struct {
	PlanCode    string    `json:"planCode"`
	StockError  string    `json:"stockError,omitempty"`
	Datacenters []StockDC `json:"datacenters"`
}

func BuildPlanStock(planCode string, dcs []DatacenterStock) PlanStock {
	out := PlanStock{PlanCode: planCode, Datacenters: []StockDC{}}
	for _, d := range dcs {
		out.Datacenters = append(out.Datacenters, StockDC{
			Name:     d.Name,
			Code:     d.Code,
			Headline: d.Status,
			Linux:    d.LinuxStatus,
			Windows:  d.WindowsStatus,
			Days:     d.Days,
		})
	}
	return out
}

func FindStockDC(dcs []StockDC, code string) StockDC {
	for _, d := range dcs {
		if d.Code == code {
			return d
		}
	}
	return StockDC{}
}

func FindStockDCByName(dcs []StockDC, name string) StockDC {
	want := strings.ToUpper(name)
	for _, d := range dcs {
		if strings.ToUpper(d.Name) == want {
			return d
		}
	}
	return StockDC{}
}

func MergePlanStock(planCode string, catalogNames []string, rule []DatacenterStock) PlanStock {
	byKey := map[string]DatacenterStock{}
	for _, d := range rule {
		if d.Name != "" {
			byKey[strings.ToUpper(d.Name)] = d
		}
		if d.Code != "" {
			byKey[strings.ToUpper(d.Code)] = d
		}
	}
	seen := map[string]bool{}
	out := PlanStock{PlanCode: planCode, Datacenters: []StockDC{}}
	appendDC := func(d DatacenterStock, fallbackName string) {
		name := d.Name
		if name == "" {
			name = fallbackName
		}
		key := strings.ToUpper(name)
		if key == "" || seen[key] {
			return
		}
		seen[key] = true
		code := d.Code
		if code == "" {
			code = strings.ToLower(name)
		}
		out.Datacenters = append(out.Datacenters, StockDC{
			Name:                name,
			Code:                code,
			Headline:            d.Status,
			Linux:               d.LinuxStatus,
			Windows:             d.WindowsStatus,
			Days:                d.Days,
			Continent:           ContinentOfDC(name, code),
			OutsideUnitedStates: OutsideUnitedStates(name, code),
		})
	}
	for _, name := range catalogNames {
		if name == "" {
			continue
		}
		if d, ok := byKey[strings.ToUpper(name)]; ok {
			appendDC(d, name)
			continue
		}
		appendDC(DatacenterStock{Name: name}, name)
	}
	for _, d := range rule {
		appendDC(d, d.Name)
	}
	return out
}

func AnnotateOrderPlans(dcs []StockDC, siblings []CatalogPlan) []StockDC {
	for i := range dcs {
		dcs[i].OrderPlanCode = AssignOrderPlanCode(dcs[i].Name, siblings)
		dcs[i].Continent = ContinentOfDC(dcs[i].Name, dcs[i].Code)
		dcs[i].OutsideUnitedStates = OutsideUnitedStates(dcs[i].Name, dcs[i].Code)
	}
	return dcs
}

func PlanHasBuyableStock(supportsWindows bool, dcs []DatacenterStock) bool {
	for _, d := range dcs {
		if TrackAvailable(d, "linux") {
			return true
		}
		if supportsWindows && TrackAvailable(d, "windows") {
			return true
		}
	}
	return false
}

type CatalogPricing struct {
	Price        int64  `json:"price"`
	Mode         string `json:"mode"`
	Interval     int    `json:"interval"`
	IntervalUnit string `json:"intervalUnit"`
}

func MonthlyPrice(p CatalogPlan) (float64, bool) {
	for _, pr := range p.Pricings {
		if pr.Mode == "default" && pr.Interval == 1 && strings.EqualFold(pr.IntervalUnit, "month") && pr.Price > 0 {
			return float64(pr.Price) / 1e8, true
		}
	}
	return 0, false
}

func CurrencyForSubsidiary(sub string) string {
	switch strings.ToUpper(sub) {
	case "US", "ASIA":
		return "USD"
	case "GB":
		return "GBP"
	case "CA", "QC":
		return "CAD"
	case "PL":
		return "PLN"
	case "SG":
		return "SGD"
	case "AU":
		return "AUD"
	case "IN":
		return "INR"
	default:
		return "EUR"
	}
}
