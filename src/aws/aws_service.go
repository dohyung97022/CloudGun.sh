package aws

import (
	"context"
	"errors"
	"fmt"
	uuid2 "fyc/uuid"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var ctx context.Context

const baseTagName string = "CloudGun"
const baseTagValue string = "CloudGun"
const baseUUIDTagName string = "CloudGunUUID"

var BaseUUIDTagValue string

func init() {
	ctx, _ = context.WithTimeout(context.Background(), 720*time.Second)
}

func initConfig(region *string) (aws.Config, error) {
	return config.LoadDefaultConfig(ctx, config.WithRegion(*region))
}

type DefaultCredentials struct {
	Region          string
	AccessKey       string
	SecretAccessKey string
}

func CheckRoute53ForDomain(region *string, domain *string) error {
	_, err := getHostedZoneId(region, domain)
	if err != nil {
		return err
	}
	return nil
}

func GetUUID(region *string) (*string, error) {
	currentUser, err := user.Current()
	if err != nil {
		return nil, errors.New("error getting current user")
	}
	uuidDir := filepath.Join(currentUser.HomeDir, fmt.Sprintf("cloudGun-%s.id", *region))
	file, err := os.OpenFile(uuidDir, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	buffer := make([]byte, 1024)
	bytesRead, err := file.Read(buffer)
	uuid := ""
	if err != nil && err == io.EOF {
		ptr, _ := uuid2.CreateUUID()
		uuid = *ptr
	} else if err != nil {
		return nil, err
	} else {
		uuid = string(buffer[:bytesRead])
	}
	uuid = strings.ReplaceAll(uuid, "\n", "")
	err = os.WriteFile(uuidDir, []byte(uuid), 0644)
	if err != nil {
		return nil, err
	}

	return &uuid, nil
}

func ReplaceUUID(region *string) (*string, error) {
	currentUser, err := user.Current()
	if err != nil {
		return nil, errors.New("error getting current user")
	}
	uuidDir := filepath.Join(currentUser.HomeDir, fmt.Sprintf("cloudGun-%s.id", *region))
	file, err := os.OpenFile(uuidDir, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	uuidStr, _ := uuid2.CreateUUID()
	_, err = file.WriteString(*uuidStr)
	if err != nil {
		return nil, err
	}
	return uuidStr, nil
}

func GetCredentials() (*DefaultCredentials, error) {
	currentUser, err := user.Current()
	if err != nil {
		return nil, errors.New("error getting current user")
	}

	awsDir := filepath.Join(currentUser.HomeDir, ".aws/credentials")
	credentials, err := os.ReadFile(awsDir)
	if err != nil {
		return nil, errors.New("no aws credential if found in .aws/credentials")
	}
	fmt.Println(string(credentials))
	r, _ := regexp.Compile("\\[default\\][\\r\\n]([^[]+)\\[*")
	regexResults := r.FindStringSubmatch(string(credentials))
	if len(regexResults) == 0 {
		return nil, errors.New("no [default] credential found in .aws/credentials")
	}
	credentials = []byte(regexResults[1])

	result := DefaultCredentials{}
	//r, _ = regexp.Compile("region\\s*=\\s*([^\\n^\\s]+)\\n*")
	//regexResults = r.FindStringSubmatch(string(credentials))
	//if len(regexResults) == 0 {
	//	return nil, errors.New("no region in [default] credential is found")
	//}
	//result.Region = regexResults[1]

	r, _ = regexp.Compile("aws_access_key_id\\s*=\\s*([^\\n\\s\\r]+)[\\r\\n]*")
	regexResults = r.FindStringSubmatch(string(credentials))
	if len(regexResults) == 0 {
		return nil, errors.New("no aws_access_key_id in [default] credential is found")
	}
	fmt.Println(regexResults[1])
	result.AccessKey = regexResults[1]
	r, _ = regexp.Compile("aws_secret_access_key\\s*=\\s*([^\\n\\s\\r]+)[\\r\\n]*")
	regexResults = r.FindStringSubmatch(string(credentials))
	if len(regexResults) == 0 {
		return nil, errors.New("no aws_secret_access_key in [default] credential is found")
	}
	result.SecretAccessKey = regexResults[1]
	fmt.Println(regexResults[1])
	return &result, nil
}

func CreateResourceGroup(name *string, region *string) error {
	err := createResourceGroup(name, region)
	if err != nil {
		return err
	}
	err = createResourceGroup(name, aws.String("us-east-1"))
	if err != nil {
		return err
	}
	return nil
}

func CreateS3Website(name *string, domain *string, region *string) (*string, error) {
	fmt.Println("createBucket")
	err := createBucket(name, region)
	if err != nil {
		return nil, err
	}
	fmt.Println("requestCertificate")
	domains := []string{*domain, "www." + *domain}
	certArn, err := requestCertificate(domain, &domains, aws.String("us-east-1"), 120)
	if err != nil {
		return nil, err
	}
	fmt.Println("deletePublicAccessBlock")
	err = deletePublicAccessBlock(name, region)
	if err != nil {
		return nil, err
	}
	fmt.Println("putBucketPolicy")
	err = putBucketPolicy(name, region)
	if err != nil {
		return nil, err
	}
	fmt.Println("putBucketWebsite")
	err = putBucketWebsite(name, region)
	if err != nil {
		return nil, err
	}
	fmt.Println("createCloudfront")
	cloudEndpoint, distributionId, err := createCloudfront(region, name, domain, certArn)
	if err != nil {
		return nil, err
	}
	fmt.Println("createCloudfrontRecord")
	err = createCloudfrontRecord(region, domain, cloudEndpoint)
	if err != nil {
		return nil, err
	}

	fmt.Println("www")
	wwwName := "www." + *name
	wwwDomain := "www." + *domain
	fmt.Println("createBucket")
	err = createBucket(&wwwName, region)
	if err != nil {
		return nil, err
	}
	fmt.Println("deletePublicAccessBlock")
	err = deletePublicAccessBlock(&wwwName, region)
	if err != nil {
		return nil, err
	}
	fmt.Println("putBucketPolicy")
	err = putBucketPolicy(&wwwName, region)
	if err != nil {
		return nil, err
	}
	fmt.Println("putBucketWebsite")
	err = putBucketWebsite(&wwwName, region)
	if err != nil {
		return nil, err
	}
	fmt.Println("createCloudfront")
	cloudEndpoint, _, err = createCloudfront(region, &wwwName, &wwwDomain, certArn)
	if err != nil {
		return nil, err
	}
	fmt.Println("createCloudfrontRecord")
	err = createCloudfrontRecord(region, &wwwDomain, cloudEndpoint)
	if err != nil {
		return nil, err
	}

	return distributionId, nil
}

func CreateECSCluster(region *string, clusterName *string, taskFamilyName *string, containerName *string, min *int32, max *int32, desired *int32,
	instanceType ec2Types.InstanceType, image Image) (*string, error) {
	fmt.Println("createAutoScalingGroup")
	asgArn, err := createAutoScalingGroup(region, clusterName, max, min, desired, instanceType, image)
	if err != nil {
		return nil, err
	}
	fmt.Println("deleteCapacityProvider")
	_ = deleteCapacityProvider(region, clusterName)
	fmt.Println("createCapacityProvider")
	capacityProviderName, err := createCapacityProvider(region, clusterName, asgArn)
	if err != nil {
		return nil, err
	}
	fmt.Println("createECSCluster")
	arn, err := createECSCluster(region, clusterName, capacityProviderName)
	if err != nil {
		return nil, err
	}
	// TODO : we need these parameters out of the function
	var containerCPU int32 = 512
	var containerMiB int32 = 102
	var containerPort int32 = 80
	var hostPort int32 = 80
	fmt.Println("createECSTaskDefinition")
	_, err = createECSTaskDefinition(region, taskFamilyName, containerName, &containerCPU, &containerMiB,
		&containerPort, &hostPort)
	if err != nil {
		return nil, err
	}
	return arn, nil
}

func CreateECR(region *string, name *string) error {
	err := createECRRepository(region, name)
	if err != nil {
		return err
	}
	return nil
}

func CreateELB(region *string, domain *string, targetDomain *string, albName *string, targetGroupName *string) error {
	fmt.Println("requestCertificate")
	requestDomains := []string{*domain, "www." + *domain, *targetDomain}
	certificateArn, err := requestCertificate(domain, &requestDomains, region, 120)
	if err != nil {
		return err
	}
	fmt.Println("getSecurityGroupId")
	securityGroupId, err := getSecurityGroupId(region)
	if err != nil {
		return err
	}
	fmt.Println("createALB")
	elbArn, err := createALB(region, albName, securityGroupId)
	if err != nil {
		return err
	}
	fmt.Println("createTargetGroup")
	targetGroupArn, err := createTargetGroup(region, targetGroupName)
	if err != nil {
		return err
	}
	//fmt.Println("registerTargetInGroup")
	//err = registerTargetInGroup(region, targetGroupArn)
	//if err != nil {
	//	return err
	//}
	fmt.Println("waitCertificateIssued")
	err = waitCertificateIssued(region, certificateArn, 360)
	if err != nil {
		return err
	}
	fmt.Println("addELBListener")
	err = addELBListener(region, elbArn, targetGroupArn, certificateArn)
	if err != nil {
		return err
	}
	fmt.Println("createELBRecord")
	err = createELBRecord(region, domain, targetDomain, elbArn)
	if err != nil {
		return err
	}
	return nil
}

func ConnectECSServiceToALB(region *string, serviceName *string, ecsArn *string, taskFamilyName *string,
	albName *string, containerName *string, targetGroupArn *string) error {
	fmt.Println("createECSService")
	err := createECSService(region, serviceName, ecsArn, taskFamilyName, albName, containerName, targetGroupArn)
	if err != nil {
		return err
	}
	return nil
}

func CreateRDS(region *string, name *string, username *string, storage *int32) error {
	err := createRDS(region, name, storage, username)
	if err != nil {
		return err
	}
	return nil
}

func DeleteResources(region *string, name *string) error {
	err := deleteResourcesInGroup(name, region)
	if err != nil {
		return err
	}
	err = deleteResourcesInGroup(name, aws.String("us-east-1"))
	if err != nil {
		return err
	}
	err = deleteLaunchTemplate(region, name) // resource group 안에 없음
	if err != nil && !strings.Contains(err.Error(), "InvalidLaunchTemplateName.NotFoundException") {
		return err
	}
	return nil
}
