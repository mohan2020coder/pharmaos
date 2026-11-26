import { randomUUID } from 'expo-crypto';

import { getUser } from './secure';
import { getDatabase } from './database';
import { enqueueOperation } from './outbox';

export type CreateVisitInput = {
  employeeId: string;
  entityType: 'DOCTOR' | 'CHEMIST' | 'STOCKIST';
  entityId: string;
  remarks: string;
  latitude?: number;
  longitude?: number;
};

export async function createDraftVisit(input: CreateVisitInput): Promise<string> {
  const user = await getUser();
  if (!user) {
    throw new Error('No active session');
  }
  const database = await getDatabase();
  const visitId = randomUUID();
  const visitedAt = new Date().toISOString();
  const localRecord = {
    id: visitId,
    employee_id: input.employeeId,
    entity_type: input.entityType,
    entity_id: input.entityId,
    remarks: input.remarks,
    visited_at: visitedAt,
    status: 'DRAFT',
    check_in_lat: input.latitude,
    check_in_lng: input.longitude,
  };
  const payload = {
    id: visitId,
    employee_id: input.employeeId,
    entity_type: input.entityType,
    entity_id: input.entityId,
    visited_at: visitedAt,
    remarks: input.remarks,
    check_in_lat: input.latitude,
    check_in_lng: input.longitude,
    client_reference_id: visitId,
  };

  await database.withTransactionAsync(async () => {
    await database.runAsync(
      `INSERT INTO read_models (tenant_id, entity, entity_id, payload_json, updated_at, op)
       VALUES (?, 'visit', ?, ?, ?, 'UPSERT')`,
      user.tenant_id,
      visitId,
      JSON.stringify(localRecord),
      visitedAt,
    );
    await enqueueOperation(database, {
      id: randomUUID(),
      tenantId: user.tenant_id,
      userId: user.id,
      entity: 'visit',
      entityId: visitId,
      operation: 'visit.create',
      payload,
      idempotencyKey: randomUUID(),
    });
  });
  return visitId;
}
