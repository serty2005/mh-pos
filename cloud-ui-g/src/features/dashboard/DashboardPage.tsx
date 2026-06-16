import { BarChart3, ClipboardCheck, Store, TrendingUp, UploadCloud } from 'lucide-react';
import { useI18n } from '../../shared/i18n/I18nProvider';
import PanelHeader from '../../shared/ui/PanelHeader';
import LaunchReadinessPanel from './LaunchReadinessPanel';
import PublicationPanel from '../publications/PublicationPanel';
import { usePublication } from '../publications/usePublication';

type DashboardPageProps = {
  restaurantId: string;
};

export default function DashboardPage({ restaurantId }: DashboardPageProps) {
  const { t } = useI18n();
  const { publication } = usePublication(restaurantId);
  const kpis = [
    {
      icon: TrendingUp,
      title: t('dashboard.kpis.salesOverviewTitle'),
      value: t('dashboard.kpis.salesOverviewValue'),
      body: t('dashboard.kpis.salesOverviewBody'),
      tone: 'amber',
    },
    {
      icon: Store,
      title: t('dashboard.kpis.restaurantScopeTitle'),
      value: restaurantId ? t('dashboard.kpis.restaurantSelected') : t('dashboard.kpis.restaurantRequired'),
      body: t('dashboard.kpis.restaurantScopeBody'),
      tone: restaurantId ? 'emerald' : 'amber',
    },
    {
      icon: UploadCloud,
      title: t('dashboard.kpis.publicationTitle'),
      value: publication ? t('dashboard.kpis.publicationReady') : t('dashboard.kpis.publicationPending'),
      body: t('dashboard.kpis.publicationBody'),
      tone: publication ? 'emerald' : 'blue',
    },
  ];

  return (
    <div className="space-y-5">
      <section className="rounded-2xl border border-slate-200 bg-white p-5 sm:p-6">
        <PanelHeader
          icon={BarChart3}
          title={t('dashboard.pageTitle')}
          description={t('dashboard.pageDescription')}
          meta={(
            <span className="inline-flex rounded-lg border border-blue-100 bg-blue-50 px-2 py-1 font-mono text-[10px] font-semibold uppercase tracking-wider text-blue-700">
              {t('dashboard.routeBackedBadge')}
            </span>
          )}
          action={(
            <div className="rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 text-right">
              <p className="text-xs font-semibold text-slate-500">{t('dashboard.selectedRestaurant')}</p>
              <p className="mt-1 max-w-[12rem] truncate font-mono text-sm font-semibold text-slate-950">{restaurantId || t('restaurants.notSelected')}</p>
            </div>
          )}
        />
      </section>

      <section className="grid gap-3 md:grid-cols-3">
        {kpis.map((item) => {
          const Icon = item.icon;
          const toneClass = item.tone === 'emerald'
            ? 'border-emerald-100 bg-emerald-50 text-emerald-700'
            : item.tone === 'amber'
              ? 'border-amber-100 bg-amber-50 text-amber-700'
              : 'border-blue-100 bg-blue-50 text-blue-700';

          return (
            <article key={item.title} className="rounded-2xl border border-slate-200 bg-white p-4">
              <div className="flex items-start justify-between gap-3">
                <div className={`flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border ${toneClass}`}>
                  <Icon className="h-4 w-4" />
                </div>
                <ClipboardCheck className="h-4 w-4 shrink-0 text-slate-300" />
              </div>
              <p className="mt-4 text-xs font-semibold text-slate-500">{item.title}</p>
              <p className="mt-1 text-xl font-semibold tracking-tight text-slate-950">{item.value}</p>
              <p className="mt-2 text-sm leading-6 text-slate-600">{item.body}</p>
            </article>
          );
        })}
      </section>

      <LaunchReadinessPanel restaurantId={restaurantId} hasPublication={Boolean(publication)} />
      <PublicationPanel restaurantId={restaurantId} canPublish={Boolean(restaurantId)} />
    </div>
  );
}
