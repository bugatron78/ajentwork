package domain

import "time"

type ArtifactKind string

const (
	ArtifactKindFile    ArtifactKind = "file"
	ArtifactKindReceipt ArtifactKind = "receipt"
)

type Artifact struct {
	ID           string       `json:"id"`
	ItemID       string       `json:"item_id"`
	Kind         ArtifactKind `json:"kind"`
	Summary      string       `json:"summary"`
	Label        string       `json:"label,omitempty"`
	OriginalPath string       `json:"original_path,omitempty"`
	StoredPath   string       `json:"stored_path,omitempty"`
	Command      string       `json:"command,omitempty"`
	ExitCode     *int         `json:"exit_code,omitempty"`
	CreatedAt    time.Time    `json:"created_at"`
	Actor        string       `json:"actor"`
}
