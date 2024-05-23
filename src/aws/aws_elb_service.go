package aws

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	elb "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	elbTypes "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	"time"
)

func initELBClient(region *string) (*elb.Client, error) {
	config, err := initConfig(region)
	if err != nil {
		return nil, err
	}
	return elb.NewFromConfig(config), nil
}

func createALB(region *string, name *string, securityGroupId *string) (*string, error) {
	client, err := initELBClient(region)
	if err != nil {
		return nil, err
	}
	subnetIds, err := describeSubnetIds(region)
	if err != nil {
		return nil, err
	}
	input := elb.CreateLoadBalancerInput{
		SecurityGroups: []string{*securityGroupId},
		Name:           name,
		Subnets:        subnetIds,
		IpAddressType:  elbTypes.IpAddressTypeIpv4,
		Scheme:         elbTypes.LoadBalancerSchemeEnumInternetFacing,
		Type:           elbTypes.LoadBalancerTypeEnumApplication,
		Tags: []elbTypes.Tag{
			{
				Key:   aws.String(baseTagName),
				Value: aws.String(baseTagValue),
			},
			{
				Key:   aws.String(baseUUIDTagName),
				Value: aws.String(BaseUUIDTagValue),
			},
		},
	}

	balancer, err := client.CreateLoadBalancer(ctx, &input)
	if err != nil {
		return nil, err
	}
	return balancer.LoadBalancers[0].LoadBalancerArn, nil
}

func deleteALB(region *string, arn *string) error {
	client, err := initELBClient(region)
	if err != nil {
		return err
	}
	_, err = client.DeleteLoadBalancer(ctx, &elb.DeleteLoadBalancerInput{LoadBalancerArn: arn})
	if err != nil {
		return err
	}
	return nil
}

func describeSubnetIds(region *string) ([]string, error) {
	client, err := initEC2Client(region)
	if err != nil {
		return nil, err
	}
	azInput := ec2.DescribeAvailabilityZonesInput{
		AllAvailabilityZones: aws.Bool(false),
		Filters: []ec2Types.Filter{
			{
				Name:   aws.String("group-name"),
				Values: []string{*region},
			},
		},
	}
	zones, err := client.DescribeAvailabilityZones(ctx, &azInput)
	if err != nil {
		return nil, err
	}
	zoneIds := make([]string, len(zones.AvailabilityZones))
	for i, az := range zones.AvailabilityZones {
		zoneIds[i] = *az.ZoneId
	}

	subnetInput := ec2.DescribeSubnetsInput{
		Filters: []ec2Types.Filter{
			{
				Name:   aws.String("availability-zone-id"),
				Values: zoneIds,
			},
		},
	}
	subnets, err := client.DescribeSubnets(ctx, &subnetInput)
	if err != nil {
		return nil, err
	}
	subnetIds := make([]string, len(subnets.Subnets))
	for i, subnet := range subnets.Subnets {
		subnetIds[i] = *subnet.SubnetId
	}
	return subnetIds, nil
}

func describeBootingEC2s(region *string, retry int) ([]string, error) {
	client, err := initEC2Client(region)
	if err != nil {
		return nil, err
	}
	input := ec2.DescribeInstancesInput{
		Filters: []ec2Types.Filter{
			{
				Name:   aws.String("instance-state-name"),
				Values: []string{"pending", "running"},
			},
			{
				Name:   aws.String(fmt.Sprintf("tag:%s", baseTagName)),
				Values: []string{baseTagValue},
			},
			{
				Name:   aws.String(fmt.Sprintf("tag:%s", baseUUIDTagName)),
				Values: []string{BaseUUIDTagValue},
			},
		},
	}
	var output *ec2.DescribeInstancesOutput
	for i := 0; i < retry; i++ {
		output, err = client.DescribeInstances(ctx, &input)
		if err != nil || len(output.Reservations) == 0 {
			time.Sleep(time.Second)
			continue
		}
		break
	}
	if err != nil {
		return nil, err
	} else if len(output.Reservations) == 0 {
		return nil, errors.New(fmt.Sprintf("no instances was found in region %s with tag %s", *region, baseTagName))
	}

	var instanceIds []string
	for _, reservation := range output.Reservations {
		for _, instance := range reservation.Instances {
			instanceIds = append(instanceIds, *instance.InstanceId)
		}
	}
	return instanceIds, nil
}

func waitEc2s(region *string, instanceIds *[]string, retry int) error {
	client, err := initEC2Client(region)
	if err != nil {
		return err
	}
	input := ec2.DescribeInstancesInput{
		InstanceIds: *instanceIds,
	}

	var output *ec2.DescribeInstancesOutput
	for i := 0; i < retry; i++ {
		output, err = client.DescribeInstances(ctx, &input)
		if err != nil {
			return err
		}
		if isInstancesRunning(output.Reservations) {
			break
		}
		time.Sleep(time.Second)
	}
	if !isInstancesRunning(output.Reservations) {
		return errors.New("there is no running instance, retry failed")
	}
	return nil
}

func isInstancesRunning(reservations []ec2Types.Reservation) bool {
	for _, reservation := range reservations {
		for _, instance := range reservation.Instances {
			if *instance.State.Code != 16 {
				return false
			}
		}
	}
	return true
}

func describeVPCs(region *string) (*string, error) {
	client, err := initEC2Client(region)
	if err != nil {
		return nil, err
	}
	input := ec2.DescribeVpcsInput{
		Filters: []ec2Types.Filter{
			{
				Name:   aws.String("is-default"),
				Values: []string{"true"},
			},
		},
	}
	vpcs, err := client.DescribeVpcs(ctx, &input)
	if err != nil {
		return nil, err
	}
	if len(vpcs.Vpcs) == 0 {
		return nil, errors.New(fmt.Sprintf("no default vpc is found in region %s", *region))
	}
	return vpcs.Vpcs[0].VpcId, nil
}

func addELBListener(region *string, elbArn *string, targetGroupArn *string, certificateArn *string) error {
	client, err := initELBClient(region)
	if err != nil {
		return err
	}
	input := elb.CreateListenerInput{
		LoadBalancerArn: elbArn,
		DefaultActions: []elbTypes.Action{
			{
				Type:           elbTypes.ActionTypeEnumForward,
				TargetGroupArn: targetGroupArn,
			},
		},
		Certificates: []elbTypes.Certificate{{
			CertificateArn: certificateArn,
		}},
		Protocol: elbTypes.ProtocolEnumHttps,
		Port:     aws.Int32(443),
		Tags: []elbTypes.Tag{
			{
				Key:   aws.String(baseTagName),
				Value: aws.String(baseTagValue),
			},
			{
				Key:   aws.String(baseUUIDTagName),
				Value: aws.String(BaseUUIDTagValue),
			},
		},
	}
	_, err = client.CreateListener(ctx, &input)
	if err != nil {
		return err
	}
	return nil
}

func createTargetGroup(region *string, name *string) (*string, error) {
	client, err := initELBClient(region)
	if err != nil {
		return nil, err
	}
	vpcId, err := describeVPCs(region)
	if err != nil {
		return nil, err
	}
	input := elb.CreateTargetGroupInput{
		Name:                name,
		HealthCheckEnabled:  aws.Bool(true),
		HealthCheckProtocol: elbTypes.ProtocolEnumHttp,
		IpAddressType:       elbTypes.TargetGroupIpAddressTypeEnumIpv4,
		Port:                aws.Int32(80),
		Protocol:            elbTypes.ProtocolEnumHttp,
		TargetType:          elbTypes.TargetTypeEnumInstance,
		VpcId:               vpcId,
		Tags: []elbTypes.Tag{
			{
				Key:   aws.String(baseTagName),
				Value: aws.String(baseTagValue),
			},
			{
				Key:   aws.String(baseUUIDTagName),
				Value: aws.String(BaseUUIDTagValue),
			},
		},
	}
	group, err := client.CreateTargetGroup(ctx, &input)
	if err != nil {
		return nil, err
	}
	return group.TargetGroups[0].TargetGroupArn, nil
}

func deleteTargetGroup(region *string, arn *string) error {
	client, err := initELBClient(region)
	if err != nil {
		return err
	}
	_, err = client.DeleteTargetGroup(ctx, &elb.DeleteTargetGroupInput{TargetGroupArn: arn})
	if err != nil {
		return err
	}
	return nil
}

func registerTargetInGroup(region *string, targetGroupArn *string) error {
	client, err := initELBClient(region)
	if err != nil {
		return err
	}
	ec2Ids, err := describeBootingEC2s(region, 120)
	if err != nil {
		return err
	}
	err = waitEc2s(region, &ec2Ids, 120)
	if err != nil {
		return err
	}
	targets := make([]elbTypes.TargetDescription, len(ec2Ids))
	for i, id := range ec2Ids {
		targets[i] = elbTypes.TargetDescription{
			Id:   &id,
			Port: aws.Int32(80),
		}
	}
	input := elb.RegisterTargetsInput{
		TargetGroupArn: targetGroupArn,
		Targets:        targets,
	}
	_, err = client.RegisterTargets(ctx, &input)
	if err != nil {
		return err
	}
	return nil
}
