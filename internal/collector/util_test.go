// SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
// SPDX-License-Identifier: Apache-2.0

package collector

import "testing"

func TestParseGpuGres(t *testing.T) {
	tests := []struct {
		name string
		gres string
		want int32
	}{
		{
			name: "empty",
			gres: "",
			want: 0,
		},
		{
			name: "no gpu",
			gres: "cpu:16",
			want: 0,
		},
		{
			name: "single gpu",
			gres: "gpu:1",
			want: 1,
		},
		{
			name: "multiple gpus",
			gres: "gpu:4",
			want: 4,
		},
		{
			name: "gpu with type",
			gres: "gpu:tesla:2",
			want: 2,
		},
		{
			name: "multiple resources",
			gres: "cpu:16,gpu:2,mem:32G",
			want: 2,
		},
		{
			name: "gpu with complex spec",
			gres: "gpu:nvidia_a100_80gb:8",
			want: 8,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseGpuGres(tt.gres); got != tt.want {
				t.Errorf("ParseGpuGres() = %v, want %v", got, tt.want)
			}
		})
	}
}