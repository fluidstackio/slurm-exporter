// SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"math"
	"regexp"
	"strconv"
	"strings"

	"k8s.io/utils/ptr"

	api "github.com/SlinkyProject/slurm-client/api/v0041"
)

func ParseUint64NoVal(noVal *api.V0041Uint64NoValStruct) uint64 {
	if noVal == nil {
		return 0
	}
	if isSet := ptr.Deref(noVal.Set, false); !isSet {
		return 0
	}
	if isInfinite := ptr.Deref(noVal.Infinite, false); isInfinite {
		return math.MaxUint64
	}
	number := uint64(ptr.Deref(noVal.Number, 0))
	return number
}

func ParseUint32NoVal(noVal *api.V0041Uint32NoValStruct) uint32 {
	if noVal == nil {
		return 0
	}
	if isSet := ptr.Deref(noVal.Set, false); !isSet {
		return 0
	}
	if isInfinite := ptr.Deref(noVal.Infinite, false); isInfinite {
		return math.MaxUint32
	}
	number := uint32(ptr.Deref(noVal.Number, 0))
	return number
}

var gpuGresRegex = regexp.MustCompile(`\bgpu(?::[^:,]+)?:(\d+)`)

func ParseGpuGres(s string) int32 {
	matches := gpuGresRegex.FindStringSubmatch(s)

	if len(matches) < 2 {
		return 0
	}

	num, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0
	}

	return int32(num)
}

// ParseGpuFromTres parses GPU count from a TRES allocation string
// Example: "cpu=96,mem=1612647M,node=1,billing=96,gres/gpu=8" returns 8
func ParseGpuFromTres(tresStr string) uint {
	if tresStr == "" {
		return 0
	}
	
	// Split by comma to get individual TRES components
	parts := strings.Split(tresStr, ",")
	for _, part := range parts {
		// Look for gres/gpu=N pattern
		if strings.HasPrefix(part, "gres/gpu=") {
			valueStr := strings.TrimPrefix(part, "gres/gpu=")
			if value, err := strconv.Atoi(valueStr); err == nil && value >= 0 {
				return uint(value)
			}
		}
	}
	return 0
}
