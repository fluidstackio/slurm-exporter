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
func NewNodeTresCollector(slurmClient client.Client) prometheus.Collector {
	return &nodeTresCollector{
		slurmClient: slurmClient,

		// CPUs
		CpusTotal: prometheus.NewDesc("slurm_node_cpus_total", "Total number of CPUs on the node", nodeLabels, nil),
		CpusAlloc: prometheus.NewDesc("slurm_node_cpus_alloc_total", "Number of Allocated CPUs on the node", nodeLabels, nil),
		CpusIdle:  prometheus.NewDesc("slurm_node_cpus_idle_total", "Number of Idle CPUs on the node", nodeLabels, nil),
		// Memory
		MemoryTotal: prometheus.NewDesc("slurm_node_memory_bytes", "Totcal amount of Allocated Memory (MB) on the node", nodeLabels, nil),
		MemoryAlloc: prometheus.NewDesc("slurm_node_memory_alloc_bytes", "Amount of Allocated Memory (MB) on the node", nodeLabels, nil),
		MemoryFree:  prometheus.NewDesc("slurm_node_memory_free_bytes", "Amount of Free Memory (MB) on the node", nodeLabels, nil),
	}
}

type nodeTresCollector struct {
	slurmClient client.Client

	// CPUs
	CpusTotal *prometheus.Desc
	CpusAlloc *prometheus.Desc
	CpusIdle  *prometheus.Desc
	// Memory
	MemoryTotal *prometheus.Desc
	MemoryAlloc *prometheus.Desc
	MemoryFree  *prometheus.Desc
}

func (c *nodeTresCollector) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(c, ch)
}

func (c *nodeTresCollector) Collect(ch chan<- prometheus.Metric) {
	ctx := context.TODO()
	logger := log.FromContext(ctx).WithName("NodeTresCollector")

	logger.V(1).Info("collecting data")

	metrics, err := c.getNodeTres(ctx)
	if err != nil {
		logger.Error(err, "failed to collect node TRES")
		return
	}

	for node, data := range metrics {
		// CPUs
		ch <- prometheus.MustNewConstMetric(c.CpusTotal, prometheus.GaugeValue, float64(data.CpusTotal), node)
		ch <- prometheus.MustNewConstMetric(c.CpusAlloc, prometheus.GaugeValue, float64(data.CpusAlloc), node)
		ch <- prometheus.MustNewConstMetric(c.CpusIdle, prometheus.GaugeValue, float64(data.CpusIdle), node)
		// Memory
		ch <- prometheus.MustNewConstMetric(c.MemoryTotal, prometheus.GaugeValue, float64(data.MemoryTotal), node)
		ch <- prometheus.MustNewConstMetric(c.MemoryAlloc, prometheus.GaugeValue, float64(data.MemoryAlloc), node)
		ch <- prometheus.MustNewConstMetric(c.MemoryFree, prometheus.GaugeValue, float64(data.MemoryFree), node)
	}
}

func (c *nodeTresCollector) getNodeTres(ctx context.Context) (map[string]*NodeTres, error) {
	nodeList := &types.V0041NodeList{}
	if err := c.slurmClient.List(ctx, nodeList); err != nil {
		return nil, err
	}
	metrics := parseNodeTres(nodeList)
	return metrics, nil
}

func parseNodeTres(nodeList *types.V0041NodeList) map[string]*NodeTres {
	metrics := make(map[string]*NodeTres, len(nodeList.Items))
	for _, node := range nodeList.Items {
		key := string(node.GetKey())
		data := &NodeTres{
			CpusTotal:   uint(ptr.Deref(node.Cpus, 0)),
			CpusAlloc:   uint(ptr.Deref(node.AllocCpus, 0)),
			CpusIdle:    uint(ptr.Deref(node.AllocIdleCpus, 0)),
			MemoryTotal: uint(ptr.Deref(node.RealMemory, 0)),
			MemoryAlloc: uint(ptr.Deref(node.AllocMemory, 0)),
			MemoryFree:  uint(ParseUint64NoVal(node.FreeMem)),
			partitions:  ptr.Deref(node.Partitions, []string{}),
		}
		metrics[key] = data
	}
	return metrics
}

type NodeTres struct {
	// CPUs
	CpusTotal uint
	CpusAlloc uint
	CpusIdle  uint
	// Memory
	MemoryTotal uint
	MemoryAlloc uint
	MemoryFree  uint
	// Metadata
	partitions []string
}
