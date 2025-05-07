// SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"context"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/SlinkyProject/slurm-client/pkg/client"
	"github.com/SlinkyProject/slurm-client/pkg/types"
	"github.com/SlinkyProject/slurm-exporter/internal/utils"
)

// Ref: https://prometheus.io/docs/practices/naming/#metric-names
func NewPartitionJobCollector(slurmClient client.Client) prometheus.Collector {
	return &partitionJobCollector{
		slurmClient: slurmClient,

		// States
		Total:   prometheus.NewDesc("slurm_partition_jobs_total", "Total number of jobs in the partition", partitionLabels, nil),
		Pending: prometheus.NewDesc("slurm_partition_jobs_pending_total", "Number of jobs in Pending state in the partition", partitionLabels, nil),
		Running: prometheus.NewDesc("slurm_partition_jobs_running_total", "Number of jobs in Running state in the partition", partitionLabels, nil),
		Hold:    prometheus.NewDesc("slurm_partition_jobs_hold_total", "Number of jobs with Hold flag in the partition", partitionLabels, nil),
		// Tres
		CpusAlloc: prometheus.NewDesc("slurm_partition_jobs_cpus_alloc_total", "Number of Allocated CPUs among jobs in the partition", partitionLabels, nil),
		// Other
		PendingNodeCount: prometheus.NewDesc("slurm_partition_jobs_pending_maxnodecount_total", "Largest number of nodes required among pending jobs in the partition", partitionLabels, nil),
	}
}

type partitionJobCollector struct {
	slurmClient client.Client

	// States
	Total   *prometheus.Desc
	Pending *prometheus.Desc
	Running *prometheus.Desc
	Hold    *prometheus.Desc
	// Tres
	CpusAlloc *prometheus.Desc
	// Other
	PendingNodeCount *prometheus.Desc
}

func (c *partitionJobCollector) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(c, ch)
}

func (c *partitionJobCollector) Collect(ch chan<- prometheus.Metric) {
	ctx := context.TODO()
	logger := log.FromContext(ctx).WithName("PartitionJobCollector")

	logger.V(1).Info("collecting data")

	metrics, err := c.getPartitionJobs(ctx)
	if err != nil {
		logger.Error(err, "failed to collect partition jobs")
		return
	}

	for partition, data := range metrics {
		// States
		ch <- prometheus.MustNewConstMetric(c.Total, prometheus.GaugeValue, float64(data.Total), partition)
		ch <- prometheus.MustNewConstMetric(c.Pending, prometheus.GaugeValue, float64(data.Pending), partition)
		ch <- prometheus.MustNewConstMetric(c.Running, prometheus.GaugeValue, float64(data.Running), partition)
		ch <- prometheus.MustNewConstMetric(c.Hold, prometheus.GaugeValue, float64(data.Hold), partition)
		// Tres
		ch <- prometheus.MustNewConstMetric(c.CpusAlloc, prometheus.GaugeValue, float64(data.CpusAlloc), partition)
		// Other
		ch <- prometheus.MustNewConstMetric(c.PendingNodeCount, prometheus.GaugeValue, float64(data.PendingNodeCount), partition)
	}
}

func (c *partitionJobCollector) getPartitionJobs(ctx context.Context) (map[string]*PartitionJobs, error) {
	partitionList := &types.V0041PartitionInfoList{}
	if err := c.slurmClient.List(ctx, partitionList); err != nil {
		return nil, err
	}
	jobList := &types.V0041JobInfoList{}
	if err := c.slurmClient.List(ctx, jobList); err != nil {
		return nil, err
	}
	metrics := parsePartitionJobs(partitionList, jobList)
	return metrics, nil
}

func parsePartitionJobs(
	partitionList *types.V0041PartitionInfoList,
	jobList *types.V0041JobInfoList,
) map[string]*PartitionJobs {
	metrics := make(map[string]*PartitionJobs, len(partitionList.Items))

	for _, partition := range partitionList.Items {
		key := string(partition.GetKey())
		metrics[key] = &PartitionJobs{}
	}

	for _, job := range jobList.Items {
		partitionCsv := ptr.Deref(job.Partition, "")
		jobPartitions := utils.PruneEmpty(strings.Split(partitionCsv, ","))
		for _, key := range jobPartitions {
			if metrics[key] == nil {
				metrics[key] = &PartitionJobs{}
			}
			parseJobState(&metrics[key].JobStates, job)
			res := getJobResourceAlloc(job)
			metrics[key].CpusAlloc += res.Cpus
			metrics[key].PendingNodeCount = max(metrics[key].PendingNodeCount, getJobPendingNodeCount(job))
		}
	}

	return metrics
}

type PartitionJobs struct {
	// States
	JobStates
	// Tres
	CpusAlloc uint
	// Other
	PendingNodeCount uint
}
