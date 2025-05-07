// SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"k8s.io/utils/ptr"

	api "github.com/SlinkyProject/slurm-client/api/v0041"
	"github.com/SlinkyProject/slurm-client/pkg/types"
)

func getJobResourceAlloc(job types.V0041JobInfo) jobResources {
	var res jobResources
	jobRes := ptr.Deref(job.JobResources, api.V0041JobRes{})
	if jobRes.Nodes == nil {
		return res
	}
	jobResNode := ptr.Deref(jobRes.Nodes.Allocation, []api.V0041JobResNode{})
	for _, resNode := range jobResNode {
		if resNode.Cpus != nil {
			res.Cpus += uint(ptr.Deref(resNode.Cpus.Count, 0))
		}
		if resNode.Memory != nil {
			res.Memory += uint(ptr.Deref(resNode.Memory.Allocated, 0))
		}
	}
	return res
}

type jobResources struct {
	Cpus   uint
	Memory uint
}

// getJobPendingNodeCount returns the requested node count if the job is
// pending, otherwise returns zero.
func getJobPendingNodeCount(job types.V0041JobInfo) uint {
	isPending := job.GetStateAsSet().Has(api.V0041JobInfoJobStatePENDING)
	isHold := ptr.Deref(job.Hold, false)
	if !isPending || isHold {
		return 0
	}
	nodeCount := ParseUint32NoVal(job.NodeCount)
	return uint(nodeCount)
}
