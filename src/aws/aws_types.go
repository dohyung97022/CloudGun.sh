package aws

type Image struct {
	name        string
	description string
}

var (
	AmazonLinux2 = Image{name: "amzn2-ami-ecs-hvm-2.0.20240424-x86_64-ebs", description: "Amazon Linux AMI 2.0.20240424 x86_64 ECS HVM GP2"}
)

type ResourceIdentifier string

var (
	EC2Instance      ResourceIdentifier = "AWS::EC2::Instance"
	EC2SecurityGroup ResourceIdentifier = "AWS::ECS::SecurityGroup"

	ECSService          ResourceIdentifier = "AWS::ECS::Service"
	ECSTaskDefinition   ResourceIdentifier = "AWS::ECS::TaskDefinition"
	ECSCapacityProvider ResourceIdentifier = "AWS::ECS::CapacityProvider"
	ECSCluster          ResourceIdentifier = "AWS::ECS::Cluster"

	ECRRepository ResourceIdentifier = "AWS::ECR::Repository"

	S3Bucket ResourceIdentifier = "AWS::S3::Bucket"

	CertificateManagerCertificate ResourceIdentifier = "AWS::CertificateManager::Certificate"

	CloudFrontDistribution ResourceIdentifier = "AWS::CloudFront::Distribution"

	ElasticLoadBalancingListener     ResourceIdentifier = "AWS::ElasticLoadBalancingV2::Listener"
	ElasticLoadBalancingLoadBalancer ResourceIdentifier = "AWS::ElasticLoadBalancingV2::LoadBalancer"
	ElasticLoadBalancingTargetGroup  ResourceIdentifier = "AWS::ElasticLoadBalancingV2::TargetGroup"
)
