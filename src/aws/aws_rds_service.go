package aws

import (
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
)

func initRDSClient(region *string) (*rds.Client, error) {
	config, err := initConfig(region)
	if err != nil {
		return nil, err
	}
	return rds.NewFromConfig(config), nil
}

func createRDS(region *string, name *string, storage *int32, username *string) error {
	client, err := initRDSClient(region)
	if err != nil {
		return err
	}
	input := rds.CreateDBInstanceInput{
		DBInstanceClass:          aws.String("db.t3.micro"),
		DBInstanceIdentifier:     name,
		DBName:                   name,
		Engine:                   aws.String("mysql"),
		AllocatedStorage:         storage,
		AutoMinorVersionUpgrade:  aws.Bool(false),
		BackupRetentionPeriod:    aws.Int32(10),
		BackupTarget:             aws.String("region"),
		EngineVersion:            aws.String("8.0.34"),
		ManageMasterUserPassword: aws.Bool(true),
		MasterUsername:           username,
	}
	instance, err := client.CreateDBInstance(ctx, &input)
	if err != nil {
		return err
	}
	fmt.Println(instance)
	return nil
}
