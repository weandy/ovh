package vps

import "testing"

func TestClassifyPlan(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"vps-2027-model1", Family2027},
		{"vps-2027-model2", Family2027},
		{"vps-2027-model4", Family2027},
		{"vps-2027-model2.LZ", Family2027LZ},
		{"vps-2027-model1.LZ", Family2027LZ},
		{"vps-2025-model1", FamilyOther},
		{"vps-2025-model1.LZ", FamilyOther},
		{"vps-2027-model2-eu", Family2027},
		{"vps-2027-model2-ca", Family2027},
		{"vps-2027-model2.LZ-eu", Family2027LZ},
		{"vps-2027-model2-degressivity12", FamilyDrop},
		{"vps-comfort-4-8-80-vps-2025-model2-10percent", FamilyDrop},
	}
	for _, tc := range cases {
		if got := ClassifyPlan(tc.in); got != tc.want {
			t.Fatalf("ClassifyPlan(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}
