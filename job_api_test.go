package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	retryCount = 10
)

func TestAPI(t *testing.T) {
	wsContainer := initWSContainer()

	ts := httptest.NewServer(wsContainer)
	defer ts.Close()

	jobURL := fmt.Sprintf("%s/%s", ts.URL, "jobs")

	jobPOST := createJob(t, jobURL, exampleJobBody)

	assert.Equal(t, "queued", string(jobPOST.Status), "Status should be queued")
	assert.Equal(t, exampleJob.ImageName, jobPOST.ImageName, "Image name should match")
	assert.Equal(t, exampleJob.Cmds, jobPOST.Cmds, "Commands should match")

	// wait while the job completes
	waitUntilDone(t, jobURL, jobPOST.ID)
	jobGET := getJob(t, jobURL, jobPOST.ID)
	assert.Equal(t, "successful", string(jobGET.Status), "Status should be successful")
}

func createJob(t *testing.T, jobURL string, body string) *Job {
	resp, err := http.Post(jobURL, "application/json", strings.NewReader(body))
	if err != nil {
		t.Errorf("Error sending post request: %s", err)
	}
	assert.Equal(t, http.StatusCreated, resp.StatusCode, "Status code should be 201")
	return decodeBody(t, resp.Body)
}

func getJob(t *testing.T, jobURL string, jobID JobID) *Job {
	resp, err := http.Get(fmt.Sprintf("%s/%d", jobURL, jobID))
	if err != nil {
		t.Errorf("Error sending get request: %s", err)
	}
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Status code should be 200")

	return decodeBody(t, resp.Body)
}

func waitUntilDone(t *testing.T, jobURL string, jobID JobID) {
	for i := 0; i < retryCount; i++ {
		jobGET := getJob(t, jobURL, jobID)
		if jobGET.Status != "running" {
			return
		}
		time.Sleep(1 * time.Second)
	}
	t.Fatal("Waiting too long for job to complete ")
}

func decodeBody(t *testing.T, respBody io.ReadCloser) *Job {
	body, err := ioutil.ReadAll(respBody)
	defer respBody.Close()
	if err != nil {
		t.Errorf("Error reading response body: %s", err)
	}

	job := &Job{}
	err = json.Unmarshal(body, job)
	if err != nil {
		t.Errorf("Error decoding response body: %s", err)
	}

	return job
}

var exampleJob = Job{
	ImageName: "ubuntu:14.04",
	Cmds: []Cmd{
		[]string{"sh", "-c", "echo \"test\" > /test.txt"},
		[]string{"sleep", "1"},
		[]string{"cat", "/test.txt"},
	},
}

const exampleJobBody = `{
  "image": "ubuntu:14.04",
  "cmds": [
    ["sh", "-c", "echo \"test\" > /test.txt"],
    ["sleep", "1"],
    ["cat", "/test.txt"]
  ]
}
`
