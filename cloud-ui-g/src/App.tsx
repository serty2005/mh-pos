import { useCallback, useEffect, useState } from 'react';
import AnalyticsPanel from './components/AnalyticsPanel';
import MenuPanel from './components/MenuPanel';
import Sidebar from './components/Sidebar';
import StaffPanel from './components/StaffPanel';
import SyncPanel from './components/SyncPanel';
import { t } from './i18n';
import { restaurantsSchema, type ProbeResult } from './types';

const apiBase = (import.meta.env.VITE_CLOUD_API_BASE ?? 'http://localhost:8090/api/v1').replace(/\/$/, '');
const probeRoute = '/restaurants';

function nowIso() {
  return new Date().toISOString();
}

function blockedResult(errorMessageKey: ProbeResult['errorMessageKey']): ProbeResult {
  return {
    status: 'blocked',
    checkedAt: nowIso(),
    route: probeRoute,
    restaurantCount: 0,
    errorMessageKey,
  };
}

export default function App() {
  const [probe, setProbe] = useState<ProbeResult>({
    status: 'loading',
    checkedAt: nowIso(),
    route: probeRoute,
    restaurantCount: 0,
  });

  const checkRoute = useCallback(async () => {
    setProbe((prev) => ({ ...prev, status: 'loading', checkedAt: nowIso(), errorMessageKey: undefined }));

    try {
      const response = await fetch(`${apiBase}${probeRoute}`, {
        method: 'GET',
        headers: {
          Accept: 'application/json',
        },
      });

      if (!response.ok) {
        setProbe(blockedResult('errors.unavailable'));
        return;
      }

      const payload: unknown = await response.json();
      const restaurants = restaurantsSchema.safeParse(payload);
      if (!restaurants.success) {
        setProbe(blockedResult('errors.invalidResponse'));
        return;
      }

      setProbe({
        status: 'ready',
        checkedAt: nowIso(),
        route: probeRoute,
        restaurantCount: restaurants.data.length,
      });
    } catch {
      setProbe(blockedResult('errors.unavailable'));
    }
  }, []);

  useEffect(() => {
    void checkRoute();
  }, [checkRoute]);

  return (
    <main className="min-h-screen bg-slate-50 px-4 py-6 lg:px-8">
      <div className="mx-auto flex w-full max-w-7xl flex-col gap-6 lg:flex-row">
        <Sidebar appTitle={t('app.title')} appSubtitle={t('app.subtitle')} />

        <section className="flex-1 rounded-2xl border border-slate-200 bg-white p-6">
          <div className="grid gap-3 text-sm text-slate-700 sm:grid-cols-2">
            <div>
              <span className="text-slate-500">{t('app.environment')}:</span> {import.meta.env.MODE}
            </div>
            <div>
              <span className="text-slate-500">{t('app.apiBase')}:</span> {apiBase}
            </div>
            <div>
              <span className="text-slate-500">{t('readiness.route')}:</span> {probe.route}
            </div>
            <div>
              <span className="text-slate-500">{t('app.status')}:</span> {t(`status.${probe.status}`)}
            </div>
          </div>

          <div className="mt-4 rounded-xl border border-slate-200 bg-slate-50 p-4 text-sm text-slate-700">
            <p className="font-medium text-slate-900">{t('readiness.title')}</p>
            <p className="mt-1 text-slate-600">{t('readiness.description')}</p>
            <p className="mt-3">{t('readiness.lastCheck')}: {probe.checkedAt}</p>
            <p className="mt-1">restaurants: {probe.restaurantCount}</p>
            {probe.errorMessageKey ? <p className="mt-1 text-rose-600">{t(probe.errorMessageKey)}</p> : null}
            <button
              type="button"
              onClick={() => void checkRoute()}
              className="mt-4 rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm font-medium text-slate-700 hover:bg-slate-100"
            >
              {t('readiness.retry')}
            </button>
          </div>

          <div className="mt-6 grid gap-4 md:grid-cols-2">
            <AnalyticsPanel title={t('sections.analytics')} description={t('sections.blocked')} />
            <MenuPanel title={t('sections.menu')} description={t('sections.blocked')} />
            <StaffPanel title={t('sections.staff')} description={t('sections.blocked')} />
            <SyncPanel title={t('sections.sync')} description={t('sections.blocked')} />
          </div>

          <div className="mt-6 rounded-xl border border-dashed border-slate-300 p-4 text-sm text-slate-600">
            <p className="font-medium text-slate-900">{t('readiness.emptyTitle')}</p>
            <p className="mt-1">{t('readiness.emptyBody')}</p>
          </div>
        </section>
      </div>
    </main>
  );
}
