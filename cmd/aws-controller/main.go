/*
Copyright 2015 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang/glog"

	"github.com/kopeio/aws-controller/pkg/awscontroller/instances"
	"github.com/kopeio/aws-controller/pkg/kope"
	"github.com/kopeio/aws-controller/pkg/kope/kopeaws"
)

const (
	healthPort = 10245
)

var (
	// value overwritten during build. This can be used to resolve issues.
	version = "0.5"
	gitRepo = "https://github.com/kopeio/aws-controller"

	//flags = pflag.NewFlagSet("", pflag.ExitOnError)

	//resyncPeriod = flags.Duration("sync-period", 30*time.Second,
	//	`Relist and confirm cloud resources this often.`)

	resyncPeriod = 30 * time.Second

	healthzPort = flag.Int("healthz-port", healthPort, "port for healthz endpoint.")

	//kubeConfig = flags.String("kubeconfig", "", "Path to kubeconfig file with authorization information.")

	//nodeName       = flags.String("node-name", "", "name of this node")
	flagZoneName = flag.String("zone-name", "", "DNS zone name to use (if managing DNS)")
	flagClusterID = flag.String("cluster-id", "", "cluster id")
	//systemUUIDPath = flags.String("system-uuid", "", "path to file containing system-uuid (as set in node status)")
	//bootIDPath     = flags.String("boot-id", "", "path to file containing boot-id (as set in node status)")
	//providerID     = flags.String("provider", "gre", "route backend to use")

	// I can't figure out how to get a serviceaccount in a manifest-controlled pod
	//inCluster = flags.Bool("running-in-cluster", true,
	//	`Optional, if this controller is running in a kubernetes cluster, use the
	//	 pod secrets for creating a Kubernetes client.`)

	profiling = flag.Bool("profiling", true, `Enable profiling via web interface host:port/debug/pprof/`)
)

func main() {
	//flags.AddGoFlagSet(flag.CommandLine)
	flag.Set("logtostderr", "true")
	flag.Parse()

	glog.Infof("Using build: %v - %v", gitRepo, version)

	cloud, err := kopeaws.NewAWSCloud()
	if err != nil {
		glog.Fatalf("error building cloud: %v", err)
	}

	clusterID := *flagClusterID
	if clusterID == "" && cloud != nil {
		clusterID = cloud.ClusterID()
	}
	if clusterID == "" {
		glog.Fatalf("cluster-id flag must be set")
	}

	var dns kope.DNSProvider
	zoneName := *flagZoneName
	if zoneName != "" {
		dns = kopeaws.NewRoute53DNSProvider(zoneName)
	}

	c := instances.NewInstancesController(cloud, resyncPeriod, dns)

	sourceDestCheck := false
	c.SourceDestCheck = &sourceDestCheck

	go registerHandlers(c)
	go handleSigterm(c)

	c.Run()

	for {
		glog.Infof("Handled quit, awaiting pod deletion")
		time.Sleep(30 * time.Second)
	}
}

func registerHandlers(c *instances.InstancesController) {
	mux := http.NewServeMux()
	// TODO: healthz
	//healthz.InstallHandler(mux, lbc.nginx)

	http.HandleFunc("/build", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "build: %v - %v", gitRepo, version)
	})

	http.HandleFunc("/stop", func(w http.ResponseWriter, r *http.Request) {
		c.Stop()
	})

	if *profiling {
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	}

	server := &http.Server{
		Addr:    fmt.Sprintf(":%v", *healthzPort),
		Handler: mux,
	}
	glog.Fatal(server.ListenAndServe())
}

func handleSigterm(c *instances.InstancesController) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGTERM)
	<-signalChan
	glog.Infof("Received SIGTERM, shutting down")

	exitCode := 0
	if err := c.Stop(); err != nil {
		glog.Infof("Error during shutdown %v", err)
		exitCode = 1
	}
	glog.Infof("Exiting with %v", exitCode)
	os.Exit(exitCode)
}
