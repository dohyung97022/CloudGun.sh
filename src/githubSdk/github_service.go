package githubSdk

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"github.com/google/go-github/v61/github"
	"strings"
	"time"
)

type Client github.Client

//go:embed all:embed
var embedded embed.FS
var ctx context.Context
var client *Client
var user *github.User

func init() {
	ctx, _ = context.WithTimeout(context.Background(), 720*time.Second)
}
func InitClient(accessToken *string) error {
	client = (*Client)(github.NewClient(nil).WithAuthToken(*accessToken))
	result, resp, err := client.Users.Get(ctx, "")
	if err != nil {
		return err
	}
	auth := resp.Response.Header.Get("X-Oauth-Scopes")
	if !strings.Contains(auth, "repo") {
		return errors.New("github token does not have repository authorization")
	} else if !strings.Contains(auth, "workflow") {
		return errors.New("github token does not have workflow authorization")
	}
	user = result
	return nil
}

func CreateS3WebsiteRepository(region *string, repoName *string, bucketName *string, awsAccessKey *string,
	awsSecretAccessKey *string, cloudFrontDistributionId *string, template FrontendTemplate, commitMessage *string, branch *string) error {
	fmt.Println("createRepository")
	err := client.createRepository("", *repoName)
	if err != nil { // 404 라면 권한이 없는 것일 수도 있다.
		return err
	}
	fmt.Println("createReadme")
	err = client.createReadme(repoName, "", commitMessage, branch)
	if err != nil {
		return err
	}
	fmt.Println("saveSecret")
	err = client.saveSecret(*repoName, "AWS_ACCESS_KEY_ID", *awsAccessKey)
	if err != nil {
		return err
	}
	fmt.Println("saveSecret")
	err = client.saveSecret(*repoName, "AWS_SECRET_ACCESS_KEY", *awsSecretAccessKey)
	if err != nil {
		return err
	}
	fmt.Println("saveSecret")
	err = client.saveSecret(*repoName, "AWS_CLOUDFRONT_DISTRIBUTION_ID", *cloudFrontDistributionId) // E265G1FI21SHCH
	if err != nil {
		return err
	}
	fmt.Println("saveSecret")
	err = client.saveSecret(*repoName, "AWS_REGION", *region)
	if err != nil {
		return err
	}
	fmt.Println("saveSecret")
	err = client.saveSecret(*repoName, "AWS_BUCKET_NAME", *bucketName)
	if err != nil {
		return err
	}
	fmt.Println("createFolder")
	entries := make([]*github.TreeEntry, 0)
	err = client.createFolder(embedded, *repoName, &entries, template.path, template.removePath, &template.gitIgnore)
	if err != nil {
		return err
	}
	fmt.Println("getBranch")
	repoCommit, baseTree, err := client.getBranch(repoName, branch)
	if err != nil {
		return err
	}
	fmt.Println("createBlobTree")
	createdTree, err := client.createBlobTree(repoName, baseTree, &entries)
	if err != nil { // 404 라면 workflow 권한이 없을 수도 있다.
		return err
	}
	fmt.Println("createCommit")
	createdCommit, err := client.createCommit(repoName, commitMessage, createdTree, &github.Commit{SHA: repoCommit.SHA})
	if err != nil {
		return err
	}
	fmt.Println("updateRef")
	err = client.updateRef(repoName, createdCommit)
	if err != nil {
		return err
	}
	return nil
}

func CreateCodeRepository(region *string, awsAccessKey *string, awsSecretAccessKey *string, ecrName *string,
	clusterName *string, serviceName *string, taskFamilyName *string, containerName *string, repoName *string,
	branch *string, template BackendTemplate) error {
	commitMessage := "good first commit from codeTemplate"
	fmt.Println("createRepository")
	err := client.createRepository("", *repoName)
	fmt.Println("createReadme")
	err = client.createReadme(repoName, "", &commitMessage, branch)
	if err != nil {
		return err
	}
	fmt.Println("saveSecret")
	err = client.saveSecret(*repoName, "AWS_ACCESS_KEY_ID", *awsAccessKey)
	if err != nil {
		return err
	}
	fmt.Println("saveSecret")
	err = client.saveSecret(*repoName, "AWS_SECRET_ACCESS_KEY", *awsSecretAccessKey)
	if err != nil {
		return err
	}
	fmt.Println("saveSecret")
	err = client.saveSecret(*repoName, "AWS_REGION", *region)
	if err != nil {
		return err
	}
	fmt.Println("saveSecret")
	err = client.saveSecret(*repoName, "AWS_ECR_REPOSITORY", *ecrName)
	if err != nil {
		return err
	}
	fmt.Println("saveSecret")
	err = client.saveSecret(*repoName, "AWS_ECS_CLUSTER", *clusterName)
	if err != nil {
		return err
	}
	fmt.Println("saveSecret")
	err = client.saveSecret(*repoName, "AWS_ECS_SERVICE", *serviceName)
	if err != nil {
		return err
	}
	fmt.Println("saveSecret")
	err = client.saveSecret(*repoName, "AWS_ECS_TASK_DEFINITION", *taskFamilyName)
	if err != nil {
		return err
	}
	fmt.Println("saveSecret")
	err = client.saveSecret(*repoName, "AWS_ECS_TASK_CONTAINER_NAME", *containerName)
	if err != nil {
		return err
	}
	fmt.Println("createFolder")
	entries := make([]*github.TreeEntry, 0)
	err = client.createFolder(embedded, *repoName, &entries, template.path, template.removePath, &template.gitIgnore)
	if err != nil {
		return err
	}
	fmt.Println("getBranch")
	repoCommit, baseTree, err := client.getBranch(repoName, branch)
	if err != nil {
		return err
	}
	fmt.Println("createBlobTree")
	createdTree, err := client.createBlobTree(repoName, baseTree, &entries)
	if err != nil {
		return err
	}
	fmt.Println("createCommit")
	createdCommit, err := client.createCommit(repoName, &commitMessage, createdTree, &github.Commit{SHA: repoCommit.SHA})
	if err != nil {
		return err
	}
	fmt.Println("updateRef")
	err = client.updateRef(repoName, createdCommit)
	if err != nil {
		return err
	}
	return nil
}
