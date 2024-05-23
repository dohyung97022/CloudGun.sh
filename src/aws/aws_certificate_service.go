package aws

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/acm"
	"github.com/aws/aws-sdk-go-v2/service/acm/types"
	"time"
)

func initCertificateClient(region *string) (*acm.Client, error) {
	config, err := initConfig(region)
	if err != nil {
		return nil, err
	}
	return acm.NewFromConfig(config), nil
}

func requestCertificate(domain *string, domains *[]string, region *string, retry int) (*string, error) {
	client, err := initCertificateClient(region)
	if err != nil {
		return nil, err
	}
	requestCertInput := acm.RequestCertificateInput{
		DomainName:              aws.String(*domain),
		ValidationMethod:        types.ValidationMethodDns,
		SubjectAlternativeNames: *domains,
		Tags: []types.Tag{
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
	certificate, err := client.RequestCertificate(ctx, &requestCertInput)
	if err != nil {
		return nil, err
	}

	// wait for aws to give records to be set
	var describeCertOutput *acm.DescribeCertificateOutput
	describeCertInput := acm.DescribeCertificateInput{CertificateArn: certificate.CertificateArn}
	for i := 0; i < retry; i++ {
		describeCertOutput, err = client.DescribeCertificate(ctx, &describeCertInput)
		if err != nil {
			return nil, err
		}
		isResourceRecordSet := true
		for _, options := range describeCertOutput.Certificate.DomainValidationOptions {
			if options.ResourceRecord == nil {
				time.Sleep(time.Second)
				isResourceRecordSet = false
			}
		}
		if isResourceRecordSet == true {
			break
		}
	}

	// setup records
	for _, options := range describeCertOutput.Certificate.DomainValidationOptions {
		if options.ValidationStatus == types.DomainStatusSuccess {
			continue
		}
		if options.ResourceRecord == nil {
			return nil, errors.New("DomainValidationOptions[].ResourceRecord is not set from aws after retry")
		}
		fmt.Println("createCertificateRecord")
		err = createCertificateRecord(region, domain, options.ResourceRecord.Name, options.ResourceRecord.Value)
		if err != nil {
			return nil, err
		}
	}
	return certificate.CertificateArn, nil
}

func waitCertificateIssued(region *string, certificateArn *string, retry int) error {
	client, err := initCertificateClient(region)
	if err != nil {
		return err
	}
	var describeCertOutput *acm.DescribeCertificateOutput
	describeCertInput := acm.DescribeCertificateInput{CertificateArn: certificateArn}
	for i := 0; i < retry; i++ {
		fmt.Println(fmt.Sprintf("checking if acm certificate is issued retry %d", i))
		describeCertOutput, err = client.DescribeCertificate(ctx, &describeCertInput)
		if err != nil {
			return nil
		}
		if describeCertOutput.Certificate.Status != types.CertificateStatusIssued {
			time.Sleep(time.Second)
			continue
		}
		break
	}
	if describeCertOutput.Certificate.Status != types.CertificateStatusIssued {
		return errors.New(fmt.Sprintf("certificate %s is not yet issued by aws", *certificateArn))
	}
	return nil
}

func deleteCertificate(region *string, arn *string) error {
	client, err := initCertificateClient(region)
	if err != nil {
		return err
	}
	_, err = client.DeleteCertificate(ctx, &acm.DeleteCertificateInput{CertificateArn: arn})
	if err != nil {
		return err
	}
	return nil
}
