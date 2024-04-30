package aws

import (
	"errors"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/acm"
	"github.com/aws/aws-sdk-go-v2/service/acm/types"
	"strings"
)

func initCertificateClient(region *string) (*acm.Client, error) {
	config, err := initConfig(region)
	if err != nil {
		return nil, err
	}
	return acm.NewFromConfig(config), nil
}

func retryDescribeCertificate(client *acm.Client, retry int, certificateArn *string) (*acm.DescribeCertificateOutput, error) {
	describeCertInput := acm.DescribeCertificateInput{CertificateArn: certificateArn}
	for i := 0; i < retry; i++ {
		describeCertOutput, err := client.DescribeCertificate(ctx, &describeCertInput)
		if err == nil {
			return describeCertOutput, err
		}
	}
	return client.DescribeCertificate(ctx, &describeCertInput)
}

func RequestCertificate(domain *string) (*string, error) {
	region := "us-east-1"
	if strings.HasPrefix(*domain, "www.") {
		return nil, errors.New(*domain + " domain should not start with www.")
	}
	client, err := initCertificateClient(&region)
	if err != nil {
		return nil, err
	}
	requestCertInput := acm.RequestCertificateInput{
		DomainName:              aws.String(*domain),
		ValidationMethod:        types.ValidationMethodDns,
		SubjectAlternativeNames: []string{*domain, "www." + *domain},
	}
	certificate, err := client.RequestCertificate(ctx, &requestCertInput)
	if err != nil {
		return nil, err
	}

	describeCertOutput, err := retryDescribeCertificate(client, 30, certificate.CertificateArn)
	if err != nil {
		return nil, err
	}

	for _, validationOption := range describeCertOutput.Certificate.DomainValidationOptions {
		err = createCertificateRecord(&region, domain, validationOption.ResourceRecord.Name, validationOption.ResourceRecord.Value)
		if err != nil {
			return nil, err
		}
	}
	return certificate.CertificateArn, nil
}
