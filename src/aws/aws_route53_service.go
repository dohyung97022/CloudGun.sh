package aws

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	elb "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	elbTypes "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
	"strings"
)

func initRoute53Client(region *string) (*route53.Client, error) {
	config, err := initConfig(region)
	if err != nil {
		return nil, err
	}
	return route53.NewFromConfig(config), nil
}

func getHostedZoneId(region *string, domain *string) (*string, error) {
	client, err := initRoute53Client(region)
	if err != nil {
		return nil, err
	}
	input := route53.ListHostedZonesInput{}
	zones, err := client.ListHostedZones(ctx, &input)
	if err != nil {
		return nil, err
	}
	for _, zone := range zones.HostedZones {
		if *domain+"." == *zone.Name {
			result := strings.Replace(*zone.Id, "/hostedzone/", "", 1)
			return &result, nil
		}
	}
	return nil, errors.New(fmt.Sprintf("domain %s match not found in route53", *domain))
}

func createCertificateRecord(region *string, domain *string, fullDomain *string, target *string) error {
	routeZoneId, err := getHostedZoneId(region, domain)
	if err != nil {
		return err
	}

	client, err := initRoute53Client(region)
	if err != nil {
		return err
	}

	input := route53.ChangeResourceRecordSetsInput{
		HostedZoneId: routeZoneId,
		ChangeBatch: &types.ChangeBatch{
			Changes: []types.Change{{
				Action: types.ChangeActionUpsert,
				ResourceRecordSet: &types.ResourceRecordSet{
					Name:            fullDomain,
					Type:            types.RRTypeCname,
					ResourceRecords: []types.ResourceRecord{{Value: target}},
					TTL:             aws.Int64(300),
				},
			}},
		},
	}
	_, err = client.ChangeResourceRecordSets(ctx, &input)
	if err != nil {
		return err
	}
	return nil
}

func createCloudfrontRecord(region *string, fullDomain *string, target *string) error {
	domain := strings.TrimPrefix(*fullDomain, "www.")
	routeZoneId, err := getHostedZoneId(region, &domain)
	if err != nil {
		return err
	}

	client, err := initRoute53Client(region)
	if err != nil {
		return err
	}

	input := route53.ChangeResourceRecordSetsInput{
		HostedZoneId: routeZoneId,
		ChangeBatch: &types.ChangeBatch{
			Changes: []types.Change{{
				Action: types.ChangeActionUpsert,
				ResourceRecordSet: &types.ResourceRecordSet{
					Name: fullDomain,
					Type: types.RRTypeA,
					AliasTarget: &types.AliasTarget{
						DNSName:              target,
						EvaluateTargetHealth: false,
						HostedZoneId:         aws.String("Z2FDTNDATAQYW2"), // cloudfront
					},
				},
			}},
		},
	}
	_, err = client.ChangeResourceRecordSets(ctx, &input)
	if err != nil {
		return err
	}
	return nil
}

func describeELB(region *string, elbArn *string) (*elbTypes.LoadBalancer, error) {
	client, err := initELBClient(region)
	if err != nil {
		return nil, err
	}
	input := elb.DescribeLoadBalancersInput{
		LoadBalancerArns: []string{*elbArn},
	}
	balancers, err := client.DescribeLoadBalancers(ctx, &input)
	if err != nil {
		return nil, err
	}
	if len(balancers.LoadBalancers) == 0 {
		return nil, errors.New(fmt.Sprintf("no elb found of arn %s", *elbArn))
	}
	elb := balancers.LoadBalancers[0]
	return &elb, nil
}

func createELBRecord(region *string, domain *string, targetDomain *string, elbArn *string) error {
	if strings.HasPrefix(*domain, "www.") {
		return errors.New(fmt.Sprintf("parameter domain %s should not include www. ", *domain))
	}
	routeZoneId, err := getHostedZoneId(region, domain)
	if err != nil {
		return err
	}

	client, err := initRoute53Client(region)
	if err != nil {
		return err
	}

	loadBalancer, err := describeELB(region, elbArn)
	if err != nil {
		return err
	}

	input := route53.ChangeResourceRecordSetsInput{
		HostedZoneId: routeZoneId,
		ChangeBatch: &types.ChangeBatch{
			Changes: []types.Change{{
				Action: types.ChangeActionUpsert,
				ResourceRecordSet: &types.ResourceRecordSet{
					Name: targetDomain,
					Type: types.RRTypeA,
					AliasTarget: &types.AliasTarget{
						DNSName:              loadBalancer.DNSName,
						EvaluateTargetHealth: false,
						HostedZoneId:         loadBalancer.CanonicalHostedZoneId,
					},
				},
			}},
		},
	}
	_, err = client.ChangeResourceRecordSets(ctx, &input)
	if err != nil {
		if strings.Contains(err.Error(), "but it already exists") {
			return nil
		}
		return err
	}
	return nil
}
