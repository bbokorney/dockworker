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

func testSetup(t *testing.T) (*httptest.Server, *webhookRecorder, *httptest.Server) {
	wsContainer := InitWSContainer()

	ts := httptest.NewServer(wsContainer)
	whRecorder := webhookRecorder{
		t:               t,
		tcNum:           0,
		webhookRequests: []*Job{},
	}

	webhookServer := httptest.NewServer(http.HandlerFunc(whRecorder.webhookHandler))

	return ts, &whRecorder, webhookServer
}

type webhookRecorder struct {
	t               *testing.T
	tcNum           int
	webhookRequests []*Job
}

func (recorder *webhookRecorder) webhookHandler(w http.ResponseWriter, r *http.Request) {
	defer func() { recorder.tcNum++ }()
	job := decodeBody(recorder.t, recorder.tcNum, r.Body)
	recorder.webhookRequests = append(recorder.webhookRequests, job)
	w.WriteHeader(http.StatusAccepted)
}

// TODO: use the client for these functions
func createJob(t *testing.T, tcNum int, jobURL string, body string) *Job {
	resp, err := http.Post(jobURL, "application/json", strings.NewReader(body))
	if err != nil {
		t.Errorf("Error sending create POST request: %s", err)
	}
	assert.Equal(t, http.StatusCreated, resp.StatusCode, "Case %d: Status code should be 201", tcNum)
	return decodeBody(t, tcNum, resp.Body)
}

func stopJob(t *testing.T, tcNum int, jobURL string, jobID JobID) {
	resp, err := http.Post(fmt.Sprintf("%s/%d/stop", jobURL, jobID), "application/json", nil)
	if err != nil {
		t.Errorf("Error sending stop POST request: %s", err)
	}
	assert.Equal(t, http.StatusAccepted, resp.StatusCode, "Case %d: Status code should be 202", tcNum)
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

// TODO: refactor these into one function
func waitUntilDone(t *testing.T, tcNum int, jobURL string, jobID JobID) {
	for i := 0; i < retryCount; i++ {
		jobGET := getJob(t, tcNum, jobURL, jobID)
		if jobGET.Status != JobStatusRunning && jobGET.Status != JobStatusQueued {
			return
		}
		time.Sleep(1 * time.Second)
	}
	t.Fatalf("Case %d: Waiting too long for job to complete", tcNum)
}

func waitUntilRunning(t *testing.T, tcNum int, jobURL string, jobID JobID) {
	for i := 0; i < retryCount; i++ {
		jobGET := getJob(t, tcNum, jobURL, jobID)
		if jobGET.Status == JobStatusRunning {
			return
		}
		time.Sleep(1 * time.Second)
	}
	t.Fatalf("Case %d: Waiting too long for job to start running", tcNum)
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
