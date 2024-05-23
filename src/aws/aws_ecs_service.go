package aws

import (
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	asgTypes "github.com/aws/aws-sdk-go-v2/service/autoscaling/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecsTypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	elb "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"strconv"
	"strings"
)

func initECSClient(region *string) (*ecs.Client, error) {
	config, err := initConfig(region)
	if err != nil {
		return nil, err
	}
	return ecs.NewFromConfig(config), nil
}

func initEC2Client(region *string) (*ec2.Client, error) {
	config, err := initConfig(region)
	if err != nil {
		return nil, err
	}
	return ec2.NewFromConfig(config), nil
}

func initAutoScalingClient(region *string) (*autoscaling.Client, error) {
	config, err := initConfig(region)
	if err != nil {
		return nil, err
	}
	return autoscaling.NewFromConfig(config), nil
}

func createECSCluster(region *string, name *string, capacityProviderName *string) (*string, error) {
	client, err := initECSClient(region)
	if err != nil {
		return nil, err
	}

	if err != nil {
		return nil, err
	}
	input := ecs.CreateClusterInput{
		ClusterName:       name,
		CapacityProviders: []string{*capacityProviderName},
		DefaultCapacityProviderStrategy: []ecsTypes.CapacityProviderStrategyItem{{
			CapacityProvider: capacityProviderName,
			Weight:           1,
		}},
		Tags: []ecsTypes.Tag{
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
	cluster, err := client.CreateCluster(ctx, &input)
	if err != nil {
		return nil, err
	}
	return cluster.Cluster.ClusterArn, nil
}

func deleteECSCluster(region *string, arn *string) error {
	client, err := initECSClient(region)
	if err != nil {
		return err
	}
	_, err = client.DeleteCluster(ctx, &ecs.DeleteClusterInput{Cluster: arn})
	if err != nil {
		return err
	}
	return nil
}

func terminateEC2Instance(region *string, arn *string) error {
	splitArn := strings.Split(*arn, "instance/")
	client, err := initEC2Client(region)
	if err != nil {
		return err
	}
	_, err = client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{InstanceIds: []string{splitArn[1]}})
	if err != nil {
		return err
	}
	return nil
}

func deleteCapacityProviderASG(region *string, arn *string) error {
	client, err := initECSClient(region)
	if err != nil {
		return err
	}

	output, err := client.DescribeCapacityProviders(ctx, &ecs.DescribeCapacityProvidersInput{CapacityProviders: []string{*arn}})
	if err != nil {
		return err
	}
	if len(output.CapacityProviders) == 0 {
		return errors.New(fmt.Sprintf("no capacity provider was found of arn %s in region %s", *arn, *region))
	}
	_, err = client.DeleteCapacityProvider(ctx, &ecs.DeleteCapacityProviderInput{CapacityProvider: arn})
	if err != nil {
		return err
	}

	asgName := strings.Split(*output.CapacityProviders[0].AutoScalingGroupProvider.AutoScalingGroupArn, "autoScalingGroupName/")
	asgClient, err := initAutoScalingClient(region)
	input := autoscaling.DeleteAutoScalingGroupInput{AutoScalingGroupName: &asgName[1], ForceDelete: aws.Bool(true)}
	_, err = asgClient.DeleteAutoScalingGroup(ctx, &input)
	if err != nil && !strings.Contains(err.Error(), " AutoScalingGroup name not found") {
		return err
	}
	return nil
}

func createECSService(region *string, serviceName *string, clusterArn *string, taskDefinition *string,
	albName *string, containerName *string, targetGroupName *string) error {
	elbClient, err := initELBClient(region)
	if err != nil {
		return err
	}
	groups, err := elbClient.DescribeTargetGroups(ctx, &elb.DescribeTargetGroupsInput{Names: []string{*targetGroupName}})
	if err != nil {
		return err
	}
	if len(groups.TargetGroups) == 0 {
		return errors.New(fmt.Sprintf("no target groups were found of name %s", *targetGroupName))
	}

	client, err := initECSClient(region)
	if err != nil {
		return err
	}
	input := ecs.CreateServiceInput{
		ServiceName:    serviceName,
		Cluster:        clusterArn,
		DesiredCount:   aws.Int32(1),
		TaskDefinition: taskDefinition,
		LaunchType:     ecsTypes.LaunchTypeEc2,
		LoadBalancers: []ecsTypes.LoadBalancer{
			{
				//LoadBalancerName: albName,
				ContainerName:  containerName,
				ContainerPort:  aws.Int32(80),
				TargetGroupArn: groups.TargetGroups[0].TargetGroupArn,
			},
		},
		Tags: []ecsTypes.Tag{
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
	_, err = client.CreateService(ctx, &input)
	if err != nil {
		return err
	}
	return nil
}

func deleteECSService(region *string, arn *string) error {
	split := strings.Split(*arn, "service/")
	clusterService := strings.Split(split[1], "/")
	fmt.Println(*arn)
	client, err := initECSClient(region)
	if err != nil {
		return err
	}

	_, err = client.UpdateService(ctx, &ecs.UpdateServiceInput{Service: &clusterService[1], Cluster: &clusterService[0], DesiredCount: aws.Int32(0)})
	if err != nil && !strings.Contains(err.Error(), "Service was not ACTIVE") {
		return err
	}

	_, err = client.DeleteService(ctx, &ecs.DeleteServiceInput{Service: &clusterService[1], Cluster: &clusterService[0]})
	if err != nil {
		return err
	}
	return nil
}

func createECSTaskDefinition(region *string, taskFamilyName *string, containerName *string, containerCpu *int32,
	containerMemory *int32, containerPort *int32, hostPort *int32) (*string, error) {
	client, err := initECSClient(region)
	if err != nil {
		return nil, err
	}

	input := ecs.RegisterTaskDefinitionInput{
		Family: taskFamilyName,
		//Cpu:    aws.String(strconv.Itoa(int(*containerCpu))),
		Memory: aws.String(strconv.Itoa(int(*containerMemory))),
		ContainerDefinitions: []ecsTypes.ContainerDefinition{
			{
				Name:  containerName,
				Image: aws.String("nginx"), // TODO : https://docs.aws.amazon.com/ko_kr/AmazonECR/latest/userguide/docker-pull-ecr-image.html
				//Memory:            containerMemory,
				//MemoryReservation: containerMemory,
				//Cpu:               *containerCpu,
				PortMappings: []ecsTypes.PortMapping{
					{
						AppProtocol:   ecsTypes.ApplicationProtocolHttp,
						ContainerPort: containerPort,
						HostPort:      aws.Int32(0), // dynamic port hosting
					},
				},
			},
		},
		Tags: []ecsTypes.Tag{
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
	taskDefinition, err := client.RegisterTaskDefinition(ctx, &input)
	if err != nil {
		return nil, err
	}
	return taskDefinition.TaskDefinition.TaskDefinitionArn, nil
}

func deregisterTaskDefinition(region *string, arn *string) error {
	client, err := initECSClient(region)
	if err != nil {
		return err
	}
	_, err = client.DeregisterTaskDefinition(ctx, &ecs.DeregisterTaskDefinitionInput{TaskDefinition: arn})
	if err != nil {
		return err
	}
	_, err = client.DeleteTaskDefinitions(ctx, &ecs.DeleteTaskDefinitionsInput{TaskDefinitions: []string{*arn}})
	if err != nil {
		return err
	}
	return nil
}

func createCapacityProvider(region *string, name *string, asgArn *string) (*string, error) {
	client, err := initECSClient(region)
	if err != nil {
		return nil, err
	}

	input := ecs.CreateCapacityProviderInput{
		Name: name,
		AutoScalingGroupProvider: &ecsTypes.AutoScalingGroupProvider{
			AutoScalingGroupArn: asgArn,
		},
		Tags: []ecsTypes.Tag{
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
	capacityProvider, err := client.CreateCapacityProvider(ctx, &input)
	if err != nil {
		return nil, err
	}
	return capacityProvider.CapacityProvider.Name, nil
}

func deleteCapacityProvider(region *string, name *string) error {
	client, err := initECSClient(region)
	if err != nil {
		return err
	}
	input := ecs.DeleteCapacityProviderInput{CapacityProvider: name}
	_, err = client.DeleteCapacityProvider(ctx, &input)
	if err != nil {
		return err
	}
	return nil
}

func createAutoScalingGroup(region *string, name *string, max *int32, min *int32, desired *int32,
	instanceType ec2Types.InstanceType, image Image) (*string, error) {
	client, err := initAutoScalingClient(region)
	if err != nil {
		return nil, err
	}
	_ = deleteLaunchTemplate(region, name)
	securityGroupId, err := createSecurityGroup(region, name)
	if err != nil {
		return nil, err
	}
	templateId, err := createLaunchTemplate(region, name, securityGroupId, instanceType, image)
	if err != nil {
		return nil, err
	}
	zones, err := describeAvailabilityZones(region)
	if err != nil {
		return nil, err
	}
	input := autoscaling.CreateAutoScalingGroupInput{
		AutoScalingGroupName: name,
		MaxSize:              max,
		MinSize:              min,
		DesiredCapacity:      desired,
		LaunchTemplate:       &asgTypes.LaunchTemplateSpecification{LaunchTemplateId: templateId},
		AvailabilityZones:    zones,
		Tags: []asgTypes.Tag{
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
	_, err = client.CreateAutoScalingGroup(ctx, &input)
	if err != nil {
		return nil, err
	}

	arn, err := getAutoScalingGroupArn(region, name)
	if err != nil {
		return nil, err
	}
	return arn, nil
}

func describeAvailabilityZones(region *string) ([]string, error) {
	client, err := initEC2Client(region)
	if err != nil {
		return nil, err
	}
	input := ec2.DescribeAvailabilityZonesInput{
		AllAvailabilityZones: aws.Bool(false),
		Filters: []ec2Types.Filter{
			{
				Name:   aws.String("group-name"),
				Values: []string{*region},
			},
		},
	}
	zones, err := client.DescribeAvailabilityZones(ctx, &input)
	if err != nil {
		return nil, err
	}
	var zoneIds = make([]string, len(zones.AvailabilityZones))
	for i, zone := range zones.AvailabilityZones {
		zoneIds[i] = *zone.ZoneName
	}
	return zoneIds, nil
}

func getAutoScalingGroupArn(region *string, name *string) (*string, error) {
	client, err := initAutoScalingClient(region)
	if err != nil {
		return nil, err
	}
	input := autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []string{*name},
	}
	groups, err := client.DescribeAutoScalingGroups(ctx, &input)
	if err != nil {
		return nil, err
	}
	if len(groups.AutoScalingGroups) == 0 {
		return nil, errors.New(fmt.Sprintf("no autoScalingGroup of name %s found in region %s", *name, *region))
	}
	return groups.AutoScalingGroups[0].AutoScalingGroupARN, nil
}

func createSecurityGroup(region *string, name *string) (*string, error) {
	client, err := initEC2Client(region)
	if err != nil {
		return nil, err
	}
	input := ec2.CreateSecurityGroupInput{

		Description: aws.String("cloudGun security group"),
		GroupName:   name,
		TagSpecifications: []ec2Types.TagSpecification{
			{
				ResourceType: ec2Types.ResourceTypeSecurityGroup,
				Tags: []ec2Types.Tag{
					{
						Key:   aws.String(baseTagName),
						Value: aws.String(baseTagValue),
					},
					{
						Key:   aws.String(baseUUIDTagName),
						Value: aws.String(BaseUUIDTagValue),
					},
				},
			},
		},
	}
	group, err := client.CreateSecurityGroup(ctx, &input)
	if err != nil {
		return nil, err
	}

	ingressInput := ec2.AuthorizeSecurityGroupIngressInput{
		GroupId: group.GroupId,
		IpPermissions: []ec2Types.IpPermission{
			{
				FromPort:   aws.Int32(32768), // dynamic port mapping range for ALB
				ToPort:     aws.Int32(65535), // https://repost.aws/knowledge-center/dynamic-port-mapping-ecs
				IpProtocol: aws.String("tcp"),
				IpRanges: []ec2Types.IpRange{
					{CidrIp: aws.String("0.0.0.0/0")},
				},
				Ipv6Ranges: []ec2Types.Ipv6Range{
					{CidrIpv6: aws.String("::/0")},
				},
			},
			{
				FromPort:   aws.Int32(443), // port ingress for alb
				ToPort:     aws.Int32(443),
				IpProtocol: aws.String("tcp"),
				IpRanges: []ec2Types.IpRange{
					{CidrIp: aws.String("0.0.0.0/0")},
				},
				Ipv6Ranges: []ec2Types.Ipv6Range{
					{CidrIpv6: aws.String("::/0")},
				},
			},
		},
	}

	_, err = client.AuthorizeSecurityGroupIngress(ctx, &ingressInput)
	if err != nil {
		return nil, err
	}
	return group.GroupId, nil
}

func getSecurityGroupId(region *string) (*string, error) {
	client, err := initEC2Client(region)
	if err != nil {
		return nil, err
	}
	input := ec2.DescribeSecurityGroupsInput{
		Filters: []ec2Types.Filter{
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
	groups, err := client.DescribeSecurityGroups(ctx, &input)
	if err != nil {
		return nil, err
	}
	if len(groups.SecurityGroups) != 1 {
		return nil, errors.New(fmt.Sprintf("no security group was found with tag %s:%s", baseUUIDTagName, BaseUUIDTagValue))
	}
	return groups.SecurityGroups[0].GroupId, nil
}
func deleteSecurityGroup(region *string, arn *string) error {
	split := strings.Split(*arn, "security-group/")
	if len(split) != 2 {
		return errors.New(fmt.Sprintf("arn %s could not be split with security-group/", *arn))
	}
	client, err := initEC2Client(region)
	if err != nil {
		return err
	}
	input := ec2.DeleteSecurityGroupInput{
		GroupId: &split[1],
	}
	_, err = client.DeleteSecurityGroup(ctx, &input)
	if err != nil {
		return err
	}
	return nil
}

func deleteLaunchTemplate(region *string, name *string) error {
	client, err := initEC2Client(region)
	if err != nil {
		return err
	}
	input := ec2.DeleteLaunchTemplateInput{
		LaunchTemplateName: name,
	}
	_, err = client.DeleteLaunchTemplate(ctx, &input)
	if err != nil {
		return err
	}
	return nil
}

func createLaunchTemplate(region *string, name *string, securityGroupId *string, instanceType ec2Types.InstanceType, image Image) (*string, error) {
	client, err := initEC2Client(region)
	if err != nil {
		return nil, err
	}
	imageId, err := describeImage(region, image)
	if err != nil {
		return nil, err
	}

	userData := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("#!/bin/bash\necho ECS_CLUSTER=%s >> /etc/ecs/ecs.config;", *name)))
	input := ec2.CreateLaunchTemplateInput{
		LaunchTemplateData: &ec2Types.RequestLaunchTemplateData{
			SecurityGroupIds: []string{*securityGroupId},
			ImageId:          imageId,
			InstanceType:     instanceType,
			IamInstanceProfile: &ec2Types.LaunchTemplateIamInstanceProfileSpecificationRequest{
				Name: aws.String("ecsInstanceRole"),
			},
			UserData: aws.String(userData),
			TagSpecifications: []ec2Types.LaunchTemplateTagSpecificationRequest{
				{
					ResourceType: ec2Types.ResourceTypeInstance,
					Tags: []ec2Types.Tag{
						{
							Key:   aws.String(baseTagName),
							Value: aws.String(baseTagValue),
						},
						{
							Key:   aws.String(baseUUIDTagName),
							Value: aws.String(BaseUUIDTagValue),
						},
					},
				},
			},
		},
		LaunchTemplateName: name,
	}
	template, err := client.CreateLaunchTemplate(ctx, &input)
	if err != nil {
		return nil, err
	}
	return template.LaunchTemplate.LaunchTemplateId, nil
}

func describeImage(region *string, image Image) (*string, error) {
	client, err := initEC2Client(region)
	if err != nil {
		return nil, err
	}
	input := ec2.DescribeImagesInput{
		Filters: []ec2Types.Filter{
			{
				Name:   aws.String("architecture"),
				Values: []string{"x86_64"}, // , "arm64"
			},
			{
				Name:   aws.String("name"),
				Values: []string{image.name},
			},
			{
				Name:   aws.String("owner-alias"),
				Values: []string{"amazon"},
			},
		},
	}
	images, err := client.DescribeImages(ctx, &input)
	if err != nil {
		return nil, err
	}
	if len(images.Images) == 0 {
		return nil, errors.New(fmt.Sprintf("no images found in region %s of name %s", *region, image.name))
	}
	return images.Images[0].ImageId, err
}
