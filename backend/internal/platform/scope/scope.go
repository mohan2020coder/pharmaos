package scope

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/pharmaos/pharmaos/backend/internal/auth"
	"github.com/pharmaos/pharmaos/backend/internal/employees"
	apperrors "github.com/pharmaos/pharmaos/backend/internal/platform/errors"
)

func freshSession(db *gorm.DB) *gorm.DB {
	session := db.Session(&gorm.Session{NewDB: true})
	if db.Statement != nil && db.Statement.Context != nil {
		session = session.WithContext(db.Statement.Context)
	}
	return session
}

func Apply(db *gorm.DB, principal auth.Principal, employeeColumn, divisionColumn, territoryColumn string) *gorm.DB {
	if principal.IsPrivileged || principal.DataScope == "" || principal.DataScope == "TENANT" {
		return db
	}
	switch principal.DataScope {
	case "SELF":
		if employeeColumn == "" {
			return db
		}
		return db.Where(employeeColumn+" IN (?)",
			freshSession(db).Table(employees.Employee{}.TableName()).Select("id").
				Where("tenant_id = ? AND user_id = ? AND deleted_at IS NULL", principal.TenantID, principal.UserID))
	case "TEAM":
		if employeeColumn == "" {
			return db
		}
		return db.Where(employeeColumn+" IN (?)", freshSession(db).Raw(`WITH RECURSIVE team AS (
			SELECT id FROM employees WHERE tenant_id = ? AND user_id = ?
			UNION
			SELECT e.id FROM employees e JOIN team t ON e.manager_employee_id = t.id WHERE e.tenant_id = ?
		) SELECT id FROM team`, principal.TenantID, principal.UserID, principal.TenantID))
	case "DIVISION":
		if divisionColumn == "" {
			return db
		}
		if len(principal.DivisionIDs) == 0 {
			return db.Where("1 = 0")
		}
		return db.Where(divisionColumn+" IN ?", principal.DivisionIDs)
	case "TERRITORY":
		if territoryColumn == "" {
			return db
		}
		if len(principal.TerritoryIDs) == 0 {
			return db.Where("1 = 0")
		}
		return db.Where(territoryColumn+" IN ?", principal.TerritoryIDs)
	default:
		return db
	}
}

func HasLinkedEmployee(db *gorm.DB, principal auth.Principal) bool {
	var count int64
	err := freshSession(db).Table(employees.Employee{}.TableName()).
		Where("tenant_id = ? AND user_id = ? AND deleted_at IS NULL", principal.TenantID, principal.UserID).
		Count(&count).Error
	return err == nil && count > 0
}

func EmployeeInScopeError(db *gorm.DB, principal auth.Principal, employeeID uuid.UUID) error {
	if EmployeeInScope(db, principal, employeeID) {
		return nil
	}
	if (principal.DataScope == "SELF" || principal.DataScope == "TEAM") && !HasLinkedEmployee(db, principal) {
		return apperrors.ErrNoEmployeeProfile
	}
	return apperrors.ErrForbidden
}

func EmployeeInScope(db *gorm.DB, principal auth.Principal, employeeID uuid.UUID) bool {
	base := freshSession(db).Table(employees.Employee{}.TableName()).
		Where("tenant_id = ? AND id = ? AND deleted_at IS NULL", principal.TenantID, employeeID)
	var count int64
	if err := Apply(base, principal, "id", "division_id", "territory_id").Count(&count).Error; err != nil {
		return false
	}
	return count > 0
}
