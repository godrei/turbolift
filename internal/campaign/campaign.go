/*
 * Copyright 2021 Skyscanner Limited.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 * http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package campaign

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type Repo struct {
	Host         string
	OrgName      string
	RepoName     string
	FullRepoName string // expected in a [host/]owner/repo
	BranchName   string
}

type Campaign struct {
	Name    string
	Repos   []Repo
	PrTitle string
	PrBody  string
}

func (r Repo) DirName() string {
	if r.BranchName == "" {
		return r.RepoName

	}

	return r.RepoName + "-" + r.BranchName
}

func (r Repo) VisibleName() string {
	if r.BranchName == "" {
		return r.FullRepoName
	}
	return r.FullRepoName + "@" + r.BranchName
}

func (r Repo) FullRepoPath() string {
	repoName := r.RepoName
	if r.BranchName != "" {
		repoName += "-" + r.BranchName
	}
	return path.Join("work", r.OrgName, repoName) // i.e. work/org/repo
}

type CampaignOptions struct {
	RepoFilename string
}

func NewCampaignOptions() *CampaignOptions {
	return &CampaignOptions{RepoFilename: "repos.txt"}
}

func OpenCampaign(options *CampaignOptions) (*Campaign, error) {
	dir, _ := os.Getwd()
	dirBasename := filepath.Base(dir)

	repos, err := readReposTxtFile(options.RepoFilename)
	if err != nil {
		return nil, err
	}

	prTitle, prBody, err := readPrDescriptionFile()
	if err != nil {
		return nil, err
	}

	return &Campaign{
		Name:    dirBasename,
		Repos:   repos,
		PrTitle: prTitle,
		PrBody:  prBody,
	}, nil
}

func readReposTxtFile(filename string) ([]Repo, error) {
	if filename == "" {
		return nil, errors.New("no repos filename to open")
	}
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("unable to open repo file: %s", filename)
	}
	defer func() {
		closeErr := file.Close()
		if err == nil {
			err = closeErr
		}
	}()

	scanner := bufio.NewScanner(file)
	uniq := map[string]interface{}{}
	var repos []Repo
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "#") && len(line) > 0 {
			if _, seen := uniq[line]; seen {
				continue
			}
			uniq[line] = struct{}{}

			var host, orgName, repoName string
			splitLine := strings.Split(line, "/")

			switch len(splitLine) {
			case 2:
				orgName, repoName = splitLine[0], splitLine[1]
			case 3:
				host, orgName, repoName = splitLine[0], splitLine[1], splitLine[2]
			default:
				return nil, fmt.Errorf("unable to parse entry in %s file: %s", filename, line)
			}

			var fullRepoName, branchName string
			splitRepoName := strings.Split(repoName, "@")

			switch len(splitRepoName) {
			case 1:
				fullRepoName = line
			case 2:
				repoName = splitRepoName[0]
				branchName = splitRepoName[1]
				fullRepoName = strings.TrimSuffix(line, "@"+branchName)
			}

			repo := Repo{
				Host:         host,
				OrgName:      orgName,
				RepoName:     repoName,
				FullRepoName: fullRepoName,
				BranchName:   branchName,
			}

			repos = append(repos, repo)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("unable to open %s file: %w", filename, err)
	}

	return repos, nil
}

func readPrDescriptionFile() (string, string, error) {
	file, err := os.Open("README.md")
	if err != nil {
		return "", "", errors.New("unable to open README.md file")
	}
	defer func() {
		closeErr := file.Close()
		if err == nil {
			err = closeErr
		}
	}()

	scanner := bufio.NewScanner(file)
	prTitle := ""
	prBodyLines := []string{}
	for scanner.Scan() {
		line := scanner.Text()

		if prTitle == "" {
			trimmedFirstLine := strings.TrimLeft(line, "# ")
			prTitle = trimmedFirstLine
		} else {
			prBodyLines = append(prBodyLines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return "", "", errors.New("unable to read README.md file")
	}

	return prTitle, strings.Join(prBodyLines, "\n"), nil
}
