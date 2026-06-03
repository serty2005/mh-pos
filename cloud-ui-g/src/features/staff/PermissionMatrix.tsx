import { useMemo } from 'react';
import { useI18n } from '../../shared/i18n/I18nProvider';
import { permissionCatalog, permissionGroupIds } from './permissionCatalog';

type Props = {
  selectedIds: string[];
  readonly?: boolean;
  onChange?: (nextIds: string[]) => void;
};

// PermissionMatrix отображает стабильные backend permission IDs и не принимает произвольный ввод прав.
export default function PermissionMatrix({ selectedIds, readonly = false, onChange }: Props) {
  const { t } = useI18n();
  const selected = useMemo(() => new Set<string>(selectedIds), [selectedIds]);

  const toggle = (id: string) => {
    if (readonly || !onChange) return;
    const next = new Set<string>(selected);
    if (next.has(id)) {
      next.delete(id);
    } else {
      next.add(id);
    }
    onChange(Array.from(next).sort());
  };

  return (
    <div className="space-y-3">
      {permissionGroupIds.map((groupId) => {
        const permissions = permissionCatalog.filter((permission) => permission.group === groupId);
        return (
          <section key={groupId} className="rounded-xl border border-slate-200 bg-white p-3">
            <h4 className="text-sm font-semibold text-slate-900">{t(`staff.permissions.groups.${groupId}`)}</h4>
            <div className="mt-2 grid gap-2 sm:grid-cols-2">
              {permissions.map((permission) => {
                const checked = selected.has(permission.id);
                return (
                  <label key={permission.id} className="flex items-start gap-2 rounded-lg border border-slate-200 px-3 py-2 text-sm text-slate-700">
                    <input
                      type="checkbox"
                      checked={checked}
                      disabled={readonly}
                      onChange={() => toggle(permission.id)}
                      className="mt-1 h-4 w-4 rounded border-slate-300"
                    />
                    <span>
                      <span className="block text-slate-900">{t(permission.labelKey)}</span>
                      <span className="block font-mono text-xs text-slate-500">{permission.id}</span>
                    </span>
                  </label>
                );
              })}
            </div>
          </section>
        );
      })}
    </div>
  );
}
