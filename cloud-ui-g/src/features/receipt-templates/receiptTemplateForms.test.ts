import { describe, expect, it } from 'vitest';
import {
  buildCreateTemplatePayload,
  buildUpdateTemplatePayload,
  toTemplateFormValues,
  validateTemplateForm,
  defaultTemplateFormValues,
} from './receiptTemplateForms';
import type { ReceiptTemplate } from '../../shared/api/schemas';

const minTemplate: ReceiptTemplate = {
  id: 'tpl-1',
  org_id: 'org-1',
  restaurant_id: '',
  document_type: 'precheck',
  name: 'Default Precheck',
  description: '',
  content: '{a:center}{{.restaurant_name}}',
  level: 0,
  cpl: 48,
  printer_class: 'thermal_80',
  is_default: true,
  version: 1,
  is_active: true,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
};

describe('receiptTemplateForms', () => {
  it('toTemplateFormValues maps backend template to form values', () => {
    expect(toTemplateFormValues(minTemplate)).toEqual({
      name: 'Default Precheck',
      description: '',
      document_type: 'precheck',
      cpl: 48,
      printer_class: 'thermal_80',
      content: '{a:center}{{.restaurant_name}}',
      restaurant_id: '',
    });
  });

  it('buildCreateTemplatePayload trims name and description', () => {
    const result = buildCreateTemplatePayload({
      ...defaultTemplateFormValues,
      name: '  Ticket Template  ',
      description: '  For events  ',
      document_type: 'ticket',
      cpl: 32,
      content: '{a:center}TICKET',
      printer_class: 'thermal_58',
      restaurant_id: '',
    });
    expect(result.name).toBe('Ticket Template');
    expect(result.description).toBe('For events');
    expect(result.document_type).toBe('ticket');
    expect(result.cpl).toBe(32);
    expect(result.restaurant_id).toBeUndefined();
  });

  it('buildCreateTemplatePayload sets restaurant_id when provided', () => {
    const result = buildCreateTemplatePayload({
      ...defaultTemplateFormValues,
      name: 'Restaurant-specific',
      content: '{a:center}Hello',
      restaurant_id: 'rest-abc',
    });
    expect(result.restaurant_id).toBe('rest-abc');
  });

  it('buildUpdateTemplatePayload produces same shape as create', () => {
    const values = {
      ...defaultTemplateFormValues,
      name: 'Updated',
      content: '{cut}',
    };
    expect(buildUpdateTemplatePayload(values)).toEqual(buildCreateTemplatePayload(values));
  });

  it('validateTemplateForm returns errors for empty required fields', () => {
    expect(validateTemplateForm({ ...defaultTemplateFormValues, name: '', content: '' }))
      .toContain('name');
    expect(validateTemplateForm({ ...defaultTemplateFormValues, name: '', content: '' }))
      .toContain('content');
  });

  it('validateTemplateForm returns no errors for valid form', () => {
    expect(validateTemplateForm({
      ...defaultTemplateFormValues,
      name: 'Good template',
      content: '{a:center}Hello',
    })).toEqual([]);
  });

  it('validateTemplateForm rejects CPL not in allowed list', () => {
    expect(validateTemplateForm({
      ...defaultTemplateFormValues,
      name: 'Good',
      content: '{a:center}x',
      cpl: 99,
    })).toContain('cpl');
  });
});
