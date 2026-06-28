import type { ReceiptTemplate, ReceiptTemplateDocumentType } from '../../shared/api/schemas';

export type ReceiptTemplateFormValues = {
  name: string;
  description: string;
  document_type: ReceiptTemplateDocumentType;
  cpl: number;
  printer_class: string;
  content: string;
  restaurant_id: string;
};

export const defaultTemplateFormValues: ReceiptTemplateFormValues = {
  name: '',
  description: '',
  document_type: 'precheck',
  cpl: 48,
  printer_class: 'thermal_80',
  content: '',
  restaurant_id: '',
};

export const DOCUMENT_TYPES: ReceiptTemplateDocumentType[] = [
  'precheck',
  'check_nonfiscal',
  'ticket',
  'kitchen_service',
  'cash_in_out',
  'acceptance',
];

export const CPL_OPTIONS = [32, 40, 42, 48, 56, 80];

export function toTemplateFormValues(t: ReceiptTemplate): ReceiptTemplateFormValues {
  return {
    name: t.name,
    description: t.description ?? '',
    document_type: t.document_type,
    cpl: t.cpl,
    printer_class: t.printer_class,
    content: t.content,
    restaurant_id: t.restaurant_id ?? '',
  };
}

export function buildCreateTemplatePayload(values: ReceiptTemplateFormValues) {
  return {
    name: values.name.trim(),
    description: values.description.trim(),
    document_type: values.document_type,
    cpl: values.cpl,
    printer_class: values.printer_class.trim(),
    content: values.content,
    restaurant_id: values.restaurant_id || undefined,
  };
}

export function buildUpdateTemplatePayload(values: ReceiptTemplateFormValues) {
  return buildCreateTemplatePayload(values);
}

export function validateTemplateForm(values: ReceiptTemplateFormValues): string[] {
  const errors: string[] = [];
  if (!values.name.trim()) errors.push('name');
  if (!values.content.trim()) errors.push('content');
  if (!CPL_OPTIONS.includes(values.cpl)) errors.push('cpl');
  return errors;
}
