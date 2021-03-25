package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"github.com/google/go-github/github"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

// Build the struct for the response from the build endpoint
// see https://docs.travis-ci.com/api/#builds in case this doesn't work anymore
type BuildItem struct {
	Builds []struct {
		ID                int    `json:"id"`
		RepositoryID      int    `json:"repository_id"`
		CommitID          int    `json:"commit_id,omitempty"`
		Number            string `json:"number,omitempty"`
		EventType         string `json:"event_type,omitempty"`
		PullRequest       bool   `json:"pull_request,omitempty"`
		PullRequestTitle  string `json:"pull_request_title,omitempty"`
		PullRequestNumber int    `json:"pull_request_number,omitempty"`
		Config            struct {
			Script   []string `json:"s ripte,omitempty`
			Result   string   `json:".result,omitempty"`
			Language string   `json:"language,omitempty"`
			Group    string   `json:"group,omitempty"`
			Dist     string   `json:"dist,omitempty"`
		} `json:"config"`
		State      string    `json:"state,omitempty"`
		StartedAt  time.Time `json:"started_at,omitempty"`
		FinishedAt time.Time `json:"finished_at,omitempty"`
		Duration   int       `json:"duration,omitempty"`
		JobIds     []int     `json:"job_ids,omitempty"`
	} `json:"builds"`
}

func main() {
	// List the command line flags and assign them to pointers
	orgPtr := flag.String("org", "", "the org to scan (this is case sensitive)")
	tokenPtr := flag.String("github-token", "", "GitHub oAuth token used for authentication with GitHub to not instantly get rate limited")
	travisTokenPtr := flag.String("travis-token", "", "Travis auth token you can get from https://travis-ci.org/account/preferences")
	expandUsersPtr := flag.Bool("expand", true, "By default travis-grabber expands to all mebers of an org, set to false to disable")
	// Parse the flags
	flag.Parse()
	// Make sure org and token are set
	if *orgPtr == "" {
		log.Fatal("You have to specify an org to scan!")
	}
	if *tokenPtr == "" {
		log.Fatal("You have to specify a GitHub token!")
	}
	if *travisTokenPtr == "" {
		log.Fatal("You have to specify a Travis token!")
	}
	// Print what we got so we know what we're scanning
	log.Info("Org to scan on Travis CI:", *orgPtr)
	// define wg
	var wg sync.WaitGroup
	// Set context
	ctx := context.Background()
	// Authenticate
	tokensource := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: *tokenPtr},
	)
	tokenclient := oauth2.NewClient(ctx, tokensource)
	// Start the client with authentication
	client := github.NewClient(tokenclient)
	// Define any options to use for GitHub
	// We want to poaginate
	optRepos := &github.RepositoryListByOrgOptions{
		ListOptions: github.ListOptions{PerPage: 10},
	}
	// Not pretty but we also want to paginated for the users
	optUsers := &github.ListMembersOptions{
		ListOptions: github.ListOptions{PerPage: 10},
	}
	optUserRepo := &github.RepositoryListOptions{
		ListOptions: github.ListOptions{PerPage: 10},
	}
	// If we want to expand, get all users in Org
	var orgMembers []*github.User
	if *expandUsersPtr == true {
		for {
			users, resp, err := client.Organizations.ListMembers(ctx, *orgPtr, optUsers)
			if err != nil {
				log.Fatal(err)
			}
			orgMembers = append(orgMembers, users...)
			if resp.NextPage == 0 {
				break
			}
			optUsers.Page = resp.NextPage
		}
		log.Info("Collected Members")
	}
	// Now get all Repos in that Org
	var allRepos []*github.Repository
	for {
		repos, resp, err := client.Repositories.ListByOrg(ctx, *orgPtr, optRepos)
		if err != nil {
			log.Fatal(err)
		}
		allRepos = append(allRepos, repos...)
		if resp.NextPage == 0 {
			break
		}
		optRepos.Page = resp.NextPage
	}
	if *expandUsersPtr == true {
		for _, user := range orgMembers {
			repos, resp, err := client.Repositories.List(ctx, user.GetLogin(), optUserRepo)
			if err != nil {
				log.Fatal(err)
			}
			allRepos = append(allRepos, repos...)
			if resp.NextPage == 0 {
				break
			}
			optRepos.Page = resp.NextPage
		}
	}
	log.Info("Collected Org Repos")
	// Remove comment to print the repos we have
	log.Debug(allRepos)

	// Define our URL
	baseUrl := "https://api.travis-ci.org/repos/"
	buildsPostfix := "/builds?limit=100"
	// Everything from here is happening for all repos in the given org
	for _, repo := range allRepos {
		buildsUrl := baseUrl + repo.GetFullName() + buildsPostfix
		// Let the user know from where we're getting the builds
		log.Info("Requesting builds from:", buildsUrl)

		// Request the builds
		tr := &http.Transport{
			MaxIdleConns:       10,
			IdleConnTimeout:    30 * time.Second,
			DisableCompression: true,
		}
		buildsClient := &http.Client{Transport: tr}
		reqBuilds, err := http.NewRequest("GET", buildsUrl, nil)
		if err != nil {
			log.Fatal(err)
		}
		// Set the header to define the Travis API version
		reqBuilds.Header.Add("Accept", "application/json; version=2")
		respBuilds, err := buildsClient.Do(reqBuilds)
		if err != nil {
			log.Fatal(err)
		}
		// Make some sense of those build responses
		bodyBuilds, err := ioutil.ReadAll(respBuilds.Body)
		if err != nil {
			log.Fatal(err)
		}
		// Make sure we use the right structure depending on the version
		buildInfo := BuildItem{}
		jsonErr := json.Unmarshal(bodyBuilds, &buildInfo)
		if jsonErr != nil {
			log.Info("Not an array of strings, trying single string struc")
			type BuildItem struct {
				Builds []struct {
					ID                int    `json:"id"`
					RepositoryID      int    `json:"repository_id"`
					CommitID          int    `json:"commit_id,omitempty"`
					Number            string `json:"number,omitempty"`
					EventType         string `json:"event_type,omitempty"`
					PullRequest       bool   `json:"pull_request,omitempty"`
					PullRequestTitle  string `json:"pull_request_title,omitempty"`
					PullRequestNumber int    `json:"pull_request_number,omitempty"`
					Config            struct {
						Script   string `json:"s ripte,omitempty`
						Result   string `json:".result,omitempty"`
						Language string `json:"language,omitempty"`
						Group    string `json:"group,omitempty"`
						Dist     string `json:"dist,omitempty"`
					} `json:"config"`
					State      string    `json:"state,omitempty"`
					StartedAt  time.Time `json:"started_at,omitempty"`
					FinishedAt time.Time `json:"finished_at,omitempty"`
					Duration   int       `json:"duration,omitempty"`
					JobIds     []int     `json:"job_ids,omitempty"`
				} `json:"builds"`
			}
			wg.Add(len(buildInfo.Builds))
			for index, build := range buildInfo.Builds {
				go func(index int) {
					defer wg.Done()
					log.Info("Gathering Jobs")
					log.Info("Build:", build.ID)
					log.Info("RepositoryID:", build.RepositoryID)
					// Request the logs for each build and dump them to files
					for index, job := range build.JobIds {
						go func(index int) {
							defer wg.Done()
							// Print the Jobs IDs
							log.Info("JobID:", strconv.Itoa(job))
							logString := strconv.Itoa(job)
							baseUrl := "https://api.travis-ci.org/v3/job/"
							logsPostfix := "/log.txt"
							logsUrl := baseUrl + logString + logsPostfix
							// Let the user know from where we're getting the logs
							log.Info("Requesting logs from:", logsUrl)

							// Request the Logs
							tr := &http.Transport{
								MaxIdleConns:       10,
								IdleConnTimeout:    30 * time.Second,
								DisableCompression: true,
							}
							logsClient := &http.Client{Transport: tr}
							reqLogs, err := http.NewRequest("GET", logsUrl, nil)
							if err != nil {
								log.Fatal(err)
							}
							// Set the header to define the Travis API version
							reqLogs.Header.Add("Accept", "text/plain; version=3")
							reqLogs.Header.Add("Authorization", "token "+*travisTokenPtr)
							respLogs, err := logsClient.Do(reqLogs)
							if err != nil {
								log.Fatal(err)
							}
							// Better use a buffer this time as we have to otherwise copy the whole byte array to conver it to string
							bodyBuffer := new(bytes.Buffer)
							bodyBuffer.ReadFrom(respLogs.Body)
							bodyString := bodyBuffer.String()

							// Write each log to file of format "repositoryID-buildID-logID.log"
							logFile, err := os.Create(strconv.Itoa(build.RepositoryID) + "-" + strconv.Itoa(build.ID) + "-" + logString + ".log")
							if err != nil {
								log.Fatal(err)
							}
							logLength, err := logFile.WriteString(bodyString)
							if err != nil {
								log.Fatal(err)
								logFile.Close()
							}
							log.Info(logLength, "bytes written successfully")

						}(index)
					}

				}(index)
			}
			wg.Wait()

		}
		// Uncomment to following to debug what you're getting back
		log.Debug(buildInfo)
		for _, build := range buildInfo.Builds {
			log.Info("Gathering Jobs")
			log.Info("Build:", build.ID)
			log.Info("RepositoryID:", build.RepositoryID)
			// Request the logs for each build and dump them to files
			for _, job := range build.JobIds {
				// Print the Jobs IDs
				log.Info("JobID:", strconv.Itoa(job))
				logString := strconv.Itoa(job)
				baseUrl := "https://api.travis-ci.org/v3/job/"
				logsPostfix := "/log.txt"
				logsUrl := baseUrl + logString + logsPostfix
				// Let the user know from where we're getting the logs
				log.Info("Requesting logs from:", logsUrl)

				// Request the Logs
				tr := &http.Transport{
					MaxIdleConns:       10,
					IdleConnTimeout:    30 * time.Second,
					DisableCompression: true,
				}
				logsClient := &http.Client{Transport: tr}
				reqLogs, err := http.NewRequest("GET", logsUrl, nil)
				if err != nil {
					log.Fatal(err)
				}
				// Set the header to define the Travis API version
				reqLogs.Header.Add("Accept", "text/plain; version=3")
				reqLogs.Header.Add("Authorization", "token "+*travisTokenPtr)
				respLogs, err := logsClient.Do(reqLogs)
				if err != nil {
					log.Fatal(err)
				}
				// Better use a buffer this time as we have to otherwise copy the whole byte array to conver it to string
				bodyBuffer := new(bytes.Buffer)
				bodyBuffer.ReadFrom(respLogs.Body)
				bodyString := bodyBuffer.String()

				// Write each log to file of format "repositoryID-buildID-logID.log"
				logFile, err := os.Create(strconv.Itoa(build.RepositoryID) + "-" + strconv.Itoa(build.ID) + "-" + logString + ".log")
				if err != nil {
					log.Fatal(err)
				}
				logLength, err := logFile.WriteString(bodyString)
				if err != nil {
					log.Fatal(err)
					logFile.Close()
				}
				log.Info(logLength, "bytes written successfully")

			}

		}
	}
	log.Info("Done!")
}
