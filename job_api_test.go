package dockworker

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
	requestBody   string
	job           Job
	resultStatus  JobStatus
	numContainers int
	logs          string
}

var testcases = []testCase{
	testCase{
		requestBody: `{
	  "image": "ubuntu:14.04",
	  "cmds": [
	    ["sh", "-c", "echo \"test\" > /test.txt"],
	    ["sleep", "1"],
	    ["cat", "/test.txt"]
	  ],
		"webhook_url": "%s"
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
		resultStatus:  JobStatusSuccessful,
		numContainers: 3,
		logs:          "test\n",
	},
	testCase{
		requestBody: `{
	  "image": "ubuntu:14.04",
	  "cmds": [
	    ["sh", "-c", "echo \"test\" > /test.txt"],
	    ["sleep", "1"],
	    ["cat", "/notthere.txt"],
	    ["echo", "'I shouldn't run"]
	  ],
		"webhook_url": "%s"
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
		resultStatus:  JobStatusFailed,
		numContainers: 3,
		logs:          "cat: /notthere.txt: No such file or directory\n",
	},
	testCase{
		requestBody: `{
	  "image": "ubuntu:14.04",
	  "cmds": [
	    ["notacommand"]
	  ],
		"webhook_url": "%s"
	}`,
		job: Job{
			ImageName: "ubuntu:14.04",
			Cmds: []Cmd{
				[]string{"notacommand"},
			},
		},
		resultStatus:  JobStatusError,
		numContainers: 1,
		logs:          "exec: \"notacommand\": executable file not found in $PATH\n",
	},
	testCase{
		requestBody: `{
  "image": "ubuntu:14.04",
  "cmds": [
    ["sh", "-c", "echo $TEST_VAR1"],
		["sh", "-c", "echo $TEST_VAR2"]
  ],
	"env": {
		"TEST_VAR1": "test value 1",
		"TEST_VAR2": "test value 2"
	},
	"webhook_url": "%s"
}`,
		job: Job{
			ImageName: "ubuntu:14.04",
			Cmds: []Cmd{
				[]string{"sh", "-c", "echo $TEST_VAR1"},
				[]string{"sh", "-c", "echo $TEST_VAR2"},
			},
			Env: map[string]string{
				"TEST_VAR1": "test value 1",
				"TEST_VAR2": "test value 2",
			},
			Results: []CmdResult{0, 0},
		},
		resultStatus:  JobStatusSuccessful,
		numContainers: 2,
		logs:          "test value 1\ntest value 2\n",
	},
	testCase{
		requestBody: `{
	  "image": "doesnotexist",
	  "cmds": [
	    ["echo", "$TEST_VAR1"],
			["echo", "$TEST_VAR2"]
	  ],
		"env": {
			"TEST_VAR1": "test value 1",
			"TEST_VAR2": "test value 2"
		},
		"webhook_url": "%s"
	}`,
		job: Job{
			ImageName: "doesnotexist",
			Cmds: []Cmd{
				[]string{"echo", "$TEST_VAR1"},
				[]string{"echo", "$TEST_VAR2"},
			},
			Env: map[string]string{
				"TEST_VAR1": "test value 1",
				"TEST_VAR2": "test value 2",
			},
		},
		resultStatus:  JobStatusError,
		numContainers: 0,
	},
}

type webhookRecorder struct {
	t     *testing.T
	tcNum int
}

var webhookRequests []*Job

func (recorder *webhookRecorder) webhookHandler(w http.ResponseWriter, r *http.Request) {
	defer func() { recorder.tcNum++ }()
	job := decodeBody(recorder.t, recorder.tcNum, r.Body)
	webhookRequests = append(webhookRequests, job)
	w.WriteHeader(http.StatusAccepted)
}

func TestAPI(t *testing.T) {
	wsContainer := InitWSContainer()

	ts := httptest.NewServer(wsContainer)
	defer ts.Close()

	whRecorder := webhookRecorder{
		t:     t,
		tcNum: 0,
	}

	webhookServer := httptest.NewServer(http.HandlerFunc(whRecorder.webhookHandler))
	defer webhookServer.Close()

	jobURL := fmt.Sprintf("%s/%s", ts.URL, "jobs")

	for i, tc := range testcases {

		jobPOST := createJob(t, i, jobURL, fmt.Sprintf(tc.requestBody, webhookServer.URL))

		assert.Equal(t, JobStatusQueued, jobPOST.Status, "Case %d: Status should be queued", i)
		assert.Equal(t, tc.job.ImageName, jobPOST.ImageName, "Case %d: Image name should match", i)
		assert.Equal(t, tc.job.Cmds, jobPOST.Cmds, "Case %d: Commands should match", i)
		assert.Equal(t, tc.job.Env, jobPOST.Env, "Case %d: Env should match", i)
		assert.Equal(t, 0, len(jobPOST.Results), "Case %d: Should be no results initially", i)
		assert.Equal(t, 0, len(jobPOST.Containers), "Case %d: Should be no containers initially", i)
		assert.Equal(t, webhookServer.URL, jobPOST.WebhookURL, "Case %d: Webhook URLs should match", i)

		// wait while the job completes
		waitUntilDone(t, i, jobURL, jobPOST.ID)
		jobGET := getJob(t, i, jobURL, jobPOST.ID)
		assert.Equal(t, tc.resultStatus, jobGET.Status, "Case %d: Status should match", i)
		assert.Equal(t, tc.job.Results, jobGET.Results, "Case %d: Results should match", i)
		assert.Equal(t, tc.numContainers, len(jobGET.Containers), "Case %d: Number of containers should match", i)

		// check the logs of the job
		logs := getLogs(t, i, jobURL, jobPOST.ID)
		assert.Equal(t, tc.logs, logs, "Case %d: Logs should match", i)

		// check the webhook results
		assert.Condition(t, func() bool { return i < len(webhookRequests) }, "Case %d: Webhook requests length not great enough", i)
		whJob := webhookRequests[i]
		assert.Equal(t, tc.resultStatus, whJob.Status, "Case %d: Webhook status should match", i)
		assert.Equal(t, tc.job.Results, whJob.Results, "Case %d: Webhook results should match", i)
		assert.Equal(t, tc.numContainers, len(whJob.Containers), "Case %d: Webhook number of containers should match", i)
	}
}

func createJob(t *testing.T, tcNum int, jobURL string, body string) *Job {
	resp, err := http.Post(jobURL, "application/json", strings.NewReader(body))
	if err != nil {
		t.Errorf("Error sending post request: %s", err)
	}
	assert.Equal(t, http.StatusCreated, resp.StatusCode, "Case %d: Status code should be 201", tcNum)
	return decodeBody(t, tcNum, resp.Body)
}

func getJob(t *testing.T, tcNum int, jobURL string, jobID JobID) *Job {
	resp, err := http.Get(fmt.Sprintf("%s/%d", jobURL, jobID))
	if err != nil {
		t.Errorf("Case %d: Error sending get request: %s", tcNum, err)
	}
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Case %d: Status code should be 200", tcNum)

	return decodeBody(t, tcNum, resp.Body)
}

func getLogs(t *testing.T, tcNum int, jobURL string, jobID JobID) string {
	resp, err := http.Get(fmt.Sprintf("%s/%d/logs", jobURL, jobID))
	if err != nil {
		t.Errorf("Case %d: Error sending get logs request: %s", tcNum, err)
	}
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Case %d: Status code should be 200", tcNum)

	return bodyToString(t, tcNum, resp.Body)
}

func waitUntilDone(t *testing.T, tcNum int, jobURL string, jobID JobID) {
	for i := 0; i < retryCount; i++ {
		jobGET := getJob(t, tcNum, jobURL, jobID)
		if jobGET.Status != "running" && jobGET.Status != "queued" {
			return
		}
		time.Sleep(1 * time.Second)
	}
	t.Fatalf("Case %d: Waiting too long for job to complete", tcNum)
}

func bodyToString(t *testing.T, tcNum int, respBody io.ReadCloser) string {
	body, err := ioutil.ReadAll(respBody)
	defer respBody.Close()
	if err != nil {
		t.Errorf("Case %d: Error converting response body to string: %s", tcNum, err)
	}
	return string(body)
}

func decodeBody(t *testing.T, tcNum int, respBody io.ReadCloser) *Job {
	body, err := ioutil.ReadAll(respBody)
	defer respBody.Close()
	if err != nil {
		t.Errorf("Case %d: Error reading response body: %s", tcNum, err)
	}

	job := &Job{}
	err = json.Unmarshal(body, job)
	if err != nil {
		t.Errorf("Case %d: Error decoding response body: %s", tcNum, err)
	}

	return job
}
