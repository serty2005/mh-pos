import fs from 'node:fs';

export const bootstrapEnvName = 'POS_E2E_BOOTSTRAP_JSON';

export function loadBootstrapJson() {
  const source = process.env[bootstrapEnvName]?.trim() ?? '';
  if (!source) return '';
  if (source.startsWith('{') || source.startsWith('[')) return source;
  if (fs.existsSync(source)) return fs.readFileSync(source, 'utf-8');
  if (source.endsWith('.json') || source.startsWith('/') || source.startsWith('.')) return '';
  return source;
}

export function bootstrapRequiredMessage() {
  return 'Run stack bootstrap and set POS_E2E_BOOTSTRAP_JSON to JSON content or /workspace/myhoreca-pos/.e2e/bootstrap.json';
}
