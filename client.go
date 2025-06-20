package heartbeat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type Client struct {
	config        ClientConfig
	mu            sync.Mutex
	successCount  int
	errorCount    int
	isForcedError bool
	currentStatus Status
	startOnce     sync.Once
}

type Thresholds struct {
	DegradedErrorRate float64 // es. 0.2 = 20%
	ErrorErrorRate    float64 // es. 0.5 = 50%
	MinTotalOps       int
}

type ClientConfig struct {
	ServerURL      string
	BeatDelay      time.Duration
	ClientID       string
	Thresholds     Thresholds
	OnStatusChange func(old, new Status)
}

func NewClient(cfg ClientConfig) *Client {
	return &Client{
		config:        cfg,
		currentStatus: StatusOK,
	}
}

func (c *Client) Success() {
	c.mu.Lock()
	c.successCount++
	c.isForcedError = false
	c.mu.Unlock()

	c.updateStatus()
}

func (c *Client) Error() {
	c.mu.Lock()
	c.errorCount++
	c.mu.Unlock()

	c.updateStatus()
}

func (c *Client) ForceError() {
	c.mu.Lock()
	c.isForcedError = true
	c.mu.Unlock()

	c.updateStatus()
}

func (c *Client) Start() {
	c.startOnce.Do(func() {
		go func() {
			ticker := time.NewTicker(c.config.BeatDelay)
			defer ticker.Stop()
			for range ticker.C {

				c.sendBeat()
			}
		}()
	})
}

func (c *Client) updateStatus() {

	c.mu.Lock()
	newStatus := c.calcStatus()
	isStatusChange := newStatus != c.currentStatus
	c.currentStatus = newStatus
	c.mu.Unlock()

	if isStatusChange {

		if c.config.OnStatusChange != nil {
			c.config.OnStatusChange(c.currentStatus, newStatus)
		}

		c.sendBeat()
	}
}

func (c *Client) calcStatus() Status {

	if c.isForcedError {
		return StatusError
	}

	total := c.successCount + c.errorCount
	if total < c.config.Thresholds.MinTotalOps {
		return c.currentStatus
	}

	errorRate := float64(c.errorCount) / float64(total)

	switch {
	case errorRate >= c.config.Thresholds.ErrorErrorRate:
		return StatusError
	case errorRate >= c.config.Thresholds.DegradedErrorRate:
		return StatusDegraded
	default:
		return StatusOK
	}
}

func (c *Client) trySendBeat() error {

	payload := BeatPayload{
		ClientID:     c.config.ClientID,
		Status:       c.currentStatus,
		SuccessCount: c.successCount,
		ErrorCount:   c.errorCount,
		Timestamp:    time.Now(),
		BeatDelay:    c.config.BeatDelay,
	}

	body, _ := json.Marshal(payload)
	resp, err := http.Post(c.config.ServerURL, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("response code: %d", resp.StatusCode)
	}

	c.errorCount = 0
	c.successCount = 0
	return nil
}

func (c *Client) sendBeat() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for range 3 {
		err := c.trySendBeat()
		if err == nil {
			break
		}
	}
}
