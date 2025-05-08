// SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"slices"
	"strings"
)

func ParseCSV(in string) []string {
	list := strings.Split(in, ",")
	return pruneEmpty(list)
}

func pruneEmpty(list []string) []string {
	return slices.DeleteFunc(list, func(s string) bool { return s == "" })
}
