// SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"context"
	"testing"

	"github.com/SlinkyProject/slurm-client/pkg/client"
	"github.com/SlinkyProject/slurm-client/pkg/client/fake"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func TestSchedulerCollector_getSchedulerMetrics(t *testing.T) {
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
		want    *SchedulerMetrics
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
			want: &SchedulerMetrics{},
		},
		{
			name: "test data",
			fields: fields{
				slurmClient: testDataClient,
			},
			args: args{
				ctx: context.TODO(),
			},
			want: &SchedulerMetrics{
				ScheduleCycleDepth:     1,
				ScheduleCycleLast:      1,
				ScheduleCycleMax:       1,
				ScheduleCycleMean:      1,
				ScheduleCycleMeanDepth: 1,
				ScheduleCyclePerMinute: 1,
				ScheduleCycleSum:       1,
				ScheduleCycleTotal:     1,
				ScheduleQueueLength:    1,
				ScheduleExit: ScheduleExitFields{
					DefaultQueueDepth: 2,
					EndJobQueue:       2,
					Licenses:          2,
					MaxJobStart:       2,
					MaxRpcCnt:         2,
					MaxSchedTime:      2,
				},

				BfActive:             true,
				BfBackfilledHetJobs:  3,
				BfBackfilledJobs:     3,
				BfCycleCounter:       3,
				BfCycleLast:          3,
				BfCycleMax:           3,
				BfCycleMean:          3,
				BfCycleSum:           3,
				BfDepthMean:          3,
				BfDepthMeanTry:       3,
				BfDepthSum:           3,
				BfDepthTrySum:        3,
				BfLastBackfilledJobs: 3,
				BfLastDepth:          3,
				BfLastDepthTry:       3,
				BfQueueLen:           3,
				BfQueueLenMean:       3,
				BfQueueLenSum:        3,
				BfTableSize:          3,
				BfTableSizeMean:      3,
				BfTableSizeSum:       3,
				BfWhenLastCycle:      3,
				BfExit: BfExitFields{
					BfMaxJobStart:   4,
					BfMaxJobTest:    4,
					BfMaxTime:       4,
					BfNodeSpaceSize: 4,
					EndJobQueue:     4,
					StateChanged:    4,
				},

				JobStatesTs:   5,
				JobsCanceled:  5,
				JobsCompleted: 5,
				JobsFailed:    5,
				JobsPending:   5,
				JobsRunning:   5,
				JobsStarted:   5,
				JobsSubmitted: 5,

				AgentCount:       6,
				AgentQueueSize:   6,
				AgentThreadCount: 6,

				ServerThreadCount: 7,

				DbdAgentQueueSize: 8,
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
			c := &schedulerCollector{
				slurmClient: tt.fields.slurmClient,
			}
			got, err := c.getSchedulerMetrics(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("schedulerCollector.getSchedulerMetrics() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			opts := []cmp.Option{
				cmpopts.IgnoreUnexported(SchedulerMetrics{}),
			}
			if diff := cmp.Diff(tt.want, got, opts...); diff != "" {
				t.Errorf("schedulerCollector.getSchedulerMetrics() = (-want,+got):\n%s", diff)
			}
		})
	}
}

func TestSchedulerCollector_Collect(t *testing.T) {
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
			c := NewSchedulerCollector(tt.fields.slurmClient)
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

func TestSchedulerCollector_Describe(t *testing.T) {
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
			c := NewSchedulerCollector(tt.fields.slurmClient)
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
