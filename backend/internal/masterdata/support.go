package masterdata

import (
	"strings"

	"github.com/google/uuid"

	apperrors "github.com/pharmaos/pharmaos/backend/internal/platform/errors"
	"github.com/pharmaos/pharmaos/backend/internal/platform/jsontypes"
)

type DoctorRequest struct {
	ExternalID     string         `json:"external_id"`
	Name           string         `json:"name" validate:"required,min=2,max=180"`
	Specialty      string         `json:"specialty"`
	Qualification  string         `json:"qualification"`
	RegistrationNo string         `json:"registration_no"`
	Email          string         `json:"email" validate:"omitempty,email"`
	Phone          string         `json:"phone"`
	Address        string         `json:"address"`
	TerritoryID    *uuid.UUID     `json:"territory_id"`
	DivisionID     *uuid.UUID     `json:"division_id"`
	IsActive       *bool          `json:"is_active"`
	Settings       map[string]any `json:"settings"`
}

type ChemistRequest struct {
	ExternalID  string         `json:"external_id"`
	Name        string         `json:"name" validate:"required,min=2,max=180"`
	OwnerName   string         `json:"owner_name"`
	LicenseNo   string         `json:"license_no"`
	Email       string         `json:"email" validate:"omitempty,email"`
	Phone       string         `json:"phone"`
	Address     string         `json:"address"`
	TerritoryID *uuid.UUID     `json:"territory_id"`
	DivisionID  *uuid.UUID     `json:"division_id"`
	IsActive    *bool          `json:"is_active"`
	Settings    map[string]any `json:"settings"`
}

type StockistRequest struct {
	ExternalID       string         `json:"external_id"`
	Name             string         `json:"name" validate:"required,min=2,max=180"`
	LicenseNo        string         `json:"license_no"`
	GSTNumber        string         `json:"gst_number"`
	Email            string         `json:"email" validate:"omitempty,email"`
	Phone            string         `json:"phone"`
	Address          string         `json:"address"`
	TerritoryID      *uuid.UUID     `json:"territory_id"`
	DivisionID       *uuid.UUID     `json:"division_id"`
	CreditLimit      float64        `json:"credit_limit"`
	PaymentTermsDays int            `json:"payment_terms_days"`
	IsActive         *bool          `json:"is_active"`
	Settings         map[string]any `json:"settings"`
}

type ProductCategoryRequest struct {
	Code             string     `json:"code" validate:"required,min=2,max=40"`
	Name             string     `json:"name" validate:"required,min=2,max=150"`
	ParentCategoryID *uuid.UUID `json:"parent_category_id"`
	IsActive         *bool      `json:"is_active"`
}

type ProductRequest struct {
	CategoryID   *uuid.UUID     `json:"category_id"`
	Code         string         `json:"code" validate:"required,min=2,max=60"`
	Name         string         `json:"name" validate:"required,min=2,max=180"`
	Brand        string         `json:"brand"`
	Manufacturer string         `json:"manufacturer"`
	Composition  string         `json:"composition"`
	PackSize     string         `json:"pack_size"`
	Unit         string         `json:"unit"`
	MRP          float64        `json:"mrp"`
	IsActive     *bool          `json:"is_active"`
	Settings     map[string]any `json:"settings"`
}

func boolOr(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}

func settingsOrEmpty(value map[string]any) jsontypes.Map {
	if value == nil {
		return jsontypes.Map{}
	}
	return jsontypes.Map(value)
}

func externalID(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	trimmed := strings.ToUpper(strings.TrimSpace(value))
	return &trimmed
}

func conflict(err error) error {
	if err == nil {
		return nil
	}
	if strings.Contains(strings.ToLower(err.Error()), "unique") {
		return apperrors.ErrConflict
	}
	return err
}

func optionalUUIDFromQuery(value string) uuid.UUID {
	if value == "" {
		return uuid.Nil
	}
	parsed, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil
	}
	return parsed
}
