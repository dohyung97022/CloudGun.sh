package githubSdk

import (
	"embed"
	"encoding/base64"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/google/go-github/v61/github"
	"strings"
)

func (client *Client) createRepository(organization string, repoName string) error {
	repo := github.Repository{Name: &repoName}
	_, _, err := client.Repositories.Create(ctx, organization, &repo)
	return err
}

func (client *Client) createReadme(repoName *string, content string, message *string, branch *string) error {
	options := github.RepositoryContentFileOptions{
		Message: message,
		Content: []byte(content),
		Branch:  branch,
	}
	_, _, err := client.Repositories.CreateFile(ctx, *user.Login, *repoName, "README.md", &options)
	return err
}

func (client *Client) createFolder(embedded embed.FS, repoName string, blobs *[]*github.TreeEntry, path string, removePath string, gitIgnore *[]string) error {
	open, err := embedded.Open(path)
	if err != nil {
		return err
	}
	stat, err := open.Stat()
	for _, ignore := range *gitIgnore {
		if stat.Name() == ignore {
			return nil
		}
	}
	if err != nil {
		return err
	}
	if stat.IsDir() {
		dir, err := embedded.ReadDir(path)
		if err != nil {
			return err
		}
		for _, entry := range dir {
			err := client.createFolder(embedded, repoName, blobs, path+"/"+entry.Name(), removePath, gitIgnore)
			if err != nil {
				return err
			}
		}
	} else {
		content, err := embedded.ReadFile(path)
		if err != nil {
			return err
		}
		err = client.createFileBlob(repoName, content, blobs, path, removePath)
		if err != nil {
			return err
		}
	}
	return nil
}

func (client *Client) createFileBlob(repoName string, content []byte, entries *[]*github.TreeEntry, gitPath string, removePath string) error {
	gitPath = strings.Replace(gitPath, removePath, "", 1)
	gitPath = strings.TrimPrefix(gitPath, "/")

	var input github.Blob
	if strings.HasSuffix(gitPath, ".png") || strings.HasSuffix(gitPath, ".jpg") || strings.HasSuffix(gitPath, ".ico") {
		base64Content := base64.StdEncoding.EncodeToString(content)
		input = github.Blob{
			Encoding: github.String("base64"),
			Content:  github.String(base64Content),
			Size:     aws.Int(len(base64Content)),
		}
	} else {
		input = github.Blob{
			Encoding: github.String("utf-8"),
			Content:  github.String(string(content)),
			Size:     aws.Int(len(content)),
		}
	}

	blob, r, err := client.Git.CreateBlob(ctx, *user.Login, repoName, &input)
	if err != nil || r.Response.StatusCode != 201 {
		return err
	}
	entry := github.TreeEntry{
		SHA:  blob.SHA,
		Path: &gitPath,
		Mode: aws.String("100644"),
		Type: aws.String("blob"),
	}
	*entries = append(*entries, &entry)
	return nil
}
func (client *Client) getBranch(repoName *string, branchName *string) (*github.RepositoryCommit, *github.Tree, error) {
	branch, r, err := client.Repositories.GetBranch(ctx, *user.Login, *repoName, *branchName, 3)
	if err != nil || r.Response.StatusCode != 200 {
		return nil, nil, err
	}
	return branch.Commit, branch.Commit.Commit.Tree, nil
}

func (client *Client) createBlobTree(repoName *string, baseTree *github.Tree, entries *[]*github.TreeEntry) (*github.Tree, error) {
	tree, _, err := client.Git.CreateTree(ctx, *user.Login, *repoName, *baseTree.SHA, *entries)
	if err != nil {
		return tree, err
	}
	return tree, nil
}

func (client *Client) createCommit(repoName *string, message *string, blobTree *github.Tree, parentCommit *github.Commit) (*github.Commit, error) {
	comment := github.Commit{
		Tree:    &github.Tree{SHA: blobTree.SHA},
		Message: message,
		Parents: []*github.Commit{parentCommit},
	}
	commit, _, err := client.Git.CreateCommit(ctx, *user.Login, *repoName, &comment, nil)
	if err != nil {
		return nil, err
	}
	return commit, nil
}

func (client *Client) updateRef(repoName *string, createdCommit *github.Commit) error {
	ref := github.Reference{
		Ref:    aws.String("refs/heads/main"),
		Object: &github.GitObject{SHA: createdCommit.SHA},
	}
	_, _, err := client.Git.UpdateRef(ctx, *user.Login, *repoName, &ref, false)
	if err != nil {
		return err
	}
	return nil
}
