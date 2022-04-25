package cluster

import "testing"

func Test_cpuToCores(t *testing.T) {
	tests := []struct {
		val  string
		want float64
	}{
		{
			"500m",
			0.5,
		},
		{
			"3000m",
			3,
		},
		{
			"1m",
			0.001,
		},
		{
			"5",
			5,
		},
	}
	for _, tt := range tests {
		t.Run(tt.val, func(t *testing.T) {
			got, err := cpuToCores(tt.val)
			if err != nil {
				t.Errorf("cpuToCores() error = %v", err)
				return
			}

			if got != tt.want {
				t.Errorf("cpuToCores() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_memToGb(t *testing.T) {
	tests := []struct {
		val  string
		want float64
	}{
		{
			"512M",
			0.5,
		},
		{
			"512Mi",
			0.476837158203125,
		},
		{
			"5G",
			5,
		},
		{
			"1Gi",
			0.9313225746154784,
		},
	}
	for _, tt := range tests {
		t.Run(tt.val, func(t *testing.T) {
			got, err := memToGb(tt.val)
			if err != nil {
				t.Errorf("memToGb() error = %v", err)
				return
			}

			if got != tt.want {
				t.Errorf("memToGb() got = %v, want %v", got, tt.want)
			}
		})
	}
}