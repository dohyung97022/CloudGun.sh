package aws

import (
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	resource "github.com/aws/aws-sdk-go-v2/service/resourcegroups"
	resourceTypes "github.com/aws/aws-sdk-go-v2/service/resourcegroups/types"
	"strings"
	"time"
)

func initResourceClient(region *string) (*resource.Client, error) {
	config, err := initConfig(region)
	if err != nil {
		return nil, err
	}
	return resource.NewFromConfig(config), nil
}

func createResourceGroup(name *string, region *string) error {
	client, err := initResourceClient(region)
	if err != nil {
		return err
	}
	query := fmt.Sprintf("{\"ResourceTypeFilters\":[\"AWS::AllSupported\"],\"TagFilters\":[{\"Key\":\"%s\",\"Values\":[\"%s\"]},{\"Key\":\"%s\",\"Values\":[\"%s\"]}]}", baseTagName, baseTagValue, baseUUIDTagName, BaseUUIDTagValue)
	input := resource.CreateGroupInput{
		Name: name,
		ResourceQuery: &resourceTypes.ResourceQuery{
			Query: aws.String(query),
			Type:  resourceTypes.QueryTypeTagFilters10,
		},
	}
	_, err = client.CreateGroup(ctx, &input)
	if err != nil {
		return err
	}
	return err
}

func deleteResourcesInGroup(name *string, region *string) error {
	client, err := initResourceClient(region)
	if err != nil {
		return err
	}

	group, err := client.ListGroupResources(ctx, &resource.ListGroupResourcesInput{Group: name})
	if err != nil {
		return err
	}

	resources := getResourcesOf(&group.Resources, CloudFrontDistribution)
	for _, resource := range resources {
		fmt.Println("disableCloudfront")
		fmt.Println(*resource.Identifier.ResourceArn)
		fmt.Println(*resource.Identifier.ResourceType)
		err := disableCloudfront(region, resource.Identifier.ResourceArn)
		if err != nil {
			return err
		}
	}

	resources = getResourcesOf(&group.Resources, EC2Instance)
	for _, resource := range resources {
		fmt.Println("terminateEC2Instance")
		fmt.Println(*resource.Identifier.ResourceArn)
		fmt.Println(*resource.Identifier.ResourceType)
		err := terminateEC2Instance(region, resource.Identifier.ResourceArn)
		if err != nil && !strings.Contains(err.Error(), "InvalidInstanceID.NotFound") {
			return err
		}
	}

	// 이상하게 asg 는 조회가 되지 않아 직접 삭제한다.
	resources = getResourcesOf(&group.Resources, ECSCapacityProvider)
	for _, resource := range resources {
		fmt.Println("deleteCapacityProviderASG")
		fmt.Println(*resource.Identifier.ResourceArn)
		fmt.Println(*resource.Identifier.ResourceType)
		err := deleteCapacityProviderASG(region, resource.Identifier.ResourceArn)
		if err != nil {
			return err
		}
	}

	resources = getResourcesOf(&group.Resources, ElasticLoadBalancingLoadBalancer)
	for _, resource := range resources {
		fmt.Println("deleteALB")
		fmt.Println(*resource.Identifier.ResourceArn)
		fmt.Println(*resource.Identifier.ResourceType)
		err := deleteALB(region, resource.Identifier.ResourceArn)
		if err != nil {
			return err
		}
	}

	resources = getResourcesOf(&group.Resources, ECSService)
	for _, resource := range resources {
		fmt.Println("deleteECSService")
		fmt.Println(*resource.Identifier.ResourceArn)
		fmt.Println(*resource.Identifier.ResourceType)
		err := deleteECSService(region, resource.Identifier.ResourceArn)
		if err != nil {
			return err
		}
	}

	resources = getResourcesOf(&group.Resources, ECSCluster)
	for _, resource := range resources {
		fmt.Println("deleteECSCluster")
		fmt.Println(*resource.Identifier.ResourceArn)
		fmt.Println(*resource.Identifier.ResourceType)
		err := deleteECSCluster(region, resource.Identifier.ResourceArn)
		if err != nil {
			return err
		}
	}

	resources = getResourcesOf(&group.Resources, ECSTaskDefinition)
	for _, resource := range resources {
		fmt.Println("deregisterTaskDefinition")
		fmt.Println(*resource.Identifier.ResourceArn)
		fmt.Println(*resource.Identifier.ResourceType)
		err := deregisterTaskDefinition(region, resource.Identifier.ResourceArn)
		if err != nil && !strings.Contains(err.Error(), "it is in the process of being deleted") {
			return err
		}
	}

	resources = getResourcesOf(&group.Resources, S3Bucket)
	for _, resource := range resources {
		fmt.Println("deleteBucket")
		fmt.Println(*resource.Identifier.ResourceArn)
		fmt.Println(*resource.Identifier.ResourceType)
		err := deleteBucket(region, resource.Identifier.ResourceArn)
		if err != nil {
			return err
		}
	}

	// 시간 겁나 걸림
	resources = getResourcesOf(&group.Resources, CloudFrontDistribution)
	for _, resource := range resources {
		fmt.Println("deleteCloudfront")
		fmt.Println("deleting cloudfront distributions will take some time.")
		fmt.Println("this is because aws is removing their cache from cdn servers all over the world!")
		fmt.Println(*resource.Identifier.ResourceArn)
		fmt.Println(*resource.Identifier.ResourceType)
		err = retry(360, time.Second*1, func() error {
			err := deleteCloudfront(region, resource.Identifier.ResourceArn)
			if err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			return err
		}
	}

	resources = getResourcesOf(&group.Resources, ElasticLoadBalancingTargetGroup)
	for _, resource := range resources {
		fmt.Println("deleteTargetGroup")
		fmt.Println(*resource.Identifier.ResourceArn)
		fmt.Println(*resource.Identifier.ResourceType)
		err = retry(360, time.Second*1, func() error {
			err := deleteTargetGroup(region, resource.Identifier.ResourceArn)
			if err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			return err
		}
	}

	// TODO : 이거 근데 굳이 삭제를 해야 하나?
	// 이슈가 너무 크다.
	// 나중에 서버리스로 가거나 멀티프로세스 형으로 가면 해결할 수 있다.
	// known issue
	// https://stackoverflow.com/questions/69424636/unable-to-delete-aws-certificate-certificate-is-in-use
	resources = getResourcesOf(&group.Resources, CertificateManagerCertificate)
	for _, resource := range resources {
		fmt.Println("deleteCertificate")
		fmt.Println(*resource.Identifier.ResourceArn)
		fmt.Println(*resource.Identifier.ResourceType)
		err = retry(180, time.Second*1, func() error {
			err := deleteCertificate(region, resource.Identifier.ResourceArn)
			if err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			return err
		}
	}

	resources = getResourcesOf(&group.Resources, ECRRepository)
	for _, resource := range resources {
		fmt.Println("deleteECRRepository")
		fmt.Println(*resource.Identifier.ResourceArn)
		fmt.Println(*resource.Identifier.ResourceType)
		err := deleteECRRepository(region, resource.Identifier.ResourceArn)
		if err != nil {
			return err
		}
	}

	resources = getResourcesOf(&group.Resources, EC2SecurityGroup)
	for _, resource := range resources {
		fmt.Println("deleteSecurityGroup")
		fmt.Println(*resource.Identifier.ResourceArn)
		fmt.Println(*resource.Identifier.ResourceType)
		err := deleteSecurityGroup(region, resource.Identifier.ResourceArn)
		if err != nil {
			return err
		}
	}

	return nil
}

func getResourcesOf(items *[]resourceTypes.ListGroupResourcesItem, identifier ResourceIdentifier) []resourceTypes.ListGroupResourcesItem {
	result := make([]resourceTypes.ListGroupResourcesItem, 0)
	for _, item := range *items {
		if ResourceIdentifier(*item.Identifier.ResourceType) == identifier {
			result = append(result, item)
		}
	}
	return result
}

func retry(attempts int, sleep time.Duration, f func() error) (err error) {
	for i := 0; i < attempts; i++ {
		err = f()
		if err == nil {
			return nil
		}
		time.Sleep(sleep)
	}
	return err
}
