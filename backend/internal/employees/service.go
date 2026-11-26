package employees

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/pharmaos/pharmaos/backend/internal/auth"
	"github.com/pharmaos/pharmaos/backend/internal/platform/crud"
	apperrors "github.com/pharmaos/pharmaos/backend/internal/platform/errors"
	"github.com/pharmaos/pharmaos/backend/internal/platform/query"
)

type CreateRequest struct {
	EmployeeCode string     `json:"employee_code" validate:"required,min=2,max=40"`
	UserID       *uuid.UUID `json:"user_id"`
	FirstName    string     `json:"first_name" validate:"required,min=2,max=80"`
	LastName     string     `json:"last_name" validate:"required,min=2,max=80"`
	Email        string     `json:"email" validate:"omitempty,email"`
	Phone        string     `json:"phone"`
	JoiningDate  string     `json:"joining_date"`
	DivisionID   *uuid.UUID `json:"division_id"`
	RegionID     *uuid.UUID `json:"region_id"`
	TerritoryID  *uuid.UUID `json:"territory_id"`
	ManagerID    *uuid.UUID `json:"manager_employee_id"`
	PositionID   *uuid.UUID `json:"position_id"`
}

type TransferRequest struct {
	DivisionID    *uuid.UUID `json:"division_id"`
	RegionID      *uuid.UUID `json:"region_id"`
	TerritoryID   *uuid.UUID `json:"territory_id"`
	ManagerID     *uuid.UUID `json:"manager_employee_id"`
	PositionID    *uuid.UUID `json:"position_id"`
	EffectiveFrom string     `json:"effective_from" validate:"required"`
	Reason        string     `json:"reason"`
}

type Service struct {
	employees *crud.Repository[Employee]
	positions *crud.Repository[Position]
	db        *gorm.DB
}

func NewService(db *gorm.DB, employees *crud.Repository[Employee], positions *crud.Repository[Position]) *Service {
	return &Service{db: db, employees: employees, positions: positions}
}

func (s *Service) Create(ctx context.Context, principal auth.Principal, request CreateRequest) (*Employee, error) {
	employee := &Employee{
		ID: uuid.New(), TenantID: principal.TenantID,
		EmployeeCode: strings.ToUpper(strings.TrimSpace(request.EmployeeCode)),
		UserID:       request.UserID, FirstName: request.FirstName, LastName: request.LastName,
		Email: request.Email, Phone: request.Phone, Status: "ACTIVE",
		DivisionID: request.DivisionID, RegionID: request.RegionID, TerritoryID: request.TerritoryID,
		ManagerEmployeeID: request.ManagerID, PositionID: request.PositionID,
	}
	if date, err := time.Parse("2006-01-02", request.JoiningDate); err == nil {
		employee.JoiningDate = &date
	}
	if err := s.employees.Create(ctx, employee); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return nil, apperrors.ErrConflict
		}
		return nil, err
	}
	tx := s.db.WithContext(ctx)
	if employee.TerritoryID != nil {
		if err := tx.Exec(`INSERT INTO employee_territory_assignments (tenant_id, employee_id, territory_id, valid_from)
			VALUES (?, ?, ?, COALESCE(?, CURRENT_DATE))`, principal.TenantID, employee.ID, *employee.TerritoryID, employee.JoiningDate).Error; err != nil {
			return nil, err
		}
	}
	if employee.ManagerEmployeeID != nil {
		if err := tx.Exec(`INSERT INTO employee_manager_history (tenant_id, employee_id, manager_employee_id, valid_from)
			VALUES (?, ?, ?, COALESCE(?, CURRENT_DATE))`, principal.TenantID, employee.ID, *employee.ManagerEmployeeID, employee.JoiningDate).Error; err != nil {
			return nil, err
		}
	}
	return employee, nil
}

func (s *Service) List(ctx context.Context, principal auth.Principal, page query.Page, divisionID, territoryID, managerID uuid.UUID) ([]Employee, int64, error) {
	var filters []func(*gorm.DB) *gorm.DB
	if divisionID != uuid.Nil {
		filters = append(filters, func(db *gorm.DB) *gorm.DB { return db.Where("division_id = ?", divisionID) })
	}
	if territoryID != uuid.Nil {
		filters = append(filters, func(db *gorm.DB) *gorm.DB { return db.Where("territory_id = ?", territoryID) })
	}
	if managerID != uuid.Nil {
		filters = append(filters, func(db *gorm.DB) *gorm.DB { return db.Where("manager_employee_id = ?", managerID) })
	}
	return s.employees.List(ctx, principal.TenantID, page, filters...)
}

func (s *Service) Current(ctx context.Context, principal auth.Principal) (*Employee, error) {
	var employee Employee
	err := s.db.WithContext(ctx).
		Where("tenant_id = ? AND user_id = ?", principal.TenantID, principal.UserID).
		First(&employee).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperrors.ErrEmployeeProfileNotFound
	}
	if err != nil {
		return nil, err
	}
	return &employee, nil
}

func (s *Service) Get(ctx context.Context, principal auth.Principal, id uuid.UUID) (*Employee, error) {
	return s.employees.Get(ctx, principal.TenantID, id)
}

func (s *Service) Transfer(ctx context.Context, principal auth.Principal, id uuid.UUID, request TransferRequest) error {
	employee, err := s.employees.Get(ctx, principal.TenantID, id)
	if err != nil {
		return err
	}
	effectiveFrom, err := time.Parse("2006-01-02", request.EffectiveFrom)
	if err != nil {
		return apperrors.New("INVALID_DATE", "effective_from must be YYYY-MM-DD", 422)
	}
	previousFrom := effectiveFrom.AddDate(0, 0, -1)
	tx := s.db.WithContext(ctx)

	if request.TerritoryID != nil && (employee.TerritoryID == nil || *request.TerritoryID != *employee.TerritoryID) {
		if err := tx.Exec(`UPDATE employee_territory_assignments SET valid_to = ?
			WHERE tenant_id = ? AND employee_id = ? AND valid_to IS NULL`, previousFrom, principal.TenantID, id).Error; err != nil {
			return err
		}
		if err := tx.Exec(`INSERT INTO employee_territory_assignments (tenant_id, employee_id, territory_id, valid_from)
			VALUES (?, ?, ?, ?)`, principal.TenantID, id, *request.TerritoryID, effectiveFrom).Error; err != nil {
			return err
		}
	}
	if request.ManagerID != nil && (employee.ManagerEmployeeID == nil || *request.ManagerID != *employee.ManagerEmployeeID) {
		if err := tx.Exec(`UPDATE employee_manager_history SET valid_to = ?
			WHERE tenant_id = ? AND employee_id = ? AND valid_to IS NULL`, previousFrom, principal.TenantID, id).Error; err != nil {
			return err
		}
		if err := tx.Exec(`INSERT INTO employee_manager_history (tenant_id, employee_id, manager_employee_id, valid_from)
			VALUES (?, ?, ?, ?)`, principal.TenantID, id, *request.ManagerID, effectiveFrom).Error; err != nil {
			return err
		}
	}
	if err := tx.Exec(`INSERT INTO employee_transfers
		(tenant_id, employee_id, from_division_id, to_division_id, from_territory_id, to_territory_id, from_manager_id, to_manager_id, effective_from, reason)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, principal.TenantID, id, employee.DivisionID, request.DivisionID,
		employee.TerritoryID, request.TerritoryID, employee.ManagerEmployeeID, request.ManagerID, effectiveFrom, request.Reason).Error; err != nil {
		return err
	}
	updates := map[string]interface{}{}
	if request.DivisionID != nil {
		updates["division_id"] = *request.DivisionID
	}
	if request.RegionID != nil {
		updates["region_id"] = *request.RegionID
	}
	if request.TerritoryID != nil {
		updates["territory_id"] = *request.TerritoryID
	}
	if request.ManagerID != nil {
		updates["manager_employee_id"] = *request.ManagerID
	}
	if request.PositionID != nil {
		updates["position_id"] = *request.PositionID
	}
	if len(updates) > 0 {
		return tx.Model(&Employee{}).Where("tenant_id = ? AND id = ?", principal.TenantID, id).Updates(updates).Error
	}
	return nil
}

func (s *Service) History(ctx context.Context, principal auth.Principal, employeeID uuid.UUID) (map[string]interface{}, error) {
	if _, err := s.employees.Get(ctx, principal.TenantID, employeeID); err != nil {
		return nil, err
	}
	tx := s.db.WithContext(ctx)
	var territories []TerritoryAssignment
	if err := tx.Where("tenant_id = ? AND employee_id = ?", principal.TenantID, employeeID).Order("valid_from DESC").Find(&territories).Error; err != nil {
		return nil, err
	}
	var managers []ManagerHistory
	if err := tx.Where("tenant_id = ? AND employee_id = ?", principal.TenantID, employeeID).Order("valid_from DESC").Find(&managers).Error; err != nil {
		return nil, err
	}
	var transfers []Transfer
	if err := tx.Where("tenant_id = ? AND employee_id = ?", principal.TenantID, employeeID).Order("effective_from DESC").Find(&transfers).Error; err != nil {
		return nil, err
	}
	return map[string]interface{}{"territories": territories, "managers": managers, "transfers": transfers}, nil
}

func (s *Service) CreatePosition(ctx context.Context, principal auth.Principal, code, title string, level int, management bool, parent *uuid.UUID) (*Position, error) {
	position := &Position{
		ID: uuid.New(), TenantID: principal.TenantID, Code: strings.ToUpper(strings.TrimSpace(code)),
		Title: strings.TrimSpace(title), Level: level, IsManagement: management, ParentPositionID: parent,
	}
	if err := s.positions.Create(ctx, position); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return nil, apperrors.ErrConflict
		}
		return nil, err
	}
	return position, nil
}

func (s *Service) ListPositions(ctx context.Context, principal auth.Principal) ([]Position, error) {
	var positions []Position
	err := s.db.WithContext(ctx).Where("tenant_id = ?", principal.TenantID).Order("level ASC, title ASC").Find(&positions).Error
	return positions, err
}
