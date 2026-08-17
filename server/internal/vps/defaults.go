package vps

import (
	"fmt"
	"strings"
)

func familyAddons(plan CatalogPlan, name string) []string {
	for _, f := range plan.AddonFamilies {
		if f.Name == name {
			return f.Addons
		}
	}
	return nil
}

func SupportsWindows(plan CatalogPlan) bool {
	for _, a := range familyAddons(plan, "os") {
		if strings.Contains(strings.ToLower(a), "windows") {
			return true
		}
	}
	return false
}

func configValues(plan CatalogPlan, name string) []string {
	for _, c := range plan.Configurations {
		if c.Name == name {
			return c.Values
		}
	}
	return nil
}

func DefaultOSImage(plan CatalogPlan, track string) string {
	vals := configValues(plan, "vps_os")
	if track == "windows" {
		var lastWin string
		for _, v := range vals {
			if strings.HasPrefix(v, "Windows") {
				if v == "Windows Server 2022 Standard (Desktop)" {
					return v
				}
				lastWin = v
			}
		}
		return lastWin
	}
	for _, prefer := range []string{"Ubuntu 24.04", "Debian 12"} {
		for _, v := range vals {
			if v == prefer {
				return v
			}
		}
	}
	for _, v := range vals {
		if !strings.HasPrefix(v, "Windows") {
			return v
		}
	}
	return ""
}

func pickBackup(addons []string, planDays string) string {
	needle := "-1-"
	if planDays == "7" {
		needle = "-7-"
	}
	for _, a := range addons {
		if strings.Contains(a, needle) {
			return a
		}
	}
	if len(addons) > 0 {
		return addons[0]
	}
	return ""
}

func DefaultAddons(plan CatalogPlan, track, backupPlan string) ([]string, error) {
	osAddons := familyAddons(plan, "os")
	var osPick string
	if track == "windows" {
		for _, a := range osAddons {
			if strings.Contains(strings.ToLower(a), "windows") {
				osPick = a
				break
			}
		}
		if osPick == "" {
			return nil, fmt.Errorf("plan %s has no Windows addon", plan.PlanCode)
		}
	} else {
		for _, a := range osAddons {
			if a == "option-linux" {
				osPick = a
				break
			}
		}
		if osPick == "" && len(osAddons) > 0 {
			osPick = osAddons[0]
		}
	}

	storage := familyAddons(plan, "storage")
	if len(storage) == 0 {
		return nil, fmt.Errorf("plan %s missing storage addon", plan.PlanCode)
	}
	backup := pickBackup(familyAddons(plan, "automatedBackup"), backupPlan)
	if backup == "" {
		return nil, fmt.Errorf("plan %s missing backup addon", plan.PlanCode)
	}
	return []string{osPick, storage[0], backup}, nil
}

func RegionFor(datacenterName string) string {
	if strings.HasPrefix(datacenterName, "US-") {
		return "united_states"
	}
	if datacenterName == "BHS" {
		return "canada"
	}
	return "europe"
}

func DefaultInfrastructure() string {
	return "production"
}
