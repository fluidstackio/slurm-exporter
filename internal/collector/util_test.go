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
			name: "empty gres (non-GPU node)",
			gres: "",
			want: 0,
		},
		{
			name: "gpu with socket spec (single socket)",
			gres: "gpu:8(S:0-1)",
			want: 8,
		},
		{
			name: "gpu with socket spec (single socket variant)",
			gres: "gpu:8(S:0)",
			want: 8,
		},
		{
			name: "gpu with socket spec (multiple gpus)",
			gres: "gpu:4(S:0-1)",
			want: 4,
		},
		{
			name: "single gpu with socket",
			gres: "gpu:1(S:0)",
			want: 1,
		},
		{
			name: "no gres prefix",
			gres: "something else",
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseNodeGresGpu(tt.gres); got != tt.want {
				t.Errorf("ParseGpuGres() = %v, want %v", got, tt.want)
			}
		})
	}
}