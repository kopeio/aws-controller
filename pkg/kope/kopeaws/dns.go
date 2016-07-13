package kopeaws

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/golang/glog"
	"github.com/kopeio/aws-controller/pkg/kope"
	"github.com/kopeio/aws-controller/pkg/kope/utils"
	"strings"
	"time"
)

var defaultTTL = time.Minute

// TODO: Replace with k8s built-in helpers

type Route53DNSProvider struct {
	zoneName string
	route53  *route53.Route53

	zone *route53.HostedZone
}

var _ kope.DNSProvider = &Route53DNSProvider{}

func NewRoute53DNSProvider(zoneName string) *Route53DNSProvider {
	s := session.New()
	s.Handlers.Send.PushFront(func(r *request.Request) {
		// Log requests
		glog.V(4).Infof("AWS API Request: %s/%s", r.ClientInfo.ServiceName, r.Operation.Name)
	})

	config := aws.NewConfig()

	route53 := route53.New(s, config)

	return &Route53DNSProvider{
		route53:  route53,
		zoneName: zoneName,
	}
}

func (d *Route53DNSProvider) ApplyDNSChanges(dns map[string][]string) error {
	return d.set(dns, defaultTTL)
}

func (d *Route53DNSProvider) getZone() (*route53.HostedZone, error) {
	if d.zone != nil {
		return d.zone, nil
	}

	if !strings.Contains(d.zoneName, ".") {
		// Looks like a zone ID
		zoneID := d.zoneName
		glog.Infof("Querying for hosted zone by id: %q", zoneID)

		request := &route53.GetHostedZoneInput{
			Id: aws.String(zoneID),
		}

		response, err := d.route53.GetHostedZone(request)
		if err != nil {
			if AWSErrorCode(err) == "NoSuchHostedZone" {
				glog.Infof("Zone not found with id %q; will reattempt by name", zoneID)
			} else {
				return nil, fmt.Errorf("error querying for DNS HostedZones %q: %v", zoneID, err)
			}
		} else {
			d.zone = response.HostedZone
			return d.zone, nil
		}
	}

	glog.Infof("Querying for hosted zone by name: %q", d.zoneName)

	findZone := d.zoneName
	if !strings.HasSuffix(findZone, ".") {
		findZone += "."
	}
	request := &route53.ListHostedZonesByNameInput{
		DNSName: aws.String(findZone),
	}

	response, err := d.route53.ListHostedZonesByName(request)
	if err != nil {
		return nil, fmt.Errorf("error querying for DNS HostedZones %q: %v", findZone, err)
	}

	var zones []*route53.HostedZone
	for _, zone := range response.HostedZones {
		if aws.StringValue(zone.Name) == findZone {
			zones = append(zones, zone)
		}
	}
	if len(zones) == 0 {
		return nil, nil
	}
	if len(zones) != 1 {
		return nil, fmt.Errorf("found multiple hosted zones matched name %q", findZone)
	}

	d.zone = zones[0]

	return d.zone, nil
}

func (d *Route53DNSProvider) set(records map[string][]string, ttl time.Duration) error {
	zone, err := d.getZone()
	if err != nil {
		return err
	}

	changeBatch := &route53.ChangeBatch{}
	for name, hosts := range records {
		rrs := &route53.ResourceRecordSet{
			Name: aws.String(name),
			Type: aws.String("A"),
			TTL:  aws.Int64(int64(ttl.Seconds())),
		}

		for _, host := range hosts {
			rr := &route53.ResourceRecord{
				Value: aws.String(host),
			}
			rrs.ResourceRecords = append(rrs.ResourceRecords, rr)
		}

		change := &route53.Change{
			Action:            aws.String("UPSERT"),
			ResourceRecordSet: rrs,
		}
		changeBatch.Changes = append(changeBatch.Changes, change)
	}

	request := &route53.ChangeResourceRecordSetsInput{}
	request.HostedZoneId = zone.Id
	request.ChangeBatch = changeBatch

	glog.V(2).Infof("Updating DNS records %q", records)
	glog.V(4).Infof("route53 request: %s", utils.DebugString(request))

	response, err := d.route53.ChangeResourceRecordSets(request)
	if err != nil {
		return fmt.Errorf("error creating ResourceRecordSets: %v", err)
	}

	glog.V(2).Infof("Change id is %q", aws.StringValue(response.ChangeInfo.Id))

	return nil
}

// AWSErrorCode returns the aws error code, if it is an awserr.Error, otherwise ""
func AWSErrorCode(err error) string {
	if awsError, ok := err.(awserr.Error); ok {
		return awsError.Code()
	}
	return ""
}
