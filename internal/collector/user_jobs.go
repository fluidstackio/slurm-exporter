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

		// Other
		Total: prometheus.NewDesc("slurm_user_jobs_total", "Total number of user jobs", userLabels, nil),
		// Base States
		Pending: prometheus.NewDesc("slurm_user_jobs_pending_total", "Number of user jobs in Pending state", userLabels, nil),
		Running: prometheus.NewDesc("slurm_user_jobs_running_total", "Number of user jobs in Running state", userLabels, nil),
		// Other States
		Hold: prometheus.NewDesc("slurm_user_jobs_hold_total", "Number of user jobs with Hold flag", userLabels, nil),
	}
}

type userJobsCollector struct {
	slurmClient client.Client

	// Other
	Total *prometheus.Desc
	// Base States
	Pending *prometheus.Desc
	Running *prometheus.Desc
	// Other States
	Hold *prometheus.Desc
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
		// Other
		ch <- prometheus.MustNewConstMetric(c.Total, prometheus.GaugeValue, float64(usage.Total), user)
		// Base States
		ch <- prometheus.MustNewConstMetric(c.Pending, prometheus.GaugeValue, float64(usage.Pending), user)
		ch <- prometheus.MustNewConstMetric(c.Running, prometheus.GaugeValue, float64(usage.Running), user)
		// Other States
		ch <- prometheus.MustNewConstMetric(c.Hold, prometheus.GaugeValue, float64(usage.Hold), user)
	}
}

func (c *userJobsCollector) getUserJobs(ctx context.Context) (map[string]*JobStates, error) {
	jobList := &types.V0041JobInfoList{}
	if err := c.slurmClient.List(ctx, jobList); err != nil {
		return nil, err
	}
	metrics := parseUserJobs(jobList)
	return metrics, nil
}

func parseUserJobs(jobList *types.V0041JobInfoList) map[string]*JobStates {
	metrics := make(map[string]*JobStates)
	for _, job := range jobList.Items {
		var key string = strconv.Itoa(int(*job.UserId))
		if ptr.Deref(job.UserName, "") != "" {
			key = *job.UserName
		}
		if metrics[key] == nil {
			metrics[key] = &JobStates{}
		}
		parseJobState(metrics[key], job)
	}
	return metrics
}
