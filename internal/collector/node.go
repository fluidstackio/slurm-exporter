// SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"context"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/log"

	api "github.com/SlinkyProject/slurm-client/api/v0041"
	"github.com/SlinkyProject/slurm-client/pkg/client"
	"github.com/SlinkyProject/slurm-client/pkg/types"
)

// Ref: https://prometheus.io/docs/practices/naming/#metric-names
func NewNodeCollector(slurmClient client.Client) prometheus.Collector {
	return &nodeCollector{
		slurmClient: slurmClient,

		NodeCount: prometheus.NewDesc("slurm_nodes_total", "Total number of nodes", nil, nil),
		NodeStates: nodeStatesCollector{
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
		},
		NodeCombinedState: prometheus.NewDesc("slurm_state_combined", "Combined Slurm node state (0=available, 1=unavailable)", combinedStateLabels, nil),
		NodeTres: nodeTresCollector{
			// CPUs
			CpusTotal:     prometheus.NewDesc("slurm_node_cpus_total", "Total number of CPUs on the node", nodeLabels, nil),
			CpusEffective: prometheus.NewDesc("slurm_node_cpus_effective_total", "Total number of effective CPUs on the node, excludes CoreSpec", nodeLabels, nil),
			CpusAlloc:     prometheus.NewDesc("slurm_node_cpus_alloc_total", "Number of Allocated CPUs on the node", nodeLabels, nil),
			CpusIdle:      prometheus.NewDesc("slurm_node_cpus_idle_total", "Number of Idle CPUs on the node", nodeLabels, nil),
			// Memory
			MemoryTotal:     prometheus.NewDesc("slurm_node_memory_bytes", "Total amount of Memory (MB) on the node", nodeLabels, nil),
			MemoryEffective: prometheus.NewDesc("slurm_node_memory_effective_bytes", "Total amount of effective Memory (MB) on the node, excludes MemSpec", nodeLabels, nil),
			MemoryAlloc:     prometheus.NewDesc("slurm_node_memory_alloc_bytes", "Amount of Allocated Memory (MB) on the node", nodeLabels, nil),
			MemoryFree:      prometheus.NewDesc("slurm_node_memory_free_bytes", "Amount of Free Memory (MB) on the node", nodeLabels, nil),
		},
	}
}

// Ref: https://slurm.schedmd.com/sinfo.html#SECTION_NODE-STATE-CODES
type nodeCollector struct {
	slurmClient client.Client

	NodeCount         *prometheus.Desc
	NodeStates        nodeStatesCollector
	NodeCombinedState *prometheus.Desc
	NodeTres          nodeTresCollector
}

type nodeStatesCollector struct {
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

type nodeTresCollector struct {
	// CPUs
	CpusTotal     *prometheus.Desc
	CpusEffective *prometheus.Desc
	CpusAlloc     *prometheus.Desc
	CpusIdle      *prometheus.Desc
	// Memory
	MemoryTotal     *prometheus.Desc
	MemoryEffective *prometheus.Desc
	MemoryAlloc     *prometheus.Desc
	MemoryFree      *prometheus.Desc
}

func (c *nodeCollector) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(c, ch)
}

func (c *nodeCollector) Collect(ch chan<- prometheus.Metric) {
	ctx := context.TODO()
	logger := log.FromContext(ctx).WithName("NodeCollector")

	logger.V(1).Info("collecting metrics")

	metrics, err := c.getNodeMetrics(ctx)
	if err != nil {
		logger.Error(err, "failed to collect node metrics")
		return
	}

	ch <- prometheus.MustNewConstMetric(c.NodeCount, prometheus.GaugeValue, float64(metrics.NodeCount))
	// Base State
	ch <- prometheus.MustNewConstMetric(c.NodeStates.Allocated, prometheus.GaugeValue, float64(metrics.NodeStates.Allocated))
	ch <- prometheus.MustNewConstMetric(c.NodeStates.Down, prometheus.GaugeValue, float64(metrics.NodeStates.Down))
	ch <- prometheus.MustNewConstMetric(c.NodeStates.Error, prometheus.GaugeValue, float64(metrics.NodeStates.Error))
	ch <- prometheus.MustNewConstMetric(c.NodeStates.Future, prometheus.GaugeValue, float64(metrics.NodeStates.Future))
	ch <- prometheus.MustNewConstMetric(c.NodeStates.Idle, prometheus.GaugeValue, float64(metrics.NodeStates.Idle))
	ch <- prometheus.MustNewConstMetric(c.NodeStates.Mixed, prometheus.GaugeValue, float64(metrics.NodeStates.Mixed))
	ch <- prometheus.MustNewConstMetric(c.NodeStates.Unknown, prometheus.GaugeValue, float64(metrics.NodeStates.Unknown))
	// Flag State
	ch <- prometheus.MustNewConstMetric(c.NodeStates.Completing, prometheus.GaugeValue, float64(metrics.NodeStates.Completing))
	ch <- prometheus.MustNewConstMetric(c.NodeStates.Drain, prometheus.GaugeValue, float64(metrics.NodeStates.Drain))
	ch <- prometheus.MustNewConstMetric(c.NodeStates.Fail, prometheus.GaugeValue, float64(metrics.NodeStates.Fail))
	ch <- prometheus.MustNewConstMetric(c.NodeStates.Maintenance, prometheus.GaugeValue, float64(metrics.NodeStates.Maintenance))
	ch <- prometheus.MustNewConstMetric(c.NodeStates.NotResponding, prometheus.GaugeValue, float64(metrics.NodeStates.NotResponding))
	ch <- prometheus.MustNewConstMetric(c.NodeStates.Planned, prometheus.GaugeValue, float64(metrics.NodeStates.Planned))
	ch <- prometheus.MustNewConstMetric(c.NodeStates.RebootRequested, prometheus.GaugeValue, float64(metrics.NodeStates.RebootRequested))
	ch <- prometheus.MustNewConstMetric(c.NodeStates.Reserved, prometheus.GaugeValue, float64(metrics.NodeStates.Reserved))

	// Combined State
	for node, state := range metrics.NodeCombinedStates {
		ch <- prometheus.MustNewConstMetric(c.NodeCombinedState, prometheus.GaugeValue, float64(state.Unavailable), node, state.CombinedState)
	}

	for node, data := range metrics.NodeTresPer {
		// CPUs
		ch <- prometheus.MustNewConstMetric(c.NodeTres.CpusTotal, prometheus.GaugeValue, float64(data.CpusTotal), node)
		ch <- prometheus.MustNewConstMetric(c.NodeTres.CpusEffective, prometheus.GaugeValue, float64(data.CpusEffective), node)
		ch <- prometheus.MustNewConstMetric(c.NodeTres.CpusAlloc, prometheus.GaugeValue, float64(data.CpusAlloc), node)
		ch <- prometheus.MustNewConstMetric(c.NodeTres.CpusIdle, prometheus.GaugeValue, float64(data.CpusIdle), node)
		// Memory
		ch <- prometheus.MustNewConstMetric(c.NodeTres.MemoryTotal, prometheus.GaugeValue, float64(data.MemoryTotal), node)
		ch <- prometheus.MustNewConstMetric(c.NodeTres.MemoryEffective, prometheus.GaugeValue, float64(data.MemoryEffective), node)
		ch <- prometheus.MustNewConstMetric(c.NodeTres.MemoryAlloc, prometheus.GaugeValue, float64(data.MemoryAlloc), node)
		ch <- prometheus.MustNewConstMetric(c.NodeTres.MemoryFree, prometheus.GaugeValue, float64(data.MemoryFree), node)
	}
}

func (c *nodeCollector) getNodeMetrics(ctx context.Context) (*NodeCollectorMetrics, error) {
	nodeList := &types.V0041NodeList{}
	if err := c.slurmClient.List(ctx, nodeList); err != nil {
		return nil, err
	}
	metrics := calculateNodeMetrics(nodeList)
	return metrics, nil
}

func calculateNodeMetrics(nodeList *types.V0041NodeList) *NodeCollectorMetrics {
	metrics := &NodeCollectorMetrics{
		NodeMetrics: NodeMetrics{
			NodeCount: uint(len(nodeList.Items)),
		},
		NodeTresPer:        make(map[string]*NodeTres, len(nodeList.Items)),
		NodeCombinedStates: make(map[string]*NodeCombinedState, len(nodeList.Items)),
	}
	for _, node := range nodeList.Items {
		key := string(node.GetKey())
		calculateNodeState(&metrics.NodeStates, node)
		calculateNodeTres(&metrics.NodeTres, node)
		if _, ok := metrics.NodeTresPer[key]; !ok {
			metrics.NodeTresPer[key] = &NodeTres{}
		}
		calculateNodeTres(metrics.NodeTresPer[key], node)
		// Calculate combined state
		metrics.NodeCombinedStates[key] = calculateNodeCombinedState(node)
	}
	return metrics
}

func calculateNodeState(metrics *NodeStates, node types.V0041Node) {
	metrics.total++
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

func calculateNodeTres(metrics *NodeTres, node types.V0041Node) {
	metrics.total++
	// CPUs
	metrics.CpusTotal += uint(ptr.Deref(node.Cpus, 0))
	metrics.CpusEffective += uint(ptr.Deref(node.EffectiveCpus, 0))
	metrics.CpusAlloc += uint(ptr.Deref(node.AllocCpus, 0))
	metrics.CpusIdle += uint(ptr.Deref(node.AllocIdleCpus, 0))
	// Memory
	metrics.MemoryTotal += uint(ptr.Deref(node.RealMemory, 0))
	metrics.MemoryEffective += uint(ptr.Deref(node.RealMemory, 0) - ptr.Deref(node.SpecializedMemory, 0))
	metrics.MemoryAlloc += uint(ptr.Deref(node.AllocMemory, 0))
	metrics.MemoryFree += uint(ParseUint64NoVal(node.FreeMem))
}

type NodeCollectorMetrics struct {
	NodeMetrics
	// Per Node
	NodeTresPer        map[string]*NodeTres
	NodeCombinedStates map[string]*NodeCombinedState
}

type NodeCombinedState struct {
	CombinedState string
	Unavailable   int
}

type NodeMetrics struct {
	NodeCount  uint
	NodeStates NodeStates
	NodeTres   NodeTres
}

// Ref: https://slurm.schedmd.com/sinfo.html#SECTION_NODE-STATE-CODES
type NodeStates struct {
	total uint
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

type NodeTres struct {
	total uint
	// CPUs
	CpusTotal     uint
	CpusEffective uint
	CpusAlloc     uint
	CpusIdle      uint
	// Memory
	MemoryTotal     uint
	MemoryEffective uint
	MemoryAlloc     uint
	MemoryFree      uint
}

func calculateNodeCombinedState(node types.V0041Node) *NodeCombinedState {
	states := node.GetStateAsSet()
	var stateNames []string
	unavailable := 0

	// Check each state and build the combined state string
	if states.Has(api.V0041NodeStateALLOCATED) {
		stateNames = append(stateNames, "allocated")
	}
	if states.Has(api.V0041NodeStateCLOUD) {
		stateNames = append(stateNames, "cloud")
	}
	if states.Has(api.V0041NodeStateCOMPLETING) {
		stateNames = append(stateNames, "completing")
	}
	if states.Has(api.V0041NodeStateDOWN) {
		stateNames = append(stateNames, "down")
		unavailable = 1
	}
	if states.Has(api.V0041NodeStateDRAIN) {
		stateNames = append(stateNames, "drain")
		unavailable = 1
	}
	if states.Has(api.V0041NodeStateDYNAMICFUTURE) {
		stateNames = append(stateNames, "dynamicFuture")
	}
	if states.Has(api.V0041NodeStateDYNAMICNORM) {
		stateNames = append(stateNames, "dynamicNorm")
	}
	if states.Has(api.V0041NodeStateERROR) {
		stateNames = append(stateNames, "err")
		unavailable = 1
	}
	if states.Has(api.V0041NodeStateFAIL) {
		stateNames = append(stateNames, "fail")
		unavailable = 1
	}
	if states.Has(api.V0041NodeStateFUTURE) {
		stateNames = append(stateNames, "future")
	}
	if states.Has(api.V0041NodeStateIDLE) {
		stateNames = append(stateNames, "idle")
	}
	if states.Has(api.V0041NodeStateINVALID) {
		stateNames = append(stateNames, "invalid")
		unavailable = 1
	}
	if states.Has(api.V0041NodeStateINVALIDREG) {
		stateNames = append(stateNames, "invalidReg")
		unavailable = 1
	}
	if states.Has(api.V0041NodeStateMAINTENANCE) {
		stateNames = append(stateNames, "maintenance")
		unavailable = 1
	}
	if states.Has(api.V0041NodeStateMIXED) {
		stateNames = append(stateNames, "mixed")
	}
	if states.Has(api.V0041NodeStateNOTRESPONDING) {
		stateNames = append(stateNames, "notResponding")
		unavailable = 1
	}
	if states.Has(api.V0041NodeStatePLANNED) {
		stateNames = append(stateNames, "planned")
	}
	if states.Has(api.V0041NodeStatePOWERDOWN) {
		stateNames = append(stateNames, "powerDown")
	}
	if states.Has(api.V0041NodeStatePOWERDRAIN) {
		stateNames = append(stateNames, "powerDrain")
	}
	if states.Has(api.V0041NodeStatePOWEREDDOWN) {
		stateNames = append(stateNames, "poweredDown")
		unavailable = 1
	}
	if states.Has(api.V0041NodeStatePOWERINGDOWN) {
		stateNames = append(stateNames, "poweringDown")
		unavailable = 1
	}
	if states.Has(api.V0041NodeStatePOWERINGUP) {
		stateNames = append(stateNames, "poweringUp")
		unavailable = 1
	}
	if states.Has(api.V0041NodeStatePOWERUP) {
		stateNames = append(stateNames, "powerUp")
	}
	if states.Has(api.V0041NodeStateREBOOTCANCELED) {
		stateNames = append(stateNames, "rebootCanceled")
	}
	if states.Has(api.V0041NodeStateREBOOTISSUED) {
		stateNames = append(stateNames, "rebootIssued")
	}
	if states.Has(api.V0041NodeStateREBOOTREQUESTED) {
		stateNames = append(stateNames, "rebootRequested")
	}
	if states.Has(api.V0041NodeStateRESERVED) {
		stateNames = append(stateNames, "reserved")
	}
	if states.Has(api.V0041NodeStateRESUME) {
		stateNames = append(stateNames, "resume")
	}
	if states.Has(api.V0041NodeStateUNDRAIN) {
		stateNames = append(stateNames, "undrain")
	}
	if states.Has(api.V0041NodeStateUNKNOWN) {
		stateNames = append(stateNames, "unknown")
		unavailable = 1
	}

	// Special case: drain + (allocated or mixed) = available
	if states.Has(api.V0041NodeStateDRAIN) && (states.Has(api.V0041NodeStateALLOCATED) || states.Has(api.V0041NodeStateMIXED)) {
		unavailable = 0
	}

	combinedState := strings.Join(stateNames, "+")
	if combinedState == "" {
		combinedState = "none"
	}

	return &NodeCombinedState{
		CombinedState: combinedState,
		Unavailable:   unavailable,
	}
}
