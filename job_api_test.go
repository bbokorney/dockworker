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

type testCase struct {
	requestBody  string
	job          Job
	resultStatus JobStatus
}

var testcases = []testCase{
	testCase{
		requestBody: `{
  "image": "ubuntu:14.04",
  "cmds": [
    ["sh", "-c", "echo \"test\" > /test.txt"],
    ["sleep", "1"],
    ["cat", "/test.txt"]
  ]
}`,
		job: Job{
			ImageName: "ubuntu:14.04",
			Cmds: []Cmd{
				[]string{"sh", "-c", "echo \"test\" > /test.txt"},
				[]string{"sleep", "1"},
				[]string{"cat", "/test.txt"},
			},
			Results: []CmdResult{0, 0, 0},
		},
		resultStatus: JobStatusSuccessful,
	},
	testCase{
		requestBody: `{
  "image": "ubuntu:14.04",
  "cmds": [
    ["sh", "-c", "echo \"test\" > /test.txt"],
    ["sleep", "1"],
    ["cat", "/notthere.txt"],
    ["echo", "'I shouldn't run"]
  ]
}`,
		job: Job{
			ImageName: "ubuntu:14.04",
			Cmds: []Cmd{
				[]string{"sh", "-c", "echo \"test\" > /test.txt"},
				[]string{"sleep", "1"},
				[]string{"cat", "/notthere.txt"},
				[]string{"echo", "'I shouldn't run"},
			},
			Results: []CmdResult{0, 0, 1},
		},
		resultStatus: JobStatusFailed,
	},
	testCase{
		requestBody: `{
  "image": "ubuntu:14.04",
  "cmds": [
    ["notacommand"]
  ]
}`,
		job: Job{
			ImageName: "ubuntu:14.04",
			Cmds: []Cmd{
				[]string{"notacommand"},
			},
		},
		resultStatus: JobStatusError,
	},
}

func TestAPI(t *testing.T) {
	wsContainer := initWSContainer()

	ts := httptest.NewServer(wsContainer)
	defer ts.Close()

	jobURL := fmt.Sprintf("%s/%s", ts.URL, "jobs")

	for i, tc := range testcases {

		jobPOST := createJob(t, jobURL, tc.requestBody)

		assert.Equal(t, JobStatusQueued, jobPOST.Status, "Case %d: Status should be queued", i)
		assert.Equal(t, tc.job.ImageName, jobPOST.ImageName, "Case %d: Image name should match", i)
		assert.Equal(t, tc.job.Cmds, jobPOST.Cmds, "Case %d: Commands should match", i)
		assert.Equal(t, 0, len(jobPOST.Results), "Case %d: Should be no results initially", i)

		// wait while the job completes
		waitUntilDone(t, jobURL, jobPOST.ID)
		jobGET := getJob(t, jobURL, jobPOST.ID)
		assert.Equal(t, tc.resultStatus, jobGET.Status, "Case %d: Status should match", i)
		assert.Equal(t, tc.job.Results, jobGET.Results, "Case %d: Results should match", i)
	}
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
		if jobGET.Status != "running" && jobGET.Status != "queued" {
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
