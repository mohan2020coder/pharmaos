package syncapi

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Change struct {
	Entity    string                 `json:"entity"`
	EntityID  string                 `json:"entity_id"`
	Operation string                 `json:"op"`
	UpdatedAt time.Time              `json:"updated_at"`
	Payload   map[string]interface{} `json:"payload"`
}

type ChangesResponse struct {
	NextCursor string   `json:"next_cursor"`
	HasMore    bool     `json:"has_more"`
	Changes    []Change `json:"changes"`
}

type StatusResponse struct {
	ServerTime        time.Time        `json:"server_time"`
	LatestCursor      string           `json:"latest_cursor"`
	PendingOperations int64            `json:"pending_operations"`
	SupportedEntities []string         `json:"supported_entities"`
	EntityCounts      map[string]int64 `json:"entity_counts"`
}

type Operation struct {
	ID             uuid.UUID       `json:"id"`
	Entity         string          `json:"entity"`
	EntityID       uuid.UUID       `json:"entity_id"`
	Operation      string          `json:"operation"`
	Payload        json.RawMessage `json:"payload"`
	IdempotencyKey uuid.UUID       `json:"idempotency_key"`
}

type PushRequest struct {
	Operations []Operation `json:"operations"`
}

type OperationResult struct {
	ClientOperationID uuid.UUID `json:"client_operation_id"`
	Status            string    `json:"status"`
	Entity            string    `json:"entity"`
	EntityID          uuid.UUID `json:"entity_id,omitempty"`
	ErrorCode         string    `json:"error_code,omitempty"`
	ErrorMessage      string    `json:"error_message,omitempty"`
	Retryable         bool      `json:"retryable"`
}

type PushResponse struct {
	Operations []OperationResult `json:"operations"`
}

type cursorData struct {
	UpdatedAt time.Time `json:"u"`
	Entity    string    `json:"e"`
	ID        uuid.UUID `json:"i"`
}

type entitySpec struct {
	Table                string
	Columns              []string
	UpdatedColumn        string
	EmployeeColumn       string
	TerritoryColumn      string
	DivisionColumn       string
	DeletedColumn        string
	ParentTable          string
	ParentIDColumn       string
	ParentEmployeeColumn string
	Visibility           string
}

var entitySpecs = map[string]entitySpec{
	"doctor": {
		Table: "crm_doctors", UpdatedColumn: "updated_at", DeletedColumn: "deleted_at",
		TerritoryColumn: "territory_id", DivisionColumn: "division_id",
		Columns: []string{"id::text AS id", "tenant_id::text AS tenant_id", "name", "specialty", "qualification",
			"registration_no", "territory_id::text AS territory_id", "division_id::text AS division_id",
			"is_active", "created_at", "updated_at", "deleted_at"},
	},
	"chemist": {
		Table: "crm_chemists", UpdatedColumn: "updated_at", DeletedColumn: "deleted_at",
		TerritoryColumn: "territory_id", DivisionColumn: "division_id",
		Columns: []string{"id::text AS id", "tenant_id::text AS tenant_id", "name", "owner_name", "license_no",
			"territory_id::text AS territory_id", "division_id::text AS division_id", "is_active",
			"created_at", "updated_at", "deleted_at"},
	},
	"stockist": {
		Table: "crm_stockists", UpdatedColumn: "updated_at", DeletedColumn: "deleted_at",
		TerritoryColumn: "territory_id", DivisionColumn: "division_id",
		Columns: []string{"id::text AS id", "tenant_id::text AS tenant_id", "name", "license_no", "gst_number",
			"credit_limit::float8 AS credit_limit", "payment_terms_days", "territory_id::text AS territory_id",
			"division_id::text AS division_id", "is_active", "created_at", "updated_at", "deleted_at"},
	},
	"product_category": {
		Table: "cat_product_categories", UpdatedColumn: "updated_at", DeletedColumn: "deleted_at",
		Columns: []string{"id::text AS id", "tenant_id::text AS tenant_id", "code::text AS code", "name",
			"parent_category_id::text AS parent_category_id", "is_active", "created_at", "updated_at", "deleted_at"},
	},
	"product": {
		Table: "cat_products", UpdatedColumn: "updated_at", DeletedColumn: "deleted_at",
		Columns: []string{"id::text AS id", "tenant_id::text AS tenant_id", "category_id::text AS category_id",
			"code::text AS code", "name", "brand", "manufacturer", "composition", "pack_size", "unit",
			"mrp::float8 AS mrp", "is_active", "created_at", "updated_at", "deleted_at"},
	},
	"target": {
		Table: "crm_targets", UpdatedColumn: "updated_at", DeletedColumn: "deleted_at",
		EmployeeColumn: "employee_id", TerritoryColumn: "territory_id",
		Columns: []string{"id::text AS id", "tenant_id::text AS tenant_id", "entity_type",
			"entity_id::text AS entity_id", "employee_id::text AS employee_id", "territory_id::text AS territory_id",
			"period_start", "period_end", "target_visits", "target_sales::float8 AS target_sales", "status",
			"created_at", "updated_at", "deleted_at"},
	},
	"territory": {
		Table: "territories", UpdatedColumn: "updated_at", DeletedColumn: "deleted_at",
		TerritoryColumn: "id", DivisionColumn: "division_id",
		Columns: []string{"id::text AS id", "tenant_id::text AS tenant_id", "division_id::text AS division_id",
			"region_id::text AS region_id", "branch_id::text AS branch_id", "code", "name", "city", "status",
			"created_at", "updated_at", "deleted_at"},
	},
	"employee": {
		Table: "employees", UpdatedColumn: "updated_at", DeletedColumn: "deleted_at",
		EmployeeColumn: "id", TerritoryColumn: "territory_id", DivisionColumn: "division_id",
		Columns: []string{"id::text AS id", "tenant_id::text AS tenant_id", "employee_code",
			"user_id::text AS user_id", "first_name", "last_name", "email::text AS email", "status",
			"joining_date", "division_id::text AS division_id", "region_id::text AS region_id",
			"territory_id::text AS territory_id", "manager_employee_id::text AS manager_employee_id",
			"position_id::text AS position_id", "created_at", "updated_at", "deleted_at"},
	},
	"tour_plan": {
		Table: "tour_plans", UpdatedColumn: "updated_at", DeletedColumn: "deleted_at",
		EmployeeColumn: "employee_id",
		Columns: []string{"id::text AS id", "tenant_id::text AS tenant_id", "employee_id::text AS employee_id",
			"title", "period_start", "period_end", "status", "submitted_at", "comments", "created_at",
			"updated_at", "deleted_at"},
	},
	"tour_plan_line": {
		Table: "tour_plan_lines", UpdatedColumn: "updated_at",
		ParentTable: "tour_plans", ParentIDColumn: "tour_plan_id", ParentEmployeeColumn: "employee_id",
		Columns: []string{"id::text AS id", "tenant_id::text AS tenant_id", "tour_plan_id::text AS tour_plan_id",
			"planned_date", "territory_id::text AS territory_id", "entity_type", "entity_id::text AS entity_id",
			"purpose", "sequence", "status", "created_at", "updated_at"},
	},
	"visit": {
		Table: "visits", UpdatedColumn: "updated_at", DeletedColumn: "deleted_at",
		EmployeeColumn: "employee_id", TerritoryColumn: "territory_id",
		Columns: []string{"id::text AS id", "tenant_id::text AS tenant_id", "employee_id::text AS employee_id",
			"tour_plan_line_id::text AS tour_plan_line_id", "territory_id::text AS territory_id", "entity_type",
			"entity_id::text AS entity_id", "client_reference_id::text AS client_reference_id", "visited_at",
			"check_in_lat", "check_in_lng", "check_in_accuracy", "status", "remarks", "submitted_at",
			"created_at", "updated_at", "deleted_at"},
	},
	"dcr": {
		Table: "daily_call_reports", UpdatedColumn: "updated_at", DeletedColumn: "deleted_at",
		EmployeeColumn: "employee_id",
		Columns: []string{"id::text AS id", "tenant_id::text AS tenant_id", "employee_id::text AS employee_id",
			"report_date", "total_visits", "status", "submitted_at", "remarks", "created_at", "updated_at", "deleted_at"},
	},
	"dcr_product": {
		Table: "dcr_product_details", UpdatedColumn: "updated_at",
		ParentTable: "daily_call_reports", ParentIDColumn: "dcr_id", ParentEmployeeColumn: "employee_id",
		Columns: []string{"id::text AS id", "tenant_id::text AS tenant_id", "dcr_id::text AS dcr_id",
			"product_id::text AS product_id", "quantity", "sample_count", "outcome", "remarks", "created_at", "updated_at"},
	},
	"distributor": {
		Table: "dist_distributors", UpdatedColumn: "updated_at", DeletedColumn: "deleted_at", Visibility: "distributor_profile",
		Columns: []string{"id::text AS id", "tenant_id::text AS tenant_id", "code::text AS code", "name",
			"license_no", "email::text AS email", "phone", "address", "status", "created_at", "updated_at", "deleted_at"},
	},
	"distribution_link": {
		Table: "dist_company_distributor_links", UpdatedColumn: "updated_at", Visibility: "distribution_link",
		Columns: []string{"id::text AS id", "tenant_id::text AS tenant_id", "distributor_tenant_id::text AS distributor_tenant_id",
			"status", "effective_from", "effective_to", "credit_limit::float8 AS credit_limit", "payment_terms_days",
			"created_at", "updated_at"},
	},
	"order": {
		Table: "dist_orders", UpdatedColumn: "updated_at", DeletedColumn: "deleted_at", Visibility: "order",
		Columns: []string{"id::text AS id", "tenant_id::text AS tenant_id", "distributor_tenant_id::text AS distributor_tenant_id",
			"link_id::text AS link_id", "order_number::text AS order_number", "status", "currency_code",
			"total_amount::float8 AS total_amount", "order_date", "required_by", "submitted_at", "approved_at",
			"cancelled_at", "remarks", "created_at", "updated_at", "deleted_at"},
	},
	"order_item": {
		Table: "dist_order_items", UpdatedColumn: "updated_at", Visibility: "order_child",
		Columns: []string{"id::text AS id", "tenant_id::text AS tenant_id", "order_id::text AS order_id",
			"product_id::text AS product_id", "quantity", "unit_price::float8 AS unit_price",
			"line_total::float8 AS line_total", "created_at", "updated_at"},
	},
	"inventory": {
		Table: "dist_inventory", UpdatedColumn: "updated_at", Visibility: "distributor_data",
		Columns: []string{"id::text AS id", "tenant_id::text AS tenant_id", "product_id::text AS product_id",
			"warehouse_code", "quantity_on_hand", "quantity_reserved", "version", "created_at", "updated_at"},
	},
	"inventory_movement": {
		Table: "dist_inventory_movements", UpdatedColumn: "occurred_at", Visibility: "distributor_data",
		Columns: []string{"id::text AS id", "tenant_id::text AS tenant_id", "product_id::text AS product_id",
			"warehouse_code", "movement_type", "quantity_delta", "reference_type", "reference_id::text AS reference_id",
			"idempotency_key::text AS idempotency_key", "occurred_at", "created_at"},
	},
	"secondary_sale": {
		Table: "dist_secondary_sales", UpdatedColumn: "updated_at", Visibility: "secondary_sale",
		Columns: []string{"id::text AS id", "tenant_id::text AS tenant_id", "order_id::text AS order_id",
			"company_tenant_id::text AS company_tenant_id", "product_id::text AS product_id", "quantity",
			"unit_price::float8 AS unit_price", "total_amount::float8 AS total_amount", "sale_date",
			"invoice_no::text AS invoice_no", "customer_code", "created_at", "updated_at"},
	},
}
