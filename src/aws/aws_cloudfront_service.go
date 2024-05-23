package aws

import (
	"errors"
	"fmt"
	"fyc/uuid"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront/types"
	"strings"
)

func initCloudfrontClient(region *string) (*cloudfront.Client, error) {
	config, err := initConfig(region)
	if err != nil {
		return nil, err
	}
	return cloudfront.NewFromConfig(config), nil
}

func createCloudfront(region *string, bucketName *string, domain *string, certArn *string) (*string, *string, error) {
	cloudfrontRegion := aws.String("us-east-1")
	err := waitCertificateIssued(cloudfrontRegion, certArn, 360)
	if err != nil {
		return nil, nil, err
	}

	client, err := initCloudfrontClient(cloudfrontRegion)
	if err != nil {
		return nil, nil, err
	}

	originId, err := uuid.CreateUUID()
	if err != nil {
		return nil, nil, err
	}

	s3WebsiteEndpoint := getBucketWebsiteDomain(region, bucketName)
	input := cloudfront.CreateDistributionWithTagsInput{
		DistributionConfigWithTags: &types.DistributionConfigWithTags{
			Tags: &types.Tags{
				Items: []types.Tag{
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
			DistributionConfig: &types.DistributionConfig{
				CallerReference: originId,
				Comment:         aws.String(""),
				DefaultCacheBehavior: &types.DefaultCacheBehavior{
					TargetOriginId:       originId,
					ViewerProtocolPolicy: types.ViewerProtocolPolicyRedirectToHttps,
					CachePolicyId:        aws.String("658327ea-f89d-4fab-a63d-7e88639e58f6"), // https://github.com/aws-cloudformation/cloudformation-coverage-roadmap/issues/1602
				},
				Enabled: aws.Bool(true),

				Origins: &types.Origins{
					Items: []types.Origin{{
						DomainName: &s3WebsiteEndpoint,
						Id:         originId,
						CustomOriginConfig: &types.CustomOriginConfig{
							HTTPPort:             aws.Int32(80),
							HTTPSPort:            aws.Int32(443),
							OriginProtocolPolicy: types.OriginProtocolPolicyHttpOnly,
						},
					}},
					Quantity: aws.Int32(1),
				},

				Aliases: &types.Aliases{
					Quantity: aws.Int32(1),
					Items:    []string{*domain},
				},
				ViewerCertificate: &types.ViewerCertificate{
					ACMCertificateArn: certArn,
					SSLSupportMethod:  types.SSLSupportMethodSniOnly,
				},
			},
		},
	}
	res, err := client.CreateDistributionWithTags(ctx, &input)
	if err != nil {
		return nil, nil, err
	}
	return res.Distribution.DomainName, res.Distribution.Id, nil
}

func disableCloudfront(region *string, arn *string) error {
	res := strings.Split(*arn, "distribution/")
	if len(res) != 2 {
		return errors.New(fmt.Sprintf("arn %s is not a valid cloudfron arn", *arn))
	}
	client, err := initCloudfrontClient(region)
	if err != nil {
		return err
	}
	config, err := client.GetDistributionConfig(ctx, &cloudfront.GetDistributionConfigInput{Id: &res[1]})
	if err != nil {
		return err
	}
	config.DistributionConfig.Enabled = aws.Bool(false)
	input := cloudfront.UpdateDistributionInput{
		Id:                 &res[1],
		DistributionConfig: config.DistributionConfig,
		IfMatch:            config.ETag,
	}
	_, err = client.UpdateDistribution(ctx, &input)
	if err != nil {
		return err
	}
	return nil
}

func deleteCloudfront(region *string, arn *string) error {
	res := strings.Split(*arn, "distribution/")
	if len(res) != 2 {
		return errors.New(fmt.Sprintf("arn %s is not a valid cloudfron arn", *arn))
	}
	client, err := initCloudfrontClient(region)
	if err != nil {
		return err
	}

	config, err := client.GetDistributionConfig(ctx, &cloudfront.GetDistributionConfigInput{Id: &res[1]})
	if err != nil && strings.Contains(err.Error(), "The specified distribution does not exist") {
		return nil
	} else if err != nil {
		return err
	}
	_, err = client.DeleteDistribution(ctx, &cloudfront.DeleteDistributionInput{Id: &res[1], IfMatch: config.ETag})
	if err != nil {
		return err
	}
	return nil
}
