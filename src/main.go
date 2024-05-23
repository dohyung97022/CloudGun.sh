package main

import (
	"errors"
	"fmt"
	"fyc/aws"
	"fyc/datadogSdk"
	"fyc/githubSdk"
	"fyc/uuid"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"os"
	"regexp"
	"strings"
)

type arguments struct {
	GithubToken *string
	AWSRegion   *string
	Domain      *string
	Command     *string
}

func getArgs() (*arguments, error) {
	var input = arguments{}

	args := os.Args
	for _, arg := range args {
		if strings.HasPrefix(arg, "-githubtoken=") {
			res, found := strings.CutPrefix(arg, "-githubtoken=")
			if !found {
				return nil, errors.New("value of -githubtoken=XXX... is not valid")
			}
			input.GithubToken = &res
		} else if strings.HasPrefix(arg, "-awsregion=") {
			res, found := strings.CutPrefix(arg, "-awsregion=")
			if !found {
				return nil, errors.New("value of -awsregion=XXX... is not valid")
			}
			input.AWSRegion = &res
		} else if strings.HasPrefix(arg, "-domain=") {
			res, found := strings.CutPrefix(arg, "-domain=")
			if !found {
				return nil, errors.New("value of -domain=example.com is not valid")
			} else if strings.HasPrefix(res, "www.") {
				return nil, errors.New("value of -domain=example.com should not start with www.")
			}
			r, _ := regexp.Compile("([a-z0-9-]+\\.[a-z0-9-]+)")
			regexResult := r.FindStringSubmatch(res)
			if len(regexResult) != 2 || res != regexResult[1] || regexResult[0] != regexResult[1] {
				return nil, errors.New("value of -domain=example.com is not valid")
			}
			input.Domain = &res
		} else if strings.HasPrefix(arg, "-command=") {
			res, found := strings.CutPrefix(arg, "-command=")
			if !found {
				return nil, errors.New("value of -command=XXX... is not valid")
			} else if res != "create" && res != "delete" {
				return nil, errors.New("value of -command=XXX... should be -command=create or -command=delete")
			}
			input.Command = &res
		}
	}

	if input.GithubToken == nil {
		return nil, errors.New("value of -githubtoken=XXX... is not valid")
	}
	if input.AWSRegion == nil {
		return nil, errors.New("value of -awsregion=XXX... is not valid")
	}
	if input.Domain == nil {
		return nil, errors.New("value of -domain=example.com is not valid")
	}
	if input.Command == nil {
		return nil, errors.New("value of -command=XXX... is not valid")
	}

	err := githubSdk.InitClient(input.GithubToken)
	if err != nil {
		return nil, errors.New("github token provided is not valid!")
	}
	return &input, nil
}

func main() {
	datadogSdk.Info("Someone started CloudGun!")
	// TODO : 이슈
	// 이상하게 연결이 안되는 경우가 있다. ㅜㅜ
	// s3 에서 static website 를 삭제하고 다시 생성하니, 연결이 되었다. ?
	// 시간을 기다리면 된다.

	input, err := getArgs()
	if err != nil {
		fmt.Println("an error has occurred")
		datadogSdk.Error(err.Error())
		fmt.Println(err)
		os.Exit(1)
	}
	region := input.AWSRegion
	githubToken := input.GithubToken
	domain := input.Domain
	if *region == "us-east-1" {
		datadogSdk.Error("us-east-1 is not yet supported. You would have to wait for the actual launch of our product!")
		fmt.Println("us-east-1 is not yet supported. You would have to wait for the actual launch of our product!")
		os.Exit(1)
	}
	createUUID, err := aws.GetUUID(region) // local 파일에 저장하는 방식으로 변경
	if err != nil {
		fmt.Println("an error has occurred")
		datadogSdk.Error(err.Error())
		fmt.Println(err)
		os.Exit(1)
	}
	aws.BaseUUIDTagValue = *createUUID

	// credential 이 정확한지 확인 TODO : 해당 내용 2번 호출된다.
	_, err = aws.GetCredentials()
	if err != nil {
		fmt.Println("an error has occurred")
		datadogSdk.Error(err.Error())
		fmt.Println(err)
		os.Exit(1)
	}
	// route53 안에 진짜 도메인이 있는지 확인이 필요
	err = aws.CheckRoute53ForDomain(region, domain)
	if err != nil {
		fmt.Println("an error has occurred")
		datadogSdk.Error(err.Error())
		fmt.Println(err)
		os.Exit(1)
	}

	if *input.Command == "create" {
		err := createAll(*githubToken, *region, *domain)
		if err != nil {
			fmt.Println("an error has occurred")
			datadogSdk.Error(err.Error())
			fmt.Println(err)
			fmt.Println("\nplease delete leftover resources by changing the command -command=delete")
			os.Exit(1)
		}
		datadogSdk.Info("creation success")
		fmt.Println("creation success")
	} else if *input.Command == "delete" {
		err := deleteAll(*region, aws.BaseUUIDTagValue)
		if err != nil {
			fmt.Println("an error has occurred")
			datadogSdk.Error(err.Error())
			fmt.Println(err)
			os.Exit(1)
		}
		datadogSdk.Info("deletion success")
		fmt.Println("cloudGun deletion success")
		_, err = aws.ReplaceUUID(region)
		os.Exit(1)
	}
}

func createAll(githubToken string, region string, domain string) error {
	credentials, err := aws.GetCredentials()
	if err != nil {
		return err
	}

	awsAccessKey := credentials.AccessKey
	awsSecretAccessKey := credentials.SecretAccessKey
	commitMessage := "good first commit from cloudGun"
	branchName := "main"
	resourceName := "cloudGun"
	bucketName := domain + "-" + aws.BaseUUIDTagValue
	clusterName := resourceName + "-" + aws.BaseUUIDTagValue
	serviceName := resourceName + "-" + aws.BaseUUIDTagValue
	taskFamilyName := resourceName + "-" + aws.BaseUUIDTagValue
	containerName := resourceName + "-" + aws.BaseUUIDTagValue
	albName := resourceName + "-" + aws.BaseUUIDTagValue
	targetGroupName := resourceName + "-" + aws.BaseUUIDTagValue
	resourceGroupName := resourceName + "-" + aws.BaseUUIDTagValue
	ecrName := "cloud-gun-main-api-" + aws.BaseUUIDTagValue

	fmt.Println("InitClient")
	err = githubSdk.InitClient(&githubToken)
	if err != nil {
		return err
	}

	err = aws.CreateResourceGroup(&resourceGroupName, &region)
	if err != nil {
		return err
	}

	// creating aws s3, cloudfront
	distributionId, err := aws.CreateS3Website(&bucketName, &domain, &region)
	if err != nil {
		return err
	}
	repoUUID, _ := uuid.CreateUUID()
	// creating s3 website repo
	frontendRepoName := "cloud-gun-frontend-" + *repoUUID
	err = githubSdk.CreateS3WebsiteRepository(&region, &frontendRepoName, &bucketName, &awsAccessKey, &awsSecretAccessKey,
		distributionId, githubSdk.Vue3, &commitMessage, &branchName)
	if err != nil {
		return err
	}

	//// creating ecs
	var min int32 = 1
	var max int32 = 3
	var desired int32 = 1
	ecsArn, err := aws.CreateECSCluster(&region, &clusterName, &taskFamilyName, &containerName, &min, &max, &desired,
		ec2Types.InstanceTypeT2Micro, aws.AmazonLinux2)
	if err != nil {
		return err
	}

	// create alb
	mainApiDomain := "main-api." + domain
	err = aws.CreateELB(&region, &domain, &mainApiDomain, &albName, &targetGroupName)
	if err != nil {
		return err
	}

	// connect ecs with alb
	err = aws.ConnectECSServiceToALB(&region, &serviceName, ecsArn, &taskFamilyName, &albName, &containerName, &targetGroupName)
	if err != nil {
		return err
	}

	// create ecr
	err = aws.CreateECR(&region, &ecrName) // TODO : 지정된 생성된 자동 삭제 기능 추가
	if err != nil && !strings.Contains(err.Error(), "already exists in the registry with id") {
		return err
	}

	// creating main-api repo
	backendRepoName := "cloud-gun-main-api-" + *repoUUID
	err = githubSdk.CreateCodeRepository(&region, &awsAccessKey, &awsSecretAccessKey, &ecrName, &clusterName,
		&serviceName, &taskFamilyName, &containerName, &backendRepoName, &branchName, githubSdk.NodeExpressMainApi)
	if err != nil {
		return err
	}
	return nil
}

func deleteAll(region string, uuid string) error {
	// TODO : CreateS3Website 의 createCertificateRecord() 는 삭제되지 않는다. 해당 내용 삭제 필요
	// TODO : CreateELB 의 createCertificateRecord() 또한 삭제되지 않는다. 해당 내용 삭제 필요
	// TODO : Cloudfront 가 너무너무너너너무 느리다.
	// TODO : certificate 가 내부의 ELB 로 인해 삭제가 되지 않는 이슈가 있다.
	resourceName := "cloudGun"
	aws.BaseUUIDTagValue = uuid
	resourceGroupName := resourceName + "-" + aws.BaseUUIDTagValue
	err := aws.DeleteResources(&region, &resourceGroupName)
	if err != nil && !strings.Contains(err.Error(), "NotFoundException: Cannot find group") {
		return err
	}
	fmt.Println(fmt.Sprintf("All previous resources are deleted. check out https://%s.console.aws.amazon.com/resource-groups/home?region=%s", region, region))
	return nil
}
