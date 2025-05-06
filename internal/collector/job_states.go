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
		BootFail:    prometheus.NewDesc("slurm_jobs_bootfail_total", "Number of jobs in BootFail state", nil, nil),
		Cancelled:   prometheus.NewDesc("slurm_jobs_cancelled_total", "Number of jobs in Cancelled state", nil, nil),
		Completed:   prometheus.NewDesc("slurm_jobs_completed_total", "Number of jobs in Completed state", nil, nil),
		Deadline:    prometheus.NewDesc("slurm_jobs_deadline_total", "Number of jobs in Deadline state", nil, nil),
		Failed:      prometheus.NewDesc("slurm_jobs_failed_total", "Number of jobs in Failed state", nil, nil),
		Pending:     prometheus.NewDesc("slurm_jobs_pending_total", "Number of jobs in Pending state", nil, nil),
		Preempted:   prometheus.NewDesc("slurm_jobs_preempted_total", "Number of jobs in Preempted state", nil, nil),
		Running:     prometheus.NewDesc("slurm_jobs_running_total", "Number of jobs in Running state", nil, nil),
		Suspended:   prometheus.NewDesc("slurm_jobs_suspended_total", "Number of jobs in Suspended state", nil, nil),
		Timeout:     prometheus.NewDesc("slurm_jobs_timeout_total", "Number of jobs in Timeout state", nil, nil),
		NodeFail:    prometheus.NewDesc("slurm_jobs_nodefail_total", "Number of jobs in NodeFail state", nil, nil),
		OutOfMemory: prometheus.NewDesc("slurm_jobs_outofmemory_total", "Number of jobs in OutOfMemory state", nil, nil),
		// Flag States
		Completing:  prometheus.NewDesc("slurm_jobs_completing_total", "Number of jobs with Completing flag", nil, nil),
		Configuring: prometheus.NewDesc("slurm_jobs_configuring_total", "Number of jobs with Configuring flag", nil, nil),
		PowerUpNode: prometheus.NewDesc("slurm_jobs_powerupnode_total", "Number of jobs with PowerUpNode flag", nil, nil),
		StageOut:    prometheus.NewDesc("slurm_jobs_stageout_total", "Number of jobs with StageOut flag", nil, nil),
		// Other States
		Hold: prometheus.NewDesc("slurm_jobs_hold_total", "Number of jobs with Hold flag", nil, nil),
	}
}

type jobStateCollector struct {
	slurmClient client.Client

	// Other
	Total *prometheus.Desc
	// Base States
	BootFail    *prometheus.Desc
	Cancelled   *prometheus.Desc
	Completed   *prometheus.Desc
	Deadline    *prometheus.Desc
	Failed      *prometheus.Desc
	Pending     *prometheus.Desc
	Running     *prometheus.Desc
	Preempted   *prometheus.Desc
	Suspended   *prometheus.Desc
	Timeout     *prometheus.Desc
	NodeFail    *prometheus.Desc
	OutOfMemory *prometheus.Desc
	// Flag States
	Completing  *prometheus.Desc
	Configuring *prometheus.Desc
	PowerUpNode *prometheus.Desc
	StageOut    *prometheus.Desc
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
	ch <- prometheus.MustNewConstMetric(c.BootFail, prometheus.GaugeValue, float64(metrics.BootFail))
	ch <- prometheus.MustNewConstMetric(c.Cancelled, prometheus.GaugeValue, float64(metrics.Cancelled))
	ch <- prometheus.MustNewConstMetric(c.Completed, prometheus.GaugeValue, float64(metrics.Completed))
	ch <- prometheus.MustNewConstMetric(c.Deadline, prometheus.GaugeValue, float64(metrics.Deadline))
	ch <- prometheus.MustNewConstMetric(c.Failed, prometheus.GaugeValue, float64(metrics.Failed))
	ch <- prometheus.MustNewConstMetric(c.Pending, prometheus.GaugeValue, float64(metrics.Pending))
	ch <- prometheus.MustNewConstMetric(c.Preempted, prometheus.GaugeValue, float64(metrics.Preempted))
	ch <- prometheus.MustNewConstMetric(c.Running, prometheus.GaugeValue, float64(metrics.Running))
	ch <- prometheus.MustNewConstMetric(c.Suspended, prometheus.GaugeValue, float64(metrics.Suspended))
	ch <- prometheus.MustNewConstMetric(c.Timeout, prometheus.GaugeValue, float64(metrics.Timeout))
	ch <- prometheus.MustNewConstMetric(c.NodeFail, prometheus.GaugeValue, float64(metrics.NodeFail))
	ch <- prometheus.MustNewConstMetric(c.OutOfMemory, prometheus.GaugeValue, float64(metrics.OutOfMemory))
	// Flag States
	ch <- prometheus.MustNewConstMetric(c.Completing, prometheus.GaugeValue, float64(metrics.Completing))
	ch <- prometheus.MustNewConstMetric(c.Configuring, prometheus.GaugeValue, float64(metrics.Configuring))
	ch <- prometheus.MustNewConstMetric(c.PowerUpNode, prometheus.GaugeValue, float64(metrics.PowerUpNode))
	ch <- prometheus.MustNewConstMetric(c.StageOut, prometheus.GaugeValue, float64(metrics.StageOut))
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
	case states.Has(api.V0041JobInfoJobStateBOOTFAIL):
		metrics.BootFail++
	case states.Has(api.V0041JobInfoJobStateCANCELLED):
		metrics.Cancelled++
	case states.Has(api.V0041JobInfoJobStateCOMPLETED):
		metrics.Completed++
	case states.Has(api.V0041JobInfoJobStateDEADLINE):
		metrics.Deadline++
	case states.Has(api.V0041JobInfoJobStateFAILED):
		metrics.Failed++
	case states.Has(api.V0041JobInfoJobStatePENDING):
		metrics.Pending++
	case states.Has(api.V0041JobInfoJobStatePREEMPTED):
		metrics.Preempted++
	case states.Has(api.V0041JobInfoJobStateRUNNING):
		metrics.Running++
	case states.Has(api.V0041JobInfoJobStateSUSPENDED):
		metrics.Suspended++
	case states.Has(api.V0041JobInfoJobStateTIMEOUT):
		metrics.Timeout++
	case states.Has(api.V0041JobInfoJobStateNODEFAIL):
		metrics.NodeFail++
	case states.Has(api.V0041JobInfoJobStateOUTOFMEMORY):
		metrics.OutOfMemory++
	}
	// Flag States
	if states.Has(api.V0041JobInfoJobStateCOMPLETING) {
		metrics.Completing++
	}
	if states.Has(api.V0041JobInfoJobStateCONFIGURING) {
		metrics.Configuring++
	}
	if states.Has(api.V0041JobInfoJobStatePOWERUPNODE) {
		metrics.PowerUpNode++
	}
	if states.Has(api.V0041JobInfoJobStateSTAGEOUT) {
		metrics.StageOut++
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
	BootFail    uint
	Cancelled   uint
	Completed   uint
	Deadline    uint
	Failed      uint
	Pending     uint
	Preempted   uint
	Running     uint
	Suspended   uint
	Timeout     uint
	NodeFail    uint
	OutOfMemory uint
	// Flag States
	Completing  uint
	Configuring uint
	PowerUpNode uint
	StageOut    uint
	// Other States
	Hold uint
}
