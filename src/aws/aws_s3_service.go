package aws

import (
	"bytes"
	"embed"
	_ "embed"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3Types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/gabriel-vasile/mimetype"
	"io"
	"log"
	"strings"
)

//go:embed embed/s3_website_policy
var s3WebsitePolicy string

//go:embed embed
var embedded embed.FS

func initS3Client(region *string) (*s3.Client, error) {
	config, err := initConfig(region)
	if err != nil {
		return nil, err
	}
	return s3.NewFromConfig(config), nil
}

func createBucket(bucket *string, region *string) error {
	client, err := initS3Client(region)
	if err != nil {
		return err
	}
	input := s3.CreateBucketInput{
		Bucket: bucket,
		CreateBucketConfiguration: &s3Types.CreateBucketConfiguration{
			LocationConstraint: s3Types.BucketLocationConstraint(*region),
		},
	}
	_, err = client.CreateBucket(ctx, &input)
	if err != nil {
		return err
	}

	taggingInput := s3.PutBucketTaggingInput{
		Bucket: bucket,
		Tagging: &s3Types.Tagging{
			TagSet: []s3Types.Tag{
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
	}
	_, err = client.PutBucketTagging(ctx, &taggingInput)
	if err != nil {
		return err
	}
	return nil
}

func deleteBucket(region *string, arn *string) error {
	name := strings.TrimPrefix(*arn, "arn:aws:s3:::")
	client, err := initS3Client(region)

	deleteObject := func(bucket, key, versionId *string) error {
		_, err := client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket:    bucket,
			Key:       key,
			VersionId: versionId,
		})
		if err != nil {
			return err
		}
		return nil
	}

	in := &s3.ListObjectsV2Input{Bucket: &name}
	for {
		out, err := client.ListObjectsV2(ctx, in)
		if err != nil {
			log.Fatalf("Failed to list objects: %v", err)
		}

		for _, item := range out.Contents {
			err := deleteObject(&name, item.Key, nil)
			if err != nil {
				return err
			}
		}

		if *out.IsTruncated {
			in.ContinuationToken = out.ContinuationToken
		} else {
			break
		}
	}

	inVer := &s3.ListObjectVersionsInput{Bucket: &name}
	for {
		out, err := client.ListObjectVersions(ctx, inVer)
		if err != nil {

		}

		for _, item := range out.DeleteMarkers {
			err := deleteObject(&name, item.Key, item.VersionId)
			if err != nil {
				return err
			}
		}

		for _, item := range out.Versions {
			err := deleteObject(&name, item.Key, item.VersionId)
			if err != nil {
				return err
			}
		}

		if *out.IsTruncated {
			inVer.VersionIdMarker = out.NextVersionIdMarker
			inVer.KeyMarker = out.NextKeyMarker
		} else {
			break
		}
	}

	_, err = client.DeleteBucket(ctx, &s3.DeleteBucketInput{Bucket: &name})
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
	policy := strings.Replace(s3WebsitePolicy, "$BUCKET_NAME", *bucket, 2)
	input := s3.PutBucketPolicyInput{
		Bucket: bucket,
		Policy: &policy,
	}
	_, err = client.PutBucketPolicy(ctx, &input)
	if err != nil {
		return err
	}
	return nil
}

func uploadFolder(bucket *string, region *string, path string, removePath string) error {
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
			err := uploadFolder(bucket, region, path+"/"+entry.Name(), removePath)
			if err != nil {
				return err
			}
		}
	} else {
		content, err := embedded.ReadFile(path)
		if err != nil {
			return err
		}
		err = uploadObject(bucket, path, region, &content, removePath)
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

func uploadObject(bucket *string, key string, region *string, content *[]byte, removePath string) error {
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
	var config s3Types.WebsiteConfiguration
	if strings.HasPrefix(*bucket, "www.") {
		redirectHost := strings.Replace(*bucket, "www.", "", 1)
		config = s3Types.WebsiteConfiguration{
			RedirectAllRequestsTo: &s3Types.RedirectAllRequestsTo{
				HostName: &redirectHost,
				Protocol: s3Types.ProtocolHttps,
			},
		}
	} else {
		config = s3Types.WebsiteConfiguration{
			ErrorDocument: &s3Types.ErrorDocument{Key: aws.String("index.html")},
			IndexDocument: &s3Types.IndexDocument{Suffix: aws.String("index.html")},
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

func getBucketWebsiteDomain(region *string, bucketName *string) string {
	return fmt.Sprintf("%s.s3-website.%s.amazonaws.com", *bucketName, *region)
}
