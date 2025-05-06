// SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"context"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/SlinkyProject/slurm-client/pkg/client"
	"github.com/SlinkyProject/slurm-client/pkg/types"
)

// Ref: https://prometheus.io/docs/practices/naming/#metric-names
func NewUserJobsCollector(slurmClient client.Client) prometheus.Collector {
	return &userJobsCollector{
		slurmClient: slurmClient,

		// States
		Total:       prometheus.NewDesc("slurm_user_jobs_total", "Total number of user jobs", userLabels, nil),
		BootFail:    prometheus.NewDesc("slurm_user_jobs_bootfail_total", "Number of user jobs in BootFail state", userLabels, nil),
		Cancelled:   prometheus.NewDesc("slurm_user_jobs_cancelled_total", "Number of user jobs in Cancelled state", userLabels, nil),
		Completed:   prometheus.NewDesc("slurm_user_jobs_completed_total", "Number of user jobs in Completed state", userLabels, nil),
		Deadline:    prometheus.NewDesc("slurm_user_jobs_deadline_total", "Number of user jobs in Deadline state", userLabels, nil),
		Failed:      prometheus.NewDesc("slurm_user_jobs_failed_total", "Number of user jobs in Failed state", userLabels, nil),
		Pending:     prometheus.NewDesc("slurm_user_jobs_pending_total", "Number of user jobs in Pending state", userLabels, nil),
		Running:     prometheus.NewDesc("slurm_user_jobs_running_total", "Number of user jobs in Running state", userLabels, nil),
		Suspended:   prometheus.NewDesc("slurm_user_jobs_suspended_total", "Number of user jobs in Suspended state", userLabels, nil),
		Timeout:     prometheus.NewDesc("slurm_user_jobs_timeout_total", "Number of user jobs in Timeout state", userLabels, nil),
		NodeFail:    prometheus.NewDesc("slurm_user_jobs_nodefail_total", "Number of user jobs in NodeFail state", userLabels, nil),
		OutOfMemory: prometheus.NewDesc("slurm_user_jobs_outofmemory_total", "Number of user jobs in OutOfMemory state", userLabels, nil),
		Completing:  prometheus.NewDesc("slurm_user_jobs_completing_total", "Number of user jobs with Completing flag", userLabels, nil),
		Configuring: prometheus.NewDesc("slurm_user_jobs_configuring_total", "Number of user jobs with Configuring flag", userLabels, nil),
		PowerUpNode: prometheus.NewDesc("slurm_user_jobs_powerupnode_total", "Number of user jobs with PowerUpNode flag", userLabels, nil),
		Hold:        prometheus.NewDesc("slurm_user_jobs_hold_total", "Number of user jobs with Hold flag", userLabels, nil),
		// Tres
		CpusAlloc:   prometheus.NewDesc("slurm_user_jobs_cpus_alloc_total", "Number of Allocated CPUs among user jobs", userLabels, nil),
		MemoryAlloc: prometheus.NewDesc("slurm_user_jobs_memory_alloc_bytes", "Amount of Allocated Memory (MB) among user jobs", userLabels, nil),
	}
}

type userJobsCollector struct {
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

func (c *userJobsCollector) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(c, ch)
}

func (c *userJobsCollector) Collect(ch chan<- prometheus.Metric) {
	ctx := context.TODO()
	logger := log.FromContext(ctx).WithName("UserJobsCollector")

	logger.V(1).Info("collecting data")

	metrics, err := c.getUserJobs(ctx)
	if err != nil {
		logger.Error(err, "failed to collect user jobs")
		return
	}

	for user, usage := range metrics {
		// States
		ch <- prometheus.MustNewConstMetric(c.Total, prometheus.GaugeValue, float64(usage.Total), user)
		ch <- prometheus.MustNewConstMetric(c.BootFail, prometheus.GaugeValue, float64(usage.BootFail), user)
		ch <- prometheus.MustNewConstMetric(c.Cancelled, prometheus.GaugeValue, float64(usage.Cancelled), user)
		ch <- prometheus.MustNewConstMetric(c.Completed, prometheus.GaugeValue, float64(usage.Completed), user)
		ch <- prometheus.MustNewConstMetric(c.Deadline, prometheus.GaugeValue, float64(usage.Deadline), user)
		ch <- prometheus.MustNewConstMetric(c.Failed, prometheus.GaugeValue, float64(usage.Failed), user)
		ch <- prometheus.MustNewConstMetric(c.Pending, prometheus.GaugeValue, float64(usage.Pending), user)
		ch <- prometheus.MustNewConstMetric(c.Running, prometheus.GaugeValue, float64(usage.Running), user)
		ch <- prometheus.MustNewConstMetric(c.Suspended, prometheus.GaugeValue, float64(usage.Suspended), user)
		ch <- prometheus.MustNewConstMetric(c.Timeout, prometheus.GaugeValue, float64(usage.Timeout), user)
		ch <- prometheus.MustNewConstMetric(c.NodeFail, prometheus.GaugeValue, float64(usage.NodeFail), user)
		ch <- prometheus.MustNewConstMetric(c.OutOfMemory, prometheus.GaugeValue, float64(usage.OutOfMemory), user)
		ch <- prometheus.MustNewConstMetric(c.Completing, prometheus.GaugeValue, float64(usage.Completing), user)
		ch <- prometheus.MustNewConstMetric(c.Configuring, prometheus.GaugeValue, float64(usage.Configuring), user)
		ch <- prometheus.MustNewConstMetric(c.PowerUpNode, prometheus.GaugeValue, float64(usage.PowerUpNode), user)
		ch <- prometheus.MustNewConstMetric(c.Hold, prometheus.GaugeValue, float64(usage.Hold), user)
		// Tres
		ch <- prometheus.MustNewConstMetric(c.CpusAlloc, prometheus.GaugeValue, float64(usage.CpusAlloc), user)
		ch <- prometheus.MustNewConstMetric(c.MemoryAlloc, prometheus.GaugeValue, float64(usage.MemoryAlloc), user)
	}
}

func (c *userJobsCollector) getUserJobs(ctx context.Context) (map[string]*UserJobs, error) {
	jobList := &types.V0041JobInfoList{}
	if err := c.slurmClient.List(ctx, jobList); err != nil {
		return nil, err
	}
	metrics := parseUserJobs(jobList)
	return metrics, nil
}

func parseUserJobs(jobList *types.V0041JobInfoList) map[string]*UserJobs {
	metrics := make(map[string]*UserJobs)
	for _, job := range jobList.Items {
		var key string = strconv.Itoa(int(*job.UserId))
		if ptr.Deref(job.UserName, "") != "" {
			key = *job.UserName
		}
		if metrics[key] == nil {
			metrics[key] = &UserJobs{}
		}
		parseJobState(&metrics[key].JobStates, job)
		res := getJobResourceAlloc(job)
		metrics[key].CpusAlloc += res.Cpus
		metrics[key].MemoryAlloc += res.Memory
	}
	return metrics
}

type UserJobs struct {
	// States
	JobStates
	// Tres
	CpusAlloc   uint
	MemoryAlloc uint
}
