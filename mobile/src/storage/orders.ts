import { randomUUID } from 'expo-crypto';

import type { CreateOrderItemInput } from '../types';
import { getUser } from './secure';
import { getDatabase } from './database';
import { enqueueOperation } from './outbox';

export type CreateOrderInput = {
  distributorTenantId: string;
  productId?: string;
  quantity?: number;
  unitPrice?: number;
  items?: CreateOrderItemInput[];
  orderNumber?: string;
  orderDate?: string;
  remarks?: string;
};

export async function createDraftOrder(input: CreateOrderInput): Promise<string> {
  const user = await getUser();
  if (!user) {
    throw new Error('No active session');
  }
  const rows: CreateOrderItemInput[] = input.items?.length
    ? input.items
    : [
        {
          product_id: input.productId ?? '',
          quantity: input.quantity ?? 0,
          ...(input.unitPrice ? { unit_price: input.unitPrice } : {}),
        },
      ];
  if (rows.some((item) => !item.product_id || item.quantity < 1)) {
    throw new Error('Each order item needs a product and a quantity greater than zero');
  }
  const database = await getDatabase();
  const orderId = randomUUID();
  const now = new Date().toISOString();
  const totalAmount = rows.reduce((sum, item) => sum + item.quantity * (item.unit_price ?? 0), 0);
  const localRecord = {
    id: orderId,
    distributor_tenant_id: input.distributorTenantId,
    order_number: input.orderNumber || `LOCAL-${orderId.slice(0, 8)}`,
    status: 'DRAFT',
    total_amount: totalAmount,
    created_at: now,
  };
  const payload = {
    id: orderId,
    distributor_tenant_id: input.distributorTenantId,
    order_number: input.orderNumber || `LOCAL-${orderId.slice(0, 8)}`,
    order_date: input.orderDate || now.slice(0, 10),
    remarks: input.remarks || '',
    items: rows.map((item) => ({
      id: randomUUID(),
      product_id: item.product_id,
      quantity: item.quantity,
      ...(item.unit_price ? { unit_price: item.unit_price } : {}),
    })),
  };
  await database.withTransactionAsync(async () => {
    await database.runAsync(
      `INSERT INTO read_models (tenant_id, entity, entity_id, payload_json, updated_at, op)
       VALUES (?, 'order', ?, ?, ?, 'UPSERT')`,
      user.tenant_id,
      orderId,
      JSON.stringify(localRecord),
      now,
    );
    await enqueueOperation(database, {
      id: randomUUID(),
      tenantId: user.tenant_id,
      userId: user.id,
      entity: 'order',
      entityId: orderId,
      operation: 'order.create',
      payload,
      idempotencyKey: randomUUID(),
    });
  });
  return orderId;
}
