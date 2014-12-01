package model

import (
	"time"
)

type ResetMode struct {
	Hold     bool          `json:"hold"`
	Mode     string        `json:"mode"`
	Duration time.Duration `json:"duration"`
}

type DisplayProgress struct {
	Progress float64 `json:"progress"`
	Icon     string  `json:"icon"`
}
