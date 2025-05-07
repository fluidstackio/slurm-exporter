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
func NewPartitionNodeCollector(slurmClient client.Client) prometheus.Collector {
	return &partitionNodeCollector{
		slurmClient: slurmClient,

		Total:     prometheus.NewDesc("slurm_partition_nodes_total", "Total number of nodes in the partition", partitionLabels, nil),
		CpusTotal: prometheus.NewDesc("slurm_partition_nodes_cpus_total", "Total number of CPUs among nodes in the partition", partitionLabels, nil),
		CpusAlloc: prometheus.NewDesc("slurm_partition_nodes_cpus_alloc_total", "Number of Allocated CPUs among nodes in the partition", partitionLabels, nil),
		CpusIdle:  prometheus.NewDesc("slurm_partition_nodes_cpus_idle_total", "Number of Idle CPUs among nodes in the partition", partitionLabels, nil),
	}
}

type partitionNodeCollector struct {
	slurmClient client.Client

	Total     *prometheus.Desc
	CpusTotal *prometheus.Desc
	CpusAlloc *prometheus.Desc
	CpusIdle  *prometheus.Desc
}

func (c *partitionNodeCollector) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(c, ch)
}

func (c *partitionNodeCollector) Collect(ch chan<- prometheus.Metric) {
	ctx := context.TODO()
	logger := log.FromContext(ctx).WithName("PartitionNodeCollector")

	logger.V(1).Info("collecting data")

	metrics, err := c.getPartitionNode(ctx)
	if err != nil {
		logger.Error(err, "failed to collect partition TRES")
		return
	}

	for partition, data := range metrics {
		ch <- prometheus.MustNewConstMetric(c.Total, prometheus.GaugeValue, float64(data.Total), partition)
		ch <- prometheus.MustNewConstMetric(c.CpusTotal, prometheus.GaugeValue, float64(data.CpusTotal), partition)
		ch <- prometheus.MustNewConstMetric(c.CpusAlloc, prometheus.GaugeValue, float64(data.CpusAlloc), partition)
		ch <- prometheus.MustNewConstMetric(c.CpusIdle, prometheus.GaugeValue, float64(data.CpusIdle), partition)
	}
}

func (c *partitionNodeCollector) getPartitionNode(ctx context.Context) (map[string]*PartitionNodes, error) {
	partitionList := &types.V0041PartitionInfoList{}
	if err := c.slurmClient.List(ctx, partitionList); err != nil {
		return nil, err
	}
	nodeList := &types.V0041NodeList{}
	if err := c.slurmClient.List(ctx, nodeList); err != nil {
		return nil, err
	}
	metrics := parsePartitionNode(partitionList, nodeList)
	return metrics, nil
}

func parsePartitionNode(
	partitionList *types.V0041PartitionInfoList,
	nodeList *types.V0041NodeList,
) map[string]*PartitionNodes {
	metrics := make(map[string]*PartitionNodes, len(partitionList.Items))

	for _, partition := range partitionList.Items {
		key := string(partition.GetKey())
		data := &PartitionNodes{
			Total: func() uint {
				if partition.Nodes == nil {
					return 0
				}
				return uint(ptr.Deref(partition.Nodes.Total, 0))
			}(),
			CpusTotal: func() uint {
				if partition.Cpus == nil {
					return 0
				}
				return uint(ptr.Deref(partition.Cpus.Total, 0))
			}(),
		}
		metrics[key] = data
	}

	nodeMetrics := parseNodeTres(nodeList)
	for _, nodeTres := range nodeMetrics {
		for _, key := range nodeTres.partitions {
			metrics[key].CpusAlloc += nodeTres.CpusAlloc
			metrics[key].CpusIdle += nodeTres.CpusIdle
		}
	}

	return metrics
}

type PartitionNodes struct {
	Total     uint
	CpusTotal uint
	CpusAlloc uint
	CpusIdle  uint
}
