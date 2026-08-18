package vps

import (
	"sort"
	"strings"
	"testing"
)

func TestCanonicalPlanCode(t *testing.T) {
	cases := map[string]string{
		"vps-2027-model1":       "vps-2027-model1",
		"vps-2027-model1-eu":    "vps-2027-model1",
		"vps-2027-model1-ca":    "vps-2027-model1",
		"vps-2027-model2.LZ":    "vps-2027-model2.LZ",
		"vps-2027-model2.LZ-eu": "vps-2027-model2.LZ",
	}
	for in, want := range cases {
		if got := CanonicalPlanCode(in); got != want {
			t.Fatalf("CanonicalPlanCode(%q)=%q want %q", in, got, want)
		}
	}
}

func TestContinentOfDCMatchesOfficialUSStorefront(t *testing.T) {
	// 美国官网 VPS-1 选位：Europe / North America / Asia-Oceania
	cases := []struct {
		name, code, want string
	}{
		{"SBG", "eu-west-sbg", ContinentEurope},
		{"GRA", "eu-west-gra", ContinentEurope},
		{"DE", "eu-west-lim", ContinentEurope},
		{"UK", "eu-west-eri", ContinentEurope},
		{"WAW", "eu-central-waw", ContinentEurope},
		{"EU-SOUTH-MIL", "eu-south-mil", ContinentEurope},
		{"EU-WEST-RBX", "eu-west-rbx", ContinentEurope},
		{"US-WEST-OR", "us-west-hil", ContinentNorthAmerica},
		{"US-EAST-VA", "us-east-vin", ContinentNorthAmerica},
		{"BHS", "ca-east-bhs", ContinentNorthAmerica},
		{"SGP", "ap-southeast-sgp", ContinentAsiaOceania},
		{"SYD", "ap-southeast-syd", ContinentAsiaOceania},
		{"YNM", "ap-south-mum", ContinentAsiaOceania},
	}
	for _, tc := range cases {
		if got := ContinentOfDC(tc.name, tc.code); got != tc.want {
			t.Fatalf("%s/%s → %s want %s", tc.name, tc.code, got, tc.want)
		}
	}
}

func TestGroupLocationVariantsMergesUSStorefront(t *testing.T) {
	plans := []CatalogPlan{
		{PlanCode: "vps-2027-model1", InvoiceName: "VPS-1 2027", Configurations: []CatalogConfig{
			{Name: "vps_datacenter", Values: []string{"US-EAST-VA", "US-WEST-OR"}},
		}},
		{PlanCode: "vps-2027-model1-eu", InvoiceName: "VPS-1 2027", Configurations: []CatalogConfig{
			{Name: "vps_datacenter", Values: []string{"DE", "EU-SOUTH-MIL", "EU-WEST-RBX", "GRA", "SBG", "UK", "WAW"}},
		}},
		{PlanCode: "vps-2027-model1-ca", InvoiceName: "VPS-1 2027", Configurations: []CatalogConfig{
			{Name: "vps_datacenter", Values: []string{"BHS", "SGP", "SYD", "YNM"}},
		}},
		{PlanCode: "vps-2027-model2-degressivity12", InvoiceName: "skip"},
	}
	groups := GroupLocationVariants(plans)
	if len(groups) != 1 {
		t.Fatalf("groups=%d", len(groups))
	}
	g := groups[0]
	if g.Canonical != "vps-2027-model1" {
		t.Fatal(g.Canonical)
	}
	if AssignOrderPlanCode("GRA", g.Siblings) != "vps-2027-model1-eu" {
		t.Fatal(AssignOrderPlanCode("GRA", g.Siblings))
	}
	if AssignOrderPlanCode("US-WEST-OR", g.Siblings) != "vps-2027-model1" {
		t.Fatal(AssignOrderPlanCode("US-WEST-OR", g.Siblings))
	}
	if AssignOrderPlanCode("BHS", g.Siblings) != "vps-2027-model1-ca" {
		t.Fatal(AssignOrderPlanCode("BHS", g.Siblings))
	}
	names := CatalogNamesFromGroup(g)
	if len(names) != 13 {
		t.Fatalf("catalog still lists 13 names including Asia leftovers, got %v", names)
	}
}

func TestUSStorefrontDoesNotSellAsia(t *testing.T) {
	catalog := []string{"US-EAST-VA", "US-WEST-OR", "GRA", "SBG", "BHS", "SGP", "SYD", "YNM"}
	homeRules := []DatacenterStock{
		{Name: "US-EAST-VA", Code: "us-east-vin"},
		{Name: "US-WEST-OR", Code: "us-west-hil"},
		{Name: "GRA", Code: "eu-west-gra"},
		{Name: "SBG", Code: "eu-west-sbg"},
		{Name: "BHS", Code: "ca-east-bhs"},
	}
	got := FilterStorefrontDCs(catalog, homeRules, "US")
	for _, n := range got {
		if ContinentOfDC(n, "") == ContinentAsiaOceania {
			t.Fatalf("US storefront must not sell %s", n)
		}
	}
	want := []string{"BHS", "GRA", "SBG", "US-EAST-VA", "US-WEST-OR"}
	if stringsJoin(got) != stringsJoin(want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestUSAccountRejectsAsiaDC(t *testing.T) {
	if StorefrontAllowsDC("US", "SGP") {
		t.Fatal("US cannot buy SGP")
	}
	if StorefrontAllowsDC("US", "SYD") {
		t.Fatal("US cannot buy SYD")
	}
	if !StorefrontAllowsDC("US", "GRA") {
		t.Fatal("US can buy GRA")
	}
	if !StorefrontAllowsDC("US", "BHS") {
		t.Fatal("US can buy BHS")
	}
	if !StorefrontAllowsDC("SG", "SGP") {
		t.Fatal("SG storefront can buy SGP")
	}
}

func stringsJoin(in []string) string {
	return strings.Join(sortedCopy(in), ",")
}

func sortedCopy(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}

func TestCartRegionFromOrderPlan(t *testing.T) {
	if CartRegion("vps-2027-model1-eu", "GRA") != "europe" {
		t.Fatal(CartRegion("vps-2027-model1-eu", "GRA"))
	}
	if CartRegion("vps-2027-model1-ca", "BHS") != "canada" {
		t.Fatal(CartRegion("vps-2027-model1-ca", "BHS"))
	}
	if CartRegion("vps-2027-model1", "US-EAST-VA") != "united_states" {
		t.Fatal(CartRegion("vps-2027-model1", "US-EAST-VA"))
	}
}

func TestOutsideUnitedStates(t *testing.T) {
	if OutsideUnitedStates("US-EAST-VA", "us-east-vin") {
		t.Fatal("US DC is home")
	}
	if !OutsideUnitedStates("GRA", "eu-west-gra") {
		t.Fatal("GRA is outside US")
	}
	if !OutsideUnitedStates("BHS", "ca-east-bhs") {
		t.Fatal("BHS is outside US")
	}
}
