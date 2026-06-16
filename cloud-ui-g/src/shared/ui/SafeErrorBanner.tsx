import { AlertTriangle } from 'lucide-react';
import { ApiError } from '../api/errors';
import { useI18n } from '../i18n/I18nProvider';

const SENSITIVE_PATTERN = /(payload|token|secret|pin|password|credential|sql|stack)/i;
const SAFE_KEY_PATTERN = /^[a-z0-9]+(?:\.[a-z0-9_]+)+$/i;

function redactDetails(details: Record<string, string>, redactedLabel: string) {
  return Object.entries(details).map(([key, value]) => {
    const sensitive = SENSITIVE_PATTERN.test(key) || SENSITIVE_PATTERN.test(value);
    return {
      key,
      value: sensitive ? redactedLabel : value,
    };
  });
}

function safeMessageKey(error: unknown): string {
  if (error instanceof ApiError && SAFE_KEY_PATTERN.test(error.messageKey)) {
    return error.messageKey;
  }
  return 'errors.unknown';
}

export default function SafeErrorBanner({ error }: { error: unknown }) {
  const { t } = useI18n();
  if (!error) return null;

  const message = t(safeMessageKey(error));
  const details = error instanceof ApiError ? redactDetails(error.details, t('errors.detailRedacted')) : [];

  return (
    <div className="rounded-2xl border border-rose-200 bg-rose-50/90 p-4 text-sm text-rose-900 shadow-[0_18px_40px_-34px_rgba(190,18,60,0.48)]" role="alert">
      <div className="flex items-start gap-3">
        <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl border border-rose-200 bg-white text-rose-600">
          <AlertTriangle className="h-4 w-4" />
        </div>
        <div className="min-w-0">
          <p className="font-semibold">{message}</p>
          {details.length > 0 ? (
            <ul className="mt-2 space-y-1 text-xs text-rose-700">
              {details.map((item) => (
                <li key={item.key}>
                  {item.key}: {item.value}
                </li>
              ))}
            </ul>
          ) : null}
          {error instanceof ApiError && error.correlationId ? (
            <p className="mt-2 break-all font-mono text-xs text-rose-700">correlation_id: {error.correlationId}</p>
          ) : null}
        </div>
      </div>
    </div>
  );
}
