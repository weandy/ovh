package handlers

import (
	"net/http"
	"strings"

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
