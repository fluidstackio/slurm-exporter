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

func Test_parseNodeState(t *testing.T) {
	type args struct {
		node types.V0041Node
	}
	tests := []struct {
		name string
		args args
		want *NodeStates
	}{
		{
			name: "empty",
			want: &NodeStates{Total: 1},
		},
		{
			name: "allocated",
			args: args{
				node: types.V0041Node{V0041Node: api.V0041Node{
					State: ptr.To([]api.V0041NodeState{
						api.V0041NodeStateALLOCATED,
					}),
				}},
			},
			want: &NodeStates{Total: 1, Allocated: 1},
		},
		{
			name: "down",
			args: args{
				node: types.V0041Node{V0041Node: api.V0041Node{
					State: ptr.To([]api.V0041NodeState{
						api.V0041NodeStateDOWN,
					}),
				}},
			},
			want: &NodeStates{Total: 1, Down: 1},
		},
		{
			name: "error",
			args: args{
				node: types.V0041Node{V0041Node: api.V0041Node{
					State: ptr.To([]api.V0041NodeState{
						api.V0041NodeStateERROR,
					}),
				}},
			},
			want: &NodeStates{Total: 1, Error: 1},
		},
		{
			name: "idle",
			args: args{
				node: types.V0041Node{V0041Node: api.V0041Node{
					State: ptr.To([]api.V0041NodeState{
						api.V0041NodeStateIDLE,
					}),
				}},
			},
			want: &NodeStates{Total: 1, Idle: 1},
		},
		{
			name: "mixed",
			args: args{
				node: types.V0041Node{V0041Node: api.V0041Node{
					State: ptr.To([]api.V0041NodeState{
						api.V0041NodeStateMIXED,
					}),
				}},
			},
			want: &NodeStates{Total: 1, Mixed: 1},
		},
		{
			name: "all states, all flags",
			args: args{
				node: types.V0041Node{V0041Node: api.V0041Node{
					State: ptr.To([]api.V0041NodeState{
						api.V0041NodeStateALLOCATED,
						api.V0041NodeStateCLOUD,
						api.V0041NodeStateCOMPLETING,
						api.V0041NodeStateDOWN,
						api.V0041NodeStateDRAIN,
						api.V0041NodeStateDYNAMICFUTURE,
						api.V0041NodeStateDYNAMICNORM,
						api.V0041NodeStateERROR,
						api.V0041NodeStateFAIL,
						api.V0041NodeStateFUTURE,
						api.V0041NodeStateIDLE,
						api.V0041NodeStateINVALID,
						api.V0041NodeStateINVALIDREG,
						api.V0041NodeStateMAINTENANCE,
						api.V0041NodeStateMIXED,
						api.V0041NodeStateNOTRESPONDING,
						api.V0041NodeStatePLANNED,
						api.V0041NodeStatePOWERDOWN,
						api.V0041NodeStatePOWERDRAIN,
						api.V0041NodeStatePOWEREDDOWN,
						api.V0041NodeStatePOWERINGDOWN,
						api.V0041NodeStatePOWERINGUP,
						api.V0041NodeStatePOWERUP,
						api.V0041NodeStateREBOOTCANCELED,
						api.V0041NodeStateREBOOTISSUED,
						api.V0041NodeStateREBOOTREQUESTED,
						api.V0041NodeStateRESERVED,
						api.V0041NodeStateRESUME,
						api.V0041NodeStateUNDRAIN,
						api.V0041NodeStateUNKNOWN,
					}),
				}},
			},
			want: &NodeStates{
				Total:       1,
				Allocated:   1,
				Completing:  1,
				Drain:       1,
				Maintenance: 1,
				Reserved:    1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics := &NodeStates{}
			parseNodeState(metrics, tt.args.node)
			if !apiequality.Semantic.DeepEqual(metrics, tt.want) {
				t.Errorf("parseNodeState() = %v, want %v", metrics, tt.want)
			}
		})
	}
}

func TestNodeStateCollector_getNodeStates(t *testing.T) {
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
		want    *NodeStates
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
			want: &NodeStates{},
		},
		{
			name: "test data",
			fields: fields{
				slurmClient: testDataClient,
			},
			args: args{
				ctx: context.TODO(),
			},
			want: &NodeStates{
				Total:      4,
				Allocated:  2,
				Idle:       1,
				Mixed:      1,
				Completing: 1,
				Drain:      1,
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
			c := &nodeStateCollector{
				slurmClient: tt.fields.slurmClient,
			}
			got, err := c.getNodeStates(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("nodeStateCollector.getNodeStates() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !apiequality.Semantic.DeepEqual(got, tt.want) {
				t.Errorf("nodeStateCollector.getNodeStates() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNodeStateCollector_Collect(t *testing.T) {
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
			c := NewNodeStateCollector(tt.fields.slurmClient)
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

func TestNodeStateCollector_Describe(t *testing.T) {
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
			c := NewNodeStateCollector(tt.fields.slurmClient)
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
