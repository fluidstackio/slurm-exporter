// SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"testing"

	"github.com/SlinkyProject/slurm-client/pkg/types"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
)

func Test_getJobResourceAlloc(t *testing.T) {
	type args struct {
		job types.V0041JobInfo
	}
	tests := []struct {
		name string
		args args
		want jobResources
	}{
		{
			name: "empty",
			args: args{
				job: types.V0041JobInfo{},
			},
			want: jobResources{},
		},
		{
			name: "test job 0",
			args: args{
				job: *job0,
			},
			want: jobResources{
				Cpus:   8,
				Memory: 1024,
			},
		},
		{
			name: "test job 2",
			args: args{
				job: *job2,
			},
			want: jobResources{
				Cpus:   12,
				Memory: 3072,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getJobResourceAlloc(tt.args.job); !apiequality.Semantic.DeepEqual(got, tt.want) {
				t.Errorf("getJobResourceAlloc() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getJobPendingNodeCount(t *testing.T) {
	type args struct {
		job types.V0041JobInfo
	}
	tests := []struct {
		name string
		args args
		want uint
	}{
		{
			name: "empty",
			args: args{
				job: types.V0041JobInfo{},
			},
			want: 0,
		},
		{
			name: "test job 1",
			args: args{
				job: *job1,
			},
			want: 0,
		},
		{
			name: "test job 3",
			args: args{
				job: *job3,
			},
			want: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getJobPendingNodeCount(tt.args.job); got != tt.want {
				t.Errorf("getJobPendingNodeCount() = %v, want %v", got, tt.want)
			}
		})
	}
}
