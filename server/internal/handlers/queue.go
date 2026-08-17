package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ovh-buy/server/internal/app"
	"github.com/ovh-buy/server/internal/purchase"
	"github.com/ovh-buy/server/internal/types"
)

// AddQueueItem POST /api/queue
// 多账户:body 必须带 account_id,后端用它确定下单走哪个账户
func AddQueueItem(state *app.State) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			AccountID     string              `json:"account_id"`
			PlanCode      string              `json:"planCode"`
			Datacenter    string              `json:"datacenter"`
			Options       []string            `json:"options"`
			RetryInterval int                 `json:"retryInterval"`
			ProductKind   string              `json:"productKind"`
			VpsSpec       *types.VpsOrderSpec `json:"vpsSpec"`
		}
		_ = c.ShouldBindJSON(&body)
		if body.AccountID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "error": "缺少 account_id"})
			return
		}
		acc, ok := state.FindAccount(body.AccountID)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "error": "account_id 不存在"})
			return
		}
		if body.ProductKind == "vps" {
			if body.VpsSpec == nil || body.VpsSpec.Subsidiary == "" || body.VpsSpec.DatacenterName == "" || body.VpsSpec.OSTrack == "" {
				c.JSON(http.StatusBadRequest, gin.H{"status": "error", "error": "VPS 任务缺少 vpsSpec（subsidiary / datacenterName / osTrack）"})
				return
			}
			if !strings.EqualFold(acc.Zone, body.VpsSpec.Subsidiary) {
				c.JSON(http.StatusBadRequest, gin.H{"status": "error", "error": "账户子公司 (" + acc.Zone + ") 必须与 VPS 订阅子公司 (" + body.VpsSpec.Subsidiary + ") 一致"})
				return
			}
			if strings.EqualFold(body.VpsSpec.OSTrack, "windows") && strings.TrimSpace(body.VpsSpec.OSImage) == "" {
				c.JSON(http.StatusBadRequest, gin.H{"status": "error", "error": "Windows 轨必须指定镜像名"})
				return
			}
		}
		if body.RetryInterval == 0 {
			body.RetryInterval = 30
		}
		item := types.QueueItem{
			AccountID:     body.AccountID,
			PlanCode:      body.PlanCode,
			Datacenter:    body.Datacenter,
			Options:       body.Options,
			RetryInterval: body.RetryInterval,
			ProductKind:   body.ProductKind,
			VpsSpec:       body.VpsSpec,
		}
		queued, err := purchase.Enqueue(state, item)
		if err == purchase.ErrDuplicate {
			c.JSON(http.StatusOK, gin.H{"status": "error", "error": "队列中已有相同任务"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "error": err.Error()})
			return
		}
		state.Logger.Info("添加任务 "+queued.ID+" ("+queued.PlanCode+" 在 "+queued.Datacenter+", 账户 "+body.AccountID+") 到队列并立即启动 (状态: running)", "")
		c.JSON(http.StatusOK, gin.H{"status": "success", "id": queued.ID})
	}
}

// RemoveQueueItem DELETE /api/queue/:id
func RemoveQueueItem(state *app.State) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		state.DeletedTaskIDsMu.Lock()
		state.DeletedTaskIDs[id] = struct{}{}
		state.DeletedTaskIDsMu.Unlock()
		state.Logger.Info("标记任务 "+id+" 为删除，后台线程将立即停止处理", "system")

		state.QueueMu.Lock()
		var removed *types.QueueItem
		// 重新分配新 slice，避免 [:0] 与原 backing array 共享导致快照读到已被覆盖的元素
		kept := make([]types.QueueItem, 0, len(state.Queue))
		for i := range state.Queue {
			if state.Queue[i].ID == id {
				cp := state.Queue[i]
				removed = &cp
				continue
			}
			kept = append(kept, state.Queue[i])
		}
		state.Queue = kept
		state.QueueMu.Unlock()
		_ = state.SaveQueue()
		if removed != nil {
			state.Logger.Info("Removed "+removed.PlanCode+" from queue (ID: "+id+")", "system")
		}
		c.JSON(http.StatusOK, gin.H{"status": "success"})
	}
}

// ClearQueue DELETE /api/queue/clear
func ClearQueue(state *app.State) gin.HandlerFunc {
	return func(c *gin.Context) {
		state.QueueMu.Lock()
		count := len(state.Queue)
		state.DeletedTaskIDsMu.Lock()
		for _, it := range state.Queue {
			state.DeletedTaskIDs[it.ID] = struct{}{}
		}
		state.DeletedTaskIDsMu.Unlock()
		state.Queue = []types.QueueItem{}
		state.QueueMu.Unlock()
		_ = state.SaveQueue()
		state.Logger.Info("Cleared all queue items ("+strconv.Itoa(count)+" items removed)", "")
		c.JSON(http.StatusOK, gin.H{"status": "success", "count": count})
	}
}

// UpdateQueueStatus PUT /api/queue/:id/status
func UpdateQueueStatus(state *app.State) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var body struct {
			Status string `json:"status"`
		}
		_ = c.ShouldBindJSON(&body)
		if body.Status == "" {
			body.Status = "pending"
		}
		state.QueueMu.Lock()
		for i := range state.Queue {
			if state.Queue[i].ID == id {
				state.Queue[i].Status = body.Status
				state.Queue[i].UpdatedAt = types.NowISO()
				state.Logger.Info("Updated "+state.Queue[i].PlanCode+" status to "+body.Status, "")
				break
			}
		}
		state.QueueMu.Unlock()
		_ = state.SaveQueue()
		c.JSON(http.StatusOK, gin.H{"status": "success"})
	}
}

// ClearPurchaseHistory DELETE /api/purchase-history
func ClearPurchaseHistory(state *app.State) gin.HandlerFunc {
	return func(c *gin.Context) {
		state.HistoryMu.Lock()
		state.History = state.History[:0]
		state.HistoryMu.Unlock()
		_ = state.SaveHistory()
		state.Logger.Info("Purchase history cleared", "")
		c.JSON(http.StatusOK, gin.H{"status": "success"})
	}
}
