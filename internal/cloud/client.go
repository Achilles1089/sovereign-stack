package cloud

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Achilles1089/sovereign-stack/internal/config"
)

const defaultCloudURL = "https://api.sovereign.dev/v1"

// Client is the Sovereign Cloud API client
type Client struct {
	APIKey  string
	BaseURL string
	http    *http.Client
}

// Subscription holds the cloud subscription info
type Subscription struct {
	Active    bool      `json:"active"`
	Plan      string    `json:"plan"` // "free", "pro", "enterprise"
	ExpiresAt time.Time `json:"expires_at"`
	Features  Features  `json:"features"`
}

// Features lists what the subscription includes
type Features struct {
	RemoteBackup    bool `json:"remote_backup"`
	Monitoring      bool `json:"monitoring"`
	SecureTunnel    bool `json:"secure_tunnel"`
	PrioritySupport bool `json:"priority_support"`
}

// NewClient creates a Sovereign Cloud client
func NewClient(apiKey string) *Client {
	return &Client{
		APIKey:  apiKey,
		BaseURL: defaultCloudURL,
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Connect links this server to a Sovereign Cloud account
func (c *Client) Connect(cfg *config.Config) error {
	// Verify the API key
	sub, err := c.GetSubscription()
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	if !sub.Active {
		return fmt.Errorf("subscription is not active")
	}

	// Store the API key in config
	return nil
}

// GetSubscription fetches subscription info
func (c *Client) GetSubscription() (*Subscription, error) {
	req, err := http.NewRequest("GET", c.BaseURL+"/subscription", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.http.Do(req)
	if err != nil {
		// Offline mode — return free tier
		return &Subscription{
			Active: true,
			Plan:   "free",
			Features: Features{
				RemoteBackup:    false,
				Monitoring:      false,
				SecureTunnel:    false,
				PrioritySupport: false,
			},
		}, nil
	}
	defer resp.Body.Close()

	var sub Subscription
	if err := json.NewDecoder(resp.Body).Decode(&sub); err != nil {
		return nil, err
	}
	return &sub, nil
}

// UploadBackup uploads an encrypted backup snapshot to cloud storage
func (c *Client) UploadBackup(snapshotPath string) error {
	// Placeholder — would upload to S3-compatible cloud storage
	return fmt.Errorf("cloud backup not yet available in this version")
}

// PushMetrics sends system metrics to the cloud monitoring dashboard
func (c *Client) PushMetrics(metrics map[string]interface{}) error {
	// Placeholder — would push to cloud metrics endpoint
	return fmt.Errorf("cloud monitoring not yet available in this version")
}
