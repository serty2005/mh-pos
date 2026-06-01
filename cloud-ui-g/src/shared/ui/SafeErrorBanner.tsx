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
    <div className="rounded-xl border border-rose-200 bg-rose-50 p-4 text-sm text-rose-900" role="alert">
      <p className="font-medium">{message}</p>
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
        <p className="mt-2 text-xs text-rose-700">correlation_id: {error.correlationId}</p>
      ) : null}
    </div>
  );
}
