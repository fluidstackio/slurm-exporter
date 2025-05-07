// SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"errors"
	"flag"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/SlinkyProject/slurm-exporter/internal/exporter"
)

var (
	setupLog = ctrl.Log.WithName("setup")
)

// Input flags to the command
type Flags struct {
	MetricsAddr string
	Server      string
	CacheFreq   time.Duration
}

func parseFlags(flags *Flags) {
	flag.StringVar(
		&flags.MetricsAddr,
		"metrics-bind-address",
		":8080",
		"The address the metric endpoint binds to.",
	)
	flag.StringVar(
		&flags.Server,
		"server",
		"http://localhost:6820",
		"The server url of the cluster for the exporter to monitor.",
	)
	flag.DurationVar(
		&flags.CacheFreq,
		"cache-freq",
		5*time.Second,
		"The amount of time to wait between updating the slurm restapi cache. Must be greater than 1s and must be parsable by time.ParseDuration.",
	)
	flag.Parse()
}

func main() {
	var flags Flags
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	parseFlags(&flags)
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
	setupLog.Info("With", "Flags", flags)

	if flags.CacheFreq <= 1*time.Second {
		setupLog.Error(errors.New("config"), "Must use a cache-freq > 1s.")
		os.Exit(1)
	}

	slurmClient, err := exporter.NewSlurmClient(flags.Server, flags.CacheFreq)
	if err != nil {
		setupLog.Error(err, "could not create slurm client")
		os.Exit(1)
	}
	slurmCollector := exporter.NewSlurmCollector(slurmClient)
	prometheus.MustRegister(slurmCollector)

	setupLog.Info("starting exporter")
	http.Handle("/metrics", promhttp.Handler())
	if err := http.ListenAndServe(flags.MetricsAddr, nil); err != nil {
		setupLog.Error(err, "problem running exporter")
		os.Exit(1)
	}
}
