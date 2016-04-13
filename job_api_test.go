package dockworker

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAPI(t *testing.T) {
	ts, whRecorder, webhookServer := testSetup(t)
	jobURL := fmt.Sprintf("%s/%s", ts.URL, "jobs")

	defer webhookServer.Close()
	defer ts.Close()

	for i, tc := range apiTestCases {

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
		assert.Condition(t, func() bool { return jobGET.StartTime.Before(jobGET.EndTime) || jobGET.StartTime.Equal(jobGET.EndTime) },
			"Case %d: Job start time (%s) should be before or equal to end time (%s)", i, jobGET.StartTime, jobGET.EndTime)

		// check the logs of the job
		logs := getLogs(t, i, jobURL, jobPOST.ID)
		assert.Equal(t, tc.logs, logs, "Case %d: Logs should match", i)

		// check the webhook results
		assert.Condition(t, func() bool { return i < len(whRecorder.webhookRequests) }, "Case %d: Webhook requests length not great enough", i)
		whJob := whRecorder.webhookRequests[i]
		assert.Equal(t, tc.resultStatus, whJob.Status, "Case %d: Webhook status should match", i)
		assert.Equal(t, tc.job.Results, whJob.Results, "Case %d: Webhook results should match", i)
		assert.Equal(t, tc.numContainers, len(whJob.Containers), "Case %d: Webhook number of containers should match", i)
	}
}
