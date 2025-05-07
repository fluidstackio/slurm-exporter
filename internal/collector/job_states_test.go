// SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"context"
	"testing"

	api "github.com/SlinkyProject/slurm-client/api/v0041"
	"github.com/SlinkyProject/slurm-client/pkg/client"
	"github.com/SlinkyProject/slurm-client/pkg/client/fake"
	"github.com/SlinkyProject/slurm-client/pkg/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/utils/ptr"
)

func Test_parseJobState(t *testing.T) {
	type args struct {
		job types.V0041JobInfo
	}
	tests := []struct {
		name string
		args args
		want *JobStates
	}{
		{
			name: "empty",
			want: &JobStates{Total: 1},
		},
		{
			name: "pending",
			args: args{
				job: types.V0041JobInfo{V0041JobInfo: api.V0041JobInfo{
					JobState: ptr.To([]api.V0041JobInfoJobState{
						api.V0041JobInfoJobStatePENDING,
					}),
				}},
			},
			want: &JobStates{Total: 1, Pending: 1},
		},
		{
			name: "running",
			args: args{
				job: types.V0041JobInfo{V0041JobInfo: api.V0041JobInfo{
					JobState: ptr.To([]api.V0041JobInfoJobState{
						api.V0041JobInfoJobStateRUNNING,
					}),
				}},
			},
			want: &JobStates{Total: 1, Running: 1},
		},
		{
			name: "all states, all flags",
			args: args{
				job: types.V0041JobInfo{V0041JobInfo: api.V0041JobInfo{
					JobState: ptr.To([]api.V0041JobInfoJobState{
						api.V0041JobInfoJobStateBOOTFAIL,
						api.V0041JobInfoJobStateCANCELLED,
						api.V0041JobInfoJobStateCOMPLETED,
						api.V0041JobInfoJobStateCOMPLETING,
						api.V0041JobInfoJobStateCONFIGURING,
						api.V0041JobInfoJobStateDEADLINE,
						api.V0041JobInfoJobStateFAILED,
						api.V0041JobInfoJobStateLAUNCHFAILED,
						api.V0041JobInfoJobStateNODEFAIL,
						api.V0041JobInfoJobStateOUTOFMEMORY,
						api.V0041JobInfoJobStatePENDING,
						api.V0041JobInfoJobStatePOWERUPNODE,
						api.V0041JobInfoJobStatePREEMPTED,
						api.V0041JobInfoJobStateRECONFIGFAIL,
						api.V0041JobInfoJobStateREQUEUED,
						api.V0041JobInfoJobStateREQUEUEFED,
						api.V0041JobInfoJobStateREQUEUEHOLD,
						api.V0041JobInfoJobStateRESIZING,
						api.V0041JobInfoJobStateRESVDELHOLD,
						api.V0041JobInfoJobStateREVOKED,
						api.V0041JobInfoJobStateRUNNING,
						api.V0041JobInfoJobStateSIGNALING,
						api.V0041JobInfoJobStateSPECIALEXIT,
						api.V0041JobInfoJobStateSTAGEOUT,
						api.V0041JobInfoJobStateSTOPPED,
						api.V0041JobInfoJobStateSUSPENDED,
						api.V0041JobInfoJobStateTIMEOUT,
						api.V0041JobInfoJobStateUPDATEDB,
					}),
					Hold: ptr.To(true),
				}},
			},
			want: &JobStates{
				Total:   1,
				Pending: 1,
				Hold:    1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics := &JobStates{}
			parseJobState(metrics, tt.args.job)
			if !apiequality.Semantic.DeepEqual(metrics, tt.want) {
				t.Errorf("parseJobState() metrics = %v, want %v", metrics, tt.want)
			}
		})
	}
}

func TestJobStateCollector_getJobStates(t *testing.T) {
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
		want    *JobStates
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
			want: &JobStates{},
		},
		{
			name: "test data",
			fields: fields{
				slurmClient: testDataClient,
			},
			args: args{
				ctx: context.TODO(),
			},
			want: &JobStates{
				Total:   4,
				Pending: 2,
				Running: 2,
				Hold:    1,
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
			c := &jobStateCollector{
				slurmClient: tt.fields.slurmClient,
			}
			got, err := c.getJobStates(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("jobStateCollector.getJobStates() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !apiequality.Semantic.DeepEqual(got, tt.want) {
				t.Errorf("jobStateCollector.getJobStates() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJobStateCollector_Collect(t *testing.T) {
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
			c := NewJobStateCollector(tt.fields.slurmClient)
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

func TestJobStateCollector_Describe(t *testing.T) {
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
			c := NewJobStateCollector(tt.fields.slurmClient)
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
