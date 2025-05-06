// SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/log"

	api "github.com/SlinkyProject/slurm-client/api/v0041"
	"github.com/SlinkyProject/slurm-client/pkg/client"
	"github.com/SlinkyProject/slurm-client/pkg/types"
)

// Ref: https://prometheus.io/docs/practices/naming/#metric-names
func NewNodeStateCollector(slurmClient client.Client) prometheus.Collector {
	return &nodeStateCollector{
		slurmClient: slurmClient,

		// Other
		Total: prometheus.NewDesc("slurm_nodes_total", "Total number of nodes", nil, nil),
		// Base State
		Allocated: prometheus.NewDesc("slurm_nodes_allocated_total", "Number of nodes in Allocated state", nil, nil),
		Down:      prometheus.NewDesc("slurm_nodes_down_total", "Number of nodes in Down state", nil, nil),
		Error:     prometheus.NewDesc("slurm_nodes_error_total", "Number of nodes in Error state", nil, nil),
		Future:    prometheus.NewDesc("slurm_nodes_future_total", "Number of nodes in Future state", nil, nil),
		Idle:      prometheus.NewDesc("slurm_nodes_idle_total", "Number of nodes in Idle state", nil, nil),
		Mixed:     prometheus.NewDesc("slurm_nodes_mixed_total", "Number of nodes in Mixed state", nil, nil),
		Unknown:   prometheus.NewDesc("slurm_nodes_unknown_total", "Number of nodes in Unknown state", nil, nil),
		// Flag State
		Completing:      prometheus.NewDesc("slurm_nodes_completing_total", "Number of nodes with Completing flag", nil, nil),
		Drain:           prometheus.NewDesc("slurm_nodes_drain_total", "Number of nodes with Drain flag", nil, nil),
		Fail:            prometheus.NewDesc("slurm_nodes_fail_total", "Number of nodes with Fail flag", nil, nil),
		Maintenance:     prometheus.NewDesc("slurm_nodes_maintenance_total", "Number of nodes with Maintenance flag", nil, nil),
		NotResponding:   prometheus.NewDesc("slurm_nodes_notresponding_total", "Number of nodes with NotResponding flag", nil, nil),
		Planned:         prometheus.NewDesc("slurm_nodes_planned_total", "Number of nodes with Planned flag", nil, nil),
		RebootRequested: prometheus.NewDesc("slurm_nodes_rebootrequested_total", "Number of nodes with RebootRequested flag", nil, nil),
		Reserved:        prometheus.NewDesc("slurm_nodes_reserved_total", "Number of nodes with Reserved flag", nil, nil),
	}
}

// Ref: https://slurm.schedmd.com/sinfo.html#SECTION_NODE-STATE-CODES
type nodeStateCollector struct {
	slurmClient client.Client

	// Other
	Total *prometheus.Desc
	// Base State
	Allocated *prometheus.Desc
	Down      *prometheus.Desc
	Error     *prometheus.Desc
	Future    *prometheus.Desc
	Idle      *prometheus.Desc
	Mixed     *prometheus.Desc
	Unknown   *prometheus.Desc
	// Flag State
	Completing      *prometheus.Desc
	Drain           *prometheus.Desc
	Fail            *prometheus.Desc
	Maintenance     *prometheus.Desc
	NotResponding   *prometheus.Desc
	Planned         *prometheus.Desc
	RebootRequested *prometheus.Desc
	Reserved        *prometheus.Desc
}

func (c *nodeStateCollector) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(c, ch)
}

func (c *nodeStateCollector) Collect(ch chan<- prometheus.Metric) {
	ctx := context.TODO()
	logger := log.FromContext(ctx).WithName("NodeStateCollector")

	logger.V(1).Info("collecting data")

	metrics, err := c.getNodeStates(ctx)
	if err != nil {
		logger.Error(err, "failed to collect node states")
		return
	}

	// Other
	ch <- prometheus.MustNewConstMetric(c.Total, prometheus.GaugeValue, float64(metrics.Total))
	// Base State
	ch <- prometheus.MustNewConstMetric(c.Allocated, prometheus.GaugeValue, float64(metrics.Allocated))
	ch <- prometheus.MustNewConstMetric(c.Down, prometheus.GaugeValue, float64(metrics.Down))
	ch <- prometheus.MustNewConstMetric(c.Error, prometheus.GaugeValue, float64(metrics.Error))
	ch <- prometheus.MustNewConstMetric(c.Future, prometheus.GaugeValue, float64(metrics.Future))
	ch <- prometheus.MustNewConstMetric(c.Idle, prometheus.GaugeValue, float64(metrics.Idle))
	ch <- prometheus.MustNewConstMetric(c.Mixed, prometheus.GaugeValue, float64(metrics.Mixed))
	ch <- prometheus.MustNewConstMetric(c.Unknown, prometheus.GaugeValue, float64(metrics.Unknown))
	// Flag State
	ch <- prometheus.MustNewConstMetric(c.Completing, prometheus.GaugeValue, float64(metrics.Completing))
	ch <- prometheus.MustNewConstMetric(c.Drain, prometheus.GaugeValue, float64(metrics.Drain))
	ch <- prometheus.MustNewConstMetric(c.Fail, prometheus.GaugeValue, float64(metrics.Fail))
	ch <- prometheus.MustNewConstMetric(c.Maintenance, prometheus.GaugeValue, float64(metrics.Maintenance))
	ch <- prometheus.MustNewConstMetric(c.NotResponding, prometheus.GaugeValue, float64(metrics.NotResponding))
	ch <- prometheus.MustNewConstMetric(c.Planned, prometheus.GaugeValue, float64(metrics.Planned))
	ch <- prometheus.MustNewConstMetric(c.RebootRequested, prometheus.GaugeValue, float64(metrics.RebootRequested))
	ch <- prometheus.MustNewConstMetric(c.Reserved, prometheus.GaugeValue, float64(metrics.Reserved))
}

func (c *nodeStateCollector) getNodeStates(ctx context.Context) (*NodeStates, error) {
	nodeList := &types.V0041NodeList{}
	if err := c.slurmClient.List(ctx, nodeList); err != nil {
		return nil, err
	}
	metrics := parseNodeStates(nodeList)
	return metrics, nil
}

func parseNodeStates(nodeList *types.V0041NodeList) *NodeStates {
	metrics := &NodeStates{}
	for _, node := range nodeList.Items {
		parseNodeState(metrics, node)
	}
	return metrics
}

func parseNodeState(metrics *NodeStates, node types.V0041Node) {
	metrics.Total++
	states := node.GetStateAsSet()
	// Base States
	switch {
	case states.Has(api.V0041NodeStateALLOCATED):
		metrics.Allocated++
	case states.Has(api.V0041NodeStateDOWN):
		metrics.Down++
	case states.Has(api.V0041NodeStateERROR):
		metrics.Error++
	case states.Has(api.V0041NodeStateFUTURE):
		metrics.Future++
	case states.Has(api.V0041NodeStateIDLE):
		metrics.Idle++
	case states.Has(api.V0041NodeStateMIXED):
		metrics.Mixed++
	case states.Has(api.V0041NodeStateUNKNOWN):
		metrics.Unknown++
	}
	// Flag States
	if states.Has(api.V0041NodeStateCOMPLETING) {
		metrics.Completing++
	}
	if states.Has(api.V0041NodeStateDRAIN) {
		metrics.Drain++
	}
	if states.Has(api.V0041NodeStateFAIL) {
		metrics.Fail++
	}
	if states.Has(api.V0041NodeStateMAINTENANCE) {
		metrics.Maintenance++
	}
	if states.Has(api.V0041NodeStateNOTRESPONDING) {
		metrics.NotResponding++
	}
	if states.Has(api.V0041NodeStatePLANNED) {
		metrics.Planned++
	}
	if states.Has(api.V0041NodeStateREBOOTREQUESTED) {
		metrics.RebootRequested++
	}
	if states.Has(api.V0041NodeStateRESERVED) {
		metrics.Reserved++
	}
}

// Ref: https://slurm.schedmd.com/sinfo.html#SECTION_NODE-STATE-CODES
type NodeStates struct {
	// Other
	Total uint
	// Base State
	Allocated uint
	Down      uint
	Error     uint
	Future    uint
	Idle      uint
	Mixed     uint
	Unknown   uint
	// Flag State
	Completing      uint
	Drain           uint
	Fail            uint
	Maintenance     uint
	NotResponding   uint
	Planned         uint
	RebootRequested uint
	Reserved        uint
}
