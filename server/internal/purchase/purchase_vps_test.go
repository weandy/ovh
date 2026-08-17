package purchase

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/ovh-buy/server/internal/app"
	"github.com/ovh-buy/server/internal/logger"
	"github.com/ovh-buy/server/internal/types"
)

func testState(t *testing.T) *app.State {
	t.Helper()
	return &app.State{Logger: logger.New(filepath.Join(t.TempDir(), "app.log.json"), nil)}
}

type fakeCart struct {
	calls    []string
	options  []map[string]interface{}
	failOpt  bool
	checkout bool
}

func (f *fakeCart) Get(url string, res interface{}) error {
	f.calls = append(f.calls, "GET "+url)
	if strings.Contains(url, "/vps/options") {
		if dest, ok := res.(*[]map[string]interface{}); ok {
			*dest = f.options
		}
		return nil
	}
	if strings.Contains(url, "/vps") && !strings.Contains(url, "options") {
		if dest, ok := res.(*[]map[string]interface{}); ok {
			*dest = []map[string]interface{}{
				{"planCode": "vps-2027-model2.LZ", "duration": "P1M", "pricingMode": "default"},
			}
		}
		return nil
	}
	if strings.Contains(url, "requiredConfiguration") {
		if dest, ok := res.(*[]map[string]interface{}); ok {
			*dest = []map[string]interface{}{}
		}
	}
	return nil
}

func (f *fakeCart) Post(url string, req, res interface{}) error {
	f.calls = append(f.calls, "POST "+url)
	if strings.HasSuffix(url, "/order/cart") {
		if dest, ok := res.(*map[string]interface{}); ok {
			*dest = map[string]interface{}{"cartId": "cart-1"}
		}
		return nil
	}
	if strings.HasSuffix(url, "/vps") && !strings.Contains(url, "options") {
		if dest, ok := res.(*map[string]interface{}); ok {
			*dest = map[string]interface{}{"itemId": 9}
		}
		return nil
	}
	if strings.Contains(url, "/checkout") {
		f.checkout = true
		if dest, ok := res.(*map[string]interface{}); ok {
			*dest = map[string]interface{}{"orderId": 123, "url": "https://example/order"}
		}
	}
	return nil
}

func (f *fakeCart) Delete(url string, res interface{}) error {
	f.calls = append(f.calls, "DELETE "+url)
	return nil
}

func TestPurchaseVPSUsesVpsCartNotEco(t *testing.T) {
	f := &fakeCart{options: []map[string]interface{}{
		{"planCode": "option-linux"},
		{"planCode": "option-storage-remote-2027"},
		{"planCode": "option-auto-backup-2027-1-model2.LZ"},
	}}
	state := testState(t)
	item := &types.QueueItem{
		ID: "t1", PlanCode: "vps-2027-model2.LZ", Datacenter: "us-west-lz-sea",
		ProductKind: "vps",
		VpsSpec: &types.VpsOrderSpec{
			Subsidiary: "US", DatacenterName: "US-WEST-LZ-SEA",
			DatacenterCode: "us-west-lz-sea", OSTrack: "linux", BackupPlan: "1",
		},
	}
	if !purchaseVPSWith(state, f, item) {
		t.Fatal("expected success")
	}
	joined := strings.Join(f.calls, "\n")
	if !strings.Contains(joined, "POST /order/cart/cart-1/vps") {
		t.Fatal(joined)
	}
	if strings.Contains(joined, "/eco") {
		t.Fatal("must not use eco cart")
	}
	if !f.checkout {
		t.Fatal("expected checkout")
	}
}

func TestPurchaseVPSAbortsWithoutStorage(t *testing.T) {
	f := &fakeCart{options: []map[string]interface{}{
		{"planCode": "option-linux"},
		{"planCode": "option-auto-backup-2027-1-model2.LZ"},
	}}
	state := testState(t)
	item := &types.QueueItem{
		ID: "t2", PlanCode: "vps-2027-model2.LZ", Datacenter: "us-west-lz-sea",
		ProductKind: "vps",
		VpsSpec: &types.VpsOrderSpec{
			Subsidiary: "US", DatacenterName: "US-WEST-LZ-SEA", OSTrack: "linux",
		},
	}
	if purchaseVPSWith(state, f, item) {
		t.Fatal("should fail")
	}
	if f.checkout {
		t.Fatal("must not checkout without storage")
	}
	joined := strings.Join(f.calls, "\n")
	if !strings.Contains(joined, "DELETE /order/cart/cart-1") {
		t.Fatal("should delete cart", joined)
	}
}

func TestPurchaseVPSRequiresSpec(t *testing.T) {
	state := testState(t)
	item := &types.QueueItem{PlanCode: "vps-2027-model2.LZ", ProductKind: "vps"}
	if PurchaseVPS(state, item) {
		t.Fatal("missing spec must fail")
	}
}
