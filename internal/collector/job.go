// SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"context"
	"fmt"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/log"

	api "github.com/SlinkyProject/slurm-client/api/v0041"
	"github.com/SlinkyProject/slurm-client/pkg/client"
	"github.com/SlinkyProject/slurm-client/pkg/types"
)

// Ref: https://prometheus.io/docs/practices/naming/#metric-names
func NewJobCollector(slurmClient client.Client) prometheus.Collector {
	return &jobCollector{
		slurmClient: slurmClient,

		// New unified state metric with state and hold labels
		JobState: prometheus.NewDesc("slurm_job_state", "The base state of the job", jobLabelsWithState, nil),
		// New flag metric with concatenated flags
		JobFlag: prometheus.NewDesc("slurm_job_flag", "The flag states of the job", jobLabelsWithFlag, nil),
		// Tres
		JobTres: jobTresCollector{
			// CPUs
			CpusAlloc: prometheus.NewDesc("slurm_job_cpus_alloc_total", "Number of allocated CPUs for the job", jobLabels, nil),
			// Memory
			MemoryAlloc: prometheus.NewDesc("slurm_job_memory_alloc_total", "Amount of allocated memory for the job in bytes", jobLabels, nil),
			// GPUs
			GpusAlloc: prometheus.NewDesc("slurm_job_gpus_alloc_total", "Number of allocated GPUs for the job", jobLabels, nil),
		},
	}
}

type jobCollector struct {
	slurmClient client.Client

	// New unified state metric with state and hold labels
	JobState *prometheus.Desc

	// New flag metric with concatenated flags
	JobFlag *prometheus.Desc

	// Individual Job Tres Metrics ---------------------------------------------
	JobTres jobTresCollector
}


type jobTresCollector struct {
	// CPUs
	CpusAlloc *prometheus.Desc
	// Memory
	MemoryAlloc *prometheus.Desc
	// GPUs
	GpusAlloc *prometheus.Desc
}

func (c *jobCollector) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(c, ch)
}

func (c *jobCollector) Collect(ch chan<- prometheus.Metric) {
	ctx := context.TODO()
	logger := log.FromContext(ctx).WithName("JobCollector")

	logger.V(1).Info("collecting metrics")

	metrics, err := c.getJobMetrics(ctx)
	if err != nil {
		logger.Error(err, "failed to collect job metrics")
		return
	}


	for _, jobState := range metrics.JobIndividualStates {
		jobID := jobState.JobID
		jobName := jobState.JobName
		account := jobState.Account
		partition := jobState.Partition
		userID := jobState.UserID
		userName := jobState.UserName
		for _, node := range jobState.Nodes {
			// New unified state metric with state and hold labels --------------
			// Determine the base state name
			var stateName string
			// Check base states (these are mutually exclusive)
			switch {
			case jobState.BootFail == 1:
				stateName = "bootfail"
			case jobState.Cancelled == 1:
				stateName = "cancelled"
			case jobState.Completed == 1:
				stateName = "completed"
			case jobState.Deadline == 1:
				stateName = "deadline"
			case jobState.Failed == 1:
				stateName = "failed"
			case jobState.Pending == 1:
				stateName = "pending"
			case jobState.Preempted == 1:
				stateName = "preempted"
			case jobState.Running == 1:
				stateName = "running"
			case jobState.Suspended == 1:
				stateName = "suspended"
			case jobState.Timeout == 1:
				stateName = "timeout"
			case jobState.NodeFail == 1:
				stateName = "nodefail"
			case jobState.OutOfMemory == 1:
				stateName = "outofmemory"
			default:
				stateName = "unknown"
			}
			
			// Determine hold status
			holdStatus := "false"
			if jobState.Hold == 1 {
				holdStatus = "true"
			}
			
			ch <- prometheus.MustNewConstMetric(c.JobState, prometheus.GaugeValue, 1, jobID, jobName, node, account, partition, userID, userName, stateName, holdStatus)
			
			// New flag metric with concatenated flags --------------------------
			var flags []string
			if jobState.Completing == 1 {
				flags = append(flags, "completing")
			}
			if jobState.Configuring == 1 {
				flags = append(flags, "configuring")
			}
			if jobState.PowerUpNode == 1 {
				flags = append(flags, "powerupnode")
			}
			if jobState.StageOut == 1 {
				flags = append(flags, "stageout")
			}
			
			// Only emit flag metric if there are flags
			if len(flags) > 0 {
				flagStr := strings.Join(flags, "+")
				ch <- prometheus.MustNewConstMetric(c.JobFlag, prometheus.GaugeValue, 1, jobID, jobName, node, account, partition, userID, userName, flagStr)
			}

			// Individual Job Tres Metrics -------------------------------------
			ch <- prometheus.MustNewConstMetric(c.JobTres.CpusAlloc, prometheus.GaugeValue, float64(jobState.CpusAlloc), jobID, jobName, node, account, partition, userID, userName)
			ch <- prometheus.MustNewConstMetric(c.JobTres.MemoryAlloc, prometheus.GaugeValue, float64(jobState.MemoryAlloc), jobID, jobName, node, account, partition, userID, userName)
			ch <- prometheus.MustNewConstMetric(c.JobTres.GpusAlloc, prometheus.GaugeValue, float64(jobState.GpusAlloc), jobID, jobName, node, account, partition, userID, userName)
		}
	}
}

func (c *jobCollector) getJobMetrics(ctx context.Context) (*JobMetrics, error) {
	jobList := &types.V0041JobInfoList{}
	if err := c.slurmClient.List(ctx, jobList); err != nil {
		return nil, err
	}
	metrics := calculateJobMetrics(jobList)
	return metrics, nil
}

func calculateJobMetrics(jobList *types.V0041JobInfoList) *JobMetrics {
	// Collective Metrics
	metrics := &JobMetrics{
		JobCount:            uint(len(jobList.Items)),
		JobIndividualStates: make([]JobIndividualStates, 0, len(jobList.Items)),
	}

	// Individual job metrics
	for _, job := range jobList.Items {
		calculateJobState(&metrics.JobStates, job)
		jobStates := calculateJobIndividualStates(job)
		if jobStates != nil {
			metrics.JobIndividualStates = append(metrics.JobIndividualStates, *jobStates)
		}
	}

	return metrics
}

func calculateJobState(metrics *JobStates, job types.V0041JobInfo) {
	metrics.total++
	states := job.GetStateAsSet()
	// Base States
	switch {
	case states.Has(api.V0041JobInfoJobStateBOOTFAIL):
		metrics.BootFail++
	case states.Has(api.V0041JobInfoJobStateCANCELLED):
		metrics.Cancelled++
	case states.Has(api.V0041JobInfoJobStateCOMPLETED):
		metrics.Completed++
	case states.Has(api.V0041JobInfoJobStateDEADLINE):
		metrics.Deadline++
	case states.Has(api.V0041JobInfoJobStateFAILED):
		metrics.Failed++
	case states.Has(api.V0041JobInfoJobStatePENDING):
		metrics.Pending++
	case states.Has(api.V0041JobInfoJobStatePREEMPTED):
		metrics.Preempted++
	case states.Has(api.V0041JobInfoJobStateRUNNING):
		metrics.Running++
	case states.Has(api.V0041JobInfoJobStateSUSPENDED):
		metrics.Suspended++
	case states.Has(api.V0041JobInfoJobStateTIMEOUT):
		metrics.Timeout++
	case states.Has(api.V0041JobInfoJobStateNODEFAIL):
		metrics.NodeFail++
	case states.Has(api.V0041JobInfoJobStateOUTOFMEMORY):
		metrics.OutOfMemory++
	}
	// Flag States
	if states.Has(api.V0041JobInfoJobStateCOMPLETING) {
		metrics.Completing++
	}
	if states.Has(api.V0041JobInfoJobStateCONFIGURING) {
		metrics.Configuring++
	}
	if states.Has(api.V0041JobInfoJobStatePOWERUPNODE) {
		metrics.PowerUpNode++
	}
	if states.Has(api.V0041JobInfoJobStateSTAGEOUT) {
		metrics.StageOut++
	}
	// Other States
	if isHold := ptr.Deref(job.Hold, false); isHold {
		metrics.Hold++
	}
}

// jobResources represents the allocated resources for a single job
type jobResources struct {
	Cpus   uint
	Memory uint
	Gpus   uint
}

func getJobResourceAlloc(job types.V0041JobInfo) jobResources {
	var res jobResources
	jobRes := ptr.Deref(job.JobResources, api.V0041JobRes{})
	if jobRes.Nodes != nil {
		jobResNode := ptr.Deref(jobRes.Nodes.Allocation, []api.V0041JobResNode{})
		for _, resNode := range jobResNode {
			if resNode.Cpus != nil {
				res.Cpus += uint(ptr.Deref(resNode.Cpus.Count, 0))
			}
			if resNode.Memory != nil {
				// Convert from MB to bytes
				res.Memory += uint(ptr.Deref(resNode.Memory.Allocated, 0)) * 1024 * 1024
			}
		}
	}
	// Parse GPU allocation from TresAllocStr
	if job.TresAllocStr != nil {
		res.Gpus = ParseTresGpu(ptr.Deref(job.TresAllocStr, ""))
	}
	return res
}

type JobMetrics struct {
	JobCount            uint
	JobStates           JobStates
	JobIndividualStates []JobIndividualStates
}

// Ref: https://slurm.schedmd.com/job_state_codes.html#states
// Ref: https://slurm.schedmd.com/job_state_codes.html#flags
type JobStates struct {
	total uint
	// Base States
	BootFail    uint
	Cancelled   uint
	Completed   uint
	Deadline    uint
	Failed      uint
	Pending     uint
	Preempted   uint
	Running     uint
	Suspended   uint
	Timeout     uint
	NodeFail    uint
	OutOfMemory uint
	// Flag States
	Completing  uint
	Configuring uint
	PowerUpNode uint
	StageOut    uint
	// Other States
	Hold uint
}

type jobStatesCollector struct {
	// Base States
	BootFail    *prometheus.Desc
	Cancelled   *prometheus.Desc
	Completed   *prometheus.Desc
	Deadline    *prometheus.Desc
	Failed      *prometheus.Desc
	Pending     *prometheus.Desc
	Preempted   *prometheus.Desc
	Running     *prometheus.Desc
	Suspended   *prometheus.Desc
	Timeout     *prometheus.Desc
	NodeFail    *prometheus.Desc
	OutOfMemory *prometheus.Desc
	// Flag States
	Completing  *prometheus.Desc
	Configuring *prometheus.Desc
	PowerUpNode *prometheus.Desc
	StageOut    *prometheus.Desc
	// Other States
	Hold *prometheus.Desc
}


type JobIndividualStates struct {
	JobID     string
	JobName   string
	Nodes     []string
	Account   string
	Partition string
	UserID    string
	UserName  string
	// Base States
	BootFail    int
	Cancelled   int
	Completed   int
	Deadline    int
	Failed      int
	Pending     int
	Preempted   int
	Running     int
	Suspended   int
	Timeout     int
	NodeFail    int
	OutOfMemory int
	// Flag States
	Completing  int
	Configuring int
	PowerUpNode int
	StageOut    int
	// Other States
	Hold int
	// Tres
	CpusAlloc   uint
	MemoryAlloc uint
	GpusAlloc   uint
}

func calculateJobIndividualStates(job types.V0041JobInfo) *JobIndividualStates {
	states := job.GetStateAsSet()
	jobID := fmt.Sprintf("%d", ptr.Deref(job.JobId, 0))
	jobName := ptr.Deref(job.Name, "")

	// Extract node list from job resources
	nodeList := ""
	if job.JobResources != nil && job.JobResources.Nodes != nil {
		nodeList = ptr.Deref(job.JobResources.Nodes.List, "")
	}
	nodes := parseNodeList(nodeList)

	// Even if there are no nodes, we still want to emit the metric, as it has the job_name and job_id
	if len(nodes) == 0 {
		nodes = []string{""}
	}

	jobStates := &JobIndividualStates{
		JobID:     jobID,
		JobName:   jobName,
		Nodes:     nodes,
		Account:   ptr.Deref(job.Account, ""),
		Partition: ptr.Deref(job.Partition, ""),
		UserID:    fmt.Sprintf("%d", ptr.Deref(job.UserId, 0)),
		UserName:  ptr.Deref(job.UserName, ""),
	}

	// Base States
	if states.Has(api.V0041JobInfoJobStateBOOTFAIL) {
		jobStates.BootFail = 1
	}
	if states.Has(api.V0041JobInfoJobStateCANCELLED) {
		jobStates.Cancelled = 1
	}
	if states.Has(api.V0041JobInfoJobStateCOMPLETED) {
		jobStates.Completed = 1
	}
	if states.Has(api.V0041JobInfoJobStateDEADLINE) {
		jobStates.Deadline = 1
	}
	if states.Has(api.V0041JobInfoJobStateFAILED) {
		jobStates.Failed = 1
	}
	if states.Has(api.V0041JobInfoJobStatePENDING) {
		jobStates.Pending = 1
	}
	if states.Has(api.V0041JobInfoJobStatePREEMPTED) {
		jobStates.Preempted = 1
	}
	if states.Has(api.V0041JobInfoJobStateRUNNING) {
		jobStates.Running = 1
	}
	if states.Has(api.V0041JobInfoJobStateSUSPENDED) {
		jobStates.Suspended = 1
	}
	if states.Has(api.V0041JobInfoJobStateTIMEOUT) {
		jobStates.Timeout = 1
	}
	if states.Has(api.V0041JobInfoJobStateNODEFAIL) {
		jobStates.NodeFail = 1
	}
	if states.Has(api.V0041JobInfoJobStateOUTOFMEMORY) {
		jobStates.OutOfMemory = 1
	}
	// Flag States
	if states.Has(api.V0041JobInfoJobStateCOMPLETING) {
		jobStates.Completing = 1
	}
	if states.Has(api.V0041JobInfoJobStateCONFIGURING) {
		jobStates.Configuring = 1
	}
	if states.Has(api.V0041JobInfoJobStatePOWERUPNODE) {
		jobStates.PowerUpNode = 1
	}
	if states.Has(api.V0041JobInfoJobStateSTAGEOUT) {
		jobStates.StageOut = 1
	}
	// Other States
	if isHold := ptr.Deref(job.Hold, false); isHold {
		jobStates.Hold = 1
	}

	// Get Tres allocations for this job
	res := getJobResourceAlloc(job)
	jobStates.CpusAlloc = res.Cpus
	jobStates.MemoryAlloc = res.Memory
	jobStates.GpusAlloc = res.Gpus

	return jobStates
}
