package vps

import (
	"os"
	"testing"
)

func TestTrackAvailableUsesOSStatusNotHeadline(t *testing.T) {
	raw, err := os.ReadFile("testdata/rule-us-model2-lz.json")
	if err != nil {
		t.Fatal(err)
	}
	dcs, err := ParseDatacenters(raw)
	if err != nil {
		t.Fatal(err)
	}
	sea := FindDC(dcs, "us-west-lz-sea")
	if sea.Name != "US-WEST-LZ-SEA" {
		t.Fatalf("name=%q", sea.Name)
	}
	if !TrackAvailable(sea, "linux") {
		t.Fatal("SEA linux should be available")
	}
	if TrackAvailable(sea, "windows") {
		t.Fatal("SEA windows should be out of stock")
	}
	atl := FindDC(dcs, "us-east-lz-atl")
	if TrackAvailable(atl, "linux") {
		t.Fatal("ATL linux should be out of stock")
	}
}

func TestIELocalZoneRule(t *testing.T) {
	raw, err := os.ReadFile("testdata/rule-ie-model2-lz.json")
	if err != nil {
		t.Fatal(err)
	}
	dcs, err := ParseDatacenters(raw)
	if err != nil {
		t.Fatal(err)
	}
	ams := FindDC(dcs, "eu-west-lz-ams")
	if !TrackAvailable(ams, "linux") {
		t.Fatal("AMS linux should be available")
	}
	if TrackAvailable(ams, "windows") {
		t.Fatal("AMS windows should be out of stock")
	}
}

func TestStatusKeyAndShouldAutoOrder(t *testing.T) {
	if StatusKey("us-west-lz-sea", "linux") != "us-west-lz-sea|linux" {
		t.Fatal(StatusKey("us-west-lz-sea", "linux"))
	}
	if ShouldAutoOrder(false, true, true, true, "acc") {
		t.Fatal("first seen must not auto-order")
	}
	if !ShouldAutoOrder(true, true, true, true, "acc") {
		t.Fatal("flip out-of-stock -> available should auto-order")
	}
	if ShouldAutoOrder(true, true, true, true, "") {
		t.Fatal("no account, no order")
	}
	if ShouldAutoOrder(true, true, true, false, "acc") {
		t.Fatal("autoOrder off")
	}
	if !IsUnavailable("out-of-stock-preorder-allowed") {
		t.Fatal("preorder is unavailable")
	}
}

func TestLegacyLastStatusIsFirstSeen(t *testing.T) {
	last := map[string]string{"us-west-lz-sea": "available"}
	if HasTrackStatus(last, "us-west-lz-sea", "linux") {
		t.Fatal("legacy headline key is not a track key")
	}
}
