import { useEffect, useMemo, useState } from 'react';
import type { Employee, Role } from '../../shared/api/schemas';
import { useI18n } from '../../shared/i18n/I18nProvider';
import SafeErrorBanner from '../../shared/ui/SafeErrorBanner';
import PermissionMatrix from './PermissionMatrix';
import {
  defaultEmployeeCreateValues,
  defaultEmployeeUpdateValues,
  employeeToUpdateValues,
  parsePermissionIds,
  type EmployeeCreateFormValues,
  type EmployeeUpdateFormValues,
} from './staffForms';

type Props = {
  employees: Employee[];
  roles: Role[];
  loading: boolean;
  error: unknown;
  onCreate: (values: EmployeeCreateFormValues) => Promise<void>;
  onUpdate: (id: string, values: EmployeeUpdateFormValues) => Promise<void>;
  onAssignRole: (id: string, roleId: string) => Promise<void>;
  onSuspend: (id: string) => Promise<void>;
  onActivate: (id: string) => Promise<void>;
  onArchive: (id: string) => Promise<void>;
  onRotatePin: (id: string, pin: string) => Promise<void>;
};

const statuses: EmployeeUpdateFormValues['status'][] = ['active', 'suspended', 'archived'];

// EmployeesPanel не хранит PIN в состоянии дольше текущей формы и не отображает PIN/hash из response.
export default function EmployeesPanel({ employees, roles, loading, error, onCreate, onUpdate, onAssignRole, onSuspend, onActivate, onArchive, onRotatePin }: Props) {
  const { t } = useI18n();
  const [createValues, setCreateValues] = useState<EmployeeCreateFormValues>(defaultEmployeeCreateValues);
  const [editing, setEditing] = useState<Employee | null>(null);
  const [editValues, setEditValues] = useState<EmployeeUpdateFormValues>(defaultEmployeeUpdateValues);
  const [pinEmployeeId, setPinEmployeeId] = useState('');
  const [nextPin, setNextPin] = useState('');

  const activeRoles = useMemo(() => roles.filter((role) => role.active), [roles]);
  const roleName = (id: string) => roles.find((role) => role.id === id)?.name ?? id;

  useEffect(() => {
    if (editing) setEditValues(employeeToUpdateValues(editing));
  }, [editing]);

  const roleSelect = (value: string, onChange: (roleId: string) => void) => (
    <select value={value} onChange={(event) => onChange(event.target.value)} className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm" disabled={loading}>
      <option value="">{t('staff.employees.fields.selectRole')}</option>
      {activeRoles.map((role) => <option key={role.id} value={role.id}>{role.name}</option>)}
    </select>
  );

  return (
    <section className="space-y-4 rounded-2xl border border-slate-200 bg-white p-6">
      <div>
        <h3 className="text-base font-semibold text-slate-900">{t('staff.employees.title')}</h3>
        <p className="mt-1 text-sm text-slate-600">{t('staff.employees.description')}</p>
      </div>

      <form className="space-y-3 rounded-xl border border-slate-200 bg-slate-50 p-4" onSubmit={(event) => {
        event.preventDefault();
        void onCreate(createValues).then(() => setCreateValues(defaultEmployeeCreateValues));
      }}>
        <div className="grid gap-3 md:grid-cols-3">
          <div>
            <label className="mb-1 block text-sm text-slate-700">{t('staff.employees.fields.name')}</label>
            <input value={createValues.name} onChange={(event) => setCreateValues({ ...createValues, name: event.target.value })} className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm" disabled={loading} />
          </div>
          <div>
            <label className="mb-1 block text-sm text-slate-700">{t('staff.employees.fields.role')}</label>
            {roleSelect(createValues.role_id, (role_id) => setCreateValues({ ...createValues, role_id }))}
          </div>
          <div>
            <label className="mb-1 block text-sm text-slate-700">{t('staff.employees.fields.pin')}</label>
            <input type="password" autoComplete="new-password" value={createValues.pin} onChange={(event) => setCreateValues({ ...createValues, pin: event.target.value })} className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm" disabled={loading} />
          </div>
        </div>
        <button type="submit" disabled={loading || !createValues.name.trim() || !createValues.role_id || !createValues.pin.trim()} className="rounded-lg bg-slate-900 px-3 py-2 text-sm font-medium text-white disabled:opacity-50">{t('staff.employees.actions.create')}</button>
      </form>

      {error ? <SafeErrorBanner error={error} /> : null}
      {employees.length === 0 ? <p className="text-sm text-slate-600">{t('staff.employees.empty')}</p> : null}

      {employees.map((employee) => {
        const snapshotIds = parsePermissionIds(employee.permission_snapshot_json);
        return (
          <article key={employee.id} className="space-y-3 rounded-xl border border-slate-200 p-4">
            <div className="flex flex-wrap justify-between gap-3">
              <div>
                <p className="text-sm font-medium text-slate-900">{employee.name}</p>
                <p className="text-xs text-slate-600">
                  {roleName(employee.role_id)} · {t(`staff.employees.status.${employee.status}`)} · {t('staff.employees.pinConfigured')}: {employee.pin_configured ? t('staff.common.yes') : t('staff.common.no')} · {t('staff.employees.pinVersion')}: {employee.pin_credential_version}
                </p>
              </div>
              <div className="flex flex-wrap gap-2">
                <button type="button" onClick={() => setEditing(employee)} className="rounded-lg border border-slate-300 px-2 py-1 text-xs text-slate-700" disabled={loading}>{t('catalog.shared.edit')}</button>
                <button type="button" onClick={() => void onActivate(employee.id)} className="rounded-lg border border-slate-300 px-2 py-1 text-xs text-slate-700" disabled={loading || employee.status === 'active'}>{t('staff.employees.actions.activate')}</button>
                <button type="button" onClick={() => void onSuspend(employee.id)} className="rounded-lg border border-amber-300 px-2 py-1 text-xs text-amber-700" disabled={loading || employee.status !== 'active'}>{t('staff.employees.actions.suspend')}</button>
                <button type="button" onClick={() => { if (window.confirm(t('catalog.shared.archiveConfirm'))) void onArchive(employee.id); }} className="rounded-lg border border-rose-300 px-2 py-1 text-xs text-rose-700" disabled={loading || employee.status === 'archived'}>{t('catalog.shared.archive')}</button>
              </div>
            </div>

            {editing?.id === employee.id ? (
              <div className="rounded-xl border border-slate-200 bg-slate-50 p-4">
                <form className="space-y-3" onSubmit={(event) => {
                  event.preventDefault();
                  void onUpdate(employee.id, editValues).then(() => setEditing(null));
                }}>
                  <div className="grid gap-3 md:grid-cols-3">
                    <div>
                      <label className="mb-1 block text-sm text-slate-700">{t('staff.employees.fields.name')}</label>
                      <input value={editValues.name} onChange={(event) => setEditValues({ ...editValues, name: event.target.value })} className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm" disabled={loading} />
                    </div>
                    <div>
                      <label className="mb-1 block text-sm text-slate-700">{t('staff.employees.fields.role')}</label>
                      {roleSelect(editValues.role_id, (role_id) => setEditValues({ ...editValues, role_id }))}
                    </div>
                    <div>
                      <label className="mb-1 block text-sm text-slate-700">{t('staff.employees.fields.status')}</label>
                      <select value={editValues.status} onChange={(event) => setEditValues({ ...editValues, status: event.target.value as EmployeeUpdateFormValues['status'] })} className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm" disabled={loading}>
                        {statuses.map((status) => <option key={status} value={status}>{t(`staff.employees.status.${status}`)}</option>)}
                      </select>
                    </div>
                  </div>
                  <div className="flex flex-wrap gap-2">
                    <button type="submit" disabled={loading || !editValues.name.trim() || !editValues.role_id} className="rounded-lg bg-slate-900 px-3 py-2 text-sm font-medium text-white disabled:opacity-50">{t('catalog.shared.save')}</button>
                    <button type="button" onClick={() => void onAssignRole(employee.id, editValues.role_id)} className="rounded-lg border border-slate-300 px-3 py-2 text-sm text-slate-700" disabled={loading || !editValues.role_id}>{t('staff.employees.actions.assignRole')}</button>
                    <button type="button" onClick={() => setEditing(null)} className="rounded-lg border border-slate-300 px-3 py-2 text-sm text-slate-700">{t('catalog.shared.cancel')}</button>
                  </div>
                </form>
              </div>
            ) : null}

            <div className="rounded-xl border border-slate-200 bg-slate-50 p-4">
              <div className="flex flex-wrap items-end gap-3">
                <div className="min-w-52 flex-1">
                  <label className="mb-1 block text-sm text-slate-700">{t('staff.employees.fields.rotatePin')}</label>
                  <input type="password" autoComplete="new-password" value={pinEmployeeId === employee.id ? nextPin : ''} onFocus={() => setPinEmployeeId(employee.id)} onChange={(event) => { setPinEmployeeId(employee.id); setNextPin(event.target.value); }} className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm" disabled={loading || employee.status === 'archived'} />
                </div>
                <button type="button" onClick={() => void onRotatePin(employee.id, nextPin).then(() => { setNextPin(''); setPinEmployeeId(''); })} className="rounded-lg border border-slate-300 px-3 py-2 text-sm text-slate-700" disabled={loading || pinEmployeeId !== employee.id || !nextPin.trim()}>{t('staff.employees.actions.rotatePin')}</button>
              </div>
            </div>

            <div>
              <p className="mb-2 text-sm font-medium text-slate-900">{t('staff.employees.snapshotTitle')}</p>
              <p className="mb-3 text-xs text-slate-600">{t('staff.employees.snapshotDescription')}</p>
              <PermissionMatrix selectedIds={snapshotIds} readonly />
            </div>
          </article>
        );
      })}
    </section>
  );
}
