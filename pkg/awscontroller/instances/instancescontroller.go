package instances

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/golang/glog"
	"github.com/kopeio/aws-controller/pkg/kope"
	"github.com/kopeio/aws-controller/pkg/kope/kopeaws"
	"k8s.io/kubernetes/pkg/util/runtime"
	"k8s.io/kubernetes/pkg/util/wait"
	"sort"
	"sync"
	"time"
)

type InstancesController struct {
	SourceDestCheck *bool
	cloud           *kopeaws.AWSCloud

	period time.Duration

	instances map[string]*instance
	sequence  int

	// dnsState holds the last configured DNS state
	dns      kope.DNSProvider
	dnsState map[string][]string

	// stopLock is used to enforce only a single call to Stop is active.
	// Needed because we allow stopping through an http endpoint and
	// allowing concurrent stoppers leads to stack traces.
	stopLock sync.Mutex
	shutdown bool
	stopCh   chan struct{}
}

func NewInstancesController(cloud *kopeaws.AWSCloud, period time.Duration, dns kope.DNSProvider) *InstancesController {
	c := &InstancesController{
		cloud:     cloud,
		instances: make(map[string]*instance),
		period:    period,
		dns:       dns,
		dnsState:  make(map[string][]string),
	}
	return c
}

type instance struct {
	ID       string
	sequence int
	status   *ec2.Instance
}

func (c *InstancesController) runLoop() {
	go wait.Until(func() {
		if err := c.runOnce(); err != nil {
			runtime.HandleError(err)
		}
	}, c.period, c.stopCh)
}

// Stop stops the route controller.
func (c *InstancesController) Stop() error {
	// Stop is invoked from the http endpoint.
	c.stopLock.Lock()
	defer c.stopLock.Unlock()

	if !c.shutdown {
		close(c.stopCh)
		c.shutdown = true

		return nil
	}

	return fmt.Errorf("shutdown already in progress")
}

func (c *InstancesController) Run() {
	glog.Infof("starting aws controller")

	go c.runLoop()

	<-c.stopCh
	glog.Infof("shutting down route controller")
}

func (c *InstancesController) runOnce() error {
	instances, err := c.cloud.DescribeInstances()
	if err != nil {
		return err
	}

	c.sequence = c.sequence + 1
	sequence := c.sequence

	for _, awsInstance := range instances {
		id := aws.StringValue(awsInstance.InstanceId)
		if id == "" {
			runtime.HandleError(fmt.Errorf("skipping instance with empty instanceid: %v", awsInstance))
			continue
		}

		i := c.instances[id]
		if i == nil {
			i = &instance{
				ID: id,
			}
			c.instances[id] = i
		}

		i.status = awsInstance
		i.sequence = sequence
	}

	if err != nil {
		return fmt.Errorf("error doing EC2 describe instances: %v", err)
	}

	for _, i := range c.instances {
		id := i.ID

		if i.sequence != sequence {
			glog.Infof("Instance deleted: %q", id)
			delete(c.instances, id)
			continue
		}

		canSetSourceDestCheck := false
		instanceStateName := aws.StringValue(i.status.State.Name)
		switch instanceStateName {
		case "pending":
			glog.V(2).Infof("Ignoring pending instance: %q", id)
		case "running":
			canSetSourceDestCheck = true
		case "shutting-down":
		// ignore
		case "terminated":
		// ignore
		case "stopping":
			canSetSourceDestCheck = true
		case "stopped":
			canSetSourceDestCheck = true

		default:
			runtime.HandleError(fmt.Errorf("unknown instance state for instance %q: %q", id, instanceStateName))
		}

		if canSetSourceDestCheck && c.SourceDestCheck != nil && *c.SourceDestCheck != aws.BoolValue(i.status.SourceDestCheck) {
			err := c.cloud.ConfigureInstanceSourceDestCheck(i.ID, *c.SourceDestCheck)
			if err != nil {
				runtime.HandleError(fmt.Errorf("failed to configure SourceDestCheck for instance %q: %v", i.ID, err))
			} else {
				// Update the status in-place
				i.status.SourceDestCheck = c.SourceDestCheck
			}
		}

		// Other ideas...
		//   configure route53 name?
		//   look for "failed nodes" that did not come up
		//   related - maybe only do this poll very rarely, and most of the time be driven by node changes
		//
		// non-aws ideas:
		//   automatically recycle nodes after a while (but not
		//   manage node auto-updates
	}

	glog.Infof("Found %d instances", len(c.instances))

	if c.dns != nil {
		err = c.configureDNS(c.instances)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *InstancesController) configureDNS(instances map[string]*instance) error {
	dnsState := make(map[string][]string)

	for _, i := range instances {
		internalName, _ := kopeaws.FindTag(i.status, kopeaws.TagNameKubernetesDnsInternal)
		if internalName != "" {
			internalIP := aws.StringValue(i.status.PrivateIpAddress)
			if internalIP != "" {
				dnsState[internalName] = append(dnsState[internalName], internalIP)
			}
		}
		publicName, _ := kopeaws.FindTag(i.status, kopeaws.TagNameKubernetesDnsPublic)
		if publicName != "" {
			publicIP := aws.StringValue(i.status.PublicIpAddress)
			if publicIP != "" {
				dnsState[publicName] = append(dnsState[publicName], publicIP)
			}
		}
	}

	var changes map[string][]string
	if c.dnsState == nil {
		if len(dnsState) == 0 {
			glog.V(2).Infof("No dns configuration to apply")
			c.dnsState = dnsState
			return nil
		} else {
			changes = dnsState
		}
	} else {
		changes = make(map[string][]string)
		for k, v := range dnsState {
			sort.Strings(v)
			lastV := c.dnsState[k]
			if !StringSlicesEqual(lastV, v) {
				glog.V(2).Infof("DNS change %s: %v -> %v", k, lastV, v)
				changes[k] = v
			}
		}

		if len(changes) == 0 {
			glog.V(2).Infof("DNS configuration unchanged")
			return nil
		}
	}

	err := c.dns.ApplyDNSChanges(changes)
	if err != nil {
		return fmt.Errorf("error applying DNS changes: %v", err)
	}

	glog.V(2).Infof("Applied DNS changes to %d hosts", len(changes))

	c.dnsState = dnsState
	return nil
}

func StringSlicesEqual(l, r []string) bool {
	if len(l) != len(r) {
		return false
	}
	for i, v := range l {
		if r[i] != v {
			return false
		}
	}
	return true
}
