package vps

import "time"

type MonitorTick struct {
	History       []map[string]interface{}
	LastStatus    map[string]string
	FirstSeen     []map[string]interface{}
	BecameAvail   []map[string]interface{}
	BecameUnavail []map[string]interface{}
	OrderTargets  []orderTarget
}

func cloneLastStatus(last map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range last {
		out[k] = v
	}
	return out
}

func histEntry(now time.Time, dc DatacenterStock, track, cur, changeType string, old interface{}) map[string]interface{} {
	return map[string]interface{}{
		"timestamp":      now.Format(time.RFC3339Nano),
		"datacenter":     dc.Name,
		"datacenterCode": dc.Code,
		"status":         cur,
		"changeType":     changeType,
		"oldStatus":      old,
		"osTrack":        track,
	}
}

// RecordMonitorTick 把本轮 rule 结果写进历史。
// 首次见到某个机房轨（或订阅已有 LastStatus 但 History 为空）一律记 initial，
// 包括 out-of-stock。补货监控大部分时间全无货，不记首次状态的话历史页会一直空白。
func RecordMonitorTick(hist []map[string]interface{}, last map[string]string, dcs []DatacenterStock, monitored []string, tracks []string, now time.Time) MonitorTick {
	if last == nil {
		last = map[string]string{}
	}
	if hist == nil {
		hist = []map[string]interface{}{}
	}
	out := MonitorTick{
		History:    append([]map[string]interface{}{}, hist...),
		LastStatus: cloneLastStatus(last),
	}
	if len(tracks) == 0 {
		tracks = []string{"linux"}
	}
	backfill := len(out.History) == 0
	for _, dc := range dcs {
		if len(monitored) > 0 && !dcMonitored(monitored, dc) {
			continue
		}
		for _, track := range tracks {
			cur := TrackStatus(dc, track)
			had := HasTrackStatus(out.LastStatus, dc.Code, track)
			avail := TrackAvailable(dc, track)
			key := StatusKey(dc.Code, track)
			row := map[string]interface{}{
				"name": dc.Name, "code": dc.Code, "status": cur, "days": dc.Days, "track": track,
			}
			switch {
			case !had || backfill:
				out.FirstSeen = append(out.FirstSeen, row)
				change := "initial"
				if had && backfill {
					// 升级前的订阅：LastStatus 已有、History 为空，补一条当前快照
					change = "initial"
				}
				out.History = append(out.History, histEntry(now, dc, track, cur, change, nil))
			default:
				old := out.LastStatus[key]
				if IsUnavailable(old) && avail {
					out.BecameAvail = append(out.BecameAvail, row)
					out.History = append(out.History, histEntry(now, dc, track, cur, "available", old))
					out.OrderTargets = append(out.OrderTargets, orderTarget{dc: dc, track: track})
				} else if !IsUnavailable(old) && IsUnavailable(cur) {
					out.BecameUnavail = append(out.BecameUnavail, row)
					out.History = append(out.History, histEntry(now, dc, track, cur, "unavailable", old))
				}
			}
			out.LastStatus[key] = cur
		}
	}
	if len(out.History) > 100 {
		out.History = out.History[len(out.History)-100:]
	}
	return out
}
