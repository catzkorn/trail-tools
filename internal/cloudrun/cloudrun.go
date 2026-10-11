// Package cloudrun runs Cloud Run jobs.
package cloudrun

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2/google"
)

// Job is a Cloud Run job.
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

// Run starts an execution of the job, with args in place of the container's
// arguments. It returns once the execution is started.
func (j *Job) Run(ctx context.Context, args []string) error {
	body, err := json.Marshal(map[string]any{
		"overrides": map[string]any{
			"containerOverrides": []map[string]any{{"args": args}},
		},
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://run.googleapis.com/v2/"+j.name+":run", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := j.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to run the Cloud Run job: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		message, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("failed to run the Cloud Run job: status %d: %s", resp.StatusCode, bytes.TrimSpace(message))
	}
	return nil
}
