package dockworker

import (
	"bytes"
	"encoding/json"
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/emicklei/go-restful"
)

// SendWebhook performs the webhook request for the given Job
func SendWebhook(job Job) {
	if job.WebhookURL == "" {
		// no webhook for this job
		log.Debugf("No webhook URL present for %d", job.ID)
		return
	}

	body, err := json.Marshal(job)
	if err != nil {
		log.Errorf("Failed to marshal job %+v: %s", job.ID, err)
		return
	}

	resp, err := http.Post(job.WebhookURL, restful.MIME_JSON, bytes.NewReader(body))
	if err != nil {
		log.Errorf("Failed to send webhook request for %d to %s: %s", job.ID, job.WebhookURL, err.Error())
		return
	}

	// check that we got some kind of successful response code
	if resp.StatusCode < 200 || 300 <= resp.StatusCode {
		log.Errorf("Unexpected response from send webhook request for %d to %s: %d", job.ID, job.WebhookURL, resp.StatusCode)
		return
	}

	log.Infof("Webhook request for %d to %s sent successfully", job.ID, job.WebhookURL)
}
