// SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
// SPDX-License-Identifier: Apache-2.0

package exporter

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/log"

	api "github.com/SlinkyProject/slurm-client/api/v0041"
	"github.com/SlinkyProject/slurm-client/pkg/client"
	"github.com/SlinkyProject/slurm-client/pkg/object"
	slurmtypes "github.com/SlinkyProject/slurm-client/pkg/types"
)

var (
	partitionLabel        = []string{"partition"}
	nodeLabels            = []string{"node", "hostname"}
	jobLabel              = []string{"user"}
	jobIDLabel            = []string{"jobid", "nodes"}
	jobCPUAllocationLabel = []string{"jobid", "core", "socket", "hostname"}
	combinedStateLabel    = []string{"node", "hostname", "combinedState"}
	nodeReasonLabel       = []string{"node", "hostname", "reason", "user2", "timestamp"}
)

type PartitionData struct {
	Nodes           int32
	Cpus            int32
	PendingJobs     int32
	PendingMaxNodes int32
	RunningJobs     int32
	HoldJobs        int32
	Jobs            int32
	Alloc           int32
	Idle            int32
}

type NodeData struct {
	Cpus            int32
	Alloc           int32
	Idle            int32
	States          NodeStates
	unavailable     int32
	CombinedState   string
	Reason          string
	ReasonSetByUser string
	ReasonChangedAt int64
}

type NodeStates struct {
	allocated       int32
	cloud           int32
	completing      int32
	down            int32
	drain           int32
	dynamicFuture   int32
	dynamicNorm     int32
	err             int32
	fail            int32
	future          int32
	idle            int32
	invalid         int32
	invalidReg      int32
	maintenance     int32
	mixed           int32
	notResponding   int32
	planned         int32
	powerDown       int32
	powerDrain      int32
	poweredDown     int32
	poweringDown    int32
	poweringUp      int32
	powerUp         int32
	rebootCanceled  int32
	rebootIssued    int32
	rebootRequested int32
	reserved        int32
	resume          int32
	undrain         int32
	unknown         int32
}

type JobStates struct {
	running    int32
	pending    int32
	hold       int32
	completing int32
	nodes      string
	allocation api.V0041JobResNodes
}

type JobData struct {
	Count   int32
	Pending int32
	Running int32
	Hold    int32
	Cpus    int32
}

type slurmData struct {
	partitions map[string]*PartitionData
	nodes      map[string]*NodeData
	nodestates NodeStates
	jobs       map[string]*JobData
	jobstates  map[int32]*JobStates
}

type SlurmCollector struct {
	slurmClient client.Client

	partitionNodes           *prometheus.Desc
	partitionTotalCpus       *prometheus.Desc
	partitionIdleCpus        *prometheus.Desc
	partitionAllocCpus       *prometheus.Desc
	partitionJobs            *prometheus.Desc
	partitionPendingJobs     *prometheus.Desc
	partitionMaxPendingNodes *prometheus.Desc
	partitionRunningJobs     *prometheus.Desc
	partitionHoldJobs        *prometheus.Desc
	// Node CPUs
	nodeTotalCpus            *prometheus.Desc
	nodeIdleCpus             *prometheus.Desc
	nodeAllocCpus            *prometheus.Desc
	// Node State Counts
	totalNodes               *prometheus.Desc
	allocNodes               *prometheus.Desc
	completingNodes          *prometheus.Desc
	downNodes                *prometheus.Desc
	drainNodes               *prometheus.Desc
	errNodes                 *prometheus.Desc
	idleNodes                *prometheus.Desc
	maintenanceNodes         *prometheus.Desc
	mixedNodes               *prometheus.Desc
	reservedNodes            *prometheus.Desc
	cloudNodes               *prometheus.Desc
	dynamicFutureNodes       *prometheus.Desc
	dynamicNormNodes         *prometheus.Desc
	failNodes                *prometheus.Desc
	futureNodes              *prometheus.Desc
	invalidNodes             *prometheus.Desc
	invalidRegNodes          *prometheus.Desc
	notRespondingNodes       *prometheus.Desc
	plannedNodes             *prometheus.Desc
	powerDownNodes           *prometheus.Desc
	powerDrainNodes          *prometheus.Desc
	poweredDownNodes         *prometheus.Desc
	poweringDownNodes        *prometheus.Desc
	poweringUpNodes          *prometheus.Desc
	powerUpNodes             *prometheus.Desc
	rebootCanceledNodes      *prometheus.Desc
	rebootIssuedNodes        *prometheus.Desc
	rebootRequestedNodes     *prometheus.Desc
	resumeNodes              *prometheus.Desc
	undrainNodes             *prometheus.Desc
	unknownNodes             *prometheus.Desc
	// Node States
	nodeStateCombined        *prometheus.Desc
	nodeStateAlloc           *prometheus.Desc
	nodeStateCompleting      *prometheus.Desc
	nodeStateDown            *prometheus.Desc
	nodeStateDrain           *prometheus.Desc
	nodeStateError           *prometheus.Desc
	nodeStateIdle            *prometheus.Desc
	nodeStateMaintanance     *prometheus.Desc
	nodeStateMixed           *prometheus.Desc
	nodeStateReserved        *prometheus.Desc
	nodeStateCloud           *prometheus.Desc
	nodeStateDynamicFuture   *prometheus.Desc
	nodeStateDynamicNorm     *prometheus.Desc
	nodeStateFail            *prometheus.Desc
	nodeStateFuture          *prometheus.Desc
	nodeStateInvalid         *prometheus.Desc
	nodeStateInvalidReg      *prometheus.Desc
	nodeStateNotResponding   *prometheus.Desc
	nodeStatePlanned         *prometheus.Desc
	nodeStatePowerDown       *prometheus.Desc
	nodeStatePowerDrain      *prometheus.Desc
	nodeStatePoweredDown     *prometheus.Desc
	nodeStatePoweringDown    *prometheus.Desc
	nodeStatePoweringUp      *prometheus.Desc
	nodeStatePowerUp         *prometheus.Desc
	nodeStateRebootCanceled  *prometheus.Desc
	nodeStateRebootIssued    *prometheus.Desc
	nodeStateRebootRequested *prometheus.Desc
	nodeStateResume          *prometheus.Desc
	nodeStateUndrain         *prometheus.Desc
	nodeStateUnknown         *prometheus.Desc
	// User
	userJobTotal             *prometheus.Desc
	userCPUsTotal            *prometheus.Desc
	userPendingJobs          *prometheus.Desc
	userRunningJobs          *prometheus.Desc
	userHoldJobs             *prometheus.Desc
	// Job
	jobRunning               *prometheus.Desc
	jobPending               *prometheus.Desc
	jobHold                  *prometheus.Desc
	jobCompleting            *prometheus.Desc
	jobCPUAllocation         *prometheus.Desc
}

func splitOutsideBrackets(input string) []string {
	var result []string
	bracketDepth := 0
	start := 0

	for i, char := range input {
		switch char {
		case '[':
			bracketDepth++
		case ']':
			bracketDepth--
		case ',':
			if bracketDepth == 0 {
				part := strings.TrimSpace(input[start:i])
				result = append(result, part)
				start = i + 1
			}
		}
	}

	if start < len(input) {
		result = append(result, strings.TrimSpace(input[start:]))
	}
	return result
}

// Expand items inside brackets, handling ranges and mixed types
func expandBracketed(base string, content string) []string {
	var expanded []string

	items := strings.Split(content, ",")
	for _, item := range items {
		item = strings.TrimSpace(item)

		if strings.Contains(item, "-") {
			parts := strings.Split(item, "-")
			if len(parts) != 2 {
				// malformed range, skip or log
				continue
			}

			startStr := parts[0]
			endStr := parts[1]

			// Check if both sides are numeric
			startNum, errStart := strconv.Atoi(startStr)
			endNum, errEnd := strconv.Atoi(endStr)

			if errStart == nil && errEnd == nil {
				// Keep the width of the largest side (to preserve padding)
				width := int(math.Max(float64(len(startStr)), float64(len(endStr))))
				for i := startNum; i <= endNum; i++ {
					expanded = append(expanded, fmt.Sprintf("%s%0*d", base, width, i))
				}
			} else {
				// If not numeric, treat as literal — not expanding
				expanded = append(expanded, fmt.Sprintf("%s%s", base, item))
			}
		} else {
			// Just append directly
			expanded = append(expanded, fmt.Sprintf("%s%s", base, item))
		}
	}
	return expanded
}

func GetActiveStates(states NodeStates) string {
	var activeStateNames []string

	val := reflect.ValueOf(states)
	for i := 0; i < val.NumField(); i++ {
		fieldName := val.Type().Field(i).Name
		fieldValue := val.Field(i).Int()
		if fieldValue == 1 {
			activeStateNames = append(activeStateNames, fieldName)
		}
	}
	return strings.Join(activeStateNames, "+")
}

// Initialize the slurm client to talk to slurmrestd.
// Requires that the env SLURM_JWT is set.
func NewSlurmClient(server string, cacheFreq time.Duration) (client.Client, error) {
	ctx := context.Background()
	log := log.FromContext(ctx)

	token, ok := os.LookupEnv("SLURM_JWT")
	if !ok || token == "" {
		return nil, errors.New("SLURM_JWT must be defined and not empty")
	}

	// Create slurm client
	config := &client.Config{
		Server:    server,
		AuthToken: token,
	}

	// Instruct the client to keep a cache of slurm objects
	clientOptions := client.ClientOptions{
		EnableFor: []object.Object{
			&slurmtypes.V0041PartitionInfo{},
			&slurmtypes.V0041Node{},
			&slurmtypes.V0041JobInfo{},
		},
		CacheSyncPeriod: cacheFreq,
	}
	slurmClient, err := client.NewClient(config, &clientOptions)
	if err != nil {
		return nil, err
	}

	// Start client cache
	go slurmClient.Start(ctx)

	log.Info("Created slurm client")

	return slurmClient, nil
}

// slurmCollect will read slurm objects from slurmrestd and return slurmData to represent
// partition, node, and job information of the running slurm cluster
func (r *SlurmCollector) slurmCollectType(list object.ObjectList) error {
	ctx := context.Background()
	log := log.FromContext(ctx)

	// Read slurm information from cache
	if err := r.slurmClient.List(ctx, list); err != nil {
		log.Error(err, "Could not list objects")
		return err
	}

	return nil
}

// slurmParse will return slurmData to represent partition, node, and job
// information of the running slurm cluster.
func (r *SlurmCollector) slurmParse(
	jobs *slurmtypes.V0041JobInfoList,
	nodes *slurmtypes.V0041NodeList,
	partitions *slurmtypes.V0041PartitionInfoList,
) slurmData {
	ctx := context.Background()
	log := log.FromContext(ctx)

	slurmData := slurmData{}
	ns := NodeStates{}

	// Populate partitionData indexed by partition with the number of cpus
	// and nodes.
	partitionData := make(map[string]*PartitionData)
	for _, p := range partitions.Items {
		key := string(p.GetKey())
		_, ok := partitionData[key]
		if !ok {
			partitionData[key] = &PartitionData{}
		}
		partitionData[key].Cpus = ptr.Deref(p.Cpus.Total, 0)
		partitionData[key].Nodes = ptr.Deref(p.Nodes.Total, 0)
	}

	// Populate nodeData indexed by node with the number of cpus
	// and how many are allocated/idle.
	nodeData := make(map[string]*NodeData)
	for _, n := range nodes.Items {
		key := string(n.GetKey())
		_, ok := nodeData[key]
		if !ok {
			nodeData[key] = &NodeData{}
		}
		nodeData[key].Cpus = ptr.Deref(n.Cpus, 0)
		nodeData[key].Alloc = ptr.Deref(n.AllocCpus, 0)
		nodeData[key].Idle = ptr.Deref(n.AllocIdleCpus, 0)
		nodeData[key].States = NodeStates{}
		nodeData[key].Reason = ptr.Deref(n.Reason, "")
		nodeData[key].ReasonSetByUser = ptr.Deref(n.ReasonSetByUser, "")
		nodeData[key].ReasonChangedAt = 0
		if n.ReasonChangedAt != nil {
			nodeData[key].ReasonChangedAt = *n.ReasonChangedAt.Number
		}
		for _, s := range n.GetStateAsSet().UnsortedList() {
			switch s {
			case api.V0041NodeStateALLOCATED:
				ns.allocated++
				nodeData[key].States.allocated++
			case api.V0041NodeStateCLOUD:
				ns.cloud++
				nodeData[key].States.cloud++
			case api.V0041NodeStateCOMPLETING:
				ns.completing++
				nodeData[key].States.completing++
			case api.V0041NodeStateDOWN:
				ns.down++
				nodeData[key].States.down++
				nodeData[key].unavailable = 1
			case api.V0041NodeStateDRAIN:
				ns.drain++
				nodeData[key].States.drain++
				nodeData[key].unavailable = 1
			case api.V0041NodeStateDYNAMICFUTURE:
				ns.dynamicFuture++
				nodeData[key].States.dynamicFuture++
			case api.V0041NodeStateDYNAMICNORM:
				ns.dynamicNorm++
				nodeData[key].States.dynamicNorm++
			case api.V0041NodeStateERROR:
				ns.err++
				nodeData[key].States.err++
				nodeData[key].unavailable = 1
			case api.V0041NodeStateFAIL:
				ns.fail++
				nodeData[key].States.fail++
				nodeData[key].unavailable = 1
			case api.V0041NodeStateFUTURE:
				ns.future++
				nodeData[key].States.future++
			case api.V0041NodeStateIDLE:
				ns.idle++
				nodeData[key].States.idle++
			case api.V0041NodeStateINVALID:
				ns.invalid++
				nodeData[key].States.invalid++
				nodeData[key].unavailable = 1
			case api.V0041NodeStateINVALIDREG:
				ns.invalidReg++
				nodeData[key].States.invalidReg++
				nodeData[key].unavailable = 1
			case api.V0041NodeStateMAINTENANCE:
				ns.maintenance++
				nodeData[key].States.maintenance++
				nodeData[key].unavailable = 1
			case api.V0041NodeStateMIXED:
				ns.mixed++
				nodeData[key].States.mixed++
			case api.V0041NodeStateNOTRESPONDING:
				ns.notResponding++
				nodeData[key].States.notResponding++
				nodeData[key].unavailable = 1
			case api.V0041NodeStatePLANNED:
				ns.planned++
				nodeData[key].States.planned++
			case api.V0041NodeStatePOWERDOWN:
				ns.powerDown++
				nodeData[key].States.powerDown++
			case api.V0041NodeStatePOWERDRAIN:
				ns.powerDrain++
				nodeData[key].States.powerDrain++
			case api.V0041NodeStatePOWEREDDOWN:
				ns.poweredDown++
				nodeData[key].States.poweredDown++
				nodeData[key].unavailable = 1
			case api.V0041NodeStatePOWERINGDOWN:
				ns.poweringDown++
				nodeData[key].States.poweringDown++
				nodeData[key].unavailable = 1
			case api.V0041NodeStatePOWERINGUP:
				ns.poweringUp++
				nodeData[key].States.poweringUp++
				nodeData[key].unavailable = 1
			case api.V0041NodeStatePOWERUP:
				ns.powerUp++
				nodeData[key].States.powerUp++
			case api.V0041NodeStateREBOOTCANCELED:
				ns.rebootCanceled++
				nodeData[key].States.rebootCanceled++
			case api.V0041NodeStateREBOOTISSUED:
				ns.rebootIssued++
				nodeData[key].States.rebootIssued++
			case api.V0041NodeStateREBOOTREQUESTED:
				ns.rebootRequested++
				nodeData[key].States.rebootRequested++
			case api.V0041NodeStateRESERVED:
				ns.reserved++
				nodeData[key].States.reserved++
			case api.V0041NodeStateRESUME:
				ns.resume++
				nodeData[key].States.resume++
			case api.V0041NodeStateUNDRAIN:
				ns.undrain++
				nodeData[key].States.undrain++
			case api.V0041NodeStateUNKNOWN:
				ns.unknown++
				nodeData[key].States.unknown++
				nodeData[key].unavailable = 1
			}
			nodeData[key].CombinedState = GetActiveStates(nodeData[key].States)
			if nodeData[key].States.drain == 1 && (nodeData[key].States.allocated == 1 || nodeData[key].States.mixed == 1) {
				nodeData[key].unavailable = 0
			}
		}

		// Update partitionData with per node information as this
		// is not yet available with the /partitions endpoint.
		partitions := ptr.Deref(n.Partitions, api.V0041CsvString{})
		for _, p := range partitions {
			_, ok := partitionData[p]
			if ok {
				partitionData[p].Alloc += nodeData[key].Alloc
				partitionData[p].Idle += nodeData[key].Idle
			}
		}
	}

	// Populate jobData indexed by user with the number of jobs
	// they have and their various states.
	jobData := make(map[string]*JobData)
	// Populate jobDatabyID indexed by jobID with the current state of the job
	jobDatabyID := make(map[int32]*JobStates)
	for _, j := range jobs.Items {
		jobID := ptr.Deref(j.JobId, 0)
		if _, ok := jobDatabyID[jobID]; !ok {
			jobDatabyID[jobID] = &JobStates{}
		}
		userName := ptr.Deref(j.UserName, "")
		partition := ptr.Deref(j.Partition, "")

		jobDatabyID[jobID].allocation = nil
		if j.JobResources != nil && j.JobResources.Nodes != nil && j.JobResources.Nodes.Allocation != nil {
			jobDatabyID[jobID].allocation = *j.JobResources.Nodes.Allocation
		}
		if _, key := jobData[userName]; !key {
			jobData[userName] = &JobData{}
		}
		if _, key := partitionData[partition]; !key {
			log.Info("No partition found for job")
			continue
		}
		// Update partitionData with the number of jobs scheduled
		if !j.GetStateAsSet().HasAny(
			api.V0041JobInfoJobStateCOMPLETED,
			api.V0041JobInfoJobStateCOMPLETING,
			api.V0041JobInfoJobStateCANCELLED) {
			partitionData[partition].Jobs++
			jobData[userName].Count++
		}
		isHold := ptr.Deref(j.Hold, false)
		if j.GetStateAsSet().HasAll(api.V0041JobInfoJobStatePENDING) &&
			!isHold {
			partitionData[partition].PendingJobs++
			jobData[userName].Pending++
			jobDatabyID[jobID].pending++
		}
		if j.GetStateAsSet().HasAll(api.V0041JobInfoJobStateRUNNING) {
			partitionData[partition].RunningJobs++
			jobData[userName].Running++
			jobDatabyID[jobID].running++
		}
		if j.GetStateAsSet().HasAll(api.V0041JobInfoJobStateCOMPLETING) {
			jobDatabyID[jobID].completing++
		}
		if isHold {
			partitionData[partition].HoldJobs++
			jobData[userName].Hold++
			jobDatabyID[jobID].hold++
		}
		jobData[userName].Cpus += *j.Cpus.Number
		if j.Nodes != nil {
			nodes := *j.Nodes
			parts := splitOutsideBrackets(nodes)
			var hostnames []string

			// Match something like "base[...]"
			re := regexp.MustCompile(`^([a-zA-Z0-9\-_]+)\[(.*?)\]$`)

			for _, part := range parts {
				if matches := re.FindStringSubmatch(part); len(matches) == 3 {
					base := matches[1]
					content := matches[2]
					hostnames = append(hostnames, expandBracketed(base, content)...)
				} else {
					// Standalone hostname — use as-is
					hostnames = append(hostnames, part)
				}
			}
			jobDatabyID[jobID].nodes = strings.Join(hostnames, ",")
		}
		// Track total pending nodes for a partition for jobs that
		// meet the criteria below. This exposes metrics that may be
		// used for autoscaling. Jobs with an eligible date greater
		// than one year are ignored.
		eligibleTimeNoVal := ptr.Deref(j.EligibleTime, api.V0041Uint64NoValStruct{})
		eligibleTime := ptr.Deref(eligibleTimeNoVal.Number, 0)
		if j.GetStateAsSet().HasAll(api.V0041JobInfoJobStatePENDING) &&
			eligibleTime >= 0 &&
			time.Unix(eligibleTime, 0).After(time.Now().AddDate(-1, 0, 0)) &&
			!isHold {
			nodeCountNoVal := ptr.Deref(j.NodeCount, api.V0041Uint32NoValStruct{})
			nodeCount := ptr.Deref(nodeCountNoVal.Number, 0)
			if partitionData[partition].PendingMaxNodes < nodeCount {
				partitionData[partition].PendingMaxNodes = nodeCount
			}
		}
	}

	slurmData.partitions = partitionData
	slurmData.nodes = nodeData
	slurmData.jobs = jobData
	slurmData.nodestates = ns
	slurmData.jobstates = jobDatabyID
	return slurmData
}

func NewSlurmCollector(slurmClient client.Client) *SlurmCollector {
	return &SlurmCollector{
		slurmClient:              slurmClient,
		// Partitions
		partitionNodes:           prometheus.NewDesc("slurm_partition_nodes", "Number of nodes in a slurm partition", partitionLabel, nil),
		partitionTotalCpus:       prometheus.NewDesc("slurm_partition_total_cpus", "Number of CPUs in a slurm partition", partitionLabel, nil),
		partitionIdleCpus:        prometheus.NewDesc("slurm_partition_idle_cpus", "Number of idle CPUs in a slurm partition", partitionLabel, nil),
		partitionAllocCpus:       prometheus.NewDesc("slurm_partition_alloc_cpus", "Number of allocated CPUs in a slurm partition", partitionLabel, nil),
		partitionJobs:            prometheus.NewDesc("slurm_partition_jobs", "Number of allocated CPUs in a slurm partition", partitionLabel, nil),
		partitionPendingJobs:     prometheus.NewDesc("slurm_partition_pending_jobs", "Number of pending jobs in a slurm partition", partitionLabel, nil),
		partitionMaxPendingNodes: prometheus.NewDesc("slurm_partition_max_pending_nodes", "Number of nodes pending for the largest job in the partition", partitionLabel, nil),
		partitionRunningJobs:     prometheus.NewDesc("slurm_partition_running_jobs", "Number of running jobs in a slurm partition", partitionLabel, nil),
		partitionHoldJobs:        prometheus.NewDesc("slurm_partition_hold_jobs", "Number of hold jobs in a slurm partition", partitionLabel, nil),
		// Node CPUs
		nodeTotalCpus:            prometheus.NewDesc("slurm_node_total_cpus", "Number of CPUs in a slurm node", nodeLabels, nil),
		nodeIdleCpus:             prometheus.NewDesc("slurm_node_idle_cpus", "Number of idle CPUs in a slurm node", nodeLabels, nil),
		nodeAllocCpus:            prometheus.NewDesc("slurm_node_alloc_cpus", "Number of allocated CPUs in a slurm node", nodeLabels, nil),
		// Node State Counts
		totalNodes:               prometheus.NewDesc("slurm_nodes_total", "Total number of slurm nodes", nil, nil),
		allocNodes:               prometheus.NewDesc("slurm_nodes_alloc", "Number of nodes in allocated state", nil, nil),
		completingNodes:          prometheus.NewDesc("slurm_nodes_completing", "Number of nodes in completing state", nil, nil),
		downNodes:                prometheus.NewDesc("slurm_nodes_down", "Number of nodes in down state", nil, nil),
		drainNodes:               prometheus.NewDesc("slurm_nodes_drain", "Number of nodes in drain state", nil, nil),
		errNodes:                 prometheus.NewDesc("slurm_nodes_err", "Number of nodes in error state", nil, nil),
		idleNodes:                prometheus.NewDesc("slurm_nodes_idle", "Number of nodes in idle state", nil, nil),
		maintenanceNodes:         prometheus.NewDesc("slurm_nodes_maintenance", "Number of nodes in maintenance state", nil, nil),
		mixedNodes:               prometheus.NewDesc("slurm_nodes_mixed", "Number of nodes in mixed state", nil, nil),
		reservedNodes:            prometheus.NewDesc("slurm_nodes_reserved", "Number of nodes in reserved state", nil, nil),
		cloudNodes:               prometheus.NewDesc("slurm_nodes_cloud", "Number of nodes in cloud state", nil, nil),
		dynamicFutureNodes:       prometheus.NewDesc("slurm_nodes_dynamic_future", "Number of nodes in dynamic future state", nil, nil),
		dynamicNormNodes:         prometheus.NewDesc("slurm_nodes_dynamic_norm", "Number of nodes in dynamic norm state", nil, nil),
		failNodes:                prometheus.NewDesc("slurm_nodes_fail", "Number of nodes in fail state", nil, nil),
		futureNodes:              prometheus.NewDesc("slurm_nodes_future", "Number of nodes in future state", nil, nil),
		invalidNodes:             prometheus.NewDesc("slurm_nodes_invalid", "Number of nodes in invalid state", nil, nil),
		invalidRegNodes:          prometheus.NewDesc("slurm_nodes_invalid_reg", "Number of nodes in invalid reg state", nil, nil),
		notRespondingNodes:       prometheus.NewDesc("slurm_nodes_not_responding", "Number of nodes in not responding state", nil, nil),
		plannedNodes:             prometheus.NewDesc("slurm_nodes_planned", "Number of nodes in planned state", nil, nil),
		powerDownNodes:           prometheus.NewDesc("slurm_nodes_power_down", "Number of nodes in power down state", nil, nil),
		powerDrainNodes:          prometheus.NewDesc("slurm_nodes_power_drain", "Number of nodes in power drain state", nil, nil),
		poweredDownNodes:         prometheus.NewDesc("slurm_nodes_powered_down", "Number of nodes in powered down state", nil, nil),
		poweringDownNodes:        prometheus.NewDesc("slurm_nodes_powering_down", "Number of nodes in powering down state", nil, nil),
		poweringUpNodes:          prometheus.NewDesc("slurm_nodes_powering_up", "Number of nodes in powering up state", nil, nil),
		powerUpNodes:             prometheus.NewDesc("slurm_nodes_power_up", "Number of nodes in power up state", nil, nil),
		rebootCanceledNodes:      prometheus.NewDesc("slurm_nodes_reboot_canceled", "Number of nodes in reboot canceled state", nil, nil),
		rebootIssuedNodes:        prometheus.NewDesc("slurm_nodes_reboot_issued", "Number of nodes in reboot issued state", nil, nil),
		rebootRequestedNodes:     prometheus.NewDesc("slurm_nodes_reboot_requested", "Number of nodes in reboot requested state", nil, nil),
		resumeNodes:              prometheus.NewDesc("slurm_nodes_resume", "Number of nodes in resume state", nil, nil),
		undrainNodes:             prometheus.NewDesc("slurm_nodes_undrain", "Number of nodes in undrain state", nil, nil),
		unknownNodes:             prometheus.NewDesc("slurm_nodes_unknown", "Number of nodes in unknown state", nil, nil),
		// Node States
		nodeStateCombined:        prometheus.NewDesc("slurm_state_combined", "Combined Slurm State", combinedStateLabel, nil),
		nodeStateCloud:           prometheus.NewDesc("slurm_state_cloud", "The cloud state of the node", nodeLabels, nil),
		nodeStateDynamicFuture:   prometheus.NewDesc("slurm_state_dynamic_future", "The dynamic future state of the node", nodeLabels, nil),
		nodeStateDynamicNorm:     prometheus.NewDesc("slurm_state_dynamic_norm", "The dynamic norm state of the node", nodeLabels, nil),
		nodeStateFail:            prometheus.NewDesc("slurm_state_fail", "The fail state of the node", nodeLabels, nil),
		nodeStateFuture:          prometheus.NewDesc("slurm_state_future", "The future state of the node", nodeLabels, nil),
		nodeStateInvalid:         prometheus.NewDesc("slurm_state_invalid", "The invalid state of the node", nodeLabels, nil),
		nodeStateInvalidReg:      prometheus.NewDesc("slurm_state_invalid_reg", "The invalid reg state of the node", nodeLabels, nil),
		nodeStateNotResponding:   prometheus.NewDesc("slurm_state_not_responding", "The not responding state of the node", nodeLabels, nil),
		nodeStatePlanned:         prometheus.NewDesc("slurm_state_planned", "The planned state of the node", nodeLabels, nil),
		nodeStatePowerDown:       prometheus.NewDesc("slurm_state_power_down", "The power down state of the node", nodeLabels, nil),
		nodeStatePowerDrain:      prometheus.NewDesc("slurm_state_power_drain", "The power drain state of the node", nodeLabels, nil),
		nodeStatePoweredDown:     prometheus.NewDesc("slurm_state_powered_down", "The powered down state of the node", nodeLabels, nil),
		nodeStatePoweringDown:    prometheus.NewDesc("slurm_state_powering_down", "The powering down state of the node", nodeLabels, nil),
		nodeStatePoweringUp:      prometheus.NewDesc("slurm_state_powering_up", "The powering up state of the node", nodeLabels, nil),
		nodeStatePowerUp:         prometheus.NewDesc("slurm_state_power_up", "The power up state of the node", nodeLabels, nil),
		nodeStateRebootCanceled:  prometheus.NewDesc("slurm_state_reboot_canceled", "The reboot canceled state of the node", nodeLabels, nil),
		nodeStateRebootIssued:    prometheus.NewDesc("slurm_state_reboot_issued", "The reboot issued state of the node", nodeLabels, nil),
		nodeStateRebootRequested: prometheus.NewDesc("slurm_state_reboot_requested", "The reboot requested state of the node", nodeLabels, nil),
		nodeStateResume:          prometheus.NewDesc("slurm_state_resume", "The resume state of the node", nodeLabels, nil),
		nodeStateUndrain:         prometheus.NewDesc("slurm_state_undrain", "The undrain state of the node", nodeLabels, nil),
		nodeStateUnknown:         prometheus.NewDesc("slurm_state_unknown", "The unknown state of the node", nodeLabels, nil),
		nodeStateAlloc:           prometheus.NewDesc("slurm_state_allocated", "The allocated state of the node", nodeLabels, nil),
		nodeStateCompleting:      prometheus.NewDesc("slurm_state_completing", "The completing state of the node", nodeLabels, nil),
		nodeStateDown:            prometheus.NewDesc("slurm_state_down", "The down state of the node", nodeReasonLabel, nil),
		nodeStateDrain:           prometheus.NewDesc("slurm_state_drain", "The drain state of the node", nodeReasonLabel, nil),
		nodeStateError:           prometheus.NewDesc("slurm_state_error", "The error state of the node", nodeLabels, nil),
		nodeStateIdle:            prometheus.NewDesc("slurm_state_idle", "The idle state of the node", nodeLabels, nil),
		nodeStateMaintanance:     prometheus.NewDesc("slurm_state_maintenance", "The maintenance state of the node", nodeLabels, nil),
		nodeStateMixed:           prometheus.NewDesc("slurm_state_mixed", "The mixed state of the node", nodeLabels, nil),
		nodeStateReserved:        prometheus.NewDesc("slurm_state_reserved", "The reserved state of the node", nodeLabels, nil),
		// User
		userJobTotal:             prometheus.NewDesc("slurm_user_job_total", "Number of jobs for a slurm user", jobLabel, nil),
		userCPUsTotal:            prometheus.NewDesc("slurm_user_cpus_total", "Number of cpus for a slurm user", jobLabel, nil),
		userPendingJobs:          prometheus.NewDesc("slurm_user_pending_jobs", "Number of pending jobs for a slurm user", jobLabel, nil),
		userRunningJobs:          prometheus.NewDesc("slurm_user_running_jobs", "Number of running jobs for a slurm user", jobLabel, nil),
		userHoldJobs:             prometheus.NewDesc("slurm_user_hold_jobs", "Number of hold jobs for a slurm user", jobLabel, nil),
		// Job
		jobRunning:               prometheus.NewDesc("slurm_job_running", "Job State Running", jobIDLabel, nil),
		jobPending:               prometheus.NewDesc("slurm_job_pending", "Job State Pending", jobIDLabel, nil),
		jobHold:                  prometheus.NewDesc("slurm_job_hold", "Job State Hold", jobIDLabel, nil),
		jobCompleting:            prometheus.NewDesc("slurm_job_completing", "Job State Completing", jobIDLabel, nil),
		jobCPUAllocation:         prometheus.NewDesc("slurm_job_cpu_allocation", "Job CPU Allocation", jobCPUAllocationLabel, nil),
	}
}

// Send all of the possible descriptions of the metrics to be collected by the Collector
// https://pkg.go.dev/github.com/prometheus/client_golang/prometheus#Collector
func (s *SlurmCollector) Describe(ch chan<- *prometheus.Desc) {
	// Partitions
	ch <- s.partitionNodes
	ch <- s.partitionTotalCpus
	ch <- s.partitionIdleCpus
	ch <- s.partitionAllocCpus
	ch <- s.partitionJobs
	ch <- s.partitionPendingJobs
	ch <- s.partitionMaxPendingNodes
	ch <- s.partitionRunningJobs
	ch <- s.partitionHoldJobs
	// Node CPUs
	ch <- s.nodeTotalCpus
	ch <- s.nodeIdleCpus
	ch <- s.nodeAllocCpus
	// Node State Counts
	ch <- s.totalNodes
	ch <- s.allocNodes
	ch <- s.completingNodes
	ch <- s.downNodes
	ch <- s.drainNodes
	ch <- s.errNodes
	ch <- s.idleNodes
	ch <- s.maintenanceNodes
	ch <- s.mixedNodes
	ch <- s.reservedNodes
	ch <- s.cloudNodes
	ch <- s.dynamicFutureNodes
	ch <- s.dynamicNormNodes
	ch <- s.failNodes
	ch <- s.futureNodes
	ch <- s.invalidNodes
	ch <- s.invalidRegNodes
	ch <- s.notRespondingNodes
	ch <- s.plannedNodes
	ch <- s.powerDownNodes
	ch <- s.powerDrainNodes
	ch <- s.poweredDownNodes
	ch <- s.poweringDownNodes
	ch <- s.poweringUpNodes
	ch <- s.powerUpNodes
	ch <- s.rebootCanceledNodes
	ch <- s.rebootIssuedNodes
	ch <- s.rebootRequestedNodes
	ch <- s.resumeNodes
	ch <- s.undrainNodes
	ch <- s.unknownNodes
	// Node States
	ch <- s.nodeStateCombined
	ch <- s.nodeStateCloud
	ch <- s.nodeStateDynamicFuture
	ch <- s.nodeStateDynamicNorm
	ch <- s.nodeStateFail
	ch <- s.nodeStateFuture
	ch <- s.nodeStateInvalid
	ch <- s.nodeStateInvalidReg
	ch <- s.nodeStateNotResponding
	ch <- s.nodeStatePlanned
	ch <- s.nodeStatePowerDown
	ch <- s.nodeStatePowerDrain
	ch <- s.nodeStatePoweredDown
	ch <- s.nodeStatePoweringDown
	ch <- s.nodeStatePoweringUp
	ch <- s.nodeStatePowerUp
	ch <- s.nodeStateRebootCanceled
	ch <- s.nodeStateRebootIssued
	ch <- s.nodeStateRebootRequested
	ch <- s.nodeStateResume
	ch <- s.nodeStateUndrain
	ch <- s.nodeStateUnknown
	ch <- s.nodeStateIdle
	ch <- s.nodeStateAlloc
	ch <- s.nodeStateCompleting
	ch <- s.nodeStateDown
	ch <- s.nodeStateDrain
	ch <- s.nodeStateError
	ch <- s.nodeStateMaintanance
	ch <- s.nodeStateMixed
	ch <- s.nodeStateReserved
	// User
	ch <- s.userJobTotal
	ch <- s.userCPUsTotal
	ch <- s.userRunningJobs
	ch <- s.userPendingJobs
	ch <- s.userHoldJobs
	// Job
	ch <- s.jobRunning
	ch <- s.jobPending
	ch <- s.jobHold
	ch <- s.jobCompleting
	ch <- s.jobCPUAllocation
}

// Called by the Prometheus registry when collecting metrics.
// https://pkg.go.dev/github.com/prometheus/client_golang/prometheus#Collector
func (s *SlurmCollector) Collect(ch chan<- prometheus.Metric) {
	ctx := context.Background()
	log := log.FromContext(ctx)
	log.Info("Collecting slurm data.")

	// Read slurm information from cache
	nodes := &slurmtypes.V0041NodeList{}
	if err := s.slurmCollectType(nodes); err != nil {
		log.Error(err, "Could not list nodes")
		return
	}
	jobs := &slurmtypes.V0041JobInfoList{}
	if err := s.slurmCollectType(jobs); err != nil {
		log.Error(err, "Could not list jobs")
		return
	}
	partitions := &slurmtypes.V0041PartitionInfoList{}
	if err := s.slurmCollectType(partitions); err != nil {
		log.Error(err, "Could not list partitions")
		return
	}

	slurmData := s.slurmParse(jobs, nodes, partitions)

	// Partitions
	for p := range slurmData.partitions {
		ch <- prometheus.MustNewConstMetric(s.partitionNodes, prometheus.GaugeValue, float64(slurmData.partitions[p].Nodes), p)
		ch <- prometheus.MustNewConstMetric(s.partitionTotalCpus, prometheus.GaugeValue, float64(slurmData.partitions[p].Cpus), p)
		ch <- prometheus.MustNewConstMetric(s.partitionIdleCpus, prometheus.GaugeValue, float64(slurmData.partitions[p].Idle), p)
		ch <- prometheus.MustNewConstMetric(s.partitionAllocCpus, prometheus.GaugeValue, float64(slurmData.partitions[p].Alloc), p)
		ch <- prometheus.MustNewConstMetric(s.partitionJobs, prometheus.GaugeValue, float64(slurmData.partitions[p].Jobs), p)
		ch <- prometheus.MustNewConstMetric(s.partitionPendingJobs, prometheus.GaugeValue, float64(slurmData.partitions[p].PendingJobs), p)
		ch <- prometheus.MustNewConstMetric(s.partitionMaxPendingNodes, prometheus.GaugeValue, float64(slurmData.partitions[p].PendingMaxNodes), p)
		ch <- prometheus.MustNewConstMetric(s.partitionRunningJobs, prometheus.GaugeValue, float64(slurmData.partitions[p].RunningJobs), p)
		ch <- prometheus.MustNewConstMetric(s.partitionHoldJobs, prometheus.GaugeValue, float64(slurmData.partitions[p].HoldJobs), p)
	}

	// States
	ch <- prometheus.MustNewConstMetric(s.totalNodes, prometheus.GaugeValue, float64(len(slurmData.nodes)))
	ch <- prometheus.MustNewConstMetric(s.allocNodes, prometheus.GaugeValue, float64(slurmData.nodestates.allocated))
	ch <- prometheus.MustNewConstMetric(s.completingNodes, prometheus.GaugeValue, float64(slurmData.nodestates.completing))
	ch <- prometheus.MustNewConstMetric(s.downNodes, prometheus.GaugeValue, float64(slurmData.nodestates.down))
	ch <- prometheus.MustNewConstMetric(s.drainNodes, prometheus.GaugeValue, float64(slurmData.nodestates.drain))
	ch <- prometheus.MustNewConstMetric(s.errNodes, prometheus.GaugeValue, float64(slurmData.nodestates.err))
	ch <- prometheus.MustNewConstMetric(s.idleNodes, prometheus.GaugeValue, float64(slurmData.nodestates.idle))
	ch <- prometheus.MustNewConstMetric(s.maintenanceNodes, prometheus.GaugeValue, float64(slurmData.nodestates.maintenance))
	ch <- prometheus.MustNewConstMetric(s.mixedNodes, prometheus.GaugeValue, float64(slurmData.nodestates.mixed))
	ch <- prometheus.MustNewConstMetric(s.reservedNodes, prometheus.GaugeValue, float64(slurmData.nodestates.reserved))
	ch <- prometheus.MustNewConstMetric(s.cloudNodes, prometheus.GaugeValue, float64(slurmData.nodestates.cloud))
	ch <- prometheus.MustNewConstMetric(s.dynamicFutureNodes, prometheus.GaugeValue, float64(slurmData.nodestates.dynamicFuture))
	ch <- prometheus.MustNewConstMetric(s.dynamicNormNodes, prometheus.GaugeValue, float64(slurmData.nodestates.dynamicNorm))
	ch <- prometheus.MustNewConstMetric(s.failNodes, prometheus.GaugeValue, float64(slurmData.nodestates.fail))
	ch <- prometheus.MustNewConstMetric(s.futureNodes, prometheus.GaugeValue, float64(slurmData.nodestates.future))
	ch <- prometheus.MustNewConstMetric(s.invalidNodes, prometheus.GaugeValue, float64(slurmData.nodestates.invalid))
	ch <- prometheus.MustNewConstMetric(s.invalidRegNodes, prometheus.GaugeValue, float64(slurmData.nodestates.invalidReg))
	ch <- prometheus.MustNewConstMetric(s.notRespondingNodes, prometheus.GaugeValue, float64(slurmData.nodestates.notResponding))
	ch <- prometheus.MustNewConstMetric(s.plannedNodes, prometheus.GaugeValue, float64(slurmData.nodestates.planned))
	ch <- prometheus.MustNewConstMetric(s.powerDownNodes, prometheus.GaugeValue, float64(slurmData.nodestates.powerDown))
	ch <- prometheus.MustNewConstMetric(s.powerDrainNodes, prometheus.GaugeValue, float64(slurmData.nodestates.powerDrain))
	ch <- prometheus.MustNewConstMetric(s.poweredDownNodes, prometheus.GaugeValue, float64(slurmData.nodestates.poweredDown))
	ch <- prometheus.MustNewConstMetric(s.poweringDownNodes, prometheus.GaugeValue, float64(slurmData.nodestates.poweringDown))
	ch <- prometheus.MustNewConstMetric(s.poweringUpNodes, prometheus.GaugeValue, float64(slurmData.nodestates.poweringUp))
	ch <- prometheus.MustNewConstMetric(s.powerUpNodes, prometheus.GaugeValue, float64(slurmData.nodestates.powerUp))
	ch <- prometheus.MustNewConstMetric(s.rebootCanceledNodes, prometheus.GaugeValue, float64(slurmData.nodestates.rebootCanceled))
	ch <- prometheus.MustNewConstMetric(s.rebootIssuedNodes, prometheus.GaugeValue, float64(slurmData.nodestates.rebootIssued))
	ch <- prometheus.MustNewConstMetric(s.rebootRequestedNodes, prometheus.GaugeValue, float64(slurmData.nodestates.rebootRequested))
	ch <- prometheus.MustNewConstMetric(s.resumeNodes, prometheus.GaugeValue, float64(slurmData.nodestates.resume))
	ch <- prometheus.MustNewConstMetric(s.undrainNodes, prometheus.GaugeValue, float64(slurmData.nodestates.undrain))
	ch <- prometheus.MustNewConstMetric(s.unknownNodes, prometheus.GaugeValue, float64(slurmData.nodestates.unknown))

	for n := range slurmData.nodes {
		// Node CPUs
		ch <- prometheus.MustNewConstMetric(s.nodeTotalCpus, prometheus.GaugeValue, float64(slurmData.nodes[n].Cpus), n, n)
		ch <- prometheus.MustNewConstMetric(s.nodeIdleCpus, prometheus.GaugeValue, float64(slurmData.nodes[n].Idle), n, n)
		ch <- prometheus.MustNewConstMetric(s.nodeAllocCpus, prometheus.GaugeValue, float64(slurmData.nodes[n].Alloc), n, n)
		// States 2
		ch <- prometheus.MustNewConstMetric(s.nodeStateCombined, prometheus.GaugeValue, float64(slurmData.nodes[n].unavailable), n, n, slurmData.nodes[n].CombinedState)
		ch <- prometheus.MustNewConstMetric(s.nodeStateIdle, prometheus.GaugeValue, float64(slurmData.nodes[n].States.idle), n, n)
		ch <- prometheus.MustNewConstMetric(s.nodeStateAlloc, prometheus.GaugeValue, float64(slurmData.nodes[n].States.allocated), n, n)
		ch <- prometheus.MustNewConstMetric(s.nodeStateCompleting, prometheus.GaugeValue, float64(slurmData.nodes[n].States.completing), n, n)
		ch <- prometheus.MustNewConstMetric(s.nodeStateDown, prometheus.GaugeValue, float64(slurmData.nodes[n].States.down), n, n, slurmData.nodes[n].Reason, slurmData.nodes[n].ReasonSetByUser, strconv.FormatInt(slurmData.nodes[n].ReasonChangedAt, 10))
		ch <- prometheus.MustNewConstMetric(s.nodeStateDrain, prometheus.GaugeValue, float64(slurmData.nodes[n].States.drain), n, n, slurmData.nodes[n].Reason, slurmData.nodes[n].ReasonSetByUser, strconv.FormatInt(slurmData.nodes[n].ReasonChangedAt, 10))
		ch <- prometheus.MustNewConstMetric(s.nodeStateError, prometheus.GaugeValue, float64(slurmData.nodes[n].States.err), n, n)
		ch <- prometheus.MustNewConstMetric(s.nodeStateMaintanance, prometheus.GaugeValue, float64(slurmData.nodes[n].States.maintenance), n, n)
		ch <- prometheus.MustNewConstMetric(s.nodeStateMixed, prometheus.GaugeValue, float64(slurmData.nodes[n].States.mixed), n, n)
		ch <- prometheus.MustNewConstMetric(s.nodeStateReserved, prometheus.GaugeValue, float64(slurmData.nodes[n].States.reserved), n, n)
		ch <- prometheus.MustNewConstMetric(s.nodeStateCloud, prometheus.GaugeValue, float64(slurmData.nodes[n].States.cloud), n, n)
		ch <- prometheus.MustNewConstMetric(s.nodeStateDynamicFuture, prometheus.GaugeValue, float64(slurmData.nodes[n].States.dynamicFuture), n, n)
		ch <- prometheus.MustNewConstMetric(s.nodeStateDynamicNorm, prometheus.GaugeValue, float64(slurmData.nodes[n].States.dynamicNorm), n, n)
		ch <- prometheus.MustNewConstMetric(s.nodeStateFail, prometheus.GaugeValue, float64(slurmData.nodes[n].States.fail), n, n)
		ch <- prometheus.MustNewConstMetric(s.nodeStateFuture, prometheus.GaugeValue, float64(slurmData.nodes[n].States.future), n, n)
		ch <- prometheus.MustNewConstMetric(s.nodeStateInvalid, prometheus.GaugeValue, float64(slurmData.nodes[n].States.invalid), n, n)
		ch <- prometheus.MustNewConstMetric(s.nodeStateInvalidReg, prometheus.GaugeValue, float64(slurmData.nodes[n].States.invalidReg), n, n)
		ch <- prometheus.MustNewConstMetric(s.nodeStateNotResponding, prometheus.GaugeValue, float64(slurmData.nodes[n].States.notResponding), n, n)
		ch <- prometheus.MustNewConstMetric(s.nodeStatePlanned, prometheus.GaugeValue, float64(slurmData.nodes[n].States.planned), n, n)
		ch <- prometheus.MustNewConstMetric(s.nodeStatePowerDown, prometheus.GaugeValue, float64(slurmData.nodes[n].States.powerDown), n, n)
		ch <- prometheus.MustNewConstMetric(s.nodeStatePowerDrain, prometheus.GaugeValue, float64(slurmData.nodes[n].States.powerDrain), n, n)
		ch <- prometheus.MustNewConstMetric(s.nodeStatePoweredDown, prometheus.GaugeValue, float64(slurmData.nodes[n].States.poweredDown), n, n)
		ch <- prometheus.MustNewConstMetric(s.nodeStatePoweringDown, prometheus.GaugeValue, float64(slurmData.nodes[n].States.poweringDown), n, n)
		ch <- prometheus.MustNewConstMetric(s.nodeStatePoweringUp, prometheus.GaugeValue, float64(slurmData.nodes[n].States.poweringUp), n, n)
		ch <- prometheus.MustNewConstMetric(s.nodeStatePowerUp, prometheus.GaugeValue, float64(slurmData.nodes[n].States.powerUp), n, n)
		ch <- prometheus.MustNewConstMetric(s.nodeStateRebootCanceled, prometheus.GaugeValue, float64(slurmData.nodes[n].States.rebootCanceled), n, n)
		ch <- prometheus.MustNewConstMetric(s.nodeStateRebootIssued, prometheus.GaugeValue, float64(slurmData.nodes[n].States.rebootIssued), n, n)
		ch <- prometheus.MustNewConstMetric(s.nodeStateRebootRequested, prometheus.GaugeValue, float64(slurmData.nodes[n].States.rebootRequested), n, n)
		ch <- prometheus.MustNewConstMetric(s.nodeStateResume, prometheus.GaugeValue, float64(slurmData.nodes[n].States.resume), n, n)
		ch <- prometheus.MustNewConstMetric(s.nodeStateUndrain, prometheus.GaugeValue, float64(slurmData.nodes[n].States.undrain), n, n)
		ch <- prometheus.MustNewConstMetric(s.nodeStateUnknown, prometheus.GaugeValue, float64(slurmData.nodes[n].States.unknown), n, n)
	}

	// User
	for j := range slurmData.jobs {
		ch <- prometheus.MustNewConstMetric(s.userCPUsTotal, prometheus.GaugeValue, float64(slurmData.jobs[j].Cpus), j)
		ch <- prometheus.MustNewConstMetric(s.userJobTotal, prometheus.GaugeValue, float64(slurmData.jobs[j].Count), j)
		ch <- prometheus.MustNewConstMetric(s.userPendingJobs, prometheus.GaugeValue, float64(slurmData.jobs[j].Pending), j)
		ch <- prometheus.MustNewConstMetric(s.userRunningJobs, prometheus.GaugeValue, float64(slurmData.jobs[j].Running), j)
		ch <- prometheus.MustNewConstMetric(s.userHoldJobs, prometheus.GaugeValue, float64(slurmData.jobs[j].Hold), j)
	}

	// Job
	for j := range slurmData.jobstates {
		ch <- prometheus.MustNewConstMetric(s.jobRunning, prometheus.GaugeValue, float64(slurmData.jobstates[j].running), strconv.Itoa(int(j)), slurmData.jobstates[j].nodes)
		ch <- prometheus.MustNewConstMetric(s.jobPending, prometheus.GaugeValue, float64(slurmData.jobstates[j].pending), strconv.Itoa(int(j)), slurmData.jobstates[j].nodes)
		ch <- prometheus.MustNewConstMetric(s.jobHold, prometheus.GaugeValue, float64(slurmData.jobstates[j].hold), strconv.Itoa(int(j)), slurmData.jobstates[j].nodes)
		ch <- prometheus.MustNewConstMetric(s.jobCompleting, prometheus.GaugeValue, float64(slurmData.jobstates[j].completing), strconv.Itoa(int(j)), slurmData.jobstates[j].nodes)

		if slurmData.jobstates[j].allocation != nil {
			for _, allocation := range slurmData.jobstates[j].allocation {
				for _, socket := range allocation.Sockets {
					for _, core := range socket.Cores {
						var alloc int
						for _, status := range core.Status {
							if status == "ALLOCATED" {
								alloc = 1
							}
						}
						ch <- prometheus.MustNewConstMetric(s.jobCPUAllocation, prometheus.GaugeValue, float64(alloc), strconv.Itoa(int(j)), strconv.Itoa(int(core.Index)), strconv.Itoa(int(socket.Index)), allocation.Name)
					}
				}
			}
		}
	}
}
