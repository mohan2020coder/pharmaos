package masterdata

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/pharmaos/pharmaos/backend/internal/platform/httputil"
	"github.com/pharmaos/pharmaos/backend/internal/platform/query"
	"github.com/pharmaos/pharmaos/backend/internal/platform/response"
	"github.com/pharmaos/pharmaos/backend/internal/platform/validate"
)

func (h *Handler) listDoctors(c *gin.Context) {
	page := query.Parse(c)
	var filters []func(*gorm.DB) *gorm.DB
	if territoryID := optionalUUIDFromQuery(c.Query("territory_id")); territoryID != uuid.Nil {
		filters = append(filters, func(db *gorm.DB) *gorm.DB { return db.Where("territory_id = ?", territoryID) })
	}
	if divisionID := optionalUUIDFromQuery(c.Query("division_id")); divisionID != uuid.Nil {
		filters = append(filters, func(db *gorm.DB) *gorm.DB { return db.Where("division_id = ?", divisionID) })
	}
	if active, ok := activeFilter(c); !ok {
		response.Fail(c, http.StatusBadRequest, "Invalid is_active filter", nil)
		return
	} else if active != nil {
		filters = append(filters, func(db *gorm.DB) *gorm.DB { return db.Where("is_active = ?", *active) })
	}
	items, total, err := h.doctors.List(c.Request.Context(), tenant(c), page, []string{"name", "specialty", "registration_no"}, filters...)
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, "Doctors retrieved", items, query.Meta(page, total))
}

func (h *Handler) createDoctor(c *gin.Context) {
	var request DoctorRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		httputil.BadJSON(c, err)
		return
	}
	if errors := validate.Struct(request); errors != nil {
		httputil.Invalid(c, errors)
		return
	}
	doctor := &Doctor{
		ID: uuid.New(), TenantID: tenant(c), ExternalID: externalID(request.ExternalID),
		Name: strings.TrimSpace(request.Name), Specialty: request.Specialty, Qualification: request.Qualification,
		RegistrationNo: request.RegistrationNo, Email: request.Email, Phone: request.Phone, Address: request.Address,
		TerritoryID: request.TerritoryID, DivisionID: request.DivisionID,
		IsActive: boolOr(request.IsActive, true), Settings: settingsOrEmpty(request.Settings),
	}
	if err := conflict(h.doctors.Create(c.Request.Context(), doctor)); err != nil {
		httputil.FailError(c, err)
		return
	}
	response.Created(c, "Doctor created successfully", doctor)
}

func (h *Handler) getDoctor(c *gin.Context) {
	id, ok := httputil.UUID(c, "id")
	if !ok {
		return
	}
	doctor, err := h.doctors.Get(c.Request.Context(), tenant(c), id)
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.OK(c, doctor)
}

func (h *Handler) updateDoctor(c *gin.Context) {
	id, ok := httputil.UUID(c, "id")
	if !ok {
		return
	}
	var request DoctorRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		httputil.BadJSON(c, err)
		return
	}
	if errors := validate.Struct(request); errors != nil {
		httputil.Invalid(c, errors)
		return
	}
	values := map[string]interface{}{
		"name": strings.TrimSpace(request.Name), "specialty": request.Specialty,
		"qualification": request.Qualification, "registration_no": request.RegistrationNo,
		"email": request.Email, "phone": request.Phone, "address": request.Address,
		"territory_id": request.TerritoryID, "division_id": request.DivisionID,
		"settings": settingsOrEmpty(request.Settings),
	}
	if request.IsActive != nil {
		values["is_active"] = *request.IsActive
	}
	if err := conflict(h.doctors.Update(c.Request.Context(), tenant(c), id, values)); err != nil {
		httputil.FailError(c, err)
		return
	}
	response.OK(c, gin.H{"updated": true})
}

func (h *Handler) deleteDoctor(c *gin.Context) {
	id, ok := httputil.UUID(c, "id")
	if !ok {
		return
	}
	if err := h.doctors.Delete(c.Request.Context(), tenant(c), id); err != nil {
		httputil.FailError(c, err)
		return
	}
	response.OK(c, gin.H{"deleted": true})
}

func (h *Handler) doctorTerritories(c *gin.Context) {
	id, ok := httputil.UUID(c, "id")
	if !ok {
		return
	}
	if _, err := h.doctors.Get(c.Request.Context(), tenant(c), id); err != nil {
		httputil.FailError(c, err)
		return
	}
	var items []DoctorTerritory
	err := h.db.WithContext(c.Request.Context()).
		Where("tenant_id = ? AND doctor_id = ?", tenant(c), id).
		Order("valid_from DESC").Find(&items).Error
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.OK(c, items)
}

func (h *Handler) assignDoctorTerritory(c *gin.Context) {
	id, ok := httputil.UUID(c, "id")
	if !ok {
		return
	}
	if _, err := h.doctors.Get(c.Request.Context(), tenant(c), id); err != nil {
		httputil.FailError(c, err)
		return
	}
	var request struct {
		TerritoryID uuid.UUID `json:"territory_id" validate:"required,uuid"`
		ValidFrom   string    `json:"valid_from"`
		ValidTo     string    `json:"valid_to"`
		IsPrimary   bool      `json:"is_primary"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		httputil.BadJSON(c, err)
		return
	}
	if errors := validate.Struct(request); errors != nil {
		httputil.Invalid(c, errors)
		return
	}
	validFrom := time.Now().UTC().Truncate(24 * time.Hour)
	if request.ValidFrom != "" {
		parsed, err := time.Parse("2006-01-02", request.ValidFrom)
		if err != nil {
			httputil.Invalid(c, map[string]string{"valid_from": "must be YYYY-MM-DD"})
			return
		}
		validFrom = parsed
	}
	var validTo *time.Time
	if request.ValidTo != "" {
		parsed, err := time.Parse("2006-01-02", request.ValidTo)
		if err != nil || !parsed.After(validFrom) {
			httputil.Invalid(c, map[string]string{"valid_to": "must be after valid_from"})
			return
		}
		validTo = &parsed
	}
	assignment := &DoctorTerritory{
		ID: uuid.New(), TenantID: tenant(c), DoctorID: id, TerritoryID: request.TerritoryID,
		ValidFrom: validFrom, ValidTo: validTo, IsPrimary: request.IsPrimary,
	}
	if err := conflict(h.db.WithContext(c.Request.Context()).Create(assignment).Error); err != nil {
		httputil.FailError(c, err)
		return
	}
	response.Created(c, "Doctor territory assigned", assignment)
}
