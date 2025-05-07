// SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"slices"
)

func PruneEmpty(list []string) []string {
	return slices.DeleteFunc(list, func(s string) bool { return s == "" })
}
