// SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"testing"
	"time"

	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func Test_parseFlags(t *testing.T) {
	flags := Flags{}
	os.Args = []string{"test", "--metrics-bind-address", "8081", "--server", "foo", "--cache-freq", "10s"}
	parseFlags(&flags)
	if flags.MetricsAddr != "8081" {
		t.Errorf("Test_parseFlags() MetricsAddr = %v, want %v", flags.MetricsAddr, "8081")
	}
	if flags.Server != "foo" {
		t.Errorf("Test_parseFlags() Server = %v, want %v", flags.Server, "foo")
	}
	if flags.CacheFreq != time.Second*10 {
		t.Errorf("Test_parseFlags() CacheFreq = %v, want %v", flags.CacheFreq, time.Second*10)
	}
}
