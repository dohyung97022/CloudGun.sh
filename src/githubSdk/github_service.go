package githubSdk

import (
	"context"
	"embed"
	"fmt"
	"github.com/google/go-github/v61/github"
	"os"
	"strings"
	"time"
)

type Client github.Client

var ctx context.Context

//go:embed embed
var external embed.FS

func init() {
	ctx, _ = context.WithTimeout(context.Background(), 30*time.Second)
}

func Example() {
	token := ""
	repoName := "test"
	owner := "dohyung97022"
	client, err := initClient(&token)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	err = client.createRepository("", repoName)
	if err != nil { // 404 라면 권한이 없는 것일 수도 있다.
		fmt.Println(err)
		os.Exit(1)
	}

	err = client.createFolder(external, owner, repoName, "embed/go/src", "embed/go/")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func initClient(accessToken *string) (*Client, error) {
	return (*Client)(github.NewClient(nil).WithAuthToken(*accessToken)), nil
}

func (client *Client) createRepository(organization string, repoName string) error {
	repo := github.Repository{Name: &repoName}
	_, _, err := client.Repositories.Create(ctx, organization, &repo)
	return err
}

func (client *Client) createFolder(embedded embed.FS, owner string, repoName string, path string, removePath string) error {
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
			err := client.createFolder(embedded, owner, repoName, path+"/"+entry.Name(), removePath)
			if err != nil {
				return err
			}
		}
	} else {
		content, err := external.ReadFile(path)
		if err != nil {
			return err
		}
		err = client.createFile(owner, repoName, path, content, removePath)
		if err != nil {
			return err
		}
	}
	return nil
}

func (client *Client) createFile(owner string, repoName string, gitPath string, content []byte, removePath string) error {
	gitPath = strings.Replace(gitPath, removePath, "", 1)
	options := github.RepositoryContentFileOptions{
		Message: github.String("testing123"),
		Content: content,
		Branch:  github.String("main"),
	}
	_, _, err := client.Repositories.CreateFile(ctx, owner, repoName, gitPath, &options)
	return err
}
