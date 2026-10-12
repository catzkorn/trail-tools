// Package cloudscheduler runs Cloud Scheduler jobs.
package cloudscheduler

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2/google"
)

// Job is a Cloud Scheduler job.
type Job struct {
	name   string
	client *http.Client
}

// NewJob creates a Job from its full name, e.g.
// projects/<project>/locations/<region>/jobs/<job>. It authenticates with the
// application default credentials, such as the service account of a Cloud
// Run service.
func NewJob(ctx context.Context, name string) (*Job, error) {
	client, err := google.DefaultClient(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return nil, fmt.Errorf("failed to get Google credentials: %w", err)
	}
	return &Job{name: name, client: client}, nil
}

// Run runs the job right away, outside its schedule. It returns once the
// run is started.
func (j *Job) Run(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://cloudscheduler.googleapis.com/v1/"+j.name+":run", nil)
	if err != nil {
		return err
	}
	resp, err := j.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to run the Cloud Scheduler job: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		message, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("failed to run the Cloud Scheduler job: status %d: %s", resp.StatusCode, bytes.TrimSpace(message))
	}
	return nil
}
