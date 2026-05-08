package shared

import (
	"encoding/json"
	"time"
)

// ReprintDocument фиксирует результат controlled reprint без обращения к текущему состоянию order.
type ReprintDocument struct {
	DocumentType    string          `json:"document_type"`
	SourceID        string          `json:"source_id"`
	CopyMarker      string          `json:"copy_marker"`
	ActorEmployeeID string          `json:"actor_employee_id,omitempty"`
	ReprintedAt     time.Time       `json:"reprinted_at"`
	Snapshot        json.RawMessage `json:"snapshot"`
}

// NewReprintDocument строит документ-репринт строго поверх сохраненного immutable snapshot.
func NewReprintDocument(documentType, sourceID string, snapshot json.RawMessage, actorEmployeeID string, reprintedAt time.Time) *ReprintDocument {
	return &ReprintDocument{
		DocumentType:    documentType,
		SourceID:        sourceID,
		CopyMarker:      "COPY",
		ActorEmployeeID: actorEmployeeID,
		ReprintedAt:     reprintedAt,
		Snapshot:        snapshot,
	}
}
