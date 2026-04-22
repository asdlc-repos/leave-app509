package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Notification represents a queued email notification fetched from leave-api.
// Fields use permissive names so we can decode whatever reasonable shape the
// backend produces.
type Notification struct {
	ID        string `json:"id"`
	Type      string `json:"type,omitempty"`
	To        string `json:"to,omitempty"`
	Recipient string `json:"recipient,omitempty"`
	Email     string `json:"email,omitempty"`
	Subject   string `json:"subject,omitempty"`
	Body      string `json:"body,omitempty"`
	Message   string `json:"message,omitempty"`
	CreatedAt string `json:"createdAt,omitempty"`
}

func (n Notification) Address() string {
	switch {
	case n.To != "":
		return n.To
	case n.Recipient != "":
		return n.Recipient
	case n.Email != "":
		return n.Email
	}
	return ""
}

func (n Notification) Text() string {
	if n.Body != "" {
		return n.Body
	}
	return n.Message
}

type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

func New(baseURL, token string) *Client {
	return &Client{
		baseURL: baseURL,
		token:   token,
		http:    &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *Client) authorize(req *http.Request) {
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
		req.Header.Set("X-Service-Token", c.token)
	}
	req.Header.Set("Accept", "application/json")
}

func (c *Client) PendingNotifications(ctx context.Context) ([]Notification, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/notifications/pending", nil)
	if err != nil {
		return nil, err
	}
	c.authorize(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get pending: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("leave-api /notifications/pending returned %d: %s", resp.StatusCode, string(b))
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Accept either a bare array or a wrapped object like {"notifications":[...]}.
	var arr []Notification
	if err := json.Unmarshal(raw, &arr); err == nil {
		return arr, nil
	}
	var wrap struct {
		Notifications []Notification `json:"notifications"`
		Items         []Notification `json:"items"`
		Data          []Notification `json:"data"`
	}
	if err := json.Unmarshal(raw, &wrap); err == nil {
		switch {
		case len(wrap.Notifications) > 0:
			return wrap.Notifications, nil
		case len(wrap.Items) > 0:
			return wrap.Items, nil
		case len(wrap.Data) > 0:
			return wrap.Data, nil
		}
		return nil, nil
	}
	return nil, fmt.Errorf("unexpected pending-notifications payload: %s", string(raw))
}

func (c *Client) MarkSent(ctx context.Context, id string) error {
	url := fmt.Sprintf("%s/notifications/%s/mark-sent", c.baseURL, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(nil))
	if err != nil {
		return err
	}
	c.authorize(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("mark-sent: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	b, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("mark-sent id=%s returned %d: %s", id, resp.StatusCode, string(b))
}
