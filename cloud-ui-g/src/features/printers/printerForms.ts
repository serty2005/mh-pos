import type { Printer, PrinterDocumentType } from '../../shared/api/schemas';

export type PrinterFormValues = {
  name: string;
  type: 'tcp' | 'usb';
  address: string;
  port: string;
  document_types: PrinterDocumentType[];
  codepage: '' | 'cp437' | 'cp866';
  cpl: number;
  paper_cut_type: 'partial' | 'full';
  restaurant_id: string;
};

export const defaultPrinterFormValues: PrinterFormValues = {
  name: '',
  type: 'tcp',
  address: '',
  port: '9100',
  document_types: [],
  codepage: '',
  cpl: 48,
  paper_cut_type: 'partial',
  restaurant_id: '',
};

export const PRINTER_DOCUMENT_TYPES: PrinterDocumentType[] = [
  'precheck',
  'check_nonfiscal',
  'ticket',
  'kitchen_service',
  'cash_in_out',
  'acceptance',
];

export const PRINTER_CPL_OPTIONS = [32, 42, 48, 56, 80];

export function toPrinterFormValues(p: Printer): PrinterFormValues {
  return {
    name: p.name,
    type: p.type,
    address: p.address ?? '',
    port: p.port != null ? String(p.port) : '',
    document_types: p.document_types,
    codepage: p.codepage,
    cpl: p.cpl,
    paper_cut_type: p.paper_cut_type,
    restaurant_id: p.restaurant_id,
  };
}

export function buildCreatePrinterPayload(values: PrinterFormValues) {
  const port = values.type === 'tcp' && values.port.trim() ? parseInt(values.port.trim(), 10) : undefined;
  return {
    restaurant_id: values.restaurant_id,
    name: values.name.trim(),
    type: values.type,
    address: values.type === 'tcp' ? values.address.trim() : undefined,
    port: values.type === 'tcp' ? port : undefined,
    document_types: values.document_types,
    codepage: values.codepage,
    cpl: values.cpl,
    paper_cut_type: values.paper_cut_type,
  };
}

export function buildUpdatePrinterPayload(values: PrinterFormValues) {
  const port = values.type === 'tcp' && values.port.trim() ? parseInt(values.port.trim(), 10) : null;
  return {
    name: values.name.trim(),
    type: values.type,
    address: values.type === 'tcp' ? values.address.trim() : '',
    port: values.type === 'tcp' ? port : null,
    document_types: values.document_types,
    codepage: values.codepage,
    cpl: values.cpl,
    paper_cut_type: values.paper_cut_type,
  };
}

export function validatePrinterForm(values: PrinterFormValues): string[] {
  const errors: string[] = [];
  if (!values.name.trim()) errors.push('name');
  if (values.document_types.length === 0) errors.push('document_types');
  if (!PRINTER_CPL_OPTIONS.includes(values.cpl)) errors.push('cpl');
  if (values.type === 'tcp' && !values.address.trim()) errors.push('address');
  return errors;
}
