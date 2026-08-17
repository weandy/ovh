package handlers

import (
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"

	"github.com/ovh-buy/server/internal/app"
	"github.com/ovh-buy/server/internal/vps"
)

func GetVPSCatalog(state *app.State) gin.HandlerFunc {
	return func(c *gin.Context) {
		sub := strings.ToUpper(strings.TrimSpace(c.Query("ovhSubsidiary")))
		if sub == "" {
			sub = "IE"
		}
		plans, err := vps.LoadPlans(state, sub)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "拉取 VPS catalog 失败: " + err.Error()})
			return
		}
		var ruleDCs []vps.DatacenterStock
		for _, p := range plans {
			fam := vps.ClassifyPlan(p.PlanCode)
			if fam != vps.Family2027 && fam != vps.Family2027LZ {
				continue
			}
			dcs, err := vps.FetchRuleStock(state, p.PlanCode, sub)
			if err != nil {
				continue
			}
			ruleDCs = append(ruleDCs, dcs...)
		}
		c.JSON(http.StatusOK, gin.H{
			"subsidiary": sub,
			"families":   vps.BuildFamilies(plans, ruleDCs),
		})
	}
}

type vpsStockPlan struct {
	PlanCode        string        `json:"planCode"`
	InvoiceName     string        `json:"invoiceName"`
	SupportsWindows bool          `json:"supportsWindows"`
	IsLocalZone     bool          `json:"isLocalZone"`
	MonthlyPrice    *float64      `json:"monthlyPrice,omitempty"`
	Currency        string        `json:"currency"`
	OSImages        []string      `json:"osImages,omitempty"`
	StockError      string        `json:"stockError,omitempty"`
	Datacenters     []vps.StockDC `json:"datacenters"`
}

func GetVPSStock(state *app.State) gin.HandlerFunc {
	return func(c *gin.Context) {
		sub := strings.ToUpper(strings.TrimSpace(c.Query("ovhSubsidiary")))
		if sub == "" {
			sub = "IE"
		}
		plans, err := vps.LoadPlans(state, sub)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "拉取 VPS catalog 失败: " + err.Error()})
			return
		}
		var targets []vps.CatalogPlan
		for _, p := range plans {
			fam := vps.ClassifyPlan(p.PlanCode)
			if fam == vps.Family2027 || fam == vps.Family2027LZ {
				targets = append(targets, p)
			}
		}
		out := make([]vpsStockPlan, len(targets))
		var wg sync.WaitGroup
		for i, p := range targets {
			wg.Add(1)
			go func(i int, p vps.CatalogPlan) {
				defer wg.Done()
				row := vpsStockPlan{
					PlanCode:        p.PlanCode,
					InvoiceName:     p.InvoiceName,
					SupportsWindows: vps.SupportsWindows(p),
					IsLocalZone:     vps.ClassifyPlan(p.PlanCode) == vps.Family2027LZ,
					Currency:        vps.CurrencyForSubsidiary(sub),
					OSImages:        nil,
					Datacenters:     []vps.StockDC{},
				}
				for _, c := range p.Configurations {
					if c.Name == "vps_os" {
						row.OSImages = c.Values
					}
				}
				if price, ok := vps.MonthlyPrice(p); ok {
					row.MonthlyPrice = &price
				}
				dcs, err := vps.FetchRuleStock(state, p.PlanCode, sub)
				if err != nil {
					row.StockError = err.Error()
				} else {
					row.Datacenters = vps.BuildPlanStock(p.PlanCode, dcs).Datacenters
				}
				out[i] = row
			}(i, p)
		}
		wg.Wait()
		c.JSON(http.StatusOK, gin.H{
			"subsidiary": sub,
			"currency":   vps.CurrencyForSubsidiary(sub),
			"plans":      out,
		})
	}
}
