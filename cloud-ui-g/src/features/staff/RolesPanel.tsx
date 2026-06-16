import { useMemo, useState } from 'react';
import { Archive, Edit2, Save, ShieldCheck, X } from 'lucide-react';
import type { Employee, Role } from '../../shared/api/schemas';
import { useI18n } from '../../shared/i18n/I18nProvider';
import SafeErrorBanner from '../../shared/ui/SafeErrorBanner';
import { permissionCatalog } from './permissionCatalog';
import { roleProfileById, roleProfiles, type RoleProfileId } from './roleProfiles';
import {
  defaultRoleValues,
  parsePermissionIds,
  roleToFormValues,
  type RoleFormValues,
} from './staffForms';

type Props = {
  roles: Role[];
  employees: Employee[];
  loading: boolean;
  error: unknown;
  onCreate: (values: RoleFormValues) => Promise<void>;
  onUpdate: (id: string, values: RoleFormValues) => Promise<void>;
  onArchive: (id: string) => Promise<void>;
};

const emptyProfile = '';

function cellClass(checked: boolean, readonly = false) {
  if (checked) return readonly ? 'border-emerald-700 bg-emerald-700' : 'border-lime-400 bg-lime-400 hover:bg-lime-300';
  return readonly ? 'border-slate-200 bg-white' : 'border-slate-200 bg-slate-50 hover:bg-slate-100';
}

// RolesPanel редактирует только role-level permissions; строки сотрудников пересчитываются от текущей роли без overrides.
export default function RolesPanel({ roles, employees, loading, error, onCreate, onUpdate, onArchive }: Props) {
  const { t } = useI18n();
  const [createValues, setCreateValues] = useState<RoleFormValues>(defaultRoleValues);
  const [profileId, setProfileId] = useState<RoleProfileId | typeof emptyProfile>(emptyProfile);
  const [editingRoleId, setEditingRoleId] = useState('');
  const [nameDraft, setNameDraft] = useState('');
  const [permissionQuery, setPermissionQuery] = useState('');
  const [selectedPermissionId, setSelectedPermissionId] = useState('');

  const roleById = useMemo(() => new Map(roles.map((role) => [role.id, role])), [roles]);
  const rolePermissionIds = useMemo(
    () => new Map(roles.map((role) => [role.id, parsePermissionIds(role.permissions_json)])),
    [roles],
  );
  const filteredPermissions = useMemo(() => {
    const query = permissionQuery.trim().toLocaleLowerCase();
    if (!query) return permissionCatalog;
    return permissionCatalog.filter((permission) => {
      const haystack = `${permission.code} ${permission.id} ${t(permission.labelKey)}`.toLocaleLowerCase();
      return haystack.includes(query);
    });
  }, [permissionQuery, t]);

  const applyProfile = (id: RoleProfileId | typeof emptyProfile) => {
    setProfileId(id);
    const profile = id ? roleProfileById.get(id) : null;
    if (!profile) {
      setCreateValues(defaultRoleValues);
      return;
    }
    setCreateValues({
      name: t(profile.labelKey),
      active: true,
      permission_ids: profile.permissionIds,
    });
  };

  const createRole = async () => {
    await onCreate(createValues);
    setCreateValues(defaultRoleValues);
    setProfileId(emptyProfile);
  };

  const updateRole = (role: Role, nextIds: string[]) => {
    return onUpdate(role.id, {
      name: role.name,
      active: role.active,
      permission_ids: nextIds,
    });
  };

  const togglePermission = (role: Role, permissionId: string) => {
    const next = new Set<string>(rolePermissionIds.get(role.id) ?? []);
    if (next.has(permissionId)) {
      next.delete(permissionId);
    } else {
      next.add(permissionId);
    }
    void updateRole(role, Array.from(next).sort());
  };

  const saveRoleName = async (role: Role) => {
    await onUpdate(role.id, {
      ...roleToFormValues(role),
      name: nameDraft.trim(),
    });
    setEditingRoleId('');
  };

  return (
    <section className="space-y-4">
      {error ? <SafeErrorBanner error={error} /> : null}

      <form
        className="grid gap-2 rounded-2xl border border-slate-200 bg-white p-3 shadow-[0_18px_44px_-34px_rgba(15,23,42,0.32)] md:grid-cols-[minmax(0,14rem)_minmax(0,1fr)_auto]"
        onSubmit={(event) => {
          event.preventDefault();
          void createRole();
        }}
      >
        <label className="block">
          <span className="mb-1 block text-xs font-semibold text-slate-600">{t('staff.roles.fields.profile')}</span>
          <select
            value={profileId}
            onChange={(event) => applyProfile(event.target.value as RoleProfileId | typeof emptyProfile)}
            className="w-full rounded-xl border border-slate-200 bg-slate-50 px-3 py-2.5 text-sm font-medium text-slate-800 outline-none transition-colors focus:border-blue-500 focus:bg-white"
            disabled={loading}
          >
            <option value="">{t('staff.roles.fields.customProfile')}</option>
            {roleProfiles.map((profile) => <option key={profile.id} value={profile.id}>{t(profile.labelKey)}</option>)}
          </select>
        </label>
        <label className="block">
          <span className="mb-1 block text-xs font-semibold text-slate-600">{t('staff.roles.fields.name')}</span>
          <input
            value={createValues.name}
            onChange={(event) => setCreateValues({ ...createValues, name: event.target.value })}
            className="w-full rounded-xl border border-slate-200 bg-slate-50 px-3 py-2.5 text-sm outline-none transition-colors focus:border-blue-500 focus:bg-white"
            disabled={loading}
          />
        </label>
        <button
          type="submit"
          disabled={loading || !createValues.name.trim()}
          className="self-end rounded-xl bg-blue-600 px-4 py-2.5 text-sm font-semibold text-white transition-colors hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-50"
        >
          {t('staff.roles.actions.create')}
        </button>
      </form>

      <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_18rem]">
        <div className="min-w-0 overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-[0_18px_44px_-34px_rgba(15,23,42,0.32)]">
          <div className="overflow-auto">
            <table className="min-w-[1320px] border-collapse text-xs">
              <thead>
                <tr className="bg-[#f5f4df] text-slate-900">
                  <th className="sticky left-0 z-20 w-64 border border-slate-300 bg-[#f5f4df] px-2 py-2 text-left font-semibold">
                    {t('staff.roles.matrix.name')}
                  </th>
                  {permissionCatalog.map((permission) => {
                    const isSelected = selectedPermissionId === permission.id;
                    return (
                    <th
                      key={permission.id}
                      className={[
                        'w-8 border px-1 py-2 text-center font-mono text-[10px] font-semibold',
                        isSelected ? 'border-blue-300 bg-blue-100 text-blue-800' : 'border-slate-300 text-slate-800',
                      ].join(' ')}
                      title={t(permission.labelKey)}
                    >
                      <button
                        type="button"
                        onClick={() => setSelectedPermissionId(isSelected ? '' : permission.id)}
                        className="h-full w-full"
                      >
                        {permission.code}
                      </button>
                    </th>
                    );
                  })}
                </tr>
              </thead>
              <tbody>
                {roles.length === 0 ? (
                  <tr>
                    <td className="border border-slate-200 px-3 py-4 text-sm text-slate-500" colSpan={permissionCatalog.length + 1}>
                      {t('staff.roles.empty')}
                    </td>
                  </tr>
                ) : null}
                {roles.map((role) => {
                  const selected = new Set(rolePermissionIds.get(role.id) ?? []);
                  const isEditing = editingRoleId === role.id;
                  return (
                    <tr key={role.id} className="bg-white hover:bg-slate-50">
                      <th className="sticky left-0 z-10 border border-slate-300 bg-[#f5f4df] px-2 py-1.5 text-left font-medium">
                        <div className="flex min-w-0 items-center gap-2">
                          <ShieldCheck className="h-3.5 w-3.5 shrink-0 text-blue-600" />
                          {isEditing ? (
                            <input
                              value={nameDraft}
                              onChange={(event) => setNameDraft(event.target.value)}
                              className="min-w-0 flex-1 rounded border border-slate-300 bg-white px-2 py-1 text-xs outline-none focus:border-blue-500"
                              disabled={loading}
                            />
                          ) : (
                            <span className="min-w-0 flex-1 truncate">{role.name}</span>
                          )}
                          <span className={role.active ? 'rounded border border-emerald-100 bg-emerald-50 px-1.5 py-0.5 text-[9px] font-bold text-emerald-700' : 'rounded border border-slate-200 bg-slate-100 px-1.5 py-0.5 text-[9px] font-bold text-slate-500'}>
                            {role.active ? t('staff.roles.status.active') : t('staff.roles.status.archived')}
                          </span>
                          {isEditing ? (
                            <>
                              <button
                                type="button"
                                onClick={() => void saveRoleName(role)}
                                disabled={loading || !nameDraft.trim()}
                                className="rounded border border-blue-200 bg-blue-50 p-1 text-blue-700 disabled:opacity-50"
                                aria-label={t('catalog.shared.save')}
                              >
                                <Save className="h-3 w-3" />
                              </button>
                              <button
                                type="button"
                                onClick={() => setEditingRoleId('')}
                                className="rounded border border-slate-200 bg-white p-1 text-slate-600"
                                aria-label={t('catalog.shared.cancel')}
                              >
                                <X className="h-3 w-3" />
                              </button>
                            </>
                          ) : (
                            <>
                              <button
                                type="button"
                                onClick={() => {
                                  setEditingRoleId(role.id);
                                  setNameDraft(role.name);
                                }}
                                className="rounded border border-slate-200 bg-white p-1 text-slate-600 hover:text-blue-700"
                                disabled={loading}
                                aria-label={t('catalog.shared.edit')}
                              >
                                <Edit2 className="h-3 w-3" />
                              </button>
                              <button
                                type="button"
                                onClick={() => { if (window.confirm(t('catalog.shared.archiveConfirm'))) void onArchive(role.id); }}
                                className="rounded border border-rose-200 bg-white p-1 text-rose-700 disabled:opacity-50"
                                disabled={loading || !role.active}
                                aria-label={t('catalog.shared.archive')}
                              >
                                <Archive className="h-3 w-3" />
                              </button>
                            </>
                          )}
                        </div>
                      </th>
                      {permissionCatalog.map((permission) => {
                        const checked = selected.has(permission.id);
                        const isSelected = selectedPermissionId === permission.id;
                        return (
                          <td key={permission.id} className={['border p-0.5 text-center', isSelected ? 'border-blue-200 bg-blue-50' : 'border-slate-200'].join(' ')}>
                            <button
                              type="button"
                              onClick={() => {
                                setSelectedPermissionId(permission.id);
                                togglePermission(role, permission.id);
                              }}
                              disabled={loading || !role.active}
                              className={['mx-auto block h-6 w-6 border transition-colors disabled:cursor-not-allowed disabled:opacity-45', cellClass(checked)].join(' ')}
                              aria-label={`${role.name}: ${t(permission.labelKey)}`}
                            />
                          </td>
                        );
                      })}
                    </tr>
                  );
                })}
                {employees.length > 0 ? (
                  <tr>
                    <td className="sticky left-0 z-10 border-x border-slate-300 bg-slate-100 px-2 py-1.5 font-mono text-[10px] font-bold uppercase text-slate-500" colSpan={permissionCatalog.length + 1}>
                      {t('staff.roles.matrix.employeeRows')}
                    </td>
                  </tr>
                ) : null}
                {employees.map((employee) => {
                  const selected = new Set(rolePermissionIds.get(employee.role_id) ?? parsePermissionIds(employee.permission_snapshot_json));
                  return (
                    <tr key={employee.id} className="bg-white hover:bg-slate-50">
                      <th className="sticky left-0 z-10 border border-slate-300 bg-[#f5f4df] px-2 py-1.5 text-left font-medium">
                        <div className="flex min-w-0 items-center gap-2">
                          <span className="h-2 w-2 shrink-0 rounded-full bg-slate-400" />
                          <span className="min-w-0 flex-1 truncate">{employee.name}</span>
                          <span className="truncate font-mono text-[10px] text-slate-500">{roleById.get(employee.role_id)?.name ?? employee.role_id}</span>
                          <span className="rounded border border-slate-200 bg-white px-1.5 py-0.5 text-[9px] font-bold text-slate-500">
                            {t(`staff.employees.status.${employee.status}`)}
                          </span>
                        </div>
                      </th>
                      {permissionCatalog.map((permission) => {
                        const isSelected = selectedPermissionId === permission.id;
                        return (
                        <td key={permission.id} className={['border p-0.5 text-center', isSelected ? 'border-blue-200 bg-blue-50' : 'border-slate-200'].join(' ')}>
                          <span
                            className={['mx-auto block h-6 w-6 border', cellClass(selected.has(permission.id), true)].join(' ')}
                            title={t(permission.labelKey)}
                          />
                        </td>
                        );
                      })}
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>

        <aside className="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-[0_18px_44px_-34px_rgba(15,23,42,0.32)]">
          <div className="border-b border-slate-200 bg-[#f5f4df] px-3 py-2 text-xs font-semibold text-slate-900">
            {t('staff.roles.matrix.dictionary')}
          </div>
          <label className="block border-b border-slate-200 bg-white p-2">
            <span className="sr-only">{t('staff.roles.matrix.searchLabel')}</span>
            <input
              type="search"
              value={permissionQuery}
              onChange={(event) => setPermissionQuery(event.target.value)}
              placeholder={t('staff.roles.matrix.searchPlaceholder')}
              className="w-full rounded-lg border border-slate-200 bg-slate-50 px-3 py-2 text-xs font-semibold text-slate-700 outline-none transition-colors placeholder:text-slate-400 focus:border-blue-500 focus:bg-white"
            />
          </label>
          <div className="max-h-[34rem] overflow-auto">
            <table className="w-full border-collapse text-xs">
              <tbody>
                {filteredPermissions.map((permission) => {
                  const isSelected = selectedPermissionId === permission.id;
                  return (
                  <tr
                    key={permission.id}
                    onClick={() => setSelectedPermissionId(isSelected ? '' : permission.id)}
                    className={[
                      'cursor-pointer border-b border-slate-100 hover:bg-slate-50',
                      isSelected ? 'bg-blue-50 text-blue-800' : '',
                    ].join(' ')}
                  >
                    <td className="w-14 px-3 py-2 font-mono font-semibold text-blue-700">{permission.code}</td>
                    <td className="px-3 py-2 text-slate-700">{t(permission.labelKey)}</td>
                  </tr>
                  );
                })}
                {filteredPermissions.length === 0 ? (
                  <tr>
                    <td className="px-3 py-4 text-slate-500" colSpan={2}>{t('staff.roles.matrix.noMatches')}</td>
                  </tr>
                ) : null}
              </tbody>
            </table>
          </div>
        </aside>
      </div>
    </section>
  );
}
