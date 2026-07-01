import type { Hall, RestaurantTable } from '../../shared/api/schemas';
import type { LifecycleStatus } from '../catalog/catalogForms';

export type HallFormValues = {
  name: string;
};

export type HallUpdateFormValues = HallFormValues & {
  status: LifecycleStatus;
};

export type TableFormValues = {
  hall_id: string;
  section_id: string;
  name: string;
  seats: number;
  status: LifecycleStatus;
};

export const defaultHallValues: HallFormValues = {
  name: '',
};

export const defaultHallUpdateValues: HallUpdateFormValues = {
  name: '',
  status: 'published',
};

export const defaultTableValues: TableFormValues = {
  hall_id: '',
  section_id: '',
  name: '',
  seats: 2,
  status: 'published',
};

function normalizeSeats(seats: number) {
  return Math.max(1, Math.floor(Number.isFinite(seats) ? seats : 1));
}

export function buildCreateHallPayload(values: HallFormValues) {
  return {
    name: values.name.trim(),
  };
}

export function buildUpdateHallPayload(values: HallUpdateFormValues) {
  return {
    name: values.name.trim(),
    status: values.status,
  };
}

export function buildCreateTablePayload(values: TableFormValues) {
  return {
    hall_id: values.hall_id,
    section_id: values.section_id,
    name: values.name.trim(),
    seats: normalizeSeats(values.seats),
  };
}

export function buildUpdateTablePayload(values: TableFormValues) {
  return {
    hall_id: values.hall_id,
    section_id: values.section_id,
    name: values.name.trim(),
    seats: normalizeSeats(values.seats),
    status: values.status,
  };
}

export function toHallValues(hall: Hall): HallUpdateFormValues {
  return {
    name: hall.name,
    status: hall.status,
  };
}

export function toTableValues(table: RestaurantTable): TableFormValues {
  return {
    hall_id: table.hall_id,
    section_id: table.section_id,
    name: table.name,
    seats: table.seats,
    status: table.status,
  };
}
