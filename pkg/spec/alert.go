package spec

import "time"

type AlertMessage struct {
	CheckName      string      `json:"check_name"`
	Status         CheckStatus `json:"status"`
	PreviousStatus CheckStatus `json:"previous_status"`
	Timestamp      time.Time   `json:"timestamp"`
	Detail         string      `json:"detail,omitempty"`
}
