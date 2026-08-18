package handlers

import (
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"

	"github.com/ovh-buy/server/internal/app"
	"github.com/ovh-buy/server/internal/vps"
)

func resolveVPSListQuery(c *gin.Context) vps.ListQuery {
	region := strings.TrimSpace(c.Query("region"))
	accountZone := strings.ToUpper(strings.TrimSpace(c.Query("accountZone")))
	legacy := strings.ToUpper(strings.TrimSpace(c.Query("ovhSubsidiary")))
	if region == "" && legacy != "" {
		return vps.ResolveListQuery(vps.RegionOfSubsidiary(legacy), firstNonEmpty(accountZone, legacy))
	}
	return vps.ResolveListQuery(region, accountZone)
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func GetVPSCatalog(state *app.State) gin.HandlerFunc {
	return func(c *gin.Context) {
		q := resolveVPSListQuery(c)
		sub := q.CatalogSubsidiary
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
			"region":     q.Region,
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
		q := resolveVPSListQuery(c)
		sub := q.CatalogSubsidiary
		plans, err := vps.LoadPlans(state, sub)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "拉取 VPS catalog 失败: " + err.Error()})
			return
		}
		groups := vps.GroupLocationVariants(plans)
		out := make([]vpsStockPlan, len(groups))
		var wg sync.WaitGroup
		for i, g := range groups {
			wg.Add(1)
			go func(i int, g vps.LocationGroup) {
				defer wg.Done()
				base := g.Siblings[0]
				for _, s := range g.Siblings {
					if vps.LocationSuffix(s.PlanCode) == "" {
						base = s
						break
					}
				}
				row := vpsStockPlan{
					PlanCode:        g.Canonical,
					InvoiceName:     g.InvoiceName,
					SupportsWindows: vps.SupportsWindows(base),
					IsLocalZone:     vps.ClassifyPlan(g.Canonical) == vps.Family2027LZ,
					Currency:        vps.CurrencyForSubsidiary(sub),
					OSImages:        nil,
					Datacenters:     []vps.StockDC{},
				}
				for _, c := range base.Configurations {
					if c.Name == "vps_os" {
						row.OSImages = c.Values
					}
				}
				if price, ok := vps.MonthlyPrice(base); ok {
					row.MonthlyPrice = &price
				}
				var (
					rule []vps.DatacenterStock
					errs []string
					mu   sync.Mutex
					rwg  sync.WaitGroup
				)
				type q struct{ plan, sub string }
				queries := []q{}
				seen := map[string]bool{}
				addQ := func(plan, s string) {
					k := plan + "|" + s
					if plan == "" || s == "" || seen[k] {
						return
					}
					seen[k] = true
					queries = append(queries, q{plan, s})
				}
				for _, s := range g.Siblings {
					addQ(s.PlanCode, sub)
				}
				for _, extra := range vps.ExtraRuleQueries(g.Canonical, sub) {
					addQ(extra[0], extra[1])
				}
				for _, query := range queries {
					rwg.Add(1)
					go func(query q) {
						defer rwg.Done()
						dcs, err := vps.FetchRuleStock(state, query.plan, query.sub)
						mu.Lock()
						defer mu.Unlock()
						if err != nil {
							errs = append(errs, query.plan+"@"+query.sub+": "+err.Error())
							return
						}
						rule = append(rule, dcs...)
					}(query)
				}
				rwg.Wait()
				names := vps.FilterStorefrontDCs(vps.CatalogNamesFromGroup(g), rule, sub)
				row.Datacenters = vps.AnnotateOrderPlans(vps.MergePlanStock(g.Canonical, names, rule).Datacenters, g.Siblings)
				if len(row.Datacenters) == 0 && len(errs) > 0 {
					row.StockError = strings.Join(errs, "; ")
				}
				out[i] = row
			}(i, g)
		}
		wg.Wait()
		c.JSON(http.StatusOK, gin.H{
			"region":        q.Region,
			"subsidiary":    sub,
			"accountCanBuy": q.AccountCanBuy,
			"currency":      vps.CurrencyForSubsidiary(sub),
			"plans":         out,
		})
	}
}
