import { useEffect, useState } from 'react';
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

// StaffPage связывает role-backed permission management с employee lifecycle без backend employee override.
export default function StaffPage({ restaurantId }: Props) {
  const { t } = useI18n();
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
    <section className="space-y-4">
      <div className="rounded-2xl border border-slate-200 bg-white p-6">
        <h3 className="text-base font-semibold text-slate-900">{t('staff.pageTitle')}</h3>
        <p className="mt-1 text-sm text-slate-600">{t('staff.pageDescription')}</p>
        <p className="mt-2 text-xs text-slate-500">{t('catalog.readiness')}: {status === 'ready' ? t('status.ready') : status === 'loading' ? t('status.loading') : t('status.blocked')}</p>
      </div>
      {status === 'blocked' ? <EmptyState title={t('staff.blockedTitle')} description={t('staff.blockedDescription')} /> : null}
      {status !== 'blocked' ? (
        <>
          <RolesPanel
            roles={roles}
            loading={loading}
            error={error}
            onCreate={(values: RoleFormValues) => mutate(async () => { await createRole({ restaurant_id: restaurantId, ...buildCreateRolePayload(values) }); })}
            onUpdate={(id: string, values: RoleFormValues) => mutate(async () => { await updateRole(id, buildUpdateRolePayload(values)); })}
            onArchive={(id: string) => mutate(async () => { await archiveRole(id); })}
          />
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
        </>
      ) : null}
    </section>
  );
}
