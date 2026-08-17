package vps

import "encoding/json"

type CatalogPlan struct {
	PlanCode        string              `json:"planCode"`
	InvoiceName     string              `json:"invoiceName"`
	Configurations  []CatalogConfig     `json:"configurations"`
	AddonFamilies   []CatalogAddonFamily `json:"addonFamilies"`
}

type CatalogConfig struct {
	Name   string   `json:"name"`
	Values []string `json:"values"`
}

type CatalogAddonFamily struct {
	Name      string   `json:"name"`
	Mandatory bool     `json:"mandatory"`
	Exclusive bool     `json:"exclusive"`
	Addons    []string `json:"addons"`
}

type Family struct {
	ID     string      `json:"id"`
	Label  string      `json:"label"`
	Plans  []FamilyPlan `json:"plans"`
}

type FamilyPlan struct {
	PlanCode         string          `json:"planCode"`
	InvoiceName      string          `json:"invoiceName"`
	SupportsWindows  bool            `json:"supportsWindows"`
	IsLocalZone      bool            `json:"isLocalZone"`
	Datacenters      []FamilyDatacenter `json:"datacenters"`
	OSImages         []string        `json:"osImages,omitempty"`
}

type FamilyDatacenter struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

func ParseCatalogPlans(raw []byte) ([]CatalogPlan, error) {
	var wrap struct {
		Plans []CatalogPlan `json:"plans"`
	}
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return nil, err
	}
	return wrap.Plans, nil
}

func ruleCodeByName(dcs []DatacenterStock) map[string]string {
	m := map[string]string{}
	for _, d := range dcs {
		if d.Name != "" {
			m[d.Name] = d.Code
		}
	}
	return m
}

func BuildFamilies(plans []CatalogPlan, ruleDCs []DatacenterStock) []Family {
	codes := ruleCodeByName(ruleDCs)
	out := []Family{
		{ID: Family2027, Label: "VPS 2027 常规", Plans: []FamilyPlan{}},
		{ID: Family2027LZ, Label: "VPS 2027 Local Zone", Plans: []FamilyPlan{}},
		{ID: FamilyOther, Label: "其他", Plans: []FamilyPlan{}},
	}
	idx := map[string]int{Family2027: 0, Family2027LZ: 1, FamilyOther: 2}
	for _, p := range plans {
		fam := ClassifyPlan(p.PlanCode)
		if fam == FamilyDrop {
			continue
		}
		fp := FamilyPlan{
			PlanCode:        p.PlanCode,
			InvoiceName:     p.InvoiceName,
			SupportsWindows: SupportsWindows(p),
			IsLocalZone:     fam == Family2027LZ,
			OSImages:        configValues(p, "vps_os"),
		}
		for _, name := range configValues(p, "vps_datacenter") {
			fp.Datacenters = append(fp.Datacenters, FamilyDatacenter{
				Name: name,
				Code: codes[name],
			})
		}
		i := idx[fam]
		out[i].Plans = append(out[i].Plans, fp)
	}
	return out
}
