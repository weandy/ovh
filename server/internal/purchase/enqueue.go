package purchase

import (
	"errors"

	"github.com/google/uuid"

	"github.com/ovh-buy/server/internal/app"
	"github.com/ovh-buy/server/internal/types"
)

var ErrDuplicate = errors.New("duplicate active queue item")

func QueueFingerprint(item types.QueueItem) string {
	kind := item.ProductKind
	if kind == "" {
		kind = "eco"
	}
	track := ""
	if item.VpsSpec != nil {
		track = item.VpsSpec.OSTrack
	}
	return kind + "|" + item.AccountID + "|" + item.PlanCode + "|" + item.Datacenter + "|" + track
}

func HasActiveDuplicate(queue []types.QueueItem, item types.QueueItem) bool {
	fp := QueueFingerprint(item)
	for _, q := range queue {
		if q.Status == "running" || q.Status == "pending" {
			if QueueFingerprint(q) == fp {
				return true
			}
		}
	}
	return false
}

func Enqueue(state *app.State, item types.QueueItem) (types.QueueItem, error) {
	if item.ID == "" {
		item.ID = uuid.NewString()
	}
	if item.ProductKind == "" {
		item.ProductKind = "eco"
	}
	if item.CreatedAt == "" {
		item.CreatedAt = types.NowISO()
	}
	item.UpdatedAt = types.NowISO()
	if item.Status == "" {
		item.Status = "running"
	}
	state.QueueMu.Lock()
	defer state.QueueMu.Unlock()
	if HasActiveDuplicate(state.Queue, item) {
		return item, ErrDuplicate
	}
	state.Queue = append(state.Queue, item)
	_ = state.SaveQueue()
	return item, nil
}
