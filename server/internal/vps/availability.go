package vps

import "encoding/json"

type DatacenterStock struct {
	Name          string `json:"datacenter"`
	Code          string `json:"code"`
	Status        string `json:"status"`
	LinuxStatus   string `json:"linuxStatus"`
	WindowsStatus string `json:"windowsStatus"`
	Days          int    `json:"daysBeforeDelivery"`
}

func ParseDatacenters(raw []byte) ([]DatacenterStock, error) {
	var wrap struct {
		Datacenters []DatacenterStock `json:"datacenters"`
	}
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return nil, err
	}
	return wrap.Datacenters, nil
}

func FindDC(dcs []DatacenterStock, code string) DatacenterStock {
	for _, d := range dcs {
		if d.Code == code {
			return d
		}
	}
	return DatacenterStock{}
}

func IsUnavailable(status string) bool {
	return status == "out-of-stock" || status == "out-of-stock-preorder-allowed"
}

func TrackStatus(dc DatacenterStock, track string) string {
	if track == "windows" {
		return dc.WindowsStatus
	}
	return dc.LinuxStatus
}

func TrackAvailable(dc DatacenterStock, track string) bool {
	st := TrackStatus(dc, track)
	if st == "" {
		return false
	}
	return !IsUnavailable(st)
}

func StatusKey(code, track string) string {
	return code + "|" + track
}

func HasTrackStatus(last map[string]string, code, track string) bool {
	if last == nil {
		return false
	}
	_, ok := last[StatusKey(code, track)]
	return ok
}

func ShouldAutoOrder(hadPrev, wasUnavail, nowAvail, autoOrder bool, accountID string) bool {
	return hadPrev && wasUnavail && nowAvail && autoOrder && accountID != ""
}
