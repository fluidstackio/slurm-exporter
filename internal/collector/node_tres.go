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
		CpusTotal: prometheus.NewDesc("slurm_node_cpus", "Number of CPUs in a slurm node", nodeLabels, nil),
		CpusAlloc: prometheus.NewDesc("slurm_node_alloc_cpus", "Number of allocated CPUs in a slurm node", nodeLabels, nil),
		CpusIdle:  prometheus.NewDesc("slurm_node_idle_cpus", "Number of idle CPUs in a slurm node", nodeLabels, nil),
	}
}

type nodeTresCollector struct {
	slurmClient client.Client

	// CPUs
	CpusTotal *prometheus.Desc
	CpusAlloc *prometheus.Desc
	CpusIdle  *prometheus.Desc
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
			CpusTotal:  uint(ptr.Deref(node.Cpus, 0)),
			CpusAlloc:  uint(ptr.Deref(node.AllocCpus, 0)),
			CpusIdle:   uint(ptr.Deref(node.AllocIdleCpus, 0)),
			partitions: ptr.Deref(node.Partitions, []string{}),
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
	// Metadata
	partitions []string
}
