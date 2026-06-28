import { describe, expect, it } from 'vitest';
import {
  buildCreatePrinterPayload,
  buildUpdatePrinterPayload,
  toPrinterFormValues,
  validatePrinterForm,
  defaultPrinterFormValues,
} from './printerForms';
import type { Printer } from '../../shared/api/schemas';

const minTcpPrinter: Printer = {
  id: 'prt-1',
  org_id: 'org-1',
  restaurant_id: 'rest-1',
  name: 'Kitchen TCP',
  type: 'tcp',
  address: '10.25.1.201',
  port: 9100,
  document_types: ['precheck', 'check_nonfiscal', 'ticket'],
  codepage: 'cp437',
  paper_cut_type: 'partial',
  cpl: 48,
  is_active: true,
  version: 1,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
};

const minUsbPrinter: Printer = {
  id: 'prt-2',
  org_id: 'org-1',
  restaurant_id: 'rest-1',
  name: 'Bar USB',
  type: 'usb',
  address: '',
  port: null,
  document_types: ['cash_in_out'],
  codepage: 'cp866',
  paper_cut_type: 'full',
  cpl: 42,
  is_active: true,
  version: 1,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
};

describe('printerForms', () => {
  describe('toPrinterFormValues', () => {
    it('maps TCP printer backend response to form values', () => {
      expect(toPrinterFormValues(minTcpPrinter)).toEqual({
        name: 'Kitchen TCP',
        type: 'tcp',
        address: '10.25.1.201',
        port: '9100',
        document_types: ['precheck', 'check_nonfiscal', 'ticket'],
        codepage: 'cp437',
        cpl: 48,
        paper_cut_type: 'partial',
        restaurant_id: 'rest-1',
      });
    });

    it('maps USB printer with null port to empty port string', () => {
      const form = toPrinterFormValues(minUsbPrinter);
      expect(form.type).toBe('usb');
      expect(form.port).toBe('');
      expect(form.address).toBe('');
      expect(form.codepage).toBe('cp866');
    });
  });

  describe('buildCreatePrinterPayload', () => {
    it('trims name and sets TCP address/port', () => {
      const result = buildCreatePrinterPayload({
        ...defaultPrinterFormValues,
        restaurant_id: 'rest-1',
        name: '  Bar Printer  ',
        type: 'tcp',
        address: '  192.168.1.10  ',
        port: '9100',
        document_types: ['precheck'],
        cpl: 48,
      });
      expect(result.name).toBe('Bar Printer');
      expect(result.address).toBe('192.168.1.10');
      expect(result.port).toBe(9100);
      expect(result.restaurant_id).toBe('rest-1');
    });

    it('omits address and port for USB printer', () => {
      const result = buildCreatePrinterPayload({
        ...defaultPrinterFormValues,
        restaurant_id: 'rest-1',
        name: 'USB Printer',
        type: 'usb',
        address: '/dev/usb/lp0',
        port: '9100',
        document_types: ['ticket'],
        cpl: 42,
      });
      expect(result.address).toBeUndefined();
      expect(result.port).toBeUndefined();
      expect(result.type).toBe('usb');
    });
  });

  describe('buildUpdatePrinterPayload', () => {
    it('produces consistent shape with name and document_types', () => {
      const values = {
        ...defaultPrinterFormValues,
        restaurant_id: 'rest-1',
        name: 'Updated Printer',
        type: 'tcp' as const,
        address: '10.0.0.1',
        port: '9100',
        document_types: ['precheck', 'kitchen_service'] as ['precheck', 'kitchen_service'],
        cpl: 48,
      };
      const result = buildUpdatePrinterPayload(values);
      expect(result.name).toBe('Updated Printer');
      expect(result.document_types).toEqual(['precheck', 'kitchen_service']);
      expect(result.port).toBe(9100);
    });

    it('sets address to empty string and port to null for USB on update', () => {
      const result = buildUpdatePrinterPayload({
        ...defaultPrinterFormValues,
        restaurant_id: 'rest-1',
        name: 'USB Updated',
        type: 'usb',
        address: 'should_be_cleared',
        port: '9100',
        document_types: ['ticket'],
        cpl: 42,
      });
      expect(result.address).toBe('');
      expect(result.port).toBeNull();
    });
  });

  describe('validatePrinterForm', () => {
    it('returns error for empty name', () => {
      expect(
        validatePrinterForm({ ...defaultPrinterFormValues, name: '', document_types: ['precheck'], address: '10.0.0.1' }),
      ).toContain('name');
    });

    it('returns error when document_types is empty', () => {
      expect(
        validatePrinterForm({ ...defaultPrinterFormValues, name: 'Printer', document_types: [], address: '10.0.0.1' }),
      ).toContain('document_types');
    });

    it('returns error for invalid CPL', () => {
      expect(
        validatePrinterForm({ ...defaultPrinterFormValues, name: 'P', document_types: ['precheck'], address: '10.0.0.1', cpl: 99 }),
      ).toContain('cpl');
    });

    it('returns error for TCP printer without address', () => {
      expect(
        validatePrinterForm({ ...defaultPrinterFormValues, name: 'P', type: 'tcp', document_types: ['precheck'], address: '' }),
      ).toContain('address');
    });

    it('does not require address for USB printer', () => {
      const errors = validatePrinterForm({
        ...defaultPrinterFormValues,
        name: 'USB',
        type: 'usb',
        document_types: ['precheck'],
        address: '',
        cpl: 48,
      });
      expect(errors).not.toContain('address');
    });

    it('returns no errors for valid TCP printer', () => {
      expect(
        validatePrinterForm({
          ...defaultPrinterFormValues,
          name: 'Good Printer',
          type: 'tcp',
          address: '10.25.1.201',
          document_types: ['precheck'],
          cpl: 48,
        }),
      ).toEqual([]);
    });
  });
});
