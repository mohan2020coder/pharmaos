package employees

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Employee struct {
	ID                uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID          uuid.UUID      `gorm:"type:uuid;not null" json:"tenant_id"`
	EmployeeCode      string         `gorm:"not null" json:"employee_code"`
	UserID            *uuid.UUID     `gorm:"type:uuid" json:"user_id,omitempty"`
	FirstName         string         `gorm:"not null" json:"first_name"`
	LastName          string         `gorm:"not null" json:"last_name"`
	Email             string         `gorm:"type:citext" json:"email,omitempty"`
	Phone             string         `json:"phone,omitempty"`
	Status            string         `gorm:"not null;default:ACTIVE" json:"status"`
	JoiningDate       *time.Time     `gorm:"type:date" json:"joining_date,omitempty"`
	DivisionID        *uuid.UUID     `gorm:"type:uuid" json:"division_id,omitempty"`
	RegionID          *uuid.UUID     `gorm:"type:uuid" json:"region_id,omitempty"`
	TerritoryID       *uuid.UUID     `gorm:"type:uuid" json:"territory_id,omitempty"`
	ManagerEmployeeID *uuid.UUID     `gorm:"type:uuid" json:"manager_employee_id,omitempty"`
	PositionID        *uuid.UUID     `gorm:"type:uuid" json:"position_id,omitempty"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Employee) TableName() string { return "employees" }

type Position struct {
	ID               uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID         uuid.UUID  `gorm:"type:uuid;not null" json:"tenant_id"`
	Code             string     `gorm:"not null" json:"code"`
	Title            string     `gorm:"not null" json:"title"`
	Level            int        `gorm:"not null" json:"level"`
	IsManagement     bool       `gorm:"not null" json:"is_management"`
	ParentPositionID *uuid.UUID `gorm:"type:uuid" json:"parent_position_id,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

func (Position) TableName() string { return "positions" }

type TerritoryAssignment struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID       uuid.UUID  `gorm:"type:uuid;not null" json:"tenant_id"`
	EmployeeID     uuid.UUID  `gorm:"type:uuid;not null" json:"employee_id"`
	TerritoryID    uuid.UUID  `gorm:"type:uuid;not null" json:"territory_id"`
	AssignmentType string     `gorm:"not null;default:PRIMARY" json:"assignment_type"`
	ValidFrom      time.Time  `gorm:"type:date;not null" json:"valid_from"`
	ValidTo        *time.Time `gorm:"type:date" json:"valid_to,omitempty"`
}

func (TerritoryAssignment) TableName() string { return "employee_territory_assignments" }

type ManagerHistory struct {
	ID                uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID          uuid.UUID  `gorm:"type:uuid;not null" json:"tenant_id"`
	EmployeeID        uuid.UUID  `gorm:"type:uuid;not null" json:"employee_id"`
	ManagerEmployeeID uuid.UUID  `gorm:"type:uuid;not null" json:"manager_employee_id"`
	ValidFrom         time.Time  `gorm:"type:date;not null" json:"valid_from"`
	ValidTo           *time.Time `gorm:"type:date" json:"valid_to,omitempty"`
}

func (ManagerHistory) TableName() string { return "employee_manager_history" }

type Transfer struct {
	ID              uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID        uuid.UUID  `gorm:"type:uuid;not null" json:"tenant_id"`
	EmployeeID      uuid.UUID  `gorm:"type:uuid;not null" json:"employee_id"`
	FromDivisionID  *uuid.UUID `gorm:"type:uuid" json:"from_division_id,omitempty"`
	ToDivisionID    *uuid.UUID `gorm:"type:uuid" json:"to_division_id,omitempty"`
	FromTerritoryID *uuid.UUID `gorm:"type:uuid" json:"from_territory_id,omitempty"`
	ToTerritoryID   *uuid.UUID `gorm:"type:uuid" json:"to_territory_id,omitempty"`
	FromManagerID   *uuid.UUID `gorm:"type:uuid" json:"from_manager_id,omitempty"`
	ToManagerID     *uuid.UUID `gorm:"type:uuid" json:"to_manager_id,omitempty"`
	EffectiveFrom   time.Time  `gorm:"type:date;not null" json:"effective_from"`
	Reason          string     `json:"reason,omitempty"`
}

func (Transfer) TableName() string { return "employee_transfers" }
