package kopeaws

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/golang/glog"
	"github.com/kopeio/aws-controller/pkg/kope"
	"net"
)

// The tag name we use to differentiate multiple logically independent clusters running in the same region
const TagNameKubernetesCluster = "KubernetesCluster"

// Set to expose the public IP of this instance via DNS
const TagNameKubernetesDnsPublic = "k8s.io/dns/public"

// Set to expose the internal IP of this instance via DNS
const TagNameKubernetesDnsInternal = "k8s.io/dns/internal"

type AWSCloud struct {
	ec2      *ec2.EC2
	metadata *ec2metadata.EC2Metadata

	zone       string
	instanceID string

	self       *ec2.Instance
	clusterID  string
	internalIP net.IP
}

var _ kope.Cloud = &AWSCloud{}

func NewAWSCloud() (*AWSCloud, error) {
	a := &AWSCloud{}

	s := session.New()
	s.Handlers.Send.PushFront(func(r *request.Request) {
		// Log requests
		glog.V(4).Infof("AWS API Request: %s/%s", r.ClientInfo.ServiceName, r.Operation.Name)
	})

	config := aws.NewConfig()
	a.metadata = ec2metadata.New(s, config)

	region, err := a.metadata.Region()
	if err != nil {
		return nil, fmt.Errorf("error querying ec2 metadata service (for az/region): %v", err)
	}

	a.zone, err = a.metadata.GetMetadata("placement/availability-zone")
	if err != nil {
		return nil, fmt.Errorf("error querying ec2 metadata service (for az): %v", err)
	}

	a.instanceID, err = a.metadata.GetMetadata("instance-id")
	if err != nil {
		return nil, fmt.Errorf("error querying ec2 metadata service (for instance-id): %v", err)
	}

	a.ec2 = ec2.New(s, config.WithRegion(region))

	err = a.getSelfInstance()
	if err != nil {
		return nil, err
	}

	return a, nil
}

func (a *AWSCloud) ClusterID() string {
	return a.clusterID
}

func (a *AWSCloud) getSelfInstance() error {
	instance, err := a.describeInstance(a.instanceID)
	if err != nil {
		return err
	}

	a.self = instance

	clusterID, _ := FindTag(instance, TagNameKubernetesCluster)
	if clusterID == "" {
		return fmt.Errorf("Cluster tag %q not found on this instance (%q)", TagNameKubernetesCluster, a.instanceID)
	}

	a.clusterID = clusterID

	a.internalIP = net.ParseIP(aws.StringValue(instance.PrivateIpAddress))
	if a.internalIP == nil {
		return fmt.Errorf("Internal IP not found on this instance (%q)", a.instanceID)
	}

	return nil
}

func (a *AWSCloud) describeInstance(instanceID string) (*ec2.Instance, error) {
	request := &ec2.DescribeInstancesInput{}
	request.InstanceIds = []*string{&instanceID}

	var instances []*ec2.Instance
	err := a.ec2.DescribeInstancesPages(request, func(p *ec2.DescribeInstancesOutput, lastPage bool) (shouldContinue bool) {
		for _, r := range p.Reservations {
			instances = append(instances, r.Instances...)
		}
		return true
	})

	if err != nil {
		return nil, fmt.Errorf("error querying for EC2 instance %q: %v", instanceID, err)
	}

	if len(instances) != 1 {
		return nil, fmt.Errorf("unexpected number of instances found with id %q: %d", instanceID, len(instances))
	}

	return instances[0], nil
}

// Add additional filters, to match on our tags
// This lets us run multiple k8s clusters in a single EC2 AZ
func (a *AWSCloud) addFilterTags(filters []*ec2.Filter) []*ec2.Filter {
	//for k, v := range c.filterTags {
	filters = append(filters, newEc2Filter("tag:"+TagNameKubernetesCluster, a.clusterID))
	//}
	if len(filters) == 0 {
		// We can't pass a zero-length Filters to AWS (it's an error)
		// So if we end up with no filters; just return nil
		return nil
	}

	return filters
}

func (a *AWSCloud) DescribeInstances() ([]*ec2.Instance, error) {
	request := &ec2.DescribeInstancesInput{
		Filters: a.addFilterTags(nil),
	}

	glog.Infof("Querying EC2 instances")

	var instances []*ec2.Instance

	err := a.ec2.DescribeInstancesPages(request, func(p *ec2.DescribeInstancesOutput, lastPage bool) bool {
		for _, r := range p.Reservations {
			for _, i := range r.Instances {
				instances = append(instances, i)
			}
		}
		return true
	})

	if err != nil {
		return nil, fmt.Errorf("error doing EC2 describe instances: %v", err)
	}

	return instances, nil
}

// Sets the instance attribute "source-dest-check" to the specified value
func (a *AWSCloud) ConfigureInstanceSourceDestCheck(instanceID string, sourceDestCheck bool) error {
	glog.Infof("Configuring SourceDestCheck on %q to %v", instanceID, sourceDestCheck)

	request := &ec2.ModifyInstanceAttributeInput{}
	request.InstanceId = aws.String(instanceID)
	request.SourceDestCheck = &ec2.AttributeBooleanValue{Value: aws.Bool(sourceDestCheck)}

	_, err := a.ec2.ModifyInstanceAttribute(request)
	if err != nil {
		return fmt.Errorf("error configuring source-dest-check on instance %q: %v", instanceID, err)
	}
	return nil
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

func FindTag(instance *ec2.Instance, name string) (string, bool) {
	for _, tag := range instance.Tags {
		k := aws.StringValue(tag.Key)
		if k == name {
			return aws.StringValue(tag.Value), true
		}
	}

	return "", false
}
