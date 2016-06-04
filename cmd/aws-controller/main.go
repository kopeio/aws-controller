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
	"github.com/spf13/pflag"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/kopeio/aws-controller/pkg/awscontroller/instances"
)

const (
	healthPort = 10249
)

var (
	// value overwritten during build. This can be used to resolve issues.
	version = "0.5"
	gitRepo = "https://github.com/kopeio/aws-controller"

	flags = pflag.NewFlagSet("", pflag.ExitOnError)

	resyncPeriod = flags.Duration("sync-period", 30*time.Second,
		`Relist and confirm cloud resources this often.`)

	healthzPort = flags.Int("healthz-port", healthPort, "port for healthz endpoint.")

	//kubeConfig = flags.String("kubeconfig", "", "Path to kubeconfig file with authorization information.")

	//nodeName       = flags.String("node-name", "", "name of this node")
	flagClusterID  = flags.String("cluster-id", "", "cluster id")
	//systemUUIDPath = flags.String("system-uuid", "", "path to file containing system-uuid (as set in node status)")
	//bootIDPath     = flags.String("boot-id", "", "path to file containing boot-id (as set in node status)")
	//providerID     = flags.String("provider", "gre", "route backend to use")

	// I can't figure out how to get a serviceaccount in a manifest-controlled pod
	//inCluster = flags.Bool("running-in-cluster", true,
	//	`Optional, if this controller is running in a kubernetes cluster, use the
	//	 pod secrets for creating a Kubernetes client.`)

	profiling = flags.Bool("profiling", true, `Enable profiling via web interface host:port/debug/pprof/`)
)

// The tag name we use to differentiate multiple logically independent clusters running in the same AZ
const TagNameKubernetesCluster = "KubernetesCluster"


func main() {
	flags.AddGoFlagSet(flag.CommandLine)

	flags.Parse(os.Args)

	glog.Infof("Using build: %v - %v", gitRepo, version)

	clusterID := *flagClusterID
	if clusterID == "" {
		glog.Fatalf("cluster-id flag must be set")
	}

	tags := make(map[string]string)
	tags[TagNameKubernetesCluster] = clusterID

	creds := credentials.NewChainCredentials(
		[]credentials.Provider{
			&credentials.EnvProvider{},
			&ec2rolecreds.EC2RoleProvider{
				Client: ec2metadata.New(session.New(&aws.Config{})),
			},
			&credentials.SharedCredentialsProvider{},
		})

	ec2 := ec2.New(session.New(&aws.Config{
		//Region:      &regionName,
		Credentials: creds,
	}))

	c := instances.NewInstancesController(ec2, tags)

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
