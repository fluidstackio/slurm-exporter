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

func TestParseTresGpu(t *testing.T) {
	tests := []struct {
		name    string
		tresStr string
		want    uint
	}{
		{
			name:    "empty string",
			tresStr: "",
			want:    0,
		},
		{
			name:    "TRES with 8 GPUs from nodes example",
			tresStr: "cpu=128,mem=1612647M,billing=128,gres/gpu=8",
			want:    8,
		},
		{
			name:    "TRES with 8 GPUs from jobs example",
			tresStr: "cpu=96,mem=1612647M,node=1,billing=96,gres/gpu=8",
			want:    8,
		},
		{
			name:    "TRES with single GPU",
			tresStr: "cpu=1,mem=1000M,gres/gpu=1",
			want:    1,
		},
		{
			name:    "TRES with no GPU",
			tresStr: "cpu=128,mem=1612647M,billing=128",
			want:    0,
		},
		{
			name:    "TRES with zero GPUs explicitly",
			tresStr: "cpu=128,mem=1612647M,gres/gpu=0",
			want:    0,
		},
		{
			name:    "TRES with double-digit GPUs",
			tresStr: "cpu=256,mem=3225294M,gres/gpu=16",
			want:    16,
		},
		{
			name:    "TRES with GPU at beginning",
			tresStr: "gres/gpu=4,cpu=64,mem=806323M",
			want:    4,
		},
		{
			name:    "TRES with GPU only",
			tresStr: "gres/gpu=3",
			want:    3,
		},
		{
			name:    "TRES with invalid GPU value",
			tresStr: "cpu=32,mem=403161M,gres/gpu=invalid",
			want:    0,
		},
		{
			name:    "TRES with negative GPU value",
			tresStr: "cpu=32,mem=403161M,gres/gpu=-5",
			want:    0,
		},
		{
			name:    "TRES with partial GPU string",
			tresStr: "cpu=32,mem=403161M,gres/gpu",
			want:    0,
		},
		{
			name:    "TRES with other gres types",
			tresStr: "cpu=32,mem=403161M,gres/nic=2,gres/gpu=5",
			want:    5,
		},
		{
			name:    "TRES with spaces",
			tresStr: "cpu=32, mem=403161M, gres/gpu=6",
			want:    6,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseTresGpu(tt.tresStr); got != tt.want {
				t.Errorf("ParseTresGpu(%q) = %v, want %v", tt.tresStr, got, tt.want)
			}
		})
	}
}

func Test_parseNodeList(t *testing.T) {
	tests := []struct {
		name     string
		nodeList string
		want     []string
	}{
		{
			name:     "single node",
			nodeList: "node1",
			want:     []string{"node1"},
		},
		{
			name:     "comma separated nodes",
			nodeList: "node1,node2,node3",
			want:     []string{"node1", "node2", "node3"},
		},
		{
			name:     "bracket notation with range",
			nodeList: "node[1-3]",
			want:     []string{"node1", "node2", "node3"},
		},
		{
			name:     "bracket notation with list",
			nodeList: "node[1,3,5]",
			want:     []string{"node1", "node3", "node5"},
		},
		{
			name:     "complex slurm format",
			nodeList: "node-[913,928,955,954,517,557,600,555,586,524,601,575,572,626,662,696,671,717,702,720,793,768,791,781,814,751,790,846,873,816,898,895]",
			want: []string{
				"node-913", "node-928", "node-955", "node-954",
				"node-517", "node-557", "node-600", "node-555",
				"node-586", "node-524", "node-601", "node-575",
				"node-572", "node-626", "node-662", "node-696",
				"node-671", "node-717", "node-702", "node-720",
				"node-793", "node-768", "node-791", "node-781",
				"node-814", "node-751", "node-790", "node-846",
				"node-873", "node-816", "node-898", "node-895",
			},
		},
		{
			name:     "bracket notation with mixed range and list",
			nodeList: "compute[1-3,5,7-9]",
			want:     []string{"compute1", "compute2", "compute3", "compute5", "compute7", "compute8", "compute9"},
		},
		{
			name:     "empty string",
			nodeList: "",
			want:     []string{},
		},
		{
			name:     "single node with bracket",
			nodeList: "node[42]",
			want:     []string{"node42"},
		},
		{
			name:     "node with suffix after bracket",
			nodeList: "rack[1-2]node",
			want:     []string{"rack1node", "rack2node"},
		},
		{
			name:     "comma separated with spaces",
			nodeList: "node1, node2, node3",
			want:     []string{"node1", "node2", "node3"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseNodeList(tt.nodeList)
			if len(got) != len(tt.want) {
				t.Errorf("parseNodeList() returned %d nodes, want %d nodes", len(got), len(tt.want))
				t.Errorf("got: %v", got)
				t.Errorf("want: %v", tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("parseNodeList()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}
