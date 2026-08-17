package purchase

import (
	"fmt"
	"strings"

	"github.com/ovh-buy/server/internal/app"
	"github.com/ovh-buy/server/internal/numconv"
	"github.com/ovh-buy/server/internal/types"
)

type OVHCart interface {
	Get(url string, res interface{}) error
	Post(url string, req, res interface{}) error
	Delete(url string, res interface{}) error
}

func PurchaseVPS(state *app.State, item *types.QueueItem) bool {
	if item.VpsSpec == nil || item.VpsSpec.DatacenterName == "" {
		recordFailure(state, item, "VPS 任务缺少 vpsSpec.datacenterName")
		return false
	}
	client, err := state.OVH.ClientFor(item.AccountID)
	if err != nil {
		recordFailure(state, item, "取 OVH client 失败: "+err.Error())
		return false
	}
	return purchaseVPSWith(state, client, item)
}

func purchaseVPSWith(state *app.State, client OVHCart, item *types.QueueItem) bool {
	spec := item.VpsSpec
	subsidiary := spec.Subsidiary
	if subsidiary == "" {
		if acc, ok := state.FindAccount(item.AccountID); ok {
			subsidiary = acc.Zone
		}
	}
	if subsidiary == "" {
		recordFailure(state, item, "无法确定 ovhSubsidiary")
		return false
	}

	var cart map[string]interface{}
	if err := client.Post("/order/cart", map[string]interface{}{"ovhSubsidiary": subsidiary}, &cart); err != nil {
		recordFailure(state, item, err.Error())
		return false
	}
	cartID, _ := cart["cartId"].(string)
	if cartID == "" {
		recordFailure(state, item, "购物车未返回 cartId")
		return false
	}
	success := false
	defer func() {
		if !success {
			_ = client.Delete("/order/cart/"+cartID, nil)
		}
	}()

	if err := client.Post("/order/cart/"+cartID+"/assign", map[string]interface{}{}, nil); err != nil {
		recordFailure(state, item, "assign cart: "+err.Error())
		return false
	}

	var offers []map[string]interface{}
	if err := client.Get("/order/cart/"+cartID+"/vps", &offers); err != nil {
		recordFailure(state, item, "list vps offers: "+err.Error())
		return false
	}
	duration, pricingMode := pickVPSOffer(offers, item.PlanCode)

	var added map[string]interface{}
	if err := client.Post("/order/cart/"+cartID+"/vps", map[string]interface{}{
		"planCode":    item.PlanCode,
		"duration":    duration,
		"pricingMode": pricingMode,
		"quantity":    1,
	}, &added); err != nil {
		recordFailure(state, item, "add vps: "+err.Error())
		return false
	}
	itemID, ok := numconv.ToInt64(added["itemId"])
	if !ok || itemID == 0 {
		recordFailure(state, item, fmt.Sprintf("无法解析 VPS itemId: %v", added))
		return false
	}

	osImage := spec.OSImage
	if osImage == "" {
		osImage = "Ubuntu 24.04"
	}
	cfgs := []struct{ label, value string }{
		{"vps_datacenter", spec.DatacenterName},
		{"vps_os", osImage},
		{"region", regionForVPS(spec.DatacenterName)},
	}
	var required []map[string]interface{}
	if err := client.Get(fmt.Sprintf("/order/cart/%s/item/%d/requiredConfiguration", cartID, itemID), &required); err == nil {
		for _, r := range required {
			label, _ := r["label"].(string)
			if label == "infrastructure" {
				infra := spec.Infrastructure
				if infra == "" {
					infra = "production"
				}
				cfgs = append(cfgs, struct{ label, value string }{"infrastructure", infra})
			}
		}
	}
	for _, c := range cfgs {
		if err := client.Post(fmt.Sprintf("/order/cart/%s/item/%d/configuration", cartID, itemID),
			map[string]interface{}{"label": c.label, "value": c.value}, nil); err != nil {
			recordFailure(state, item, fmt.Sprintf("config %s: %s", c.label, err.Error()))
			return false
		}
	}

	var optList []map[string]interface{}
	if err := client.Get(fmt.Sprintf("/order/cart/%s/vps/options?planCode=%s", cartID, item.PlanCode), &optList); err != nil {
		recordFailure(state, item, "list vps options: "+err.Error())
		return false
	}
	wanted, err := resolveVPSAddons(item, optList)
	if err != nil {
		recordFailure(state, item, err.Error())
		return false
	}
	for _, pc := range wanted {
		if err := client.Post("/order/cart/"+cartID+"/vps/options", map[string]interface{}{
			"itemId":      itemID,
			"planCode":    pc,
			"duration":    duration,
			"pricingMode": pricingMode,
			"quantity":    1,
		}, nil); err != nil {
			recordFailure(state, item, "add option "+pc+": "+err.Error())
			return false
		}
	}

	var checkout map[string]interface{}
	if err := client.Post("/order/cart/"+cartID+"/checkout", map[string]interface{}{
		"autoPayWithPreferredPaymentMethod": false,
		"waiveRetractationPeriod":           true,
	}, &checkout); err != nil {
		recordFailure(state, item, "checkout: "+err.Error())
		return false
	}
	success = true
	orderID := numconv.ToString(checkout["orderId"])
	orderURL, _ := checkout["url"].(string)
	recordSuccess(state, item, orderID, orderURL, "", nil)
	state.Logger.Info(fmt.Sprintf("VPS 下单成功 %s @ %s order=%s", item.PlanCode, spec.DatacenterName, orderID), "purchase")
	return true
}

func regionForVPS(datacenterName string) string {
	if strings.HasPrefix(datacenterName, "US-") {
		return "united_states"
	}
	if datacenterName == "BHS" {
		return "canada"
	}
	return "europe"
}

func pickVPSOffer(offers []map[string]interface{}, planCode string) (duration, pricingMode string) {
	duration, pricingMode = "P1M", "default"
	var fallback map[string]interface{}
	for _, o := range offers {
		if o["planCode"] != planCode {
			continue
		}
		if fallback == nil {
			fallback = o
		}
		if d, _ := o["duration"].(string); d == "P1M" {
			if d != "" {
				duration = d
			}
			if pm, _ := o["pricingMode"].(string); pm != "" {
				pricingMode = pm
			}
			return
		}
	}
	if fallback != nil {
		if d, _ := fallback["duration"].(string); d != "" {
			duration = d
		}
		if pm, _ := fallback["pricingMode"].(string); pm != "" {
			pricingMode = pm
		}
	}
	return
}

func resolveVPSAddons(item *types.QueueItem, optList []map[string]interface{}) ([]string, error) {
	available := map[string]bool{}
	for _, o := range optList {
		if pc, _ := o["planCode"].(string); pc != "" {
			available[pc] = true
		}
	}
	track := "linux"
	backup := "1"
	if item.VpsSpec != nil {
		if item.VpsSpec.OSTrack != "" {
			track = item.VpsSpec.OSTrack
		}
		if item.VpsSpec.BackupPlan != "" {
			backup = item.VpsSpec.BackupPlan
		}
	}
	// 从已返回的 options 里按 family 习惯挑，不依赖完整 catalog
	wanted := pickAddonsFromList(item.PlanCode, track, backup, available)
	hasStorage := false
	for _, w := range wanted {
		if strings.Contains(w, "option-storage-") {
			hasStorage = true
		}
		if !available[w] {
			return nil, fmt.Errorf("缺少必选 addon %s（已取消下单）", w)
		}
	}
	if !hasStorage {
		return nil, fmt.Errorf("缺少必选 addon option-storage（已取消下单）")
	}
	return wanted, nil
}

func pickAddonsFromList(planCode, track, backup string, available map[string]bool) []string {
	osPick := "option-linux"
	if track == "windows" {
		osPick = ""
		for pc := range available {
			if strings.Contains(strings.ToLower(pc), "windows") {
				osPick = pc
				break
			}
		}
	}
	storage := ""
	for pc := range available {
		if strings.Contains(pc, "option-storage-") {
			storage = pc
			if strings.Contains(planCode, ".LZ") && strings.Contains(pc, "remote") {
				break
			}
			if !strings.Contains(planCode, ".LZ") && strings.Contains(pc, "local") {
				break
			}
		}
	}
	bakNeedle := "-1-"
	if backup == "7" {
		bakNeedle = "-7-"
	}
	bak := ""
	var bakFallback string
	for pc := range available {
		if !strings.Contains(pc, "auto-backup") {
			continue
		}
		if bakFallback == "" {
			bakFallback = pc
		}
		if strings.Contains(pc, bakNeedle) {
			bak = pc
		}
	}
	if bak == "" {
		bak = bakFallback
	}
	out := []string{}
	if osPick != "" {
		out = append(out, osPick)
	}
	if storage != "" {
		out = append(out, storage)
	}
	if bak != "" {
		out = append(out, bak)
	}
	return out
}
