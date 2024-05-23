package aws

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	ecrTypes "github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"strings"
)

func initECRClient(region *string) (*ecr.Client, error) {
	config, err := initConfig(region)
	if err != nil {
		return nil, err
	}
	return ecr.NewFromConfig(config), nil
}

func createECRRepository(region *string, name *string) error {
	client, err := initECRClient(region)
	if err != nil {
		return err
	}
	input := ecr.CreateRepositoryInput{
		RepositoryName: name,
		Tags: []ecrTypes.Tag{
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
	_, err = client.CreateRepository(ctx, &input)
	if err != nil {
		return err
	}
	return nil
}

func deleteECRRepository(region *string, arn *string) error {
	arnSplit := strings.Split(*arn, "repository/")
	if len(arnSplit) != 2 {
		return errors.New(fmt.Sprintf("arn %s could not be split with string repository/. arn is not valid", *arn))
	}
	client, err := initECRClient(region)
	if err != nil {
		return err
	}
	input := ecr.DeleteRepositoryInput{
		RepositoryName: &arnSplit[1],
		Force:          true,
	}
	_, err = client.DeleteRepository(ctx, &input)
	if err != nil {
		return err
	}
	return nil
}
