package dockworker

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStop(t *testing.T) {
	ts, whRecorder, webhookServer := testSetup(t)
	jobURL := fmt.Sprintf("%s/%s", ts.URL, "jobs")

	for i, tc := range stopTestCases {
		jobPOST := createJob(t, i, jobURL, fmt.Sprintf(tc.requestBody, webhookServer.URL))
		waitUntilRunning(t, i, jobURL, jobPOST.ID)
		// let the job run for a bit
		time.Sleep(time.Second * 2)
		stopJob(t, i, jobURL, jobPOST.ID)
		waitUntilDone(t, i, jobURL, jobPOST.ID)

		jobGET := getJob(t, i, jobURL, jobPOST.ID)
		assert.Equal(t, tc.resultStatus, jobGET.Status, "Case %d: Status should match", i)
		assert.Equal(t, tc.job.Results, jobGET.Results, "Case %d: Results should match", i)
		assert.Equal(t, tc.numContainers, len(jobGET.Containers), "Case %d: Number of containers should match", i)

		// check the logs of the job
		logs := getLogs(t, i, jobURL, jobPOST.ID)
		assert.Equal(t, tc.logs, logs, "Case %d: Logs should match", i)

		assert.Condition(t, func() bool { return i < len(whRecorder.webhookRequests) }, "Case %d: Webhook requests length not great enough", i)
		whJob := whRecorder.webhookRequests[i]
		assert.Equal(t, tc.resultStatus, whJob.Status, "Case %d: Webhook status should match", i)
		assert.Equal(t, tc.job.Results, whJob.Results, "Case %d: Webhook results should match", i)
		assert.Equal(t, tc.numContainers, len(whJob.Containers), "Case %d: Webhook number of containers should match", i)
	}

	defer webhookServer.Close()
	defer ts.Close()
}

var stopTestCases = []testCase{
	testCase{
		requestBody: `{
	  "image": "ubuntu:14.04",
	  "cmds": [
      ["echo", "Sleeping..."],
      ["sleep", "30"]
    ],
		"webhook_url": "%s"
	}`,
		job: Job{
			ImageName: "ubuntu:14.04",
			Cmds: []Cmd{
				[]string{"echo", "Sleeping..."},
				[]string{"sleep", "30"},
			},
			Results: []CmdResult{0, 1},
		},
		resultStatus:  JobStatusStopped,
		numContainers: 2,
		logs:          "Sleeping...\nsleep: cannot read realtime clock: Operation not permitted\n",
	},
}
