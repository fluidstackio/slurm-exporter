// SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
// SPDX-License-Identifier: Apache-2.0

package exporter

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"

	"github.com/SlinkyProject/slurm-client/pkg/client"
	"github.com/SlinkyProject/slurm-client/pkg/client/fake"
	"github.com/SlinkyProject/slurm-client/pkg/client/interceptor"
	"github.com/SlinkyProject/slurm-client/pkg/object"

	slurmtypes "github.com/SlinkyProject/slurm-client/pkg/types"
)

var (
	exporter_url = "http://localhost:8080"
	cacheFreq    = 5 * time.Second
)

// newSlurmCollectorHelper will initialize a Slurm Collector for
// metrics and will also load a predefined set of objects into
// the cache of a fake Slurm client.
func newSlurmCollectorHelper() SlurmCollector {
	jobs := &slurmtypes.V0041JobInfoList{}
	partitions := &slurmtypes.V0041PartitionInfoList{}
	nodes := &slurmtypes.V0041NodeList{}

	jobs.AppendItem(jobInfoA)
	jobs.AppendItem(jobInfoB)
	jobs.AppendItem(jobInfoC)
	partitions.AppendItem(partitionA)
	partitions.AppendItem(partitionB)
	nodes.AppendItem(nodeA)
	nodes.AppendItem(nodeB)
	nodes.AppendItem(nodeC)

	slurmClient := newFakeClientList(interceptor.Funcs{}, jobs, partitions, nodes)
	sc := NewSlurmCollector(slurmClient, true)
	return *sc
}

// newFakeClientList will return a fake Slurm client with interceptor functions
// and lists of Slurm objects.
func newFakeClientList(interceptorFuncs interceptor.Funcs, initObjLists ...object.ObjectList) client.Client {
	return fake.NewClientBuilder().
		WithLists(initObjLists...).
		WithInterceptorFuncs(interceptorFuncs).
		Build()
}

// TestSlurmParse will test that slurmParse() correctly calculates the
// metrics for a predefined cache of Slurm objects.
func TestSlurmParse(t *testing.T) {
	os.Setenv("SLURM_JWT", "foo")
	c, err := NewSlurmClient(exporter_url, cacheFreq)
	assert.Nil(t, err)
	sc := NewSlurmCollector(c, false)

	jobs := &slurmtypes.V0041JobInfoList{}
	partitions := &slurmtypes.V0041PartitionInfoList{}
	nodes := &slurmtypes.V0041NodeList{}

	jobs.AppendItem(jobInfoA)
	jobs.AppendItem(jobInfoB)
	jobs.AppendItem(jobInfoC)
	jobs.AppendItem(jobInfoD)
	partitions.AppendItem(partitionA)
	partitions.AppendItem(partitionB)
	nodes.AppendItem(nodeA)
	nodes.AppendItem(nodeB)
	nodes.AppendItem(nodeC)

	slurmData := sc.slurmParse(jobs, nodes, partitions)
	assert.NotEmpty(t, slurmData)

	assert.Equal(t, 2, len(slurmData.jobs))
	assert.Equal(t, &JobData{Count: 2, Pending: 1, Running: 1, Hold: 1}, slurmData.jobs["slurm"])
	assert.Equal(t, &JobData{Count: 1, Pending: 1, Running: 0, Hold: 0}, slurmData.jobs["hal"])
	assert.Equal(t, 2, len(slurmData.partitions))
	assert.Equal(t, &PartitionData{Nodes: 2, Cpus: 20, PendingJobs: 1, PendingMaxNodes: 1, Jobs: 1, RunningJobs: 0, HoldJobs: 0, Alloc: 4, Idle: 16}, slurmData.partitions["purple"])
	assert.Equal(t, &PartitionData{Nodes: 2, Cpus: 32, PendingJobs: 1, PendingMaxNodes: 0, Jobs: 2, RunningJobs: 1, HoldJobs: 1, Alloc: 16, Idle: 16}, slurmData.partitions["green"])
	assert.Equal(t, len(slurmData.nodes), 3)
	assert.Equal(t, &NodeData{
		Cpus:  12,
		Alloc: 0,
		Idle:  12,
		States: NodeStates{
			allocated:  0,
			completing: 0,
			down:       0,
			drain:      0,
			err:        0,
			idle:       1,
			mixed:      0,
			reserved:   0,
		},
		unavailable:   0,
		CombinedState: "idle",
	}, slurmData.nodes["kind-worker"])
	assert.Equal(t, &NodeData{
		Cpus:  24,
		Alloc: 12,
		Idle:  12,
		States: NodeStates{
			allocated:   1,
			completing:  0,
			down:        0,
			drain:       0,
			err:         0,
			idle:        0,
			maintenance: 0,
			mixed:       0,
			reserved:    0,
		},
		unavailable:   0,
		CombinedState: "allocated",
	}, slurmData.nodes["kind-worker2"])
	assert.Equal(t, &NodeData{
		Cpus:  8,
		Alloc: 4,
		Idle:  4,
		States: NodeStates{
			allocated:   0,
			completing:  1,
			down:        1,
			drain:       1,
			err:         1,
			idle:        0,
			maintenance: 1,
			mixed:       1,
			reserved:    1,
		},
		unavailable:     0,
		CombinedState:   "completing+down+drain+err+maintenance+mixed+reserved",
		Reason:          "",
		ReasonSetByUser: "",
		ReasonChangedAt: 0,
	}, slurmData.nodes["kind-worker3"])
	assert.Equal(t, NodeStates{allocated: 1, completing: 1, down: 1, drain: 1, err: 1, idle: 1, maintenance: 1, mixed: 1, reserved: 1}, slurmData.nodestates)
}

// TestSlurmClient will test how SlurmClient() initializes
// a Slurm client.
func TestSlurmClient(t *testing.T) {
	var c client.Client
	var err error

	os.Unsetenv("SLURM_JWT")
	c, err = NewSlurmClient(exporter_url, cacheFreq)
	assert.NotNil(t, err)
	assert.Nil(t, c)

	os.Setenv("SLURM_JWT", "")
	c, err = NewSlurmClient(exporter_url, cacheFreq)
	assert.NotNil(t, err)
	assert.Nil(t, c)

	os.Setenv("SLURM_JWT", "foo")
	c, err = NewSlurmClient(exporter_url, cacheFreq)
	assert.Nil(t, err)
	assert.NotNil(t, c)
}

// TestSlurmCollectType will test that slurmCollectType returns
// the correct data from a Slurm client. A fake Slurm
// client is used and preloaded with a cache of objects. In order
// to simulate error conditions, interceptor functions are used.
func TestSlurmCollectType(t *testing.T) {

	sc := newSlurmCollectorHelper()

	jobsTest := &slurmtypes.V0041JobInfoList{}
	partitionsTest := &slurmtypes.V0041PartitionInfoList{}
	nodesTest := &slurmtypes.V0041NodeList{}

	err := sc.slurmCollectType(jobsTest)
	assert.Equal(t, nil, err)
	assert.Equal(t, 3, len(jobsTest.Items))
	err = sc.slurmCollectType(partitionsTest)
	assert.Equal(t, nil, err)
	assert.Equal(t, 2, len(partitionsTest.Items))
	err = sc.slurmCollectType(nodesTest)
	assert.Equal(t, nil, err)
	assert.Equal(t, 3, len(nodesTest.Items))

	interceptorFunc := interceptor.Funcs{
		List: func(ctx context.Context, list object.ObjectList, opts ...client.ListOption) error {
			return errors.New("error")
		},
	}
	sc.slurmClient = newFakeClientList(interceptorFunc)
	err = sc.slurmCollectType(jobsTest)
	assert.NotEqual(t, nil, err)
}

// TestCollect will test the Prometheus Collect method that implements
// the Collector interface.
func TestCollect(t *testing.T) {

	sc := newSlurmCollectorHelper()

	c := make(chan prometheus.Metric)
	var metric prometheus.Metric
	var numMetric int

	interceptorFunc := interceptor.Funcs{
		List: func(ctx context.Context, list object.ObjectList, opts ...client.ListOption) error {
			if list.GetType() == slurmtypes.ObjectTypeV0041Node {
				return errors.New("error")
			}
			return nil
		},
	}
	sc.slurmClient = newFakeClientList(interceptorFunc)
	sc.Collect(c)

	interceptorFunc = interceptor.Funcs{
		List: func(ctx context.Context, list object.ObjectList, opts ...client.ListOption) error {
			if list.GetType() == slurmtypes.ObjectTypeV0041JobInfo {
				return errors.New("error")
			}
			return nil
		},
	}
	sc.slurmClient = newFakeClientList(interceptorFunc)
	sc.Collect(c)

	interceptorFunc = interceptor.Funcs{
		List: func(ctx context.Context, list object.ObjectList, opts ...client.ListOption) error {
			if list.GetType() == slurmtypes.ObjectTypeV0041PartitionInfo {
				return errors.New("error")
			}
			return nil
		},
	}
	sc.slurmClient = newFakeClientList(interceptorFunc)
	sc.Collect(c)

	sc = newSlurmCollectorHelper()

	go func() {
		sc.Collect(c)
		close(c)
	}()
	for metric = range c {
		numMetric++
	}
	assert.NotNil(t, metric)
	assert.Equal(t, 167, numMetric)
}

// TestDescribe will test the Prometheus Describe method that implements
// the Collector interface. Verify the correct number of metric
// descriptions is returned from the channel.
func TestDescribe(t *testing.T) {
	c := make(chan *prometheus.Desc)
	var desc *prometheus.Desc
	var numDesc int
	cl, err := NewSlurmClient(exporter_url, cacheFreq)
	assert.Nil(t, err)
	sc := NewSlurmCollector(cl, false)

	go func() {
		sc.Describe(c)
		close(c)
	}()
	for desc = range c {
		numDesc++
		assert.NotNil(t, desc)
	}
	assert.NotNil(t, desc)
	assert.Equal(t, 82, numDesc)
}

// TestNewSlurmCollector will test that NewSlurmCollector
// returns the correct SlurmCollector.
func TestNewSlurmCollector(t *testing.T) {
	c, err := NewSlurmClient(exporter_url, cacheFreq)
	assert.Nil(t, err)
	assert.NotNil(t, c)
	sc := NewSlurmCollector(c, false)
	assert.NotNil(t, sc)
	assert.Equal(t, false, sc.perUserMetrics)
}
