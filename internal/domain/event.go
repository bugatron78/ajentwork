package domain

import "time"

type Event struct {
	ID      string    `json:"id"`
	ItemID  string    `json:"item_id"`
	Type    string    `json:"type"`
	At      time.Time `json:"at"`
	Actor   string    `json:"actor"`
	Summary string    `json:"summary"`
}
