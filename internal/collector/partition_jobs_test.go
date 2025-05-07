// SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"context"
	"testing"

	"github.com/SlinkyProject/slurm-client/pkg/client"
	"github.com/SlinkyProject/slurm-client/pkg/client/fake"
	"github.com/google/go-cmp/cmp"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
)

func TestPartitionJobCollector_getPartitionJobs(t *testing.T) {
	type fields struct {
		slurmClient client.Client
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    map[string]*PartitionJobs
		wantErr bool
	}{
		{
			name: "empty",
			fields: fields{
				slurmClient: fake.NewFakeClient(),
			},
			args: args{
				ctx: context.TODO(),
			},
			want: map[string]*PartitionJobs{},
		},
		{
			name: "test data",
			fields: fields{
				slurmClient: testDataClient,
			},
			args: args{
				ctx: context.TODO(),
			},
			want: map[string]*PartitionJobs{
				*partition1.Name: {
					JobStates: JobStates{
						Total:   2,
						Pending: 1,
						Running: 1,
						Hold:    1,
					},
					CpusAlloc: 8,
				},
				*partition2.Name: {
					JobStates: JobStates{
						Total:   3,
						Pending: 2,
						Running: 1,
						Hold:    1,
					},
					CpusAlloc:        12,
					PendingNodeCount: 2,
				},
			},
		},
		{
			name: "fail",
			fields: fields{
				slurmClient: testFailClient,
			},
			args: args{
				ctx: context.TODO(),
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &partitionJobCollector{
				slurmClient: tt.fields.slurmClient,
			}
			got, err := c.getPartitionJobs(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("partitionJobCollector.getPartitionJobs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !apiequality.Semantic.DeepEqual(got, tt.want) {
				t.Errorf("partitionJobCollector.getPartitionJobs() = %v, want %v", got, tt.want)
			}
			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(PartitionJobs{})); diff != "" {
				t.Errorf("partitionJobCollector.getPartitionJobs() = (-want,+got):\n%s", diff)
			}
		})
	}
}

func TestPartitionJobCollector_Collect(t *testing.T) {
	type fields struct {
		slurmClient client.Client
	}
	type args struct {
		ch chan prometheus.Metric
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		wantNone bool
	}{
		{
			name: "empty",
			fields: fields{
				slurmClient: fake.NewFakeClient(),
			},
			args: args{
				ch: make(chan prometheus.Metric),
			},
		},
		{
			name: "data",
			fields: fields{
				slurmClient: testDataClient,
			},
			args: args{
				ch: make(chan prometheus.Metric),
			},
		},
		{
			name: "failure",
			fields: fields{
				slurmClient: testFailClient,
			},
			args: args{
				ch: make(chan prometheus.Metric),
			},
			wantNone: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewPartitionJobCollector(tt.fields.slurmClient)
			go func() {
				c.Collect(tt.args.ch)
				close(tt.args.ch)
			}()
			var got int
			for range tt.args.ch {
				got++
			}
			if !tt.wantNone {
				assert.GreaterOrEqual(t, got, 0)
			} else {
				assert.Equal(t, got, 0)
			}
		})
	}
}

func TestPartitionJobCollector_Describe(t *testing.T) {
	type fields struct {
		slurmClient client.Client
	}
	type args struct {
		ch chan *prometheus.Desc
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "test",
			fields: fields{
				slurmClient: fake.NewFakeClient(),
			},
			args: args{
				ch: make(chan *prometheus.Desc),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewPartitionJobCollector(tt.fields.slurmClient)
			go func() {
				c.Describe(tt.args.ch)
				close(tt.args.ch)
			}()
			var desc *prometheus.Desc
			for desc = range tt.args.ch {
				assert.NotNil(t, desc)
			}
		})
	}
}
