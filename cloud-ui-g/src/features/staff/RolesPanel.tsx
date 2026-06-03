import { useEffect, useState } from 'react';
import type { Role } from '../../shared/api/schemas';
import { useI18n } from '../../shared/i18n/I18nProvider';
import SafeErrorBanner from '../../shared/ui/SafeErrorBanner';
import PermissionMatrix from './PermissionMatrix';
import { roleProfileById, roleProfiles, type RoleProfileId } from './roleProfiles';
import {
  defaultRoleValues,
  parsePermissionIds,
  roleToFormValues,
  type RoleFormValues,
} from './staffForms';

type Props = {
  roles: Role[];
  loading: boolean;
  error: unknown;
  onCreate: (values: RoleFormValues) => Promise<void>;
  onUpdate: (id: string, values: RoleFormValues) => Promise<void>;
  onArchive: (id: string) => Promise<void>;
};

const emptyProfile = '';

// RolesPanel управляет только role-level permissions, потому что employee override endpoint не подтвержден.
export default function RolesPanel({ roles, loading, error, onCreate, onUpdate, onArchive }: Props) {
  const { t } = useI18n();
  const [createValues, setCreateValues] = useState<RoleFormValues>(defaultRoleValues);
  const [profileId, setProfileId] = useState<RoleProfileId | typeof emptyProfile>(emptyProfile);
  const [editing, setEditing] = useState<Role | null>(null);
  const [editValues, setEditValues] = useState<RoleFormValues>(defaultRoleValues);

  useEffect(() => {
    if (editing) setEditValues(roleToFormValues(editing));
  }, [editing]);

  const applyProfile = (id: RoleProfileId | typeof emptyProfile) => {
    setProfileId(id);
    const profile = id ? roleProfileById.get(id) : null;
    if (!profile) return;
    setCreateValues({
      name: t(profile.labelKey),
      active: true,
      permission_ids: profile.permissionIds,
    });
  };

  const renderForm = (
    values: RoleFormValues,
    setValues: (next: RoleFormValues) => void,
    onSubmit: () => Promise<void>,
    actionLabel: string,
    includeProfile = false,
  ) => (
    <form className="space-y-3 rounded-xl border border-slate-200 bg-slate-50 p-4" onSubmit={(event) => { event.preventDefault(); void onSubmit(); }}>
      {includeProfile ? (
        <div>
          <label className="mb-1 block text-sm text-slate-700">{t('staff.roles.fields.profile')}</label>
          <select value={profileId} onChange={(event) => applyProfile(event.target.value as RoleProfileId | typeof emptyProfile)} className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm" disabled={loading}>
            <option value="">{t('staff.roles.fields.customProfile')}</option>
            {roleProfiles.map((profile) => <option key={profile.id} value={profile.id}>{t(profile.labelKey)}</option>)}
          </select>
        </div>
      ) : null}
      <div className="grid gap-3 md:grid-cols-2">
        <div>
          <label className="mb-1 block text-sm text-slate-700">{t('staff.roles.fields.name')}</label>
          <input value={values.name} onChange={(event) => setValues({ ...values, name: event.target.value })} className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm" disabled={loading} />
        </div>
        <label className="flex items-center gap-2 rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm text-slate-700">
          <input type="checkbox" checked={values.active} onChange={(event) => setValues({ ...values, active: event.target.checked })} disabled={loading} />
          {t('staff.roles.fields.active')}
        </label>
      </div>
      <PermissionMatrix selectedIds={values.permission_ids} onChange={(permission_ids) => setValues({ ...values, permission_ids })} />
      <button type="submit" disabled={loading || !values.name.trim()} className="rounded-lg bg-slate-900 px-3 py-2 text-sm font-medium text-white disabled:opacity-50">{actionLabel}</button>
    </form>
  );

  return (
    <section className="space-y-4 rounded-2xl border border-slate-200 bg-white p-6">
      <div>
        <h3 className="text-base font-semibold text-slate-900">{t('staff.roles.title')}</h3>
        <p className="mt-1 text-sm text-slate-600">{t('staff.roles.description')}</p>
      </div>
      {renderForm(createValues, setCreateValues, async () => {
        await onCreate(createValues);
        setCreateValues(defaultRoleValues);
        setProfileId(emptyProfile);
      }, t('staff.roles.actions.create'), true)}
      {error ? <SafeErrorBanner error={error} /> : null}
      {roles.length === 0 ? <p className="text-sm text-slate-600">{t('staff.roles.empty')}</p> : null}
      {roles.map((role) => {
        const permissionCount = parsePermissionIds(role.permissions_json).length;
        return (
          <article key={role.id} className="rounded-xl border border-slate-200 p-4">
            <div className="flex flex-wrap justify-between gap-3">
              <div>
                <p className="text-sm font-medium text-slate-900">{role.name}</p>
                <p className="text-xs text-slate-600">{t('staff.roles.permissionCount')}: {permissionCount} · {role.active ? t('staff.roles.status.active') : t('staff.roles.status.archived')}</p>
              </div>
              <div className="flex flex-wrap gap-2">
                <button type="button" onClick={() => setEditing(role)} className="rounded-lg border border-slate-300 px-2 py-1 text-xs text-slate-700" disabled={loading}>{t('catalog.shared.edit')}</button>
                <button type="button" onClick={() => { if (window.confirm(t('catalog.shared.archiveConfirm'))) void onArchive(role.id); }} className="rounded-lg border border-rose-300 px-2 py-1 text-xs text-rose-700" disabled={loading || !role.active}>{t('catalog.shared.archive')}</button>
              </div>
            </div>
            {editing?.id === role.id ? (
              <div className="mt-3">
                {renderForm(editValues, setEditValues, async () => {
                  await onUpdate(role.id, editValues);
                  setEditing(null);
                }, t('catalog.shared.save'))}
                <button type="button" onClick={() => setEditing(null)} className="mt-2 rounded-lg border border-slate-300 px-3 py-2 text-sm text-slate-700">{t('catalog.shared.cancel')}</button>
              </div>
            ) : null}
          </article>
        );
      })}
    </section>
  );
}
