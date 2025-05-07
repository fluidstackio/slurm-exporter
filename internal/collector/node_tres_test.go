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
)

func TestNodeTresCollector_getNodeTres(t *testing.T) {
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
		want    map[string]*NodeTres
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
			want: make(map[string]*NodeTres),
		},
		{
			name: "test data",
			fields: fields{
				slurmClient: testDataClient,
			},
			args: args{
				ctx: context.TODO(),
			},
			want: map[string]*NodeTres{
				*node0.Name: {
					CpusTotal:  16,
					CpusIdle:   16,
					partitions: *node0.Partitions,
				},
				*node1.Name: {
					CpusTotal:  8,
					CpusAlloc:  8,
					partitions: *node1.Partitions,
				},
				*node2.Name: {
					CpusTotal:  16,
					CpusAlloc:  16,
					partitions: *node2.Partitions,
				},
				*node3.Name: {
					CpusTotal:  6,
					CpusAlloc:  4,
					CpusIdle:   2,
					partitions: *node3.Partitions,
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
			c := &nodeTresCollector{
				slurmClient: tt.fields.slurmClient,
			}
			got, err := c.getNodeTres(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("nodeTresCollector.getNodeTres() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(NodeTres{})); diff != "" {
				t.Errorf("nodeTresCollector.getNodeTres() = (-want,+got):\n%s", diff)
			}
		})
	}
}

func TestNodeTresCollector_Collect(t *testing.T) {
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
			c := NewNodeTresCollector(tt.fields.slurmClient)
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

func TestNodeTresCollector_Describe(t *testing.T) {
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
			c := NewNodeTresCollector(tt.fields.slurmClient)
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
