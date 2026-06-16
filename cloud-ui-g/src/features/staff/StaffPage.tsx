import { useEffect, useState } from 'react';
import { ShieldCheck, Users } from 'lucide-react';
import {
  activateEmployee,
  archiveEmployee,
  archiveRole,
  assignEmployeeRole,
  createEmployee,
  createRole,
  listEmployees,
  listRoles,
  rotateEmployeePIN,
  suspendEmployee,
  updateEmployee,
  updateRole,
} from '../../shared/api/endpoints';
import type { Employee, Role } from '../../shared/api/schemas';
import { useI18n } from '../../shared/i18n/I18nProvider';
import EmptyState from '../../shared/ui/EmptyState';
import LoadingSkeleton from '../../shared/ui/LoadingSkeleton';
import PanelHeader from '../../shared/ui/PanelHeader';
import EmployeesPanel from './EmployeesPanel';
import RolesPanel from './RolesPanel';
import {
  buildCreateEmployeePayload,
  buildCreateRolePayload,
  buildUpdateEmployeePayload,
  buildUpdateRolePayload,
  type EmployeeCreateFormValues,
  type EmployeeUpdateFormValues,
  type RoleFormValues,
} from './staffForms';

type Props = {
  restaurantId: string;
};

type RouteStatus = 'loading' | 'ready' | 'blocked';
type StaffTab = 'employees' | 'roles';

// StaffPage связывает role-backed permission management с employee lifecycle без backend employee override.
export default function StaffPage({ restaurantId }: Props) {
  const { t } = useI18n();
  const [activeTab, setActiveTab] = useState<StaffTab>('employees');
  const [roles, setRoles] = useState<Role[]>([]);
  const [employees, setEmployees] = useState<Employee[]>([]);
  const [status, setStatus] = useState<RouteStatus>('loading');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);

  const reload = async () => {
    setStatus('loading');
    setError(null);
    try {
      const [nextRoles, nextEmployees] = await Promise.all([
        listRoles(restaurantId),
        listEmployees(restaurantId),
      ]);
      setRoles(nextRoles);
      setEmployees(nextEmployees);
      setStatus('ready');
    } catch (nextError) {
      setStatus('blocked');
      setError(nextError);
    }
  };

  useEffect(() => {
    void reload();
  }, [restaurantId]);

  const mutate = async (action: () => Promise<void>) => {
    setLoading(true);
    setError(null);
    try {
      await action();
      await reload();
    } catch (nextError) {
      setError(nextError);
    } finally {
      setLoading(false);
    }
  };

  return (
    <section className="space-y-5">
      <PanelHeader
        icon={ShieldCheck}
        title={t('staff.pageTitle')}
        description={t('staff.pageDescription')}
        action={(
          <p className={status === 'ready' ? 'rounded-full border border-emerald-100 bg-emerald-50 px-3 py-1.5 text-xs font-semibold text-emerald-700' : status === 'loading' ? 'rounded-full border border-blue-100 bg-blue-50 px-3 py-1.5 text-xs font-semibold text-blue-700' : 'rounded-full border border-amber-100 bg-amber-50 px-3 py-1.5 text-xs font-semibold text-amber-700'}>
            {t('catalog.readiness')}: {status === 'ready' ? t('status.ready') : status === 'loading' ? t('status.loading') : t('status.blocked')}
          </p>
        )}
      />

      <div className="rounded-2xl border border-slate-200 bg-white p-2 shadow-[0_18px_44px_-34px_rgba(15,23,42,0.32)]">
        <div className="flex flex-col gap-2 sm:flex-row">
          <button
            type="button"
            onClick={() => setActiveTab('employees')}
            className={[
              'inline-flex flex-1 items-center justify-center gap-2 rounded-xl px-4 py-3 text-sm font-semibold transition-colors',
              activeTab === 'employees' ? 'bg-blue-600 text-white' : 'text-slate-600 hover:bg-slate-50',
            ].join(' ')}
          >
            <Users className="h-4 w-4" />
            {t('staff.tabs.employees')}
          </button>
          <button
            type="button"
            onClick={() => setActiveTab('roles')}
            className={[
              'inline-flex flex-1 items-center justify-center gap-2 rounded-xl px-4 py-3 text-sm font-semibold transition-colors',
              activeTab === 'roles' ? 'bg-blue-600 text-white' : 'text-slate-600 hover:bg-slate-50',
            ].join(' ')}
          >
            <ShieldCheck className="h-4 w-4" />
            {t('staff.tabs.roles')}
          </button>
        </div>
      </div>

      {status === 'blocked' ? <EmptyState title={t('staff.blockedTitle')} description={t('staff.blockedDescription')} /> : null}
      {status === 'loading' ? <LoadingSkeleton cards={4} className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4" /> : null}
      {status === 'ready' ? (
        <>
          {activeTab === 'employees' ? (
            <EmployeesPanel
              employees={employees}
              roles={roles}
              loading={loading}
              error={error}
              onCreate={(values: EmployeeCreateFormValues) => mutate(async () => { await createEmployee({ restaurant_id: restaurantId, ...buildCreateEmployeePayload(values) }); })}
              onUpdate={(id: string, values: EmployeeUpdateFormValues) => mutate(async () => { await updateEmployee(id, buildUpdateEmployeePayload(values)); })}
              onAssignRole={(id: string, roleId: string) => mutate(async () => { await assignEmployeeRole(id, roleId); })}
              onSuspend={(id: string) => mutate(async () => { await suspendEmployee(id); })}
              onActivate={(id: string) => mutate(async () => { await activateEmployee(id); })}
              onArchive={(id: string) => mutate(async () => { await archiveEmployee(id); })}
              onRotatePin={(id: string, pin: string) => mutate(async () => { await rotateEmployeePIN(id, pin.trim()); })}
            />
          ) : null}
          {activeTab === 'roles' ? (
            <RolesPanel
              roles={roles}
              employees={employees}
              loading={loading}
              error={error}
              onCreate={(values: RoleFormValues) => mutate(async () => { await createRole({ restaurant_id: restaurantId, ...buildCreateRolePayload(values) }); })}
              onUpdate={(id: string, values: RoleFormValues) => mutate(async () => { await updateRole(id, buildUpdateRolePayload(values)); })}
              onArchive={(id: string) => mutate(async () => { await archiveRole(id); })}
            />
          ) : null}
        </>
      ) : null}
    </section>
  );
}
