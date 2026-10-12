package strava

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// PushSubscription is the application's subscription to Strava's webhook
// events. Strava allows one subscription per application.
type PushSubscription struct {
	ID          int64  `json:"id"`
	CallbackURL string `json:"callback_url"`
}

// ListPushSubscriptions lists the application's push subscriptions.
func (c *Client) ListPushSubscriptions(ctx context.Context) ([]*PushSubscription, error) {
	body, err := c.appRequest(ctx, http.MethodGet, "/push_subscriptions", nil)
	if err != nil {
		return nil, err
	}
	var subs []*PushSubscription
	if err := json.Unmarshal(body, &subs); err != nil {
		return nil, fmt.Errorf("failed to decode push subscriptions: %w", err)
	}
	return subs, nil
}

// CreatePushSubscription subscribes the application to webhook events,
// sent to callbackURL. Before responding, Strava validates the callback by
// sending it a GET request with the verify token, so the callback must
// already be served.
func (c *Client) CreatePushSubscription(ctx context.Context, callbackURL, verifyToken string) (*PushSubscription, error) {
	body, err := c.appRequest(ctx, http.MethodPost, "/push_subscriptions", url.Values{
		"callback_url": {callbackURL},
		"verify_token": {verifyToken},
	})
	if err != nil {
		return nil, err
	}
	var sub PushSubscription
	if err := json.Unmarshal(body, &sub); err != nil {
		return nil, fmt.Errorf("failed to decode push subscription: %w", err)
	}
	sub.CallbackURL = callbackURL
	return &sub, nil
}

// DeletePushSubscription deletes a push subscription.
func (c *Client) DeletePushSubscription(ctx context.Context, id int64) error {
	_, err := c.appRequest(ctx, http.MethodDelete, "/push_subscriptions/"+strconv.FormatInt(id, 10), nil)
	return err
}

// appRequest performs a request authenticated with the application's client
// ID and secret, as used by push subscriptions: in the query, or the form
// body of a POST.
func (c *Client) appRequest(ctx context.Context, method, path string, params url.Values) ([]byte, error) {
	if params == nil {
		params = url.Values{}
	}
	params.Set("client_id", c.oauth.ClientID)
	params.Set("client_secret", c.oauth.ClientSecret)
	u := c.baseURL + path
	var body io.Reader
	if method == http.MethodPost {
		body = strings.NewReader(params.Encode())
	} else {
		u += "?" + params.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Leave out the URL, which has the client secret in its query.
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			err = urlErr.Err
		}
		return nil, fmt.Errorf("failed to %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()
	c.recordUsage(resp)
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated, http.StatusNoContent:
		return data, nil
	case http.StatusNotFound:
		return nil, ErrNotFound
	case http.StatusTooManyRequests:
		return nil, c.blockUntilReset()
	default:
		// Strava explains rejected subscriptions in the body, e.g. that the
		// callback couldn't be verified.
		return nil, fmt.Errorf("unexpected status %d from Strava: %s", resp.StatusCode, strings.TrimSpace(string(data[:min(len(data), 512)])))
	}
}
