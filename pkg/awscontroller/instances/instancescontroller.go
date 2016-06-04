package instances

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/golang/glog"
	"k8s.io/kubernetes/pkg/util/runtime"
	"k8s.io/kubernetes/pkg/util/wait"
	"sync"
	"time"
)

type InstancesController struct {
	SourceDestCheck *bool
	ec2             *ec2.EC2
	filterTags      map[string]string

	period          time.Duration

	instances       map[string]*instance
	sequence        int

	// stopLock is used to enforce only a single call to Stop is active.
	// Needed because we allow stopping through an http endpoint and
	// allowing concurrent stoppers leads to stack traces.
	stopLock        sync.Mutex
	shutdown        bool
	stopCh          chan struct{}
}

func NewInstancesController(ec2 *ec2.EC2, filterTags map[string]string) *InstancesController {
	c := &InstancesController{
		ec2:       ec2,
		instances: make(map[string]*instance),
		period:    10 * time.Second,
		filterTags: filterTags,
	}
	return c
}

type instance struct {
	ID       string
	sequence int
	status   *ec2.Instance
}

func (c *InstancesController) runLoop() {
	// TODO: A better way to run right away?
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

func newEc2Filter(name string, value string) *ec2.Filter {
	filter := &ec2.Filter{
		Name: aws.String(name),
		Values: []*string{
			aws.String(value),
		},
	}
	return filter
}


// Add additional filters, to match on our tags
// This lets us run multiple k8s clusters in a single EC2 AZ
func (c *InstancesController) addFilterTags(filters []*ec2.Filter) []*ec2.Filter {
	for k, v := range c.filterTags {
		filters = append(filters, newEc2Filter("tag:" + k, v))
	}
	if len(filters) == 0 {
		// We can't pass a zero-length Filters to AWS (it's an error)
		// So if we end up with no filters; just return nil
		return nil
	}

	return filters
}

func (c *InstancesController) runOnce() error {
	request := &ec2.DescribeInstancesInput{
		Filters: c.addFilterTags(nil),
	}

	glog.Infof("Querying EC2 instances")

	c.sequence = c.sequence + 1
	sequence := c.sequence

	err := c.ec2.DescribeInstancesPages(request, func(p *ec2.DescribeInstancesOutput, lastPage bool) bool {
		for _, r := range p.Reservations {
			for _, awsInstance := range r.Instances {
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
		}
		return true
	})

	if err != nil {
		return fmt.Errorf("error doing EC2 describe instances: %v", err)
	}

	for _, i := range c.instances {
		if i.sequence != sequence {
			glog.Infof("Instance deleted: %q", i.ID)
			delete(c.instances, i.ID)
			continue
		}

		if c.SourceDestCheck != nil && *c.SourceDestCheck != aws.BoolValue(i.status.SourceDestCheck) {
			err := c.configureInstanceSourceDestCheck(i.ID, *c.SourceDestCheck)
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

	return nil
}

// Sets the instance attribute "source-dest-check" to the specified value
func (c *InstancesController) configureInstanceSourceDestCheck(instanceID string, sourceDestCheck bool) error {
	glog.Infof("Configuring SourceDestCheck on %q to %v", instanceID, sourceDestCheck)

	request := &ec2.ModifyInstanceAttributeInput{}
	request.InstanceId = aws.String(instanceID)
	request.SourceDestCheck = &ec2.AttributeBooleanValue{Value: aws.Bool(sourceDestCheck)}

	_, err := c.ec2.ModifyInstanceAttribute(request)
	if err != nil {
		return fmt.Errorf("error configuring source-dest-check on instance %q: %v", instanceID, err)
	}
	return nil
}
