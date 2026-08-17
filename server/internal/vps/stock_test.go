package vps

import (
	"os"
	"testing"
)

func TestBuildPlanStockMapsTracks(t *testing.T) {
	raw, err := os.ReadFile("testdata/rule-us-model2-lz.json")
	if err != nil {
		t.Fatal(err)
	}
	dcs, err := ParseDatacenters(raw)
	if err != nil {
		t.Fatal(err)
	}
	st := BuildPlanStock("vps-2027-model2.LZ", dcs)
	if st.PlanCode != "vps-2027-model2.LZ" {
		t.Fatal(st.PlanCode)
	}
	sea := FindStockDC(st.Datacenters, "us-west-lz-sea")
	if sea.Name != "US-WEST-LZ-SEA" || sea.Linux != "available" || sea.Windows != "out-of-stock" {
		t.Fatalf("%+v", sea)
	}
	if sea.Headline != "available" {
		t.Fatal(sea.Headline)
	}
}

func TestPlanHasBuyableStockLinuxOnly(t *testing.T) {
	raw, _ := os.ReadFile("testdata/rule-us-model2-lz.json")
	dcs, _ := ParseDatacenters(raw)
	if !PlanHasBuyableStock(false, dcs) {
		t.Fatal("LZ linux available should be buyable")
	}
	// 去掉 SEA，只剩 ATL 全无货
	onlyATL := []DatacenterStock{FindDC(dcs, "us-east-lz-atl")}
	if PlanHasBuyableStock(false, onlyATL) {
		t.Fatal("ATL out of stock")
	}
}

func TestPlanHasBuyableStockWindowsOnlyNeedsAddon(t *testing.T) {
	dcs := []DatacenterStock{{
		Name: "GRA", Code: "eu-west-gra",
		Status: "available", LinuxStatus: "out-of-stock", WindowsStatus: "available",
	}}
	if PlanHasBuyableStock(false, dcs) {
		t.Fatal("no windows product, linux oos → not buyable")
	}
	if !PlanHasBuyableStock(true, dcs) {
		t.Fatal("windows product + windows stock → buyable")
	}
}

func TestMonthlyPrice(t *testing.T) {
	raw, err := os.ReadFile("testdata/catalog-us-2027-subset.json")
	if err != nil {
		t.Fatal(err)
	}
	plans, err := ParseCatalogPlans(raw)
	if err != nil {
		t.Fatal(err)
	}
	p, ok := FindPlan(plans, "vps-2027-model2")
	if !ok {
		t.Fatal("missing plan")
	}
	got, ok := MonthlyPrice(p)
	if !ok || got != 10 {
		t.Fatalf("got %v ok=%v", got, ok)
	}
}

func TestMergePlanStockUsesCatalogDCs(t *testing.T) {
	catalog := []string{"GRA", "SBG", "US-EAST-VA"}
	rule := []DatacenterStock{{
		Name: "GRA", Code: "gra",
		Status: "available", LinuxStatus: "available", WindowsStatus: "out-of-stock",
	}}
	st := MergePlanStock("vps-2027-model2", catalog, rule)
	if len(st.Datacenters) != 3 {
		t.Fatalf("want 3 catalog DCs, got %+v", st.Datacenters)
	}
	gra := FindStockDC(st.Datacenters, "gra")
	if gra.Linux != "available" || gra.Name != "GRA" {
		t.Fatalf("GRA should keep rule stock: %+v", gra)
	}
	sbg := FindStockDCByName(st.Datacenters, "SBG")
	if sbg.Name != "SBG" || sbg.Linux != "" {
		t.Fatalf("SBG should appear even without rule: %+v", sbg)
	}
}

func TestCurrencyForSubsidiary(t *testing.T) {
	if CurrencyForSubsidiary("US") != "USD" {
		t.Fatal(CurrencyForSubsidiary("US"))
	}
	if CurrencyForSubsidiary("IE") != "EUR" {
		t.Fatal(CurrencyForSubsidiary("IE"))
	}
}
