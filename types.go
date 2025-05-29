package heartbeat

import "time"

type Status string

const (
	StatusOK       Status = "OK"
	StatusDegraded Status = "DEGRADED"
	StatusError    Status = "ERROR"
)

type BeatPayload struct {
	ClientID     string        `json:"client_id"`
	Status       Status        `json:"status"`
	SuccessCount int           `json:"success_count"`
	ErrorCount   int           `json:"error_count"`
	Timestamp    time.Time     `json:"timestamp"`
	BeatDelay    time.Duration `json:"beat_delay"`
}
