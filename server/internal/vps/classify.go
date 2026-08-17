package vps

import "regexp"

const (
	FamilyDrop   = ""
	Family2027   = "vps-2027"
	Family2027LZ = "vps-2027-lz"
	FamilyOther  = "other"
)

var (
	reShadow = regexp.MustCompile(`-(eu|ca)$|degressivity|percent`)
	re2027   = regexp.MustCompile(`^vps-2027-model[0-9]+$`)
	re2027LZ = regexp.MustCompile(`^vps-2027-model[0-9]+\.LZ$`)
)

func ClassifyPlan(planCode string) string {
	if planCode == "" || reShadow.MatchString(planCode) {
		return FamilyDrop
	}
	if re2027LZ.MatchString(planCode) {
		return Family2027LZ
	}
	if re2027.MatchString(planCode) {
		return Family2027
	}
	return FamilyOther
}
