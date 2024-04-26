package aws

import (
	"bytes"
	"embed"
	_ "embed"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/gabriel-vasile/mimetype"
	"io"
	"strings"
)

//go:embed embed/s3_website_policy
var s3WebsitePolicy string

//go:embed embed
var embedded embed.FS

func initS3Client(region *string) (*s3.Client, error) {
	config, err := initConfig(*region)
	if err != nil {
		return nil, err
	}
	return s3.NewFromConfig(config), nil
}

func headBucket(bucket *string, region *string) error {
	client, err := initS3Client(region)
	if err != nil {
		return err
	}
	input := s3.HeadBucketInput{
		Bucket: bucket,
	}
	_, err = client.HeadBucket(ctx, &input)
	if err != nil {
		return err
	}
	return nil
}

func createBucket(bucket *string, region *string) error {
	client, err := initS3Client(region)
	if err != nil {
		return err
	}
	input := s3.CreateBucketInput{
		Bucket: bucket,
		CreateBucketConfiguration: &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraint(*region),
		},
	}
	_, err = client.CreateBucket(ctx, &input)
	if err != nil {
		return err
	}
	return nil
}

func deletePublicAccessBlock(bucket *string, region *string) error {
	client, err := initS3Client(region)
	if err != nil {
		return err
	}
	input := s3.DeletePublicAccessBlockInput{Bucket: bucket}
	_, err = client.DeletePublicAccessBlock(ctx, &input)
	if err != nil {
		return err
	}
	return nil
}

func putBucketPolicy(bucket *string, region *string) error {
	client, err := initS3Client(region)
	if err != nil {
		return err
	}
	s3WebsitePolicy = strings.Replace(s3WebsitePolicy, "$BUCKET_NAME", *bucket, 2)
	input := s3.PutBucketPolicyInput{
		Bucket: bucket,
		Policy: &s3WebsitePolicy,
	}
	_, err = client.PutBucketPolicy(ctx, &input)
	if err != nil {
		return err
	}
	return nil
}

func putFolder(bucket *string, region *string, path string, removePath string) error {
	open, err := embedded.Open(path)
	if err != nil {
		return err
	}
	stat, err := open.Stat()
	if err != nil {
		return err
	}
	if stat.IsDir() {
		dir, err := embedded.ReadDir(path)
		if err != nil {
			return err
		}
		for _, entry := range dir {
			err := putFolder(bucket, region, path+"/"+entry.Name(), removePath)
			if err != nil {
				return err
			}
		}
	} else {
		content, err := embedded.ReadFile(path)
		if err != nil {
			return err
		}
		err = putObject(bucket, path, region, &content, removePath)
		if err != nil {
			return err
		}
	}
	return nil
}

func getMimeType(key *string, content *[]byte) string {
	if strings.HasSuffix(*key, ".js") {
		return "application/javascript"
	} else if strings.HasSuffix(*key, ".js.map") {
		return "binary/octet-stream"
	} else if strings.HasSuffix(*key, ".css") {
		return "text/css"
	}
	return mimetype.Detect(*content).String()
}

func putObject(bucket *string, key string, region *string, content *[]byte, removePath string) error {
	key = strings.Replace(key, removePath, "", 1)
	client, err := initS3Client(region)
	if err != nil {
		return err
	}
	contentLength := int64(len(*content))
	contentType := getMimeType(&key, content)
	input := s3.PutObjectInput{
		Bucket:        bucket,
		Key:           &key,
		Body:          io.Reader(bytes.NewReader(*content)),
		ContentLength: &contentLength,
		ContentType:   &contentType,
	}
	_, err = client.PutObject(ctx, &input)
	if err != nil {
		return err
	}
	return nil
}

func putBucketWebsite(bucket *string, region *string) error {
	client, err := initS3Client(region)
	if err != nil {
		return err
	}
	var config types.WebsiteConfiguration
	if strings.HasPrefix(*bucket, "www.") {
		redirectHost := strings.Replace(*bucket, "www.", "", 1)
		config = types.WebsiteConfiguration{
			RedirectAllRequestsTo: &types.RedirectAllRequestsTo{
				HostName: &redirectHost,
				Protocol: types.ProtocolHttps,
			},
		}
	} else {
		config = types.WebsiteConfiguration{
			ErrorDocument: &types.ErrorDocument{Key: aws.String("index.html")},
			IndexDocument: &types.IndexDocument{Suffix: aws.String("index.html")},
		}
	}
	input := s3.PutBucketWebsiteInput{
		Bucket:               bucket,
		WebsiteConfiguration: &config,
	}
	_, err = client.PutBucketWebsite(ctx, &input)
	if err != nil {
		return err
	}
	return nil
}

func Example() error {
	name := "bizstrap.io"
	region := "us-west-1"
	err := createBucket(&name, &region)
	if err != nil {
		return err
	}
	err = deletePublicAccessBlock(&name, &region)
	if err != nil {
		return err
	}
	err = putBucketPolicy(&name, &region)
	if err != nil {
		return err
	}
	err = putFolder(&name, &region, "embed/vue/dist", "embed/vue/dist/")
	if err != nil {
		return err
	}
	err = putBucketWebsite(&name, &region)
	if err != nil {
		return err
	}
	return nil
}
