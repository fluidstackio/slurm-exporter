// SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
// SPDX-License-Identifier: Apache-2.0

package collector

var (
	accountLabels = []string{"account"}

	userLabels = []string{"user_id", "user_name"}

	nodeLabels = []string{"node"}

	nodeReasonLabels = []string{"node", "reason", "user"}

	partitionLabels = []string{"partition"}

	combinedStateLabels = []string{"node", "combined_state", "reason", "user"}

	jobLabels = []string{"job_id", "job_name", "node", "account", "partition", "user_id", "user_name"}

	jobLabelsWithState = []string{"job_id", "job_name", "node", "account", "partition", "user_id", "user_name", "state", "hold"}

	jobLabelsWithFlag = []string{"job_id", "job_name", "node", "account", "partition", "user_id", "user_name", "flag"}

	rpcUserLabels = []string{"user_id", "user_name"}
)
