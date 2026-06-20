import { useEffect, useMemo, useState } from 'react';
import {
  Archive,
  CheckCircle2,
  CircleSlash2,
  Edit2,
  KeyRound,
  Search,
  UserPlus,
  X,
} from 'lucide-react';
import type { Employee, Restaurant, Role } from '../../shared/api/schemas';
import { useI18n } from '../../shared/i18n/I18nProvider';
import EmptyState from '../../shared/ui/EmptyState';
import SafeErrorBanner from '../../shared/ui/SafeErrorBanner';
import {
  defaultEmployeeCreateValues,
  defaultEmployeeUpdateValues,
  employeeToUpdateValues,
  parsePermissionIds,
  type EmployeeCreateFormValues,
  type EmployeeStatus,
  type EmployeeUpdateFormValues,
} from './staffForms';

type Props = {
  employees: Employee[];
  roles: Role[];
  restaurants: Restaurant[];
  defaultRestaurantId: string;
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

type StatusFilter = 'all' | EmployeeStatus;

const statuses: EmployeeStatus[] = ['active', 'suspended', 'archived'];

function initials(name: string) {
  return name.trim().slice(0, 2).toUpperCase() || 'ID';
}

function statusClass(status: EmployeeStatus) {
  if (status === 'active') return 'border-emerald-100 bg-emerald-50 text-emerald-700';
  if (status === 'suspended') return 'border-amber-100 bg-amber-50 text-amber-700';
  return 'border-slate-200 bg-slate-100 text-slate-500';
}

// EmployeesPanel не отображает матрицы прав сотрудников: права редактируются только в общей role matrix.
export default function EmployeesPanel({
  employees,
  roles,
  restaurants,
  defaultRestaurantId,
  loading,
  error,
  onCreate,
  onUpdate,
  onAssignRole,
  onSuspend,
  onActivate,
  onArchive,
  onRotatePin,
}: Props) {
  const { t } = useI18n();
  const [createValues, setCreateValues] = useState<EmployeeCreateFormValues>(defaultEmployeeCreateValues);
  const [editing, setEditing] = useState<Employee | null>(null);
  const [editValues, setEditValues] = useState<EmployeeUpdateFormValues>(defaultEmployeeUpdateValues);
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all');
  const [roleFilter, setRoleFilter] = useState('all');
  const [searchQuery, setSearchQuery] = useState('');
  const [isModalOpen, setModalOpen] = useState(false);
  const [nextPin, setNextPin] = useState('');

  const roleById = useMemo(() => new Map(roles.map((role) => [role.id, role])), [roles]);
  const activeRoles = useMemo(() => roles.filter((role) => role.active), [roles]);
  const roleName = (id: string) => roleById.get(id)?.name ?? id;

  const visibleEmployees = useMemo(() => {
    const query = searchQuery.trim().toLocaleLowerCase();
    return employees.filter((employee) => {
      const matchesStatus = statusFilter === 'all' || employee.status === statusFilter;
      const matchesRole = roleFilter === 'all' || employee.role_id === roleFilter;
      const haystack = `${employee.name} ${roleName(employee.role_id)} ${employee.id}`.toLocaleLowerCase();
      return matchesStatus && matchesRole && (!query || haystack.includes(query));
    });
  }, [employees, roleFilter, searchQuery, statusFilter, roleById]);

  useEffect(() => {
    if (editing) setEditValues(employeeToUpdateValues(editing));
  }, [editing]);

  const roleOptions = (selectedRoleId: string) => {
    const options = [...activeRoles];
    const selectedRole = roleById.get(selectedRoleId);
    if (selectedRole && !options.some((role) => role.id === selectedRole.id)) options.push(selectedRole);
    return options;
  };

  const openCreate = () => {
    setEditing(null);
    setCreateValues({ ...defaultEmployeeCreateValues, restaurant_ids: defaultRestaurantId ? [defaultRestaurantId] : [] });
    setNextPin('');
    setModalOpen(true);
  };

  const openEdit = (employee: Employee) => {
    setEditing(employee);
    setEditValues(employeeToUpdateValues(employee));
    setNextPin('');
    setModalOpen(true);
  };

  const closeModal = () => {
    setModalOpen(false);
    setEditing(null);
    setNextPin('');
  };

  const submitEmployee = async () => {
    if (editing) {
      await onUpdate(editing.id, { ...editValues, role_id: editing.role_id });
      if (editValues.role_id && editValues.role_id !== editing.role_id) await onAssignRole(editing.id, editValues.role_id);
      if (nextPin.trim()) await onRotatePin(editing.id, nextPin);
    } else {
      await onCreate(createValues);
      setCreateValues(defaultEmployeeCreateValues);
    }
    closeModal();
  };

  const formValues = editing ? editValues : createValues;
  const organizationManager = parsePermissionIds(roleById.get(formValues.role_id)?.permissions_json ?? '').includes('organization.manage');
  const hasActiveRoles = activeRoles.length > 0;

  return (
    <section className="space-y-4">
      <div className="rounded-2xl border border-slate-200 bg-white p-3 shadow-[0_18px_44px_-34px_rgba(15,23,42,0.32)]">
        <div className="flex flex-col gap-3 xl:flex-row xl:items-center xl:justify-between">
          <div className="flex flex-wrap gap-2">
            <button
              type="button"
              onClick={() => setStatusFilter('all')}
              className={[
                'rounded-xl border px-3 py-2 text-xs font-semibold transition-colors',
                statusFilter === 'all' ? 'border-slate-900 bg-slate-900 text-white' : 'border-slate-200 bg-white text-slate-600 hover:bg-slate-50',
              ].join(' ')}
            >
              {t('staff.employees.filters.allStatuses')}
            </button>
            {statuses.map((status) => (
              <button
                key={status}
                type="button"
                onClick={() => setStatusFilter(status)}
                className={[
                  'rounded-xl border px-3 py-2 text-xs font-semibold transition-colors',
                  statusFilter === status ? 'border-slate-900 bg-slate-900 text-white' : 'border-slate-200 bg-white text-slate-600 hover:bg-slate-50',
                ].join(' ')}
              >
                {t(`staff.employees.status.${status}`)}
              </button>
            ))}
          </div>

          <div className="grid gap-2 sm:grid-cols-[minmax(0,12rem)_minmax(0,18rem)_auto]">
            <select
              value={roleFilter}
              onChange={(event) => setRoleFilter(event.target.value)}
              className="rounded-xl border border-slate-200 bg-slate-50 px-3 py-2.5 text-xs font-semibold text-slate-700 outline-none transition-colors focus:border-blue-500 focus:bg-white"
            >
              <option value="all">{t('staff.employees.filters.allRoles')}</option>
              {roles.map((role) => (
                <option key={role.id} value={role.id}>{role.name}</option>
              ))}
            </select>
            <label className="relative block">
              <span className="sr-only">{t('staff.employees.searchLabel')}</span>
              <input
                type="search"
                value={searchQuery}
                onChange={(event) => setSearchQuery(event.target.value)}
                placeholder={t('staff.employees.searchPlaceholder')}
                className="w-full rounded-xl border border-slate-200 bg-slate-50 py-2.5 pl-3 pr-10 text-xs font-semibold text-slate-700 outline-none transition-colors placeholder:text-slate-400 focus:border-blue-500 focus:bg-white"
              />
              <Search className="pointer-events-none absolute right-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
            </label>
            <button
              type="button"
              onClick={openCreate}
              className="inline-flex items-center justify-center gap-2 rounded-xl bg-blue-600 px-4 py-2.5 text-xs font-semibold text-white transition-colors hover:bg-blue-700"
            >
              <UserPlus className="h-4 w-4" />
              {t('staff.employees.actions.new')}
            </button>
          </div>
        </div>
      </div>

      {error ? <SafeErrorBanner error={error} /> : null}
      {employees.length === 0 ? <EmptyState title={t('staff.employees.emptyTitle')} description={t('staff.employees.empty')} /> : null}
      {employees.length > 0 && visibleEmployees.length === 0 ? <EmptyState title={t('staff.employees.filteredEmptyTitle')} description={t('staff.employees.filteredEmptyDescription')} /> : null}

      {visibleEmployees.length > 0 ? (
        <div className="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-[0_18px_44px_-34px_rgba(15,23,42,0.32)]">
          <div className="overflow-x-auto">
            <table className="min-w-[920px] w-full border-collapse text-left text-xs">
              <thead className="bg-slate-50 text-[10px] font-bold uppercase tracking-wide text-slate-500">
                <tr>
                  <th className="px-4 py-3">{t('staff.employees.fields.name')}</th>
                  <th className="px-4 py-3">{t('staff.employees.fields.role')}</th>
                  <th className="px-4 py-3">{t('staff.employees.fields.status')}</th>
                  <th className="px-4 py-3">{t('staff.employees.pinConfigured')}</th>
                  <th className="px-4 py-3">{t('staff.employees.pinVersion')}</th>
                  <th className="px-4 py-3">{t('staff.employees.permissionCount')}</th>
                  <th className="px-4 py-3">{t('staff.employees.memberships')}</th>
                  <th className="px-4 py-3 text-right">{t('staff.employees.actions.title')}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100">
                {visibleEmployees.map((employee) => {
                  const permissionCount = parsePermissionIds(employee.permission_snapshot_json).length;
                  return (
                    <tr key={employee.id} className="hover:bg-slate-50/80">
                      <td className="px-4 py-3">
                        <div className="flex min-w-0 items-center gap-3">
                          <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl border border-blue-50 bg-blue-50 text-xs font-extrabold text-blue-800">
                            {initials(employee.name)}
                          </span>
                          <div className="min-w-0">
                            <p className="truncate text-sm font-semibold text-slate-950">{employee.name}</p>
                            <p className="truncate font-mono text-[10px] text-slate-400">{employee.id}</p>
                          </div>
                        </div>
                      </td>
                      <td className="px-4 py-3 font-medium text-slate-700">{roleName(employee.role_id)}</td>
                      <td className="px-4 py-3">
                        <span className={['rounded border px-2 py-0.5 text-[10px] font-bold uppercase tracking-wide', statusClass(employee.status)].join(' ')}>
                          {t(`staff.employees.status.${employee.status}`)}
                        </span>
                      </td>
                      <td className="px-4 py-3">
                        <span className="inline-flex items-center gap-1.5 font-semibold text-slate-700">
                          {employee.pin_configured ? <CheckCircle2 className="h-3.5 w-3.5 text-emerald-500" /> : <CircleSlash2 className="h-3.5 w-3.5 text-slate-400" />}
                          {employee.pin_configured ? t('staff.common.yes') : t('staff.common.no')}
                        </span>
                      </td>
                      <td className="px-4 py-3 font-mono tabular-nums text-slate-700">{employee.pin_credential_version}</td>
                      <td className="px-4 py-3 font-mono tabular-nums text-slate-700">{permissionCount}</td>
                      <td className="px-4 py-3 text-slate-600">{employee.all_restaurants ? t('staff.employees.allRestaurants') : employee.restaurant_ids.length}</td>
                      <td className="px-4 py-3">
                        <div className="flex justify-end gap-1.5">
                          <button
                            type="button"
                            onClick={() => openEdit(employee)}
                            disabled={loading || employee.status === 'archived'}
                            className="inline-flex h-8 w-8 items-center justify-center rounded-lg border border-slate-200 bg-white text-slate-600 transition-colors hover:border-blue-100 hover:bg-blue-50 hover:text-blue-700 disabled:cursor-not-allowed disabled:opacity-50"
                            aria-label={t('staff.employees.actions.edit')}
                          >
                            <Edit2 className="h-3.5 w-3.5" />
                          </button>
                          <button
                            type="button"
                            onClick={() => void onActivate(employee.id)}
                            className="rounded-lg border border-emerald-200 px-2.5 py-1.5 text-xs font-semibold text-emerald-700 transition-colors hover:bg-emerald-50 disabled:cursor-not-allowed disabled:opacity-50"
                            disabled={loading || employee.status === 'active' || employee.status === 'archived'}
                          >
                            {t('staff.employees.actions.activate')}
                          </button>
                          <button
                            type="button"
                            onClick={() => void onSuspend(employee.id)}
                            className="rounded-lg border border-amber-200 px-2.5 py-1.5 text-xs font-semibold text-amber-700 transition-colors hover:bg-amber-50 disabled:cursor-not-allowed disabled:opacity-50"
                            disabled={loading || employee.status !== 'active'}
                          >
                            {t('staff.employees.actions.suspend')}
                          </button>
                          <button
                            type="button"
                            onClick={() => { if (window.confirm(t('catalog.shared.archiveConfirm'))) void onArchive(employee.id); }}
                            className="inline-flex h-8 w-8 items-center justify-center rounded-lg border border-rose-200 bg-white text-rose-700 transition-colors hover:bg-rose-50 disabled:cursor-not-allowed disabled:opacity-50"
                            disabled={loading || employee.status === 'archived'}
                            aria-label={t('catalog.shared.archive')}
                          >
                            <Archive className="h-3.5 w-3.5" />
                          </button>
                        </div>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>
      ) : null}

      {isModalOpen ? (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/40 p-4 backdrop-blur-sm">
          <div className="w-full max-w-xl overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-2xl">
            <div className="flex items-center justify-between border-b border-slate-200 bg-slate-50 px-5 py-4">
              <div>
                <h3 className="text-sm font-semibold text-slate-950">
                  {editing ? t('staff.employees.editTitle') : t('staff.employees.createTitle')}
                </h3>
                <p className="mt-1 text-xs text-slate-500">
                  {editing ? t('staff.employees.editDescription') : t('staff.employees.createDescription')}
                </p>
              </div>

              <button
                type="button"
                onClick={closeModal}
                className="inline-flex h-9 w-9 items-center justify-center rounded-xl border border-slate-200 bg-white text-slate-500 transition-colors hover:bg-slate-100"
                aria-label={t('catalog.shared.cancel')}
              >
                <X className="h-4 w-4" />
              </button>
            </div>

            <form
              className="space-y-4 p-5"
              onSubmit={(event) => {
                event.preventDefault();
                void submitEmployee();
              }}
            >
              <label className="block">
                <span className="mb-1.5 block text-xs font-semibold text-slate-600">{t('staff.employees.fields.name')}</span>
                <input
                  value={formValues.name}
                  onChange={(event) => {
                    if (editing) setEditValues({ ...editValues, name: event.target.value });
                    else setCreateValues({ ...createValues, name: event.target.value });
                  }}
                  className="w-full rounded-xl border border-slate-200 bg-slate-50 px-3 py-2.5 text-sm outline-none transition-colors focus:border-blue-500 focus:bg-white disabled:cursor-not-allowed disabled:opacity-60"
                  disabled={loading}
                />
              </label>

              <div className="grid gap-4 sm:grid-cols-2">
                <label className="block">
                  <span className="mb-1.5 block text-xs font-semibold text-slate-600">{t('staff.employees.fields.role')}</span>
                  <select
                    value={formValues.role_id}
                    onChange={(event) => {
                      if (editing) setEditValues({ ...editValues, role_id: event.target.value });
                      else setCreateValues({ ...createValues, role_id: event.target.value });
                    }}
                    className="w-full rounded-xl border border-slate-200 bg-white px-3 py-2.5 text-sm font-medium text-slate-800 outline-none transition-colors focus:border-blue-500 disabled:cursor-not-allowed disabled:opacity-60"
                    disabled={loading}
                  >
                    <option value="">{t('staff.employees.fields.selectRole')}</option>
                    {roleOptions(formValues.role_id).map((role) => (
                      <option key={role.id} value={role.id}>{role.name}</option>
                    ))}
                  </select>
                  {!hasActiveRoles ? <p className="mt-2 text-xs leading-5 text-amber-700">{t('staff.employees.noActiveRoles')}</p> : null}
                </label>

                {editing ? (
                  <label className="block">
                    <span className="mb-1.5 block text-xs font-semibold text-slate-600">{t('staff.employees.fields.status')}</span>
                    <select
                      value={editValues.status}
                      onChange={(event) => setEditValues({ ...editValues, status: event.target.value as EmployeeStatus })}
                      className="w-full rounded-xl border border-slate-200 bg-white px-3 py-2.5 text-sm font-medium text-slate-800 outline-none transition-colors focus:border-blue-500 disabled:cursor-not-allowed disabled:opacity-60"
                      disabled={loading}
                    >
                      {statuses.map((status) => (
                        <option key={status} value={status}>{t(`staff.employees.status.${status}`)}</option>
                      ))}
                    </select>
                  </label>
                ) : (
                  <label className="block">
                    <span className="mb-1.5 block text-xs font-semibold text-slate-600">{t('staff.employees.fields.pin')}</span>
                    <input
                      type="password"
                      autoComplete="new-password"
                      value={createValues.pin}
                      onChange={(event) => setCreateValues({ ...createValues, pin: event.target.value })}
                      className="w-full rounded-xl border border-slate-200 bg-slate-50 px-3 py-2.5 text-sm outline-none transition-colors focus:border-blue-500 focus:bg-white disabled:cursor-not-allowed disabled:opacity-60"
                      disabled={loading}
                    />
                    <p className="mt-2 text-xs leading-5 text-slate-500">{t('staff.employees.pinHelp')}</p>
                  </label>
                )}
              </div>

              <fieldset className="rounded-xl border border-slate-200 p-3" disabled={loading || organizationManager}>
                <legend className="px-1 text-xs font-semibold text-slate-600">{t('staff.employees.memberships')}</legend>
                {organizationManager ? <p className="text-xs text-slate-500">{t('staff.employees.organizationScope')}</p> : (
                  <div className="grid gap-2 sm:grid-cols-2">
                    {restaurants.filter((restaurant) => restaurant.status === 'active').map((restaurant) => (
                      <label key={restaurant.id} className="flex items-center gap-2 text-sm text-slate-700">
                        <input
                          type="checkbox"
                          checked={formValues.restaurant_ids.includes(restaurant.id)}
                          onChange={(event) => {
                            const ids = event.target.checked
                              ? [...formValues.restaurant_ids, restaurant.id]
                              : formValues.restaurant_ids.filter((id) => id !== restaurant.id);
                            if (editing) setEditValues({ ...editValues, restaurant_ids: ids });
                            else setCreateValues({ ...createValues, restaurant_ids: ids });
                          }}
                        />
                        {restaurant.name}
                      </label>
                    ))}
                  </div>
                )}
              </fieldset>

              {editing ? (
                <label className="block">
                  <span className="mb-1.5 block text-xs font-semibold text-slate-600">{t('staff.employees.fields.rotatePin')}</span>
                  <div className="relative">
                    <input
                      type="password"
                      autoComplete="new-password"
                      value={nextPin}
                      onChange={(event) => setNextPin(event.target.value)}
                      className="w-full rounded-xl border border-slate-200 bg-slate-50 py-2.5 pl-3 pr-10 text-sm outline-none transition-colors focus:border-blue-500 focus:bg-white disabled:cursor-not-allowed disabled:opacity-60"
                      disabled={loading || editing.status === 'archived'}
                    />
                    <KeyRound className="pointer-events-none absolute right-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
                  </div>
                </label>
              ) : null}

              <div className="flex justify-end gap-2 border-t border-slate-100 pt-4">
                <button
                  type="button"
                  onClick={closeModal}
                  className="rounded-xl border border-slate-200 px-4 py-2.5 text-sm font-semibold text-slate-600 transition-colors hover:bg-slate-50"
                >
                  {t('catalog.shared.cancel')}
                </button>
                <button
                  type="submit"
                  disabled={loading || !formValues.name.trim() || !formValues.role_id || (!organizationManager && formValues.restaurant_ids.length === 0) || (!editing && !createValues.pin.trim())}
                  className="rounded-xl bg-blue-600 px-4 py-2.5 text-sm font-semibold text-white transition-colors hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-50"
                >
                  {editing ? t('catalog.shared.save') : t('staff.employees.actions.create')}
                </button>
              </div>
            </form>
          </div>
        </div>
      ) : null}
    </section>
  );
}
