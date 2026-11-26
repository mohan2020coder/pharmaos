package audit

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Entry struct {
	TenantID      *uuid.UUID
	UserID        *uuid.UUID
	Action        string
	Module        string
	EntityType    string
	EntityID      string
	Previous      interface{}
	New           interface{}
	IPAddress     string
	UserAgent     string
	CorrelationID string
}

type Service struct{ db *gorm.DB }

func NewService(db *gorm.DB) *Service { return &Service{db: db} }

func (s *Service) Record(ctx context.Context, entry Entry) error {
	return s.db.WithContext(ctx).Exec(`INSERT INTO audit_logs
		(tenant_id, user_id, action, module, entity_type, entity_id, previous_values, new_values, ip_address, user_agent, correlation_id)
		VALUES (?, ?, ?, ?, ?, ?, ?::jsonb, ?::jsonb, NULLIF(?, '')::inet, ?, NULLIF(?, ''))`,
		entry.TenantID, entry.UserID, entry.Action, entry.Module, entry.EntityType, entry.EntityID,
		jsonString(entry.Previous), jsonString(entry.New), entry.IPAddress, entry.UserAgent, entry.CorrelationID).Error
}

func jsonString(value interface{}) interface{} {
	if value == nil {
		return nil
	}
	body, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(body)
}
