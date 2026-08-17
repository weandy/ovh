package purchase

import (
	"testing"

	"github.com/ovh-buy/server/internal/types"
)

func TestQueueFingerprintSeparatesVPSTrack(t *testing.T) {
	eco := types.QueueItem{AccountID: "a", PlanCode: "24sk", Datacenter: "gra"}
	vpsL := types.QueueItem{
		AccountID: "a", PlanCode: "vps-2027-model2.LZ", Datacenter: "us-west-lz-sea",
		ProductKind: "vps",
		VpsSpec:     &types.VpsOrderSpec{OSTrack: "linux"},
	}
	vpsW := vpsL
	vpsW.VpsSpec = &types.VpsOrderSpec{OSTrack: "windows"}
	if QueueFingerprint(eco) == QueueFingerprint(vpsL) {
		t.Fatal("eco and vps must differ")
	}
	if QueueFingerprint(vpsL) == QueueFingerprint(vpsW) {
		t.Fatal("linux and windows tracks must differ")
	}
}

func TestHasActiveDuplicate(t *testing.T) {
	item := types.QueueItem{
		AccountID: "acc", PlanCode: "vps-2027-model2", Datacenter: "us-east-vin",
		ProductKind: "vps", Status: "running",
		VpsSpec: &types.VpsOrderSpec{OSTrack: "linux"},
	}
	q := []types.QueueItem{item}
	if !HasActiveDuplicate(q, item) {
		t.Fatal("expected duplicate")
	}
	done := item
	done.Status = "completed"
	if HasActiveDuplicate([]types.QueueItem{done}, item) {
		t.Fatal("completed should not block")
	}
}
