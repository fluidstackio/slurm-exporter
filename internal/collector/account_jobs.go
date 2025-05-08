// SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/SlinkyProject/slurm-client/pkg/client"
	"github.com/SlinkyProject/slurm-client/pkg/types"
)

// Ref: https://prometheus.io/docs/practices/naming/#metric-names
func NewAccountJobsCollector(slurmClient client.Client) prometheus.Collector {
	return &accountJobsCollector{
		slurmClient: slurmClient,

		// States
		Total:       prometheus.NewDesc("slurm_account_jobs_total", "Total number of account jobs", accountLabels, nil),
		BootFail:    prometheus.NewDesc("slurm_account_jobs_bootfail_total", "Number of account jobs in BootFail state", accountLabels, nil),
		Cancelled:   prometheus.NewDesc("slurm_account_jobs_cancelled_total", "Number of account jobs in Cancelled state", accountLabels, nil),
		Completed:   prometheus.NewDesc("slurm_account_jobs_completed_total", "Number of account jobs in Completed state", accountLabels, nil),
		Deadline:    prometheus.NewDesc("slurm_account_jobs_deadline_total", "Number of account jobs in Deadline state", accountLabels, nil),
		Failed:      prometheus.NewDesc("slurm_account_jobs_failed_total", "Number of account jobs in Failed state", accountLabels, nil),
		Pending:     prometheus.NewDesc("slurm_account_jobs_pending_total", "Number of account jobs in Pending state", accountLabels, nil),
		Running:     prometheus.NewDesc("slurm_account_jobs_running_total", "Number of account jobs in Running state", accountLabels, nil),
		Suspended:   prometheus.NewDesc("slurm_account_jobs_suspended_total", "Number of account jobs in Suspended state", accountLabels, nil),
		Timeout:     prometheus.NewDesc("slurm_account_jobs_timeout_total", "Number of account jobs in Timeout state", accountLabels, nil),
		NodeFail:    prometheus.NewDesc("slurm_account_jobs_nodefail_total", "Number of account jobs in NodeFail state", accountLabels, nil),
		OutOfMemory: prometheus.NewDesc("slurm_account_jobs_outofmemory_total", "Number of account jobs in OutOfMemory state", accountLabels, nil),
		Completing:  prometheus.NewDesc("slurm_account_jobs_completing_total", "Number of account jobs with Completing flag", accountLabels, nil),
		Configuring: prometheus.NewDesc("slurm_account_jobs_configuring_total", "Number of account jobs with Configuring flag", accountLabels, nil),
		PowerUpNode: prometheus.NewDesc("slurm_account_jobs_powerupnode_total", "Number of account jobs with PowerUpNode flag", accountLabels, nil),
		Hold:        prometheus.NewDesc("slurm_account_jobs_hold_total", "Number of account jobs with Hold flag", accountLabels, nil),
		// Tres
		CpusAlloc:   prometheus.NewDesc("slurm_account_jobs_cpus_alloc_total", "Number of Allocated CPUs among account jobs", accountLabels, nil),
		MemoryAlloc: prometheus.NewDesc("slurm_account_jobs_memory_alloc_bytes", "Amount of Allocated Memory (MB) among account jobs", accountLabels, nil),
	}
}

type accountJobsCollector struct {
	slurmClient client.Client

	// States
	Total       *prometheus.Desc
	BootFail    *prometheus.Desc
	Cancelled   *prometheus.Desc
	Completed   *prometheus.Desc
	Deadline    *prometheus.Desc
	Failed      *prometheus.Desc
	Pending     *prometheus.Desc
	Running     *prometheus.Desc
	Suspended   *prometheus.Desc
	Timeout     *prometheus.Desc
	NodeFail    *prometheus.Desc
	OutOfMemory *prometheus.Desc
	Completing  *prometheus.Desc
	Configuring *prometheus.Desc
	PowerUpNode *prometheus.Desc
	Hold        *prometheus.Desc
	// Tres
	CpusAlloc   *prometheus.Desc
	MemoryAlloc *prometheus.Desc
}

func (c *accountJobsCollector) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(c, ch)
}

func (c *accountJobsCollector) Collect(ch chan<- prometheus.Metric) {
	ctx := context.TODO()
	logger := log.FromContext(ctx).WithName("AccountJobsCollector")

	logger.V(1).Info("collecting data")

	metrics, err := c.getAccountJobs(ctx)
	if err != nil {
		logger.Error(err, "failed to collect account jobs")
		return
	}

	for data, usage := range metrics {
		// States
		ch <- prometheus.MustNewConstMetric(c.Total, prometheus.GaugeValue, float64(usage.Total), data)
		ch <- prometheus.MustNewConstMetric(c.BootFail, prometheus.GaugeValue, float64(usage.BootFail), data)
		ch <- prometheus.MustNewConstMetric(c.Cancelled, prometheus.GaugeValue, float64(usage.Cancelled), data)
		ch <- prometheus.MustNewConstMetric(c.Completed, prometheus.GaugeValue, float64(usage.Completed), data)
		ch <- prometheus.MustNewConstMetric(c.Deadline, prometheus.GaugeValue, float64(usage.Deadline), data)
		ch <- prometheus.MustNewConstMetric(c.Failed, prometheus.GaugeValue, float64(usage.Failed), data)
		ch <- prometheus.MustNewConstMetric(c.Pending, prometheus.GaugeValue, float64(usage.Pending), data)
		ch <- prometheus.MustNewConstMetric(c.Running, prometheus.GaugeValue, float64(usage.Running), data)
		ch <- prometheus.MustNewConstMetric(c.Suspended, prometheus.GaugeValue, float64(usage.Suspended), data)
		ch <- prometheus.MustNewConstMetric(c.Timeout, prometheus.GaugeValue, float64(usage.Timeout), data)
		ch <- prometheus.MustNewConstMetric(c.NodeFail, prometheus.GaugeValue, float64(usage.NodeFail), data)
		ch <- prometheus.MustNewConstMetric(c.OutOfMemory, prometheus.GaugeValue, float64(usage.OutOfMemory), data)
		ch <- prometheus.MustNewConstMetric(c.Completing, prometheus.GaugeValue, float64(usage.Completing), data)
		ch <- prometheus.MustNewConstMetric(c.Configuring, prometheus.GaugeValue, float64(usage.Configuring), data)
		ch <- prometheus.MustNewConstMetric(c.PowerUpNode, prometheus.GaugeValue, float64(usage.PowerUpNode), data)
		ch <- prometheus.MustNewConstMetric(c.Hold, prometheus.GaugeValue, float64(usage.Hold), data)
		// Tres
		ch <- prometheus.MustNewConstMetric(c.CpusAlloc, prometheus.GaugeValue, float64(usage.CpusAlloc), data)
		ch <- prometheus.MustNewConstMetric(c.MemoryAlloc, prometheus.GaugeValue, float64(usage.MemoryAlloc), data)
	}
}

func (c *accountJobsCollector) getAccountJobs(ctx context.Context) (map[string]*AccountJobs, error) {
	jobList := &types.V0041JobInfoList{}
	if err := c.slurmClient.List(ctx, jobList); err != nil {
		return nil, err
	}
	metrics := parseAccountJobs(jobList)
	return metrics, nil
}

func parseAccountJobs(jobList *types.V0041JobInfoList) map[string]*AccountJobs {
	metrics := make(map[string]*AccountJobs)
	for _, job := range jobList.Items {
		key := ptr.Deref(job.Account, "")
		if metrics[key] == nil {
			metrics[key] = &AccountJobs{}
		}
		parseJobState(&metrics[key].JobStates, job)
		res := getJobResourceAlloc(job)
		metrics[key].CpusAlloc += res.Cpus
		metrics[key].MemoryAlloc += res.Memory
	}
	return metrics
}

type AccountJobs struct {
	// States
	JobStates
	// Tres
	CpusAlloc   uint
	MemoryAlloc uint
}
