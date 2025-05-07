// SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/log"

	api "github.com/SlinkyProject/slurm-client/api/v0041"
	"github.com/SlinkyProject/slurm-client/pkg/client"
	"github.com/SlinkyProject/slurm-client/pkg/types"
)

// Ref: https://prometheus.io/docs/practices/naming/#metric-names
func NewJobStateCollector(slurmClient client.Client) prometheus.Collector {
	return &jobStateCollector{
		slurmClient: slurmClient,

		// Other
		Total: prometheus.NewDesc("slurm_jobs_total", "Total number of jobs", nil, nil),
		// Base States
		Pending: prometheus.NewDesc("slurm_jobs_pending_total", "Number of jobs in Pending state", nil, nil),
		Running: prometheus.NewDesc("slurm_jobs_running_total", "Number of jobs in Running state", nil, nil),
		// Other States
		Hold: prometheus.NewDesc("slurm_jobs_hold_total", "Number of jobs with Hold flag", nil, nil),
	}
}

type jobStateCollector struct {
	slurmClient client.Client

	// Other
	Total *prometheus.Desc
	// Base States
	Pending *prometheus.Desc
	Running *prometheus.Desc
	// Other States
	Hold *prometheus.Desc
}

func (c *jobStateCollector) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(c, ch)
}

func (c *jobStateCollector) Collect(ch chan<- prometheus.Metric) {
	ctx := context.TODO()
	logger := log.FromContext(ctx).WithName("JobStateCollector")

	logger.V(1).Info("collecting data")

	metrics, err := c.getJobStates(ctx)
	if err != nil {
		logger.Error(err, "failed to collect job states")
		return
	}

	// Other
	ch <- prometheus.MustNewConstMetric(c.Total, prometheus.GaugeValue, float64(metrics.Total))
	// Base States
	ch <- prometheus.MustNewConstMetric(c.Pending, prometheus.GaugeValue, float64(metrics.Pending))
	ch <- prometheus.MustNewConstMetric(c.Running, prometheus.GaugeValue, float64(metrics.Running))
	// Other States
	ch <- prometheus.MustNewConstMetric(c.Hold, prometheus.GaugeValue, float64(metrics.Hold))
}

func (c *jobStateCollector) getJobStates(ctx context.Context) (*JobStates, error) {
	jobList := &types.V0041JobInfoList{}
	if err := c.slurmClient.List(ctx, jobList); err != nil {
		return nil, err
	}
	metrics := parseJobStates(jobList)
	return metrics, nil
}

func parseJobStates(jobList *types.V0041JobInfoList) *JobStates {
	metrics := &JobStates{}
	for _, job := range jobList.Items {
		parseJobState(metrics, job)
	}
	return metrics
}

func parseJobState(metrics *JobStates, job types.V0041JobInfo) {
	metrics.Total++
	states := job.GetStateAsSet()
	// Base States
	switch {
	case states.Has(api.V0041JobInfoJobStatePENDING):
		metrics.Pending++
	case states.Has(api.V0041JobInfoJobStateRUNNING):
		metrics.Running++
	}
	// Other States
	if isHold := ptr.Deref(job.Hold, false); isHold {
		metrics.Hold++
	}
}

// Ref: https://slurm.schedmd.com/job_state_codes.html#states
// Ref: https://slurm.schedmd.com/job_state_codes.html#flags
type JobStates struct {
	// Other
	Total uint
	// Base States
	Pending uint
	Running uint
	// Other States
	Hold uint
}
