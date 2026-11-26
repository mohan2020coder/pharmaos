package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/pharmaos/pharmaos/backend/internal/config"
	"github.com/pharmaos/pharmaos/backend/internal/database"
	"github.com/pharmaos/pharmaos/backend/internal/platform/logging"
)

const (
	companyCode     = "DEMO"
	distributorCode = "DEMODIST"
	demoPassword    = "DemoPass123!"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	logger, err := logging.New(cfg.LogLevel, cfg.Environment == "production")
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	password := env("PHARMAOS_SEED_PASSWORD", demoPassword)
	if len(password) < 12 {
		fmt.Fprintln(os.Stderr, "PHARMAOS_SEED_PASSWORD must be at least 12 characters")
		os.Exit(1)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), cfg.PasswordHashCost)
	if err != nil {
		panic(err)
	}

	db, err := database.Connect(cfg.DatabaseURL, 5, 2, gormlogger.Warn)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	ctx := context.Background()
	tx := db.WithContext(ctx).Begin()
	defer func() {
		_ = tx.Rollback()
	}()
	if err := tx.Exec(`SELECT set_config('app.is_privileged', 'true', true)`).Error; err != nil {
		panic(err)
	}

	companyID := ensureTenant(tx, companyCode, "Demo Pharma", "PHARMA_COMPANY")
	distributorID := ensureTenant(tx, distributorCode, "Demo Distributor", "DISTRIBUTOR")
	mustExec(tx, `SELECT set_config('app.current_tenant_id', ?, true)`, companyID.String())

	adminID := ensureUser(tx, companyID, "admin@demo.test", "Demo", "Administrator", hash)
	managerID := ensureUser(tx, companyID, "manager@demo.test", "Priya", "Sharma", hash)
	mrID := ensureUser(tx, companyID, "mr@demo.test", "Arjun", "Mehta", hash)
	distributorUserID := ensureUser(tx, distributorID, "distributor@demo.test", "Rohan", "Verma", hash)
	viewerID := ensureUser(tx, companyID, "viewer@demo.test", "Neha", "Iyer", hash)

	assignRole(tx, adminID, "SUPER_ADMIN", uuid.Nil)
	assignRole(tx, managerID, "COMPANY_ADMIN", uuid.Nil)
	assignRole(tx, mrID, "MR", uuid.Nil)
	assignRole(tx, distributorUserID, "DISTRIBUTOR_ADMIN", uuid.Nil)
	assignRole(tx, viewerID, "REPORT_VIEWER", uuid.Nil)

	divisionID := fixedID("division:NORTH")
	regionID := fixedID("region:NORTH-01")
	territoryID := fixedID("territory:NORTH-01-A")
	branchID := fixedID("branch:NORTH-01")
	managerPositionID := fixedID("position:MANAGER")
	mrPositionID := fixedID("position:MR")
	adminEmployeeID := fixedID("employee:ADM-001")
	managerEmployeeID := fixedID("employee:MGR-001")
	mrEmployeeID := fixedID("employee:MR-001")
	viewerEmployeeID := fixedID("employee:VIEW-001")
	distributorEmployeeID := fixedID("employee:DIST-001")

	mustExec(tx, `INSERT INTO business_divisions (id, tenant_id, code, name, description, status)
		VALUES (?, ?, 'NORTH', 'North Division', 'Demo north territory', 'ACTIVE')
		ON CONFLICT (id) DO NOTHING`, divisionID, companyID)
	mustExec(tx, `INSERT INTO branches (id, tenant_id, division_id, code, name, city, state, status)
		VALUES (?, ?, ?, 'BR-NORTH-01', 'North Branch', 'Delhi', 'Delhi', 'ACTIVE')
		ON CONFLICT (id) DO NOTHING`, branchID, companyID, divisionID)
	mustExec(tx, `INSERT INTO regions (id, tenant_id, division_id, code, name, status)
		VALUES (?, ?, ?, 'REG-NORTH-01', 'North Region', 'ACTIVE')
		ON CONFLICT (id) DO NOTHING`, regionID, companyID, divisionID)
	mustExec(tx, `INSERT INTO territories (id, tenant_id, division_id, region_id, branch_id, code, name, city, status)
		VALUES (?, ?, ?, ?, ?, 'TER-NORTH-01-A', 'Delhi Central', 'Delhi', 'ACTIVE')
		ON CONFLICT (id) DO NOTHING`, territoryID, companyID, divisionID, regionID, branchID)
	mustExec(tx, `INSERT INTO positions (id, tenant_id, code, title, level, is_management)
		VALUES (?, ?, 'MANAGER', 'Area Sales Manager', 4, true)
		ON CONFLICT (id) DO NOTHING`, managerPositionID, companyID)
	mustExec(tx, `INSERT INTO positions (id, tenant_id, code, title, level, is_management, parent_position_id)
		VALUES (?, ?, 'MR', 'Medical Representative', 2, false, ?)
		ON CONFLICT (id) DO NOTHING`, mrPositionID, companyID, managerPositionID)

	mustExec(tx, `INSERT INTO employees
		(id, tenant_id, employee_code, user_id, first_name, last_name, email, phone, status, joining_date, division_id, region_id, territory_id, position_id)
		VALUES (?, ?, 'MGR-001', ?, 'Priya', 'Sharma', 'manager@demo.test', '9999000001', 'ACTIVE', CURRENT_DATE - 900, ?, ?, ?, ?)
		ON CONFLICT (id) DO NOTHING`, managerEmployeeID, companyID, managerID, divisionID, regionID, territoryID, managerPositionID)
	mustExec(tx, `INSERT INTO employees
		(id, tenant_id, employee_code, user_id, first_name, last_name, email, phone, status, joining_date, division_id, region_id, territory_id, manager_employee_id, position_id)
		VALUES (?, ?, 'MR-001', ?, 'Arjun', 'Mehta', 'mr@demo.test', '9999000002', 'ACTIVE', CURRENT_DATE - 420, ?, ?, ?, ?, ?)
		ON CONFLICT (id) DO NOTHING`, mrEmployeeID, companyID, mrID, divisionID, regionID, territoryID, managerEmployeeID, mrPositionID)
	mustExec(tx, `INSERT INTO employees
		(id, tenant_id, employee_code, user_id, first_name, last_name, email, phone, status, joining_date, division_id, region_id, territory_id, position_id)
		VALUES (?, ?, 'ADM-001', ?, 'Demo', 'Administrator', 'admin@demo.test', '9999000000', 'ACTIVE', CURRENT_DATE - 1000, ?, ?, ?, ?)
		ON CONFLICT DO NOTHING`, adminEmployeeID, companyID, adminID, divisionID, regionID, territoryID, managerPositionID)
	mustExec(tx, `INSERT INTO employees
		(id, tenant_id, employee_code, user_id, first_name, last_name, email, phone, status, joining_date, division_id, region_id, territory_id, position_id)
		VALUES (?, ?, 'VIEW-001', ?, 'Neha', 'Iyer', 'viewer@demo.test', '9999000004', 'ACTIVE', CURRENT_DATE - 300, ?, ?, ?, ?)
		ON CONFLICT DO NOTHING`, viewerEmployeeID, companyID, viewerID, divisionID, regionID, territoryID, managerPositionID)
	mustExec(tx, `INSERT INTO employees
		(id, tenant_id, employee_code, user_id, first_name, last_name, email, phone, status, joining_date)
		VALUES (?, ?, 'DIST-001', ?, 'Rohan', 'Verma', 'distributor@demo.test', '9999000003', 'ACTIVE', CURRENT_DATE - 300)
		ON CONFLICT DO NOTHING`, distributorEmployeeID, distributorID, distributorUserID)
	mustExec(tx, `INSERT INTO employee_positions (id, tenant_id, employee_id, position_id, valid_from, is_primary)
		VALUES (?, ?, ?, ?, CURRENT_DATE - 420, true)
		ON CONFLICT (id) DO NOTHING`, fixedID("employee-position:MR-001"), companyID, mrEmployeeID, mrPositionID)

	doctor1 := fixedID("doctor:DOC-001")
	doctor2 := fixedID("doctor:DOC-002")
	doctor3 := fixedID("doctor:DOC-003")
	chemist1 := fixedID("chemist:CHEM-001")
	chemist2 := fixedID("chemist:CHEM-002")
	stockist1 := fixedID("stockist:STK-001")
	stockist2 := fixedID("stockist:STK-002")

	mustExec(tx, `INSERT INTO crm_doctors (id, tenant_id, external_id, name, specialty, qualification, registration_no, email, phone, address, territory_id, division_id)
		VALUES (?, ?, 'DOC-001', 'Dr. Anita Rao', 'Cardiology', 'MD', 'MED-1001', 'anita.rao@demo.test', '9999100001', '12 MG Road, Delhi', ?, ?)
		ON CONFLICT (id) DO NOTHING`, doctor1, companyID, territoryID, divisionID)
	mustExec(tx, `INSERT INTO crm_doctors (id, tenant_id, external_id, name, specialty, qualification, registration_no, email, phone, address, territory_id, division_id)
		VALUES (?, ?, 'DOC-002', 'Dr. Vikram Sethi', 'General Medicine', 'MBBS', 'MED-1002', 'vikram.sethi@demo.test', '9999100002', '8 Park Street, Delhi', ?, ?)
		ON CONFLICT (id) DO NOTHING`, doctor2, companyID, territoryID, divisionID)
	mustExec(tx, `INSERT INTO crm_doctors (id, tenant_id, external_id, name, specialty, qualification, registration_no, email, phone, address, territory_id, division_id)
		VALUES (?, ?, 'DOC-003', 'Dr. Meera Iqbal', 'Pediatrics', 'MD', 'MED-1003', 'meera.iqbal@demo.test', '9999100003', '21 Sector 9, Delhi', ?, ?)
		ON CONFLICT (id) DO NOTHING`, doctor3, companyID, territoryID, divisionID)
	mustExec(tx, `INSERT INTO crm_chemists (id, tenant_id, external_id, name, owner_name, license_no, email, phone, address, territory_id, division_id)
		VALUES (?, ?, 'CHEM-001', 'Central Chemists', 'Suresh Kumar', 'CH-1001', 'central@demo.test', '9999200001', '5 Market Road, Delhi', ?, ?)
		ON CONFLICT (id) DO NOTHING`, chemist1, companyID, territoryID, divisionID)
	mustExec(tx, `INSERT INTO crm_chemists (id, tenant_id, external_id, name, owner_name, license_no, email, phone, address, territory_id, division_id)
		VALUES (?, ?, 'CHEM-002', 'Wellness Pharmacy', 'Kavita Jain', 'CH-1002', 'wellness@demo.test', '9999200002', '18 Nehru Place, Delhi', ?, ?)
		ON CONFLICT (id) DO NOTHING`, chemist2, companyID, territoryID, divisionID)
	mustExec(tx, `INSERT INTO crm_stockists (id, tenant_id, external_id, name, license_no, gst_number, email, phone, address, territory_id, division_id, credit_limit, payment_terms_days)
		VALUES (?, ?, 'STK-001', 'Delhi Medical Distributors', 'ST-1001', '07AABCU9603R1ZM', 'sales@dmd.test', '9999300001', '40 Ring Road, Delhi', ?, ?, 500000, 30)
		ON CONFLICT (id) DO NOTHING`, stockist1, companyID, territoryID, divisionID)
	mustExec(tx, `INSERT INTO crm_stockists (id, tenant_id, external_id, name, license_no, gst_number, email, phone, address, territory_id, division_id, credit_limit, payment_terms_days)
		VALUES (?, ?, 'STK-002', 'Capital Pharma House', 'ST-1002', '07AABCU9603R1ZN', 'sales@cph.test', '9999300002', '9 Connaught Place, Delhi', ?, ?, 350000, 15)
		ON CONFLICT (id) DO NOTHING`, stockist2, companyID, territoryID, divisionID)

	for index, doctorID := range []uuid.UUID{doctor1, doctor2, doctor3} {
		mustExec(tx, `INSERT INTO crm_doctor_territories (id, tenant_id, doctor_id, territory_id, valid_from, is_primary)
			VALUES (?, ?, ?, ?, CURRENT_DATE - 180, true)
			ON CONFLICT (id) DO NOTHING`, fixedID(fmt.Sprintf("doctor-territory:%d", index+1)), companyID, doctorID, territoryID)
	}

	cardioCategoryID := fixedID("category:CARDIO")
	generalCategoryID := fixedID("category:GENERAL")
	product1 := fixedID("product:CARD-10")
	product2 := fixedID("product:VIT-C")
	product3 := fixedID("product:ANTIBIOTIC-500")
	mustExec(tx, `INSERT INTO cat_product_categories (id, tenant_id, code, name, is_active)
		VALUES (?, ?, 'CARDIO', 'Cardiovascular', true)
		ON CONFLICT (id) DO NOTHING`, cardioCategoryID, companyID)
	mustExec(tx, `INSERT INTO cat_product_categories (id, tenant_id, code, name, is_active)
		VALUES (?, ?, 'GENERAL', 'General Range', true)
		ON CONFLICT (id) DO NOTHING`, generalCategoryID, companyID)
	mustExec(tx, `INSERT INTO cat_products (id, tenant_id, category_id, code, name, brand, manufacturer, composition, pack_size, unit, mrp)
		VALUES (?, ?, ?, 'CARD-10', 'Cardio Tablet 10mg', 'Cardiowell', 'Demo Pharma', 'Amlodipine 10mg', '10 tablets', 'STRIP', 125.50)
		ON CONFLICT (id) DO NOTHING`, product1, companyID, cardioCategoryID)
	mustExec(tx, `INSERT INTO cat_products (id, tenant_id, category_id, code, name, brand, manufacturer, composition, pack_size, unit, mrp)
		VALUES (?, ?, ?, 'VIT-C', 'Vitamin C 500mg', 'ImmunePlus', 'Demo Pharma', 'Ascorbic Acid 500mg', '15 tablets', 'STRIP', 89.00)
		ON CONFLICT (id) DO NOTHING`, product2, companyID, generalCategoryID)
	mustExec(tx, `INSERT INTO cat_products (id, tenant_id, category_id, code, name, brand, manufacturer, composition, pack_size, unit, mrp)
		VALUES (?, ?, ?, 'ANTIBIOTIC-500', 'Antibiotic 500mg', 'Curewell', 'Demo Pharma', 'Azithromycin 500mg', '5 tablets', 'STRIP', 210.75)
		ON CONFLICT (id) DO NOTHING`, product3, companyID, generalCategoryID)
	mustExec(tx, `INSERT INTO crm_targets (id, tenant_id, entity_type, entity_id, employee_id, territory_id, period_start, period_end, target_visits, target_sales, status)
		VALUES (?, ?, 'DOCTOR', ?, ?, ?, CURRENT_DATE - 30, CURRENT_DATE + 60, 40, 250000, 'ACTIVE')
		ON CONFLICT (id) DO NOTHING`, fixedID("target:MR-001:DOC-001"), companyID, doctor1, mrEmployeeID, territoryID)

	tourPlanID := fixedID("tour-plan:MR-001:CURRENT")
	tourLine1 := fixedID("tour-line:MR-001:1")
	tourLine2 := fixedID("tour-line:MR-001:2")
	mustExec(tx, `INSERT INTO tour_plans (id, tenant_id, employee_id, title, period_start, period_end, status, submitted_at, comments)
		VALUES (?, ?, ?, 'North Delhi weekly plan', CURRENT_DATE - 6, CURRENT_DATE, 'SUBMITTED', now(), 'Demo submitted plan')
		ON CONFLICT (id) DO NOTHING`, tourPlanID, companyID, mrEmployeeID)
	mustExec(tx, `INSERT INTO tour_plan_lines (id, tenant_id, tour_plan_id, planned_date, territory_id, entity_type, entity_id, purpose, sequence, status)
		VALUES (?, ?, ?, CURRENT_DATE - 5, ?, 'DOCTOR', ?, 'Product introduction', 1, 'COMPLETED')
		ON CONFLICT (id) DO NOTHING`, tourLine1, companyID, tourPlanID, territoryID, doctor1)
	mustExec(tx, `INSERT INTO tour_plan_lines (id, tenant_id, tour_plan_id, planned_date, territory_id, entity_type, entity_id, purpose, sequence, status)
		VALUES (?, ?, ?, CURRENT_DATE - 4, ?, 'CHEMIST', ?, 'Stock review', 2, 'PLANNED')
		ON CONFLICT (id) DO NOTHING`, tourLine2, companyID, tourPlanID, territoryID, chemist1)

	visit1 := fixedID("visit:DEMO-001")
	visit2 := fixedID("visit:DEMO-002")
	visit3 := fixedID("visit:DEMO-003")
	visit4 := fixedID("visit:DEMO-004")
	mustExec(tx, `INSERT INTO visits (id, tenant_id, employee_id, tour_plan_line_id, territory_id, entity_type, entity_id, client_reference_id, visited_at, check_in_lat, check_in_lng, check_in_accuracy, status, remarks, submitted_at)
		VALUES (?, ?, ?, ?, ?, 'DOCTOR', ?, ?, now() - interval '5 days', 28.6139, 77.2090, 12, 'SUBMITTED', 'Discussed Cardio Tablet 10mg', now() - interval '5 days')
		ON CONFLICT (id) DO NOTHING`, visit1, companyID, mrEmployeeID, tourLine1, territoryID, doctor1, fixedID("visit-ref:DEMO-001"))
	mustExec(tx, `INSERT INTO visits (id, tenant_id, employee_id, tour_plan_line_id, territory_id, entity_type, entity_id, client_reference_id, visited_at, check_in_lat, check_in_lng, check_in_accuracy, status, remarks, submitted_at)
		VALUES (?, ?, ?, ?, ?, 'CHEMIST', ?, ?, now() - interval '4 days', 28.6210, 77.2175, 15, 'SUBMITTED', 'Confirmed monthly stock', now() - interval '4 days')
		ON CONFLICT (id) DO NOTHING`, visit2, companyID, mrEmployeeID, tourLine2, territoryID, chemist1, fixedID("visit-ref:DEMO-002"))
	mustExec(tx, `INSERT INTO visits (id, tenant_id, employee_id, territory_id, entity_type, entity_id, client_reference_id, visited_at, check_in_lat, check_in_lng, check_in_accuracy, status, remarks)
		VALUES (?, ?, ?, ?, 'STOCKIST', ?, ?, now() - interval '2 days', 28.6320, 77.2240, 20, 'DRAFT', 'Pending secondary sales update')
		ON CONFLICT (id) DO NOTHING`, visit3, companyID, mrEmployeeID, territoryID, stockist1, fixedID("visit-ref:DEMO-003"))
	mustExec(tx, `INSERT INTO visits (id, tenant_id, employee_id, territory_id, entity_type, entity_id, client_reference_id, visited_at, check_in_lat, check_in_lng, check_in_accuracy, status, remarks)
		VALUES (?, ?, ?, ?, 'DOCTOR', ?, ?, now() - interval '1 day', 28.6400, 77.2300, 18, 'DRAFT', 'Follow-up required')
		ON CONFLICT (id) DO NOTHING`, visit4, companyID, mrEmployeeID, territoryID, doctor2, fixedID("visit-ref:DEMO-004"))

	dcr1 := fixedID("dcr:MR-001:CURRENT")
	dcr2 := fixedID("dcr:MR-001:YESTERDAY")
	mustExec(tx, `INSERT INTO daily_call_reports (id, tenant_id, employee_id, report_date, total_visits, status, submitted_at, remarks)
		VALUES (?, ?, ?, CURRENT_DATE, 2, 'SUBMITTED', now(), 'Demo daily call report')
		ON CONFLICT (id) DO NOTHING`, dcr1, companyID, mrEmployeeID)
	mustExec(tx, `INSERT INTO daily_call_reports (id, tenant_id, employee_id, report_date, total_visits, status, submitted_at, remarks)
		VALUES (?, ?, ?, CURRENT_DATE - 1, 1, 'SUBMITTED', now() - interval '1 day', 'Previous day demo report')
		ON CONFLICT (id) DO NOTHING`, dcr2, companyID, mrEmployeeID)
	mustExec(tx, `INSERT INTO dcr_product_details (id, tenant_id, dcr_id, product_id, quantity, sample_count, outcome, remarks)
		VALUES (?, ?, ?, ?, 12, 4, 'POSITIVE', 'Good acceptance')
		ON CONFLICT (id) DO NOTHING`, fixedID("dcr-detail:1"), companyID, dcr1, product1)
	mustExec(tx, `INSERT INTO dcr_product_details (id, tenant_id, dcr_id, product_id, quantity, sample_count, outcome, remarks)
		VALUES (?, ?, ?, ?, 20, 6, 'POSITIVE', 'High interest')
		ON CONFLICT (id) DO NOTHING`, fixedID("dcr-detail:2"), companyID, dcr1, product2)
	mustExec(tx, `INSERT INTO dcr_product_details (id, tenant_id, dcr_id, product_id, quantity, sample_count, outcome, remarks)
		VALUES (?, ?, ?, ?, 5, 2, 'FOLLOW_UP', 'Needs approval')
		ON CONFLICT (id) DO NOTHING`, fixedID("dcr-detail:3"), companyID, dcr2, product3)

	linkID := fixedID("link:DEMO:DEMODIST")
	distributorProfileID := fixedID("distributor:DEMO-001")
	mustExec(tx, `INSERT INTO dist_company_distributor_links
		(id, tenant_id, distributor_tenant_id, status, effective_from, credit_limit, payment_terms_days, created_by)
		VALUES (?, ?, ?, 'ACTIVE', CURRENT_DATE - 120, 500000, 30, ?)
		ON CONFLICT (id) DO NOTHING`, linkID, companyID, distributorID, adminID)
	mustExec(tx, `INSERT INTO dist_distributors (id, tenant_id, code, name, license_no, email, phone, address, status)
		VALUES (?, ?, 'DIST-001', 'Demo Distributor North', 'DL-1001', 'ops@demodist.test', '9999400001', '7 Wholesale Market, Delhi', 'ACTIVE')
		ON CONFLICT (id) DO NOTHING`, distributorProfileID, companyID)

	inventory1 := fixedID("inventory:CARD-10")
	inventory2 := fixedID("inventory:VIT-C")
	inventory3 := fixedID("inventory:ANTIBIOTIC-500")
	mustExec(tx, `INSERT INTO dist_inventory (id, tenant_id, product_id, warehouse_code, quantity_on_hand, quantity_reserved, version)
		VALUES (?, ?, ?, 'MAIN', 520, 20, 2)
		ON CONFLICT (id) DO NOTHING`, inventory1, companyID, product1)
	mustExec(tx, `INSERT INTO dist_inventory (id, tenant_id, product_id, warehouse_code, quantity_on_hand, quantity_reserved, version)
		VALUES (?, ?, ?, 'MAIN', 780, 0, 1)
		ON CONFLICT (id) DO NOTHING`, inventory2, companyID, product2)
	mustExec(tx, `INSERT INTO dist_inventory (id, tenant_id, product_id, warehouse_code, quantity_on_hand, quantity_reserved, version)
		VALUES (?, ?, ?, 'MAIN', 340, 0, 1)
		ON CONFLICT (id) DO NOTHING`, inventory3, companyID, product3)
	mustExec(tx, `INSERT INTO dist_inventory_movements (id, tenant_id, product_id, warehouse_code, movement_type, quantity_delta, reference_type, idempotency_key, created_by, occurred_at)
		VALUES (?, ?, ?, 'MAIN', 'RECEIPT', 500, 'SEED', ?, ?, now() - interval '10 days')
		ON CONFLICT (id) DO NOTHING`, fixedID("inventory-movement:1"), companyID, product1, fixedID("idem:movement:1"), adminID)
	mustExec(tx, `INSERT INTO dist_inventory_movements (id, tenant_id, product_id, warehouse_code, movement_type, quantity_delta, reference_type, idempotency_key, created_by, occurred_at)
		VALUES (?, ?, ?, 'MAIN', 'RECEIPT', 750, 'SEED', ?, ?, now() - interval '10 days')
		ON CONFLICT (id) DO NOTHING`, fixedID("inventory-movement:2"), companyID, product2, fixedID("idem:movement:2"), adminID)
	mustExec(tx, `INSERT INTO dist_inventory_movements (id, tenant_id, product_id, warehouse_code, movement_type, quantity_delta, reference_type, idempotency_key, created_by, occurred_at)
		VALUES (?, ?, ?, 'MAIN', 'RECEIPT', 320, 'SEED', ?, ?, now() - interval '10 days')
		ON CONFLICT (id) DO NOTHING`, fixedID("inventory-movement:3"), companyID, product3, fixedID("idem:movement:3"), adminID)

	order1 := fixedID("order:DEMO-001")
	order2 := fixedID("order:DEMO-002")
	mustExec(tx, `INSERT INTO dist_orders
		(id, tenant_id, distributor_tenant_id, link_id, order_number, status, currency_code, total_amount, idempotency_key, requested_by, approved_by, order_date, required_by, submitted_at, approved_at, remarks)
		VALUES (?, ?, ?, ?, 'DEMO-ORD-001', 'APPROVED', 'INR', 3210.00, ?, ?, ?, CURRENT_DATE - 3, CURRENT_DATE + 4, now() - interval '3 days', now() - interval '2 days', 'Approved demo order')
		ON CONFLICT (id) DO NOTHING`, order1, companyID, distributorID, linkID, fixedID("idem:order:1"), adminID, adminID)
	mustExec(tx, `INSERT INTO dist_orders
		(id, tenant_id, distributor_tenant_id, link_id, order_number, status, currency_code, total_amount, idempotency_key, requested_by, order_date, required_by, remarks)
		VALUES (?, ?, ?, ?, 'DEMO-ORD-002', 'DRAFT', 'INR', 1780.00, ?, ?, CURRENT_DATE, CURRENT_DATE + 7, 'Draft demo order')
		ON CONFLICT (id) DO NOTHING`, order2, companyID, distributorID, linkID, fixedID("idem:order:2"), adminID)
	mustExec(tx, `INSERT INTO dist_order_items (id, tenant_id, order_id, product_id, quantity, unit_price, line_total)
		VALUES (?, ?, ?, ?, 20, 125.50, 2510.00)
		ON CONFLICT (id) DO NOTHING`, fixedID("order-item:1"), companyID, order1, product1)
	mustExec(tx, `INSERT INTO dist_order_items (id, tenant_id, order_id, product_id, quantity, unit_price, line_total)
		VALUES (?, ?, ?, ?, 8, 87.50, 700.00)
		ON CONFLICT (id) DO NOTHING`, fixedID("order-item:2"), companyID, order1, product2)
	mustExec(tx, `INSERT INTO dist_order_items (id, tenant_id, order_id, product_id, quantity, unit_price, line_total)
		VALUES (?, ?, ?, ?, 8, 222.50, 1780.00)
		ON CONFLICT (id) DO NOTHING`, fixedID("order-item:3"), companyID, order2, product3)

	mustExec(tx, `INSERT INTO dist_secondary_sales (id, tenant_id, order_id, company_tenant_id, product_id, quantity, unit_price, total_amount, sale_date, invoice_no, customer_code, idempotency_key, created_by)
		VALUES (?, ?, ?, ?, ?, 12, 140.00, 1680.00, CURRENT_DATE - 1, 'DEMO-INV-001', 'CUST-001', ?, ?)
		ON CONFLICT (id) DO NOTHING`, fixedID("secondary-sale:1"), distributorID, order1, companyID, product1, fixedID("idem:secondary:1"), distributorUserID)
	mustExec(tx, `INSERT INTO dist_secondary_sales (id, tenant_id, order_id, company_tenant_id, product_id, quantity, unit_price, total_amount, sale_date, invoice_no, customer_code, idempotency_key, created_by)
		VALUES (?, ?, ?, ?, ?, 6, 95.00, 570.00, CURRENT_DATE, 'DEMO-INV-002', 'CUST-002', ?, ?)
		ON CONFLICT (id) DO NOTHING`, fixedID("secondary-sale:2"), distributorID, order1, companyID, product2, fixedID("idem:secondary:2"), distributorUserID)

	if err := tx.Commit().Error; err != nil {
		panic(err)
	}
	logger.Info(
		"demo seed complete",
		zap.String("company_tenant", companyCode),
		zap.String("distributor_tenant", distributorCode),
		zap.String("password", "[demo password]"),
	)
	fmt.Printf("Demo tenant %s and distributor %s are ready. Log in with admin@demo.test / %s.\n", companyCode, distributorCode, password)
}

func fixedID(name string) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte("https://demo.pharmaos.local/"+name))
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func mustExec(tx *gorm.DB, query string, args ...any) {
	if err := tx.Exec(query, args...).Error; err != nil {
		panic(err)
	}
}

func ensureTenant(tx *gorm.DB, code, name, tenantType string) uuid.UUID {
	var raw string
	if err := tx.Raw(`SELECT id::text FROM tenants WHERE code = ?`, code).Scan(&raw).Error; err != nil {
		panic(err)
	}
	if parsed, err := uuid.Parse(raw); err == nil && parsed != uuid.Nil {
		return parsed
	}
	tenantID := fixedID("tenant:" + code)
	mustExec(tx, `INSERT INTO tenants (id, code, name, tenant_type, status, is_platform, settings)
		VALUES (?, ?, ?, ?, 'ACTIVE', false, '{}'::jsonb)
		ON CONFLICT (id) DO NOTHING`, tenantID, code, name, tenantType)
	return tenantID
}

func ensureUser(tx *gorm.DB, tenantID uuid.UUID, email, firstName, lastName string, hash []byte) uuid.UUID {
	var raw string
	if err := tx.Raw(`SELECT id::text FROM users WHERE tenant_id = ? AND email = ?`, tenantID, email).Scan(&raw).Error; err != nil {
		panic(err)
	}
	if parsed, err := uuid.Parse(raw); err == nil && parsed != uuid.Nil {
		return parsed
	}
	userID := fixedID("user:" + email)
	mustExec(tx, `INSERT INTO users (id, tenant_id, email, password_hash, first_name, last_name, status, email_verified)
		VALUES (?, ?, ?, ?, ?, ?, 'ACTIVE', true)
		ON CONFLICT (tenant_id, email) DO NOTHING`, userID, tenantID, email, string(hash), firstName, lastName)
	if err := tx.Raw(`SELECT id::text FROM users WHERE tenant_id = ? AND email = ?`, tenantID, email).Scan(&raw).Error; err != nil {
		panic(err)
	}
	parsed, err := uuid.Parse(raw)
	if err != nil || parsed == uuid.Nil {
		panic("unable to resolve seeded user " + email)
	}
	return parsed
}

func assignRole(tx *gorm.DB, userID uuid.UUID, code string, scopeID uuid.UUID) {
	var raw string
	if err := tx.Raw(`SELECT id::text FROM roles WHERE code = ?`, code).Scan(&raw).Error; err != nil {
		panic(err)
	}
	roleID, err := uuid.Parse(raw)
	if err != nil || roleID == uuid.Nil {
		panic("role not found: " + code)
	}
	if scopeID == uuid.Nil {
		scopeID = uuid.MustParse("00000000-0000-0000-0000-000000000000")
	}
	mustExec(tx, `INSERT INTO user_roles (user_id, role_id, scope_type, scope_id, valid_from)
		VALUES (?, ?, 'TENANT', ?, CURRENT_DATE)
		ON CONFLICT (user_id, role_id, scope_type, scope_id) DO NOTHING`, userID, roleID, scopeID)
}
