import { z } from 'zod';

export const edgeNodeIdentitySchema = z.object({
  id: z.string(),
  node_device_id: z.string(),
  restaurant_id: z.string(),
  status: z.literal('paired'),
  paired_at: z.string(),
});

export const pairingStatusSchema = z.object({
  paired: z.boolean(),
  identity: edgeNodeIdentitySchema.optional(),
  node_device_id: z.string().optional(),
  restaurant_id: z.string().optional(),
});

export const authSessionSchema = z.object({
  id: z.string(),
  restaurant_id: z.string(),
  node_device_id: z.string(),
  client_device_id: z.string(),
  employee_id: z.string(),
  status: z.enum(['active', 'revoked']),
  started_at: z.string(),
  last_seen_at: z.string(),
  revoked_at: z.string().optional(),
});

export const actorContextSchema = z.object({
  employee_id: z.string(),
  restaurant_id: z.string(),
  role_id: z.string(),
  name: z.string(),
  permissions: z.array(z.string()),
});

export const pinLoginResultSchema = z.object({
  session: authSessionSchema,
  actor: actorContextSchema,
  permissions: z.array(z.string()),
});

export const hallSchema = z.object({
  id: z.string(),
  restaurant_id: z.string(),
  name: z.string(),
  active: z.boolean(),
});

export const tableSchema = z.object({
  id: z.string(),
  restaurant_id: z.string(),
  hall_id: z.string(),
  name: z.string(),
  seats: z.number(),
  active: z.boolean(),
});

export type PairingStatus = z.infer<typeof pairingStatusSchema>;
export type AuthSession = z.infer<typeof authSessionSchema>;
export type ActorContext = z.infer<typeof actorContextSchema>;
export type PinLoginResult = z.infer<typeof pinLoginResultSchema>;
export type Hall = z.infer<typeof hallSchema>;
export type RestaurantTable = z.infer<typeof tableSchema>;
