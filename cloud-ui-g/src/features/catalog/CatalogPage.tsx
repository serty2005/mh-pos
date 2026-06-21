import { useEffect, useState } from 'react';
import { ClipboardList } from 'lucide-react';
import {
  archiveCatalogFolder,
  archiveCatalogItem,
  assignCatalogItemTag,
  createCatalogFolder,
  createCatalogItem,
  createCatalogTag,
  listCatalogFolders,
  listCatalogItems,
  listCatalogTags,
  updateCatalogFolder,
  updateCatalogItem,
  updateCatalogTag,
} from '../../shared/api/endpoints';
import type { CatalogFolder, CatalogItem, CatalogTag } from '../../shared/api/schemas';
import { useI18n } from '../../shared/i18n/I18nProvider';
import EmptyState from '../../shared/ui/EmptyState';
import CatalogFoldersPanel from './CatalogFoldersPanel';
import CatalogItemsPanel from './CatalogItemsPanel';
import CatalogTagsPanel from './CatalogTagsPanel';
import ItemTagCommandPanel from './ItemTagCommandPanel';
import {
  buildCreateCatalogFolderPayload,
  buildCreateCatalogItemPayload,
  buildCreateCatalogTagPayload,
  type CatalogFolderFormValues,
  type CatalogItemFormValues,
  type CatalogTagFormValues,
  type ItemTagCommandFormValues,
} from './catalogForms';

type RouteStatus = 'loading' | 'ready' | 'blocked';

export default function CatalogPage() {
  const { t } = useI18n();
  const [items, setItems] = useState<CatalogItem[]>([]);
  const [folders, setFolders] = useState<CatalogFolder[]>([]);
  const [tags, setTags] = useState<CatalogTag[]>([]);
  const [status, setStatus] = useState<RouteStatus>('loading');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [itemTagSuccess, setItemTagSuccess] = useState(false);

  const reload = async () => {
    setStatus('loading');
    setError(null);
    try {
      const [nextItems, nextFolders, nextTags] = await Promise.all([
        listCatalogItems(),
        listCatalogFolders(),
        listCatalogTags(),
      ]);
      setItems(nextItems);
      setFolders(nextFolders);
      setTags(nextTags);
      setStatus('ready');
    } catch (nextError) {
      setStatus('blocked');
      setError(nextError);
    }
  };

  useEffect(() => {
    void reload();
  }, []);

  const mutate = async (action: () => Promise<void>) => {
    setLoading(true);
    setItemTagSuccess(false);
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

  const handleItemTagAssign = async (values: ItemTagCommandFormValues) => {
    setLoading(true);
    setItemTagSuccess(false);
    setError(null);
    try {
      await assignCatalogItemTag(values);
      setItemTagSuccess(true);
      await reload();
    } catch (nextError) {
      setError(nextError);
    } finally {
      setLoading(false);
    }
  };

  return (
    <section className="space-y-4">
      <div className="rounded-2xl border border-slate-200 bg-white p-5 sm:p-6">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
          <div className="flex items-start gap-3">
            <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border border-blue-100 bg-blue-50 text-blue-700">
              <ClipboardList className="h-4 w-4" />
            </div>
            <div>
              <h3 className="text-lg font-semibold tracking-tight text-slate-950">{t('catalog.pageTitle')}</h3>
              <p className="mt-1 max-w-3xl text-sm leading-6 text-slate-600">{t('catalog.pageDescription')}</p>
            </div>
          </div>
          <p className={status === 'ready' ? 'rounded-full border border-emerald-100 bg-emerald-50 px-3 py-1.5 text-xs font-semibold text-emerald-700' : status === 'loading' ? 'rounded-full border border-blue-100 bg-blue-50 px-3 py-1.5 text-xs font-semibold text-blue-700' : 'rounded-full border border-amber-100 bg-amber-50 px-3 py-1.5 text-xs font-semibold text-amber-700'}>
            {t('catalog.readiness')}: {status === 'ready' ? t('status.ready') : status === 'loading' ? t('status.loading') : t('status.blocked')}
          </p>
        </div>
      </div>

      {status === 'blocked' ? <EmptyState title={t('catalog.blockedTitle')} description={t('catalog.blockedDescription')} /> : null}

      {status !== 'blocked' ? (
        <>
          <CatalogItemsPanel
            items={items}
            folders={folders}
            loading={loading}
            error={error}
            onCreate={(values: CatalogItemFormValues) => mutate(async () => {
              await createCatalogItem(buildCreateCatalogItemPayload(values));
            })}
            onUpdate={(id: string, values: CatalogItemFormValues) => mutate(async () => {
              await updateCatalogItem(id, values);
            })}
            onArchive={(id: string) => mutate(async () => {
              await archiveCatalogItem(id);
            })}
          />

          <CatalogFoldersPanel
            folders={folders}
            loading={loading}
            error={error}
            onCreate={(values: CatalogFolderFormValues) => mutate(async () => {
              await createCatalogFolder(buildCreateCatalogFolderPayload(values));
            })}
            onUpdate={(id: string, values: CatalogFolderFormValues) => mutate(async () => {
              await updateCatalogFolder(id, values);
            })}
            onArchive={(id: string) => mutate(async () => {
              await archiveCatalogFolder(id);
            })}
          />

          <CatalogTagsPanel
            tags={tags}
            loading={loading}
            error={error}
            onCreate={(values: CatalogTagFormValues) => mutate(async () => {
              await createCatalogTag(buildCreateCatalogTagPayload(values));
            })}
            onUpdate={(id: string, values: CatalogTagFormValues) => mutate(async () => {
              await updateCatalogTag(id, values);
            })}
          />

          <ItemTagCommandPanel
            items={items.filter((item) => item.status !== 'archived')}
            tags={tags.filter((tag) => tag.status !== 'archived')}
            loading={loading}
            error={error}
            success={itemTagSuccess}
            onAssign={handleItemTagAssign}
          />
        </>
      ) : null}
    </section>
  );
}
