package vps

import (
	"os"
	"sort"
	"testing"
)

func mustPlan(t *testing.T, path, planCode string) CatalogPlan {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	plans, err := ParseCatalogPlans(raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range plans {
		if p.PlanCode == planCode {
			return p
		}
	}
	t.Fatalf("plan %s not in %s", planCode, path)
	return CatalogPlan{}
}

func sameSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	as := append([]string(nil), a...)
	bs := append([]string(nil), b...)
	sort.Strings(as)
	sort.Strings(bs)
	for i := range as {
		if as[i] != bs[i] {
			return false
		}
	}
	return true
}

func TestDefaultAddonsLocalZoneLinux(t *testing.T) {
	plan := mustPlan(t, "testdata/catalog-us-2027-subset.json", "vps-2027-model2.LZ")
	got, err := DefaultAddons(plan, "linux", "1")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"option-linux", "option-storage-remote-2027", "option-auto-backup-2027-1-model2.LZ"}
	if !sameSet(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestDefaultOSImage(t *testing.T) {
	plan := mustPlan(t, "testdata/catalog-us-2027-subset.json", "vps-2027-model2")
	if DefaultOSImage(plan, "linux") != "Ubuntu 24.04" {
		t.Fatal(DefaultOSImage(plan, "linux"))
	}
	if DefaultOSImage(plan, "windows") != "Windows Server 2022 Standard (Desktop)" {
		t.Fatal(DefaultOSImage(plan, "windows"))
	}
}

func TestRegionFor(t *testing.T) {
	if RegionFor("BHS") != "canada" {
		t.Fatal(RegionFor("BHS"))
	}
	if RegionFor("GRA") != "europe" {
		t.Fatal(RegionFor("GRA"))
	}
	if RegionFor("US-EAST-VA") != "united_states" {
		t.Fatal(RegionFor("US-EAST-VA"))
	}
	if RegionFor("US-WEST-LZ-SEA") != "united_states" {
		t.Fatal(RegionFor("US-WEST-LZ-SEA"))
	}
	if RegionFor("EU-WEST-LZ-AMS") != "europe" {
		t.Fatal(RegionFor("EU-WEST-LZ-AMS"))
	}
}

func TestWindowsRejectedWhenNoAddon(t *testing.T) {
	plan := mustPlan(t, "testdata/catalog-us-2027-subset.json", "vps-2027-model1")
	if SupportsWindows(plan) {
		t.Fatal("model1 has no windows addon")
	}
	if _, err := DefaultAddons(plan, "windows", "1"); err == nil {
		t.Fatal("expected error")
	}
}

func TestBuildFamiliesDropsShadowAndSplitsLZ(t *testing.T) {
	raw, err := os.ReadFile("testdata/catalog-us-2027-subset.json")
	if err != nil {
		t.Fatal(err)
	}
	plans, err := ParseCatalogPlans(raw)
	if err != nil {
		t.Fatal(err)
	}
	ruleRaw, err := os.ReadFile("testdata/rule-us-model2-lz.json")
	if err != nil {
		t.Fatal(err)
	}
	dcs, err := ParseDatacenters(ruleRaw)
	if err != nil {
		t.Fatal(err)
	}
	fams := BuildFamilies(plans, dcs)
	if len(fams) != 3 {
		t.Fatalf("families=%d", len(fams))
	}
	if fams[0].ID != Family2027 || fams[1].ID != Family2027LZ || fams[2].ID != FamilyOther {
		t.Fatalf("ids %s %s %s", fams[0].ID, fams[1].ID, fams[2].ID)
	}
	if len(fams[0].Plans) != 2 {
		t.Fatalf("regular plans=%d", len(fams[0].Plans))
	}
	if len(fams[1].Plans) != 1 || fams[1].Plans[0].PlanCode != "vps-2027-model2.LZ" {
		t.Fatalf("lz plans=%v", fams[1].Plans)
	}
	if !fams[1].Plans[0].IsLocalZone || fams[1].Plans[0].SupportsWindows {
		t.Fatal("LZ should be local zone linux-only")
	}
	if fams[1].Label != "VPS 2027 Local Zone" {
		t.Fatal(fams[1].Label)
	}
	for _, p := range fams[0].Plans {
		if p.PlanCode == "vps-2027-model2-eu" {
			t.Fatal("shadow sku leaked")
		}
	}
}
