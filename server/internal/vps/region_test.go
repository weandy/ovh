package vps

import "testing"

func TestRegionOfSubsidiary(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"US", RegionUS},
		{"us", RegionUS},
		{"IE", RegionIE},
		{"FR", RegionIE},
		{"DE", RegionIE},
		{"GB", RegionIE},
		{"IT", RegionIE},
		{"NL", RegionIE},
		{"CA", RegionCA},
		{"QC", RegionCA},
		{"ASIA", RegionCA},
		{"SG", RegionCA},
		{"AU", RegionCA},
		{"IN", RegionCA},
		{"", RegionUS},
		{"??", RegionIE},
	}
	for _, tc := range cases {
		if got := RegionOfSubsidiary(tc.in); got != tc.want {
			t.Fatalf("RegionOfSubsidiary(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestNormalizeRegion(t *testing.T) {
	if NormalizeRegion("") != RegionUS {
		t.Fatal(NormalizeRegion(""))
	}
	if NormalizeRegion("eu") != RegionIE {
		t.Fatal(NormalizeRegion("eu"))
	}
	if NormalizeRegion("ovh-ca") != RegionCA {
		t.Fatal(NormalizeRegion("ovh-ca"))
	}
	if NormalizeRegion("ovh-us") != RegionUS {
		t.Fatal(NormalizeRegion("ovh-us"))
	}
	if NormalizeRegion("FR") != RegionIE {
		t.Fatal(NormalizeRegion("FR"))
	}
}

func TestCatalogSubsidiary(t *testing.T) {
	if CatalogSubsidiary(RegionUS, "") != "US" {
		t.Fatal(CatalogSubsidiary(RegionUS, ""))
	}
	if CatalogSubsidiary(RegionIE, "") != "IE" {
		t.Fatal(CatalogSubsidiary(RegionIE, ""))
	}
	if CatalogSubsidiary(RegionCA, "") != "CA" {
		t.Fatal(CatalogSubsidiary(RegionCA, ""))
	}
	// 账号子公司落在同一区时，用账号的子公司拉目录（税/价），否则用该区默认子公司
	if CatalogSubsidiary(RegionIE, "FR") != "FR" {
		t.Fatal(CatalogSubsidiary(RegionIE, "FR"))
	}
	if CatalogSubsidiary(RegionUS, "FR") != "US" {
		t.Fatal(CatalogSubsidiary(RegionUS, "FR"))
	}
	if CatalogSubsidiary(RegionCA, "SG") != "SG" {
		t.Fatal(CatalogSubsidiary(RegionCA, "SG"))
	}
}

func TestSameRegion(t *testing.T) {
	if !SameRegion("FR", "IE") {
		t.Fatal("FR and IE are both Europe")
	}
	if !SameRegion("US", "US") {
		t.Fatal("US matches US")
	}
	if SameRegion("US", "IE") {
		t.Fatal("US must not buy IE")
	}
	if SameRegion("US", "CA") {
		t.Fatal("US must not buy CA")
	}
	if SameRegion("FR", "SG") {
		t.Fatal("IE-region must not buy CA-region")
	}
	if !SameRegion("SG", "CA") {
		t.Fatal("SG and CA are both CA-region")
	}
}

func TestResolveListQuery(t *testing.T) {
	// 没传区、没账号 → 默认 US
	got := ResolveListQuery("", "")
	if got.Region != RegionUS || got.CatalogSubsidiary != "US" {
		t.Fatalf("%+v", got)
	}
	// 账号法国 → 默认欧洲区，目录用 FR
	got = ResolveListQuery("", "FR")
	if got.Region != RegionIE || got.CatalogSubsidiary != "FR" {
		t.Fatalf("%+v", got)
	}
	// 手选 US，账号却是 FR → 仍看美国货架，目录 US（不能下单）
	got = ResolveListQuery("US", "FR")
	if got.Region != RegionUS || got.CatalogSubsidiary != "US" || got.AccountCanBuy {
		t.Fatalf("%+v", got)
	}
	// 手选 IE，账号 FR → 能买，目录 FR
	got = ResolveListQuery("IE", "FR")
	if got.Region != RegionIE || got.CatalogSubsidiary != "FR" || !got.AccountCanBuy {
		t.Fatalf("%+v", got)
	}
}
