package vps

import "strings"

type StockDC struct {
	Name     string `json:"name"`
	Code     string `json:"code"`
	Headline string `json:"headline"`
	Linux    string `json:"linux"`
	Windows  string `json:"windows"`
	Days     int    `json:"daysBeforeDelivery"`
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
