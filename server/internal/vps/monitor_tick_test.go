package vps

import (
	"testing"
	"time"
)

func TestFirstCheckRecordsOutOfStockHistory(t *testing.T) {
	dcs := []DatacenterStock{{
		Name: "US-WEST-LZ-SEA", Code: "us-west-lz-sea",
		LinuxStatus: "out-of-stock", WindowsStatus: "out-of-stock",
	}}
	got := RecordMonitorTick(nil, nil, dcs, nil, []string{"linux"}, time.Date(2026, 8, 18, 9, 0, 0, 0, time.UTC))
	if len(got.History) != 1 {
		t.Fatalf("hist=%v", got.History)
	}
	if got.History[0]["changeType"] != "initial" {
		t.Fatalf("type=%v", got.History[0]["changeType"])
	}
	if got.History[0]["status"] != "out-of-stock" {
		t.Fatalf("status=%v", got.History[0]["status"])
	}
	if got.LastStatus["us-west-lz-sea|linux"] != "out-of-stock" {
		t.Fatalf("last=%v", got.LastStatus)
	}
}

func TestNoChangeDoesNotDuplicateAfterHistoryExists(t *testing.T) {
	dcs := []DatacenterStock{{
		Name: "US-WEST-LZ-SEA", Code: "us-west-lz-sea",
		LinuxStatus: "out-of-stock",
	}}
	first := RecordMonitorTick(nil, nil, dcs, nil, []string{"linux"}, time.Unix(1, 0).UTC())
	second := RecordMonitorTick(first.History, first.LastStatus, dcs, nil, []string{"linux"}, time.Unix(2, 0).UTC())
	if len(second.History) != 1 {
		t.Fatalf("should keep the initial snapshot only, got %d", len(second.History))
	}
}

func TestRestockAppendsAvailable(t *testing.T) {
	dcs1 := []DatacenterStock{{Name: "GRA", Code: "eu-west-gra", LinuxStatus: "out-of-stock"}}
	dcs2 := []DatacenterStock{{Name: "GRA", Code: "eu-west-gra", LinuxStatus: "available"}}
	a := RecordMonitorTick(nil, nil, dcs1, nil, []string{"linux"}, time.Unix(1, 0).UTC())
	b := RecordMonitorTick(a.History, a.LastStatus, dcs2, nil, []string{"linux"}, time.Unix(2, 0).UTC())
	if len(b.History) != 2 {
		t.Fatalf("hist=%v", b.History)
	}
	if b.History[1]["changeType"] != "available" {
		t.Fatalf("second=%v", b.History[1])
	}
	if len(b.BecameAvail) != 1 {
		t.Fatal("expected restock")
	}
}

func TestBackfillWhenLastStatusExistsButHistoryEmpty(t *testing.T) {
	dcs := []DatacenterStock{{
		Name: "US-EAST-LZ-ATL", Code: "us-east-lz-atl", LinuxStatus: "out-of-stock",
	}}
	last := map[string]string{"us-east-lz-atl|linux": "out-of-stock"}
	got := RecordMonitorTick(nil, last, dcs, nil, []string{"linux"}, time.Unix(3, 0).UTC())
	if len(got.History) != 1 || got.History[0]["changeType"] != "initial" {
		t.Fatalf("backfill=%v", got.History)
	}
}
