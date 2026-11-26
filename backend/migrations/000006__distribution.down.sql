DELETE FROM role_permissions WHERE permission_id IN (
    SELECT id FROM permissions WHERE code IN (
        'link.manage','order.approve','inventory.view','inventory.manage'
    )
);
DELETE FROM permissions WHERE code IN (
    'link.manage','order.approve','inventory.view','inventory.manage'
);

DROP TABLE IF EXISTS dist_secondary_sales;
DROP TABLE IF EXISTS dist_inventory_movements;
DROP TABLE IF EXISTS dist_inventory;
DROP TABLE IF EXISTS dist_order_items;
DROP TABLE IF EXISTS dist_orders;
DROP TABLE IF EXISTS dist_distributors;
DROP TABLE IF EXISTS dist_company_distributor_links;

DROP FUNCTION IF EXISTS pharmaos_can_view_distributor_data(uuid);
DROP FUNCTION IF EXISTS pharmaos_can_write_distribution_row(uuid, uuid);
DROP FUNCTION IF EXISTS pharmaos_can_view_distribution_row(uuid, uuid);
DROP FUNCTION IF EXISTS pharmaos_active_distributor_link(uuid, uuid);
