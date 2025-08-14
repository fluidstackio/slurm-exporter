// SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"context"
	"sort"
	"testing"

	api "github.com/SlinkyProject/slurm-client/api/v0041"
	"github.com/SlinkyProject/slurm-client/pkg/client"
	"github.com/SlinkyProject/slurm-client/pkg/client/fake"
	"github.com/SlinkyProject/slurm-client/pkg/types"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/utils/ptr"
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
				Memory: 1024 * 1024 * 1024,
				Gpus:   2,
			},
		},
		{
			name: "test job 2",
			args: args{
				job: *job2,
			},
			want: jobResources{
				Cpus:   12,
				Memory: 3072 * 1024 * 1024,
				Gpus:   4,
			},
		},
		{
			name: "job with GPU only in TRES",
			args: args{
				job: types.V0041JobInfo{V0041JobInfo: api.V0041JobInfo{
					TresAllocStr: ptr.To("cpu=16,mem=8192M,node=1,billing=16,gres/gpu=8"),
				}},
			},
			want: jobResources{
				Gpus: 8,
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


func TestJobCollector_getJobMetrics(t *testing.T) {
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
		want    *JobMetrics
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
			want: &JobMetrics{
				JobIndividualStates: []JobIndividualStates{},
			},
		},
		{
			name: "test data",
			fields: fields{
				slurmClient: testDataClient,
			},
			args: args{
				ctx: context.TODO(),
			},
			want: &JobMetrics{
				JobCount:  4,
				JobStates: JobStates{Pending: 2, Running: 2, Hold: 1},
				JobIndividualStates: []JobIndividualStates{
					{JobID: "0", JobName: "test_job_0", Nodes: []string{"node1"}, Account: "root", Partition: "blue", UserID: "0", UserName: "root", Running: 1, CpusAlloc: 8, MemoryAlloc: 1024 * 1024 * 1024, GpusAlloc: 2},
					{JobID: "1", JobName: "test_job_1", Nodes: []string{""}, Account: "", Partition: "blue,green", UserID: "0", UserName: "root", Pending: 1, Hold: 1, CpusAlloc: 0, MemoryAlloc: 0, GpusAlloc: 0},
					{JobID: "2", JobName: "test_job_2", Nodes: []string{"node2", "node3"}, Account: "root", Partition: "green", UserID: "1000", UserName: "", Running: 1, CpusAlloc: 12, MemoryAlloc: 3072 * 1024 * 1024, GpusAlloc: 4},
					{JobID: "3", JobName: "test_job_3", Nodes: []string{""}, Account: "", Partition: "green", UserID: "1000", UserName: "", Pending: 1, CpusAlloc: 0, MemoryAlloc: 0, GpusAlloc: 0},
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
			c := &jobCollector{
				slurmClient: tt.fields.slurmClient,
			}
			got, err := c.getJobMetrics(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("jobCollector.getJobMetrics() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Sort JobIndividualStates for consistent comparison
			if got != nil {
				sort.Slice(got.JobIndividualStates, func(i, j int) bool {
					return got.JobIndividualStates[i].JobID < got.JobIndividualStates[j].JobID
				})
			}

			opts := []cmp.Option{
				cmpopts.IgnoreUnexported(JobMetrics{}),
				cmpopts.IgnoreFields(JobStates{}, "total"),
			}
			if diff := cmp.Diff(tt.want, got, opts...); diff != "" {
				t.Errorf("jobCollector.getJobMetrics() = (-want,+got):\n%s", diff)
			}
		})
	}
}

func TestJobCollector_Collect(t *testing.T) {
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
			c := NewJobCollector(tt.fields.slurmClient)
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

func TestJobCollector_Describe(t *testing.T) {
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
			c := NewJobCollector(tt.fields.slurmClient)
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

func TestJobCollector_JobStateMetric(t *testing.T) {
	// Create a job collector with test data
	c := NewJobCollector(testDataClient)
	ch := make(chan prometheus.Metric, 100)
	
	// Collect metrics
	c.Collect(ch)
	close(ch)
	
	// Track found metrics
	foundStates := make(map[string]bool)
	expectedStates := map[string]struct {
		state string
		hold  string
	}{
		"0": {state: "running", hold: "false"},  // job0 is RUNNING
		"1": {state: "pending", hold: "true"},   // job1 is PENDING with Hold=true
		"2": {state: "running", hold: "false"},  // job2 is RUNNING
		"3": {state: "pending", hold: "false"},  // job3 is PENDING
	}
	
	// Check all metrics
	for metric := range ch {
		metricDto := &dto.Metric{}
		err := metric.Write(metricDto)
		assert.NoError(t, err)
		
		// Look for slurm_job_state metrics
		if metricDto.Label != nil {
			// Check if this is our new metric by looking for the state label
			var hasStateLabel, hasHoldLabel bool
			var jobID, stateName, holdValue string
			
			for _, label := range metricDto.Label {
				if label.GetName() == "state" {
					hasStateLabel = true
					stateName = label.GetValue()
				}
				if label.GetName() == "hold" {
					hasHoldLabel = true
					holdValue = label.GetValue()
				}
				if label.GetName() == "job_id" {
					jobID = label.GetValue()
				}
			}
			
			if hasStateLabel && hasHoldLabel {
				// Verify the state and hold value match expected
				if expected, ok := expectedStates[jobID]; ok {
					assert.Equal(t, expected.state, stateName, "Job %s should have state %s", jobID, expected.state)
					assert.Equal(t, expected.hold, holdValue, "Job %s should have hold %s", jobID, expected.hold)
					foundStates[jobID] = true
				}
			}
		}
	}
	
	// Ensure we found all expected job states
	for jobID := range expectedStates {
		assert.True(t, foundStates[jobID], "Did not find slurm_job_state metric for job %s", jobID)
	}
}

func TestJobStateMetric_AllStates(t *testing.T) {
	// Test that all possible states are handled correctly
	testCases := []struct {
		name          string
		jobState      []api.V0041JobInfoJobState
		hold          bool
		expectedState string
		expectedHold  string
	}{
		{"bootfail", []api.V0041JobInfoJobState{api.V0041JobInfoJobStateBOOTFAIL}, false, "bootfail", "false"},
		{"cancelled", []api.V0041JobInfoJobState{api.V0041JobInfoJobStateCANCELLED}, false, "cancelled", "false"},
		{"completed", []api.V0041JobInfoJobState{api.V0041JobInfoJobStateCOMPLETED}, false, "completed", "false"},
		{"deadline", []api.V0041JobInfoJobState{api.V0041JobInfoJobStateDEADLINE}, false, "deadline", "false"},
		{"failed", []api.V0041JobInfoJobState{api.V0041JobInfoJobStateFAILED}, false, "failed", "false"},
		{"pending", []api.V0041JobInfoJobState{api.V0041JobInfoJobStatePENDING}, false, "pending", "false"},
		{"pending_with_hold", []api.V0041JobInfoJobState{api.V0041JobInfoJobStatePENDING}, true, "pending", "true"},
		{"preempted", []api.V0041JobInfoJobState{api.V0041JobInfoJobStatePREEMPTED}, false, "preempted", "false"},
		{"running", []api.V0041JobInfoJobState{api.V0041JobInfoJobStateRUNNING}, false, "running", "false"},
		{"suspended", []api.V0041JobInfoJobState{api.V0041JobInfoJobStateSUSPENDED}, false, "suspended", "false"},
		{"timeout", []api.V0041JobInfoJobState{api.V0041JobInfoJobStateTIMEOUT}, false, "timeout", "false"},
		{"nodefail", []api.V0041JobInfoJobState{api.V0041JobInfoJobStateNODEFAIL}, false, "nodefail", "false"},
		{"outofmemory", []api.V0041JobInfoJobState{api.V0041JobInfoJobStateOUTOFMEMORY}, false, "outofmemory", "false"},
		{"unknown", []api.V0041JobInfoJobState{}, false, "unknown", "false"},
		{"unknown_with_hold", []api.V0041JobInfoJobState{}, true, "unknown", "true"},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a test job with the specified state
			testJob := &types.V0041JobInfo{V0041JobInfo: api.V0041JobInfo{
				JobId:    ptr.To[int32](999),
				Name:     ptr.To("test_state_job"),
				JobState: ptr.To(tc.jobState),
				Hold:     ptr.To(tc.hold),
				UserId:   ptr.To[int32](0),
				UserName: ptr.To("root"),
				Account:  ptr.To("test"),
				Partition: ptr.To("test"),
			}}
			
			// Create a fake client with just this job
			jobList := &types.V0041JobInfoList{
				Items: []types.V0041JobInfo{*testJob},
			}
			client := fake.NewClientBuilder().WithLists(jobList).Build()
			
			// Create collector and collect metrics
			c := NewJobCollector(client)
			ch := make(chan prometheus.Metric, 100)
			c.Collect(ch)
			close(ch)
			
			// Find the slurm_job_state metric
			var foundState, foundHold string
			for metric := range ch {
				metricDto := &dto.Metric{}
				err := metric.Write(metricDto)
				assert.NoError(t, err)
				
				var hasState, hasHold bool
				for _, label := range metricDto.Label {
					if label.GetName() == "state" {
						foundState = label.GetValue()
						hasState = true
					}
					if label.GetName() == "hold" {
						foundHold = label.GetValue()
						hasHold = true
					}
				}
				
				if hasState && hasHold {
					break
				}
			}
			
			assert.Equal(t, tc.expectedState, foundState, "Expected state %s for test case %s", tc.expectedState, tc.name)
			assert.Equal(t, tc.expectedHold, foundHold, "Expected hold %s for test case %s", tc.expectedHold, tc.name)
		})
	}
}

func TestJobFlagMetric(t *testing.T) {
	// Test flag concatenation
	testCases := []struct {
		name         string
		jobState     []api.V0041JobInfoJobState
		expectedFlag string
		shouldEmit   bool
	}{
		{"no_flags", []api.V0041JobInfoJobState{api.V0041JobInfoJobStateRUNNING}, "", false},
		{"completing", []api.V0041JobInfoJobState{api.V0041JobInfoJobStateRUNNING, api.V0041JobInfoJobStateCOMPLETING}, "completing", true},
		{"configuring", []api.V0041JobInfoJobState{api.V0041JobInfoJobStatePENDING, api.V0041JobInfoJobStateCONFIGURING}, "configuring", true},
		{"powerupnode", []api.V0041JobInfoJobState{api.V0041JobInfoJobStatePENDING, api.V0041JobInfoJobStatePOWERUPNODE}, "powerupnode", true},
		{"stageout", []api.V0041JobInfoJobState{api.V0041JobInfoJobStateRUNNING, api.V0041JobInfoJobStateSTAGEOUT}, "stageout", true},
		{"multiple_flags", []api.V0041JobInfoJobState{
			api.V0041JobInfoJobStateRUNNING,
			api.V0041JobInfoJobStateCOMPLETING,
			api.V0041JobInfoJobStateSTAGEOUT,
		}, "completing+stageout", true},
		{"all_flags", []api.V0041JobInfoJobState{
			api.V0041JobInfoJobStatePENDING,
			api.V0041JobInfoJobStateCOMPLETING,
			api.V0041JobInfoJobStateCONFIGURING,
			api.V0041JobInfoJobStatePOWERUPNODE,
			api.V0041JobInfoJobStateSTAGEOUT,
		}, "completing+configuring+powerupnode+stageout", true},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a test job with the specified states
			testJob := &types.V0041JobInfo{V0041JobInfo: api.V0041JobInfo{
				JobId:     ptr.To[int32](999),
				Name:      ptr.To("test_flag_job"),
				JobState:  ptr.To(tc.jobState),
				UserId:    ptr.To[int32](0),
				UserName:  ptr.To("root"),
				Account:   ptr.To("test"),
				Partition: ptr.To("test"),
			}}
			
			// Create a fake client with just this job
			jobList := &types.V0041JobInfoList{
				Items: []types.V0041JobInfo{*testJob},
			}
			client := fake.NewClientBuilder().WithLists(jobList).Build()
			
			// Create collector and collect metrics
			c := NewJobCollector(client)
			ch := make(chan prometheus.Metric, 100)
			c.Collect(ch)
			close(ch)
			
			// Find the slurm_job_flag metric
			var foundFlag string
			var foundFlagMetric bool
			for metric := range ch {
				metricDto := &dto.Metric{}
				err := metric.Write(metricDto)
				assert.NoError(t, err)
				
				for _, label := range metricDto.Label {
					if label.GetName() == "flag" {
						foundFlag = label.GetValue()
						foundFlagMetric = true
						break
					}
				}
				
				if foundFlagMetric {
					break
				}
			}
			
			if tc.shouldEmit {
				assert.True(t, foundFlagMetric, "Expected flag metric to be emitted for test case %s", tc.name)
				assert.Equal(t, tc.expectedFlag, foundFlag, "Expected flag %s for test case %s", tc.expectedFlag, tc.name)
			} else {
				assert.False(t, foundFlagMetric, "Expected no flag metric for test case %s", tc.name)
			}
		})
	}
}
