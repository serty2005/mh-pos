import { useEffect, useMemo, useState, type FormEvent, type MouseEvent, type ReactNode } from 'react';
import { MoreVertical, Plus, RefreshCw, Search, X } from 'lucide-react';
import {
  archiveCatalogFolder,
  archiveCatalogItem,
  archiveMenuCategory,
  archiveMenuItem,
  assignCatalogItemTag,
  createCatalogFolder,
  createCatalogItem,
  createCatalogTag,
  createMenuCategory,
  createMenuItem,
  listCatalogFolders,
  listCatalogItems,
  listCatalogTags,
  listMenuCategories,
  listMenuItems,
  updateCatalogFolder,
  updateCatalogItem,
  updateCatalogTag,
  updateMenuCategory,
  updateMenuItem,
} from '../../shared/api/endpoints';
import type { CatalogFolder, CatalogItem, CatalogTag, Category, MenuItem } from '../../shared/api/schemas';
import { useI18n } from '../../shared/i18n/I18nProvider';
import EmptyState from '../../shared/ui/EmptyState';
import SafeErrorBanner from '../../shared/ui/SafeErrorBanner';
import {
  buildCreateCatalogFolderPayload,
  buildCreateCatalogItemPayload,
  buildCreateCatalogTagPayload,
  defaultCatalogFolderValues,
  defaultCatalogItemValues,
  defaultCatalogTagValues,
  defaultItemTagCommandValues,
  toCatalogFolderValues,
  toCatalogItemValues,
  toCatalogTagValues,
  type CatalogFolderFormValues,
  type CatalogItemFormValues,
  type CatalogKind,
  type CatalogTagFormValues,
  type ItemTagCommandFormValues,
  type LifecycleStatus,
} from './catalogForms';
import {
  buildCreateMenuItemPayload,
  defaultMenuCategoryValues,
  defaultMenuItemValues,
  normalizeMenuItemValues,
  toMenuItemValues,
  type MenuCategoryFormValues,
  type MenuItemFormValues,
} from '../menu/menuForms';

type Props = {
  restaurantId?: string;
  restaurantCurrency?: string;
};

type RouteStatus = 'loading' | 'ready' | 'blocked';
type ViewMode = 'catalog' | 'menu';
type NodeSelection =
  | { kind: 'catalog-folder'; id: string }
  | { kind: 'catalog-item'; id: string }
  | { kind: 'menu-category'; id: string }
  | { kind: 'menu-item'; id: string };
type DialogState =
  | { kind: 'catalog-folder-create' }
  | { kind: 'catalog-folder-edit'; folder: CatalogFolder }
  | { kind: 'catalog-item-create' }
  | { kind: 'catalog-item-edit'; item: CatalogItem }
  | { kind: 'catalog-tag-create' }
  | { kind: 'catalog-tag-edit'; tag: CatalogTag }
  | { kind: 'item-tag-assign' }
  | { kind: 'menu-category-create' }
  | { kind: 'menu-category-edit'; category: Category }
  | { kind: 'menu-item-create'; catalogItem?: CatalogItem }
  | { kind: 'menu-item-edit'; item: MenuItem };
type ContextMenuState = { x: number; y: number; selection: NodeSelection } | null;

const lifecycleStatuses: LifecycleStatus[] = ['draft', 'published', 'archived'];
const catalogKinds: CatalogKind[] = ['dish', 'good', 'semi_finished', 'service'];
const runtimeStatuses = ['available', 'unavailable', 'hidden'];

export default function CatalogPage({ restaurantId = '', restaurantCurrency = 'RUB' }: Props) {
  const { t } = useI18n();
  const [items, setItems] = useState<CatalogItem[]>([]);
  const [folders, setFolders] = useState<CatalogFolder[]>([]);
  const [tags, setTags] = useState<CatalogTag[]>([]);
  const [menuItems, setMenuItems] = useState<MenuItem[]>([]);
  const [menuCategories, setMenuCategories] = useState<Category[]>([]);
  const [status, setStatus] = useState<RouteStatus>('loading');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [dialog, setDialog] = useState<DialogState | null>(null);
  const [selection, setSelection] = useState<NodeSelection | null>(null);
  const [viewMode, setViewMode] = useState<ViewMode>('catalog');
  const [search, setSearch] = useState('');
  const [isCreateMenuOpen, setCreateMenuOpen] = useState(false);
  const [contextMenu, setContextMenu] = useState<ContextMenuState>(null);
  const [itemTagSuccess, setItemTagSuccess] = useState(false);
  const hasRestaurant = Boolean(restaurantId.trim());

  const reload = async () => {
    setStatus('loading');
    setError(null);
    try {
      const [nextItems, nextFolders, nextTags, nextMenuItems, nextMenuCategories] = await Promise.all([
        listCatalogItems(hasRestaurant ? restaurantId : undefined),
        listCatalogFolders(hasRestaurant ? restaurantId : undefined),
        listCatalogTags(hasRestaurant ? restaurantId : undefined),
        hasRestaurant ? listMenuItems(restaurantId) : Promise.resolve([]),
        hasRestaurant ? listMenuCategories(restaurantId) : Promise.resolve([]),
      ]);
      setItems(nextItems);
      setFolders(nextFolders);
      setTags(nextTags);
      setMenuItems(nextMenuItems);
      setMenuCategories(nextMenuCategories);
      setStatus('ready');
    } catch (nextError) {
      setStatus('blocked');
      setError(nextError);
    }
  };

  useEffect(() => {
    void reload();
  }, [restaurantId]);

  useEffect(() => {
    if (!hasRestaurant && viewMode === 'menu') setViewMode('catalog');
  }, [hasRestaurant, viewMode]);

  const saleByCatalog = useMemo(() => new Map(menuItems.map((item) => [item.catalog_item_id, item])), [menuItems]);
  const itemById = useMemo(() => new Map(items.map((item) => [item.id, item])), [items]);
  const folderById = useMemo(() => new Map(folders.map((folder) => [folder.id, folder])), [folders]);
  const categoryById = useMemo(() => new Map(menuCategories.map((category) => [category.id, category])), [menuCategories]);
  const menuItemById = useMemo(() => new Map(menuItems.map((item) => [item.id, item])), [menuItems]);
  const visibleItems = useMemo(() => filterCatalogItems(items, search), [items, search]);
  const visibleMenuItems = useMemo(() => filterMenuItems(menuItems, search, itemById), [menuItems, search, itemById]);
  const activeSelection = resolveSelection(selection, folderById, itemById, categoryById, menuItemById);

  const mutate = async (action: () => Promise<void>) => {
    setLoading(true);
    setItemTagSuccess(false);
    setError(null);
    try {
      await action();
      await reload();
      setDialog(null);
    } catch (nextError) {
      setError(nextError);
    } finally {
      setLoading(false);
    }
  };

  const selectNode = (next: NodeSelection) => {
    setSelection(next);
    setContextMenu(null);
  };

  const archiveSelected = (target: NodeSelection) => {
    if (!window.confirm(t('catalogMenu.archiveConfirm'))) return;
    void mutate(async () => {
      if (target.kind === 'catalog-folder') await archiveCatalogFolder(target.id);
      if (target.kind === 'catalog-item') await archiveCatalogItem(target.id);
      if (target.kind === 'menu-category') await archiveMenuCategory(target.id);
      if (target.kind === 'menu-item') await archiveMenuItem(target.id);
    });
  };

  const openEditDialog = (target: NodeSelection) => {
    if (target.kind === 'catalog-folder') {
      const folder = folderById.get(target.id);
      if (folder) setDialog({ kind: 'catalog-folder-edit', folder });
    }
    if (target.kind === 'catalog-item') {
      const item = itemById.get(target.id);
      if (item) setDialog({ kind: 'catalog-item-edit', item });
    }
    if (target.kind === 'menu-category') {
      const category = categoryById.get(target.id);
      if (category) setDialog({ kind: 'menu-category-edit', category });
    }
    if (target.kind === 'menu-item') {
      const item = menuItemById.get(target.id);
      if (item) setDialog({ kind: 'menu-item-edit', item });
    }
    setContextMenu(null);
  };

  return (
    <section className="space-y-4" onClick={() => setContextMenu(null)}>
      <header className="rounded-2xl border border-slate-200 bg-white p-5 sm:p-6">
        <div className="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
          <div>
            <h3 className="text-lg font-semibold tracking-tight text-slate-950">{t('catalogMenu.pageTitle')}</h3>
            <p className="mt-1 max-w-3xl text-sm leading-6 text-slate-600">{hasRestaurant ? t('catalogMenu.pageDescriptionRestaurant') : t('catalogMenu.pageDescriptionTenant')}</p>
          </div>
          <p className={statusBadgeClass(status)}>
            {t('catalog.readiness')}: {status === 'ready' ? t('status.ready') : status === 'loading' ? t('status.loading') : t('status.blocked')}
          </p>
        </div>
      </header>

      <div className="rounded-2xl border border-slate-200 bg-white p-4">
        <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
          <div className="flex flex-wrap gap-2">
            <button type="button" onClick={() => setViewMode('catalog')} className={segmentClass(viewMode === 'catalog')}>{t('catalogMenu.modes.catalog')}</button>
            <button type="button" onClick={() => hasRestaurant && setViewMode('menu')} disabled={!hasRestaurant} className={segmentClass(viewMode === 'menu', !hasRestaurant)}>{t('catalogMenu.modes.menu')}</button>
          </div>

          <div className="flex flex-1 flex-col gap-2 sm:flex-row lg:max-w-3xl">
            <label className="relative min-w-0 flex-1">
              <span className="sr-only">{t('catalogMenu.search.label')}</span>
              <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
              <input
                value={search}
                onChange={(event) => setSearch(event.target.value)}
                placeholder={t('catalogMenu.search.placeholder')}
                className="w-full rounded-xl border border-slate-200 bg-slate-50 py-2.5 pl-9 pr-3 text-sm outline-none transition-colors focus:border-blue-500 focus:bg-white"
              />
            </label>
            <div className="relative">
              <button type="button" onClick={() => setCreateMenuOpen((value) => !value)} className="inline-flex h-10 items-center gap-2 rounded-xl bg-slate-900 px-3 text-sm font-semibold text-white hover:bg-slate-700">
                <Plus className="h-4 w-4" />
                {t('catalogMenu.actions.new')}
              </button>
              {isCreateMenuOpen ? (
                <div className="absolute right-0 z-20 mt-2 w-64 rounded-xl border border-slate-200 bg-white p-1 shadow-xl">
                  <MenuButton onClick={() => { setDialog({ kind: 'catalog-folder-create' }); setCreateMenuOpen(false); }}>{t('catalogMenu.actions.newFolder')}</MenuButton>
                  <MenuButton onClick={() => { setDialog({ kind: 'catalog-item-create' }); setCreateMenuOpen(false); }}>{t('catalogMenu.actions.newItem')}</MenuButton>
                  <MenuButton onClick={() => { setDialog({ kind: 'catalog-tag-create' }); setCreateMenuOpen(false); }}>{t('catalogMenu.actions.newTag')}</MenuButton>
                  <MenuButton onClick={() => { setDialog({ kind: 'item-tag-assign' }); setCreateMenuOpen(false); }}>{t('catalogMenu.actions.assignTag')}</MenuButton>
                  <MenuButton disabled={!hasRestaurant} onClick={() => { setDialog({ kind: 'menu-category-create' }); setCreateMenuOpen(false); }}>{t('catalogMenu.actions.newMenuCategory')}</MenuButton>
                  <MenuButton disabled={!hasRestaurant} onClick={() => { setDialog({ kind: 'menu-item-create' }); setCreateMenuOpen(false); }}>{t('catalogMenu.actions.newMenuItem')}</MenuButton>
                </div>
              ) : null}
            </div>
            <button type="button" onClick={() => void reload()} className="inline-flex h-10 items-center justify-center gap-2 rounded-xl border border-slate-200 px-3 text-sm font-semibold text-slate-700 hover:bg-slate-50">
              <RefreshCw className="h-4 w-4" />
              {t('ui.retry')}
            </button>
          </div>
        </div>
        {!hasRestaurant ? <p className="mt-3 text-xs leading-5 text-slate-500">{t('catalogMenu.restaurantDisabledHint')}</p> : null}
      </div>

      {status === 'blocked' ? <EmptyState title={t('catalog.blockedTitle')} description={t('catalog.blockedDescription')} /> : null}
      {error ? <SafeErrorBanner error={error} /> : null}

      {status !== 'blocked' ? (
        <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_380px]">
          <section className="min-h-[520px] rounded-2xl border border-slate-200 bg-white p-4">
            {viewMode === 'catalog' ? (
              <CatalogTree
                folders={folders}
                items={visibleItems}
                saleByCatalog={saleByCatalog}
                selected={selection}
                onSelect={selectNode}
                onContextMenu={setContextMenu}
              />
            ) : (
              <MenuTree
                categories={menuCategories}
                items={visibleMenuItems}
                catalogItems={itemById}
                selected={selection}
                onSelect={selectNode}
                onContextMenu={setContextMenu}
              />
            )}
          </section>

          <DetailPanel
            selection={activeSelection}
            tags={tags}
            folders={folders}
            menuCategories={menuCategories}
            saleByCatalog={saleByCatalog}
            catalogItems={itemById}
            hasRestaurant={hasRestaurant}
            loading={loading}
            onEdit={openEditDialog}
            onArchive={archiveSelected}
            onSell={(item) => setDialog({ kind: 'menu-item-create', catalogItem: item })}
          />
        </div>
      ) : null}

      {contextMenu ? (
        <div className="fixed z-50 w-56 rounded-xl border border-slate-200 bg-white p-1 shadow-2xl" style={{ left: contextMenu.x, top: contextMenu.y }} onClick={(event) => event.stopPropagation()}>
          <MenuButton onClick={() => selectNode(contextMenu.selection)}>{t('catalogMenu.context.open')}</MenuButton>
          <MenuButton onClick={() => openEditDialog(contextMenu.selection)}>{t('catalog.shared.edit')}</MenuButton>
          <MenuButton disabled>{t('catalogMenu.context.futureActions')}</MenuButton>
        </div>
      ) : null}

      {dialog ? (
        <CatalogMenuDialog
          dialog={dialog}
          folders={folders}
          items={items}
          tags={tags}
          menuCategories={menuCategories}
          restaurantId={restaurantId}
          restaurantCurrency={restaurantCurrency}
          loading={loading}
          itemTagSuccess={itemTagSuccess}
          onClose={() => setDialog(null)}
          onCreateFolder={(values) => mutate(async () => { await createCatalogFolder(buildCreateCatalogFolderPayload(values)); })}
          onUpdateFolder={(id, values) => mutate(async () => { await updateCatalogFolder(id, values); })}
          onCreateItem={(values) => mutate(async () => { await createCatalogItem(buildCreateCatalogItemPayload(values)); })}
          onUpdateItem={(id, values) => mutate(async () => { await updateCatalogItem(id, values); })}
          onCreateTag={(values) => mutate(async () => { await createCatalogTag(buildCreateCatalogTagPayload(values)); })}
          onUpdateTag={(id, values) => mutate(async () => { await updateCatalogTag(id, values); })}
          onAssignTag={(values) => mutate(async () => { await assignCatalogItemTag(values); setItemTagSuccess(true); })}
          onCreateMenuCategory={(values) => mutate(async () => { await createMenuCategory({ ...values, restaurant_id: restaurantId }); })}
          onUpdateMenuCategory={(id, values) => mutate(async () => { await updateMenuCategory(id, values); })}
          onCreateMenuItem={(values) => mutate(async () => { await createMenuItem({ restaurant_id: restaurantId, ...buildCreateMenuItemPayload(values) }); })}
          onUpdateMenuItem={(id, values) => mutate(async () => { await updateMenuItem(id, normalizeMenuItemValues(values)); })}
        />
      ) : null}
    </section>
  );
}

function CatalogTree({ folders, items, saleByCatalog, selected, onSelect, onContextMenu }: {
  folders: CatalogFolder[];
  items: CatalogItem[];
  saleByCatalog: Map<string, MenuItem>;
  selected: NodeSelection | null;
  onSelect: (selection: NodeSelection) => void;
  onContextMenu: (state: ContextMenuState) => void;
}) {
  const { t } = useI18n();
  const rootFolders = folders.filter((folder) => !folder.parent_id).sort(bySortOrder);
  const orphanItems = items.filter((item) => !item.folder_id);

  return (
    <div className="space-y-2">
      {rootFolders.map((folder) => (
        <FolderNode key={folder.id} folder={folder} folders={folders} items={items} level={0} saleByCatalog={saleByCatalog} selected={selected} onSelect={onSelect} onContextMenu={onContextMenu} />
      ))}
      {orphanItems.length > 0 ? <p className="px-2 pt-3 text-xs font-semibold text-slate-500">{t('catalog.shared.noFolder')}</p> : null}
      {orphanItems.map((item) => <CatalogItemRow key={item.id} item={item} level={0} isForSale={saleByCatalog.has(item.id)} selected={selected} onSelect={onSelect} onContextMenu={onContextMenu} />)}
      {folders.length === 0 && items.length === 0 ? <EmptyState title={t('catalogMenu.empty.catalogTitle')} description={t('catalog.items.empty')} /> : null}
    </div>
  );
}

function FolderNode(props: {
  folder: CatalogFolder;
  folders: CatalogFolder[];
  items: CatalogItem[];
  level: number;
  saleByCatalog: Map<string, MenuItem>;
  selected: NodeSelection | null;
  onSelect: (selection: NodeSelection) => void;
  onContextMenu: (state: ContextMenuState) => void;
}) {
  const childFolders = props.folders.filter((folder) => folder.parent_id === props.folder.id).sort(bySortOrder);
  const childItems = props.items.filter((item) => item.folder_id === props.folder.id);
  return (
    <div className="space-y-1">
      <TreeButton
        level={props.level}
        active={props.selected?.kind === 'catalog-folder' && props.selected.id === props.folder.id}
        label={props.folder.name}
        meta={props.folder.status}
        onClick={() => props.onSelect({ kind: 'catalog-folder', id: props.folder.id })}
        onContextMenu={(event) => props.onContextMenu(menuState(event, { kind: 'catalog-folder', id: props.folder.id }))}
      />
      {childFolders.map((folder) => <FolderNode key={folder.id} {...props} folder={folder} level={props.level + 1} />)}
      {childItems.map((item) => <CatalogItemRow key={item.id} item={item} level={props.level + 1} isForSale={props.saleByCatalog.has(item.id)} selected={props.selected} onSelect={props.onSelect} onContextMenu={props.onContextMenu} />)}
    </div>
  );
}

function CatalogItemRow({ item, level, isForSale, selected, onSelect, onContextMenu }: {
  item: CatalogItem;
  level: number;
  isForSale: boolean;
  selected: NodeSelection | null;
  onSelect: (selection: NodeSelection) => void;
  onContextMenu: (state: ContextMenuState) => void;
}) {
  const { t } = useI18n();
  return (
    <TreeButton
      level={level}
      active={selected?.kind === 'catalog-item' && selected.id === item.id}
      label={item.name}
      meta={`${item.sku} ${t(`catalog.kinds.${item.kind}`)}`}
      badge={isForSale ? t('catalogMenu.saleState.forSale') : undefined}
      onClick={() => onSelect({ kind: 'catalog-item', id: item.id })}
      onContextMenu={(event) => onContextMenu(menuState(event, { kind: 'catalog-item', id: item.id }))}
    />
  );
}

function MenuTree({ categories, items, catalogItems, selected, onSelect, onContextMenu }: {
  categories: Category[];
  items: MenuItem[];
  catalogItems: Map<string, CatalogItem>;
  selected: NodeSelection | null;
  onSelect: (selection: NodeSelection) => void;
  onContextMenu: (state: ContextMenuState) => void;
}) {
  const { t } = useI18n();
  const sortedCategories = [...categories].sort(bySortOrder);
  const uncategorized = items.filter((item) => !item.category_id);
  return (
    <div className="space-y-2">
      {sortedCategories.map((category) => {
        const categoryItems = items.filter((item) => item.category_id === category.id);
        return (
          <div key={category.id} className="space-y-1">
            <TreeButton
              level={0}
              active={selected?.kind === 'menu-category' && selected.id === category.id}
              label={category.name}
              meta={category.status}
              onClick={() => onSelect({ kind: 'menu-category', id: category.id })}
              onContextMenu={(event) => onContextMenu(menuState(event, { kind: 'menu-category', id: category.id }))}
            />
            {categoryItems.map((item) => <MenuItemRow key={item.id} item={item} catalogItem={catalogItems.get(item.catalog_item_id)} selected={selected} onSelect={onSelect} onContextMenu={onContextMenu} />)}
          </div>
        );
      })}
      {uncategorized.length > 0 ? <p className="px-2 pt-3 text-xs font-semibold text-slate-500">{t('catalogMenu.menuTree.noCategory')}</p> : null}
      {uncategorized.map((item) => <MenuItemRow key={item.id} item={item} catalogItem={catalogItems.get(item.catalog_item_id)} selected={selected} onSelect={onSelect} onContextMenu={onContextMenu} />)}
      {categories.length === 0 && items.length === 0 ? <EmptyState title={t('catalogMenu.empty.menuTitle')} description={t('menu.items.empty')} /> : null}
    </div>
  );
}

function MenuItemRow({ item, catalogItem, selected, onSelect, onContextMenu }: {
  item: MenuItem;
  catalogItem?: CatalogItem;
  selected: NodeSelection | null;
  onSelect: (selection: NodeSelection) => void;
  onContextMenu: (state: ContextMenuState) => void;
}) {
  return (
    <TreeButton
      level={1}
      active={selected?.kind === 'menu-item' && selected.id === item.id}
      label={item.name}
      meta={`${catalogItem?.name ?? item.catalog_item_id} ${item.price} ${item.currency}`}
      badge={item.runtime_status}
      onClick={() => onSelect({ kind: 'menu-item', id: item.id })}
      onContextMenu={(event) => onContextMenu(menuState(event, { kind: 'menu-item', id: item.id }))}
    />
  );
}

function TreeButton({ level, active, label, meta, badge, onClick, onContextMenu }: {
  level: number;
  active: boolean;
  label: string;
  meta: string;
  badge?: string;
  onClick: () => void;
  onContextMenu: (event: MouseEvent) => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      onContextMenu={onContextMenu}
      className={[
        'flex min-h-12 w-full items-center justify-between gap-3 rounded-xl border px-3 py-2 text-left transition-colors',
        active ? 'border-blue-200 bg-blue-50' : 'border-transparent hover:border-slate-200 hover:bg-slate-50',
      ].join(' ')}
      style={{ paddingLeft: 12 + level * 20 }}
    >
      <span className="min-w-0">
        <span className="block truncate text-sm font-semibold text-slate-900">{label}</span>
        <span className="block truncate text-xs text-slate-500">{meta}</span>
      </span>
      <span className="flex shrink-0 items-center gap-2">
        {badge ? <span className="rounded-full border border-emerald-200 bg-emerald-50 px-2 py-0.5 text-[11px] font-semibold text-emerald-700">{badge}</span> : null}
        <MoreVertical className="h-4 w-4 text-slate-400" />
      </span>
    </button>
  );
}

function DetailPanel({ selection, tags, folders, menuCategories, saleByCatalog, catalogItems, hasRestaurant, loading, onEdit, onArchive, onSell }: {
  selection: ResolvedSelection | null;
  tags: CatalogTag[];
  folders: CatalogFolder[];
  menuCategories: Category[];
  saleByCatalog: Map<string, MenuItem>;
  catalogItems: Map<string, CatalogItem>;
  hasRestaurant: boolean;
  loading: boolean;
  onEdit: (selection: NodeSelection) => void;
  onArchive: (selection: NodeSelection) => void;
  onSell: (item: CatalogItem) => void;
}) {
  const { t } = useI18n();
  if (!selection) {
    return (
      <aside className="rounded-2xl border border-slate-200 bg-white p-5">
        <EmptyState title={t('catalogMenu.detail.emptyTitle')} description={t('catalogMenu.detail.emptyDescription')} />
      </aside>
    );
  }

  const target = selection.selection;
  return (
    <aside className="space-y-4 rounded-2xl border border-slate-200 bg-white p-5">
      <div className="flex items-start justify-between gap-3">
        <div>
          <p className="text-xs font-semibold uppercase tracking-wide text-slate-500">{t(`catalogMenu.nodeKinds.${target.kind}`)}</p>
          <h3 className="mt-1 text-base font-semibold text-slate-950">{selection.title}</h3>
        </div>
        <div className="flex gap-2">
          <button type="button" onClick={() => onEdit(target)} className="rounded-lg border border-slate-200 px-2.5 py-1.5 text-xs font-semibold text-slate-700 hover:bg-slate-50" disabled={loading}>{t('catalog.shared.edit')}</button>
          <button type="button" onClick={() => onArchive(target)} className="rounded-lg border border-rose-200 px-2.5 py-1.5 text-xs font-semibold text-rose-700 hover:bg-rose-50" disabled={loading}>{t('catalog.shared.archive')}</button>
        </div>
      </div>

      {selection.kind === 'catalog-folder' ? (
        <dl className="grid gap-3 text-sm">
          <DetailLine label={t('catalog.folders.fields.parent')} value={folders.find((folder) => folder.id === selection.value.parent_id)?.name ?? t('catalog.folders.fields.root')} />
          <DetailLine label={t('catalog.folders.fields.sortOrder')} value={String(selection.value.sort_order)} />
          <DetailLine label={t('catalog.folders.fields.status')} value={t(`catalog.statuses.${selection.value.status}`)} />
        </dl>
      ) : null}

      {selection.kind === 'catalog-item' ? (
        <CatalogItemDetail item={selection.value} sale={saleByCatalog.get(selection.value.id)} tags={tags} folders={folders} menuCategories={menuCategories} hasRestaurant={hasRestaurant} loading={loading} onSell={onSell} />
      ) : null}

      {selection.kind === 'menu-category' ? (
        <dl className="grid gap-3 text-sm">
          <DetailLine label={t('menu.categories.fields.sortOrder')} value={String(selection.value.sort_order)} />
          <DetailLine label={t('catalog.items.fields.status')} value={t(`catalog.statuses.${selection.value.status}`)} />
        </dl>
      ) : null}

      {selection.kind === 'menu-item' ? (
        <MenuItemDetail item={selection.value} catalogItem={catalogItems.get(selection.value.catalog_item_id)} tags={tags} menuCategories={menuCategories} />
      ) : null}
    </aside>
  );
}

function CatalogItemDetail({ item, sale, tags, folders, menuCategories, hasRestaurant, loading, onSell }: {
  item: CatalogItem;
  sale?: MenuItem;
  tags: CatalogTag[];
  folders: CatalogFolder[];
  menuCategories: Category[];
  hasRestaurant: boolean;
  loading: boolean;
  onSell: (item: CatalogItem) => void;
}) {
  const { t } = useI18n();
  return (
    <div className="space-y-4">
      <dl className="grid gap-3 text-sm">
        <DetailLine label={t('catalog.items.fields.sku')} value={item.sku} />
        <DetailLine label={t('catalog.items.fields.kind')} value={t(`catalog.kinds.${item.kind}`)} />
        <DetailLine label={t('catalog.items.fields.folder')} value={folders.find((folder) => folder.id === item.folder_id)?.name ?? t('catalog.shared.noFolder')} />
        <DetailLine label={t('catalog.items.fields.status')} value={t(`catalog.statuses.${item.status}`)} />
      </dl>
      {!hasRestaurant ? <p className="rounded-xl border border-slate-200 bg-slate-50 p-3 text-sm text-slate-600">{t('catalogMenu.detail.selectRestaurantForMenu')}</p> : null}
      {hasRestaurant && sale ? <MenuOverrides item={sale} tags={tags} categories={menuCategories} /> : null}
      {hasRestaurant && !sale ? (
        <button type="button" onClick={() => onSell(item)} className="w-full rounded-xl bg-slate-900 px-3 py-2.5 text-sm font-semibold text-white hover:bg-slate-700 disabled:opacity-50" disabled={loading || item.status === 'archived'}>{t('catalogMenu.actions.sell')}</button>
      ) : null}
    </div>
  );
}

function MenuItemDetail({ item, catalogItem, tags, menuCategories }: {
  item: MenuItem;
  catalogItem?: CatalogItem;
  tags: CatalogTag[];
  menuCategories: Category[];
}) {
  const { t } = useI18n();
  return (
    <div className="space-y-4">
      <dl className="grid gap-3 text-sm">
        <DetailLine label={t('menu.items.fields.catalogItem')} value={catalogItem?.name ?? item.catalog_item_id} />
        <DetailLine label={t('catalog.items.fields.status')} value={t(`catalog.statuses.${item.status}`)} />
      </dl>
      <MenuOverrides item={item} tags={tags} categories={menuCategories} />
    </div>
  );
}

function MenuOverrides({ item, tags, categories }: { item: MenuItem; tags: CatalogTag[]; categories: Category[] }) {
  const { t } = useI18n();
  return (
    <section className="space-y-3 rounded-xl border border-emerald-200 bg-emerald-50 p-3">
      <p className="text-sm font-semibold text-emerald-900">{t('catalogMenu.saleState.forSale')}</p>
      <dl className="grid gap-3 text-sm">
        <DetailLine label={t('menu.items.fields.name')} value={item.name} />
        <DetailLine label={t('menu.items.fields.price')} value={`${item.price} ${item.currency}`} />
        <DetailLine label={t('menu.items.fields.category')} value={categories.find((category) => category.id === item.category_id)?.name ?? item.category_id} />
        <DetailLine label={t('menu.items.fields.tag')} value={tags.find((tag) => tag.id === item.tag_id)?.name ?? item.tag_id} />
        <DetailLine label={t('menu.items.fields.taxProfile')} value={item.tax_profile_id} />
        <DetailLine label={t('menu.items.fields.availability')} value={item.availability_json} />
        <DetailLine label={t('menu.items.fields.runtimeStatus')} value={t(`menu.items.runtimeStatuses.${item.runtime_status}`)} />
      </dl>
    </section>
  );
}

function DetailLine({ label, value }: { label: string; value: string }) {
  return (
    <div className="grid gap-1">
      <dt className="text-xs font-semibold text-slate-500">{label}</dt>
      <dd className="break-words text-sm font-medium text-slate-900">{value || '-'}</dd>
    </div>
  );
}

function CatalogMenuDialog(props: {
  dialog: DialogState;
  folders: CatalogFolder[];
  items: CatalogItem[];
  tags: CatalogTag[];
  menuCategories: Category[];
  restaurantId: string;
  restaurantCurrency: string;
  loading: boolean;
  itemTagSuccess: boolean;
  onClose: () => void;
  onCreateFolder: (values: CatalogFolderFormValues) => Promise<void>;
  onUpdateFolder: (id: string, values: CatalogFolderFormValues) => Promise<void>;
  onCreateItem: (values: CatalogItemFormValues) => Promise<void>;
  onUpdateItem: (id: string, values: CatalogItemFormValues) => Promise<void>;
  onCreateTag: (values: CatalogTagFormValues) => Promise<void>;
  onUpdateTag: (id: string, values: CatalogTagFormValues) => Promise<void>;
  onAssignTag: (values: ItemTagCommandFormValues) => Promise<void>;
  onCreateMenuCategory: (values: MenuCategoryFormValues) => Promise<void>;
  onUpdateMenuCategory: (id: string, values: MenuCategoryFormValues) => Promise<void>;
  onCreateMenuItem: (values: MenuItemFormValues) => Promise<void>;
  onUpdateMenuItem: (id: string, values: MenuItemFormValues) => Promise<void>;
}) {
  const { t } = useI18n();
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/40 p-4 backdrop-blur-sm">
      <div className="max-h-[92dvh] w-full max-w-3xl overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-2xl">
        <div className="flex items-center justify-between border-b border-slate-200 bg-slate-50 px-5 py-4">
          <h3 className="text-sm font-semibold text-slate-950">{dialogTitle(props.dialog, t)}</h3>
          <button type="button" onClick={props.onClose} className="inline-flex h-9 w-9 items-center justify-center rounded-xl border border-slate-200 bg-white text-slate-500 hover:bg-slate-100" aria-label={t('catalog.shared.cancel')}>
            <X className="h-4 w-4" />
          </button>
        </div>
        <div className="max-h-[calc(92dvh-73px)] overflow-y-auto p-5">
          <DialogBody {...props} />
        </div>
      </div>
    </div>
  );
}

function DialogBody(props: Parameters<typeof CatalogMenuDialog>[0]) {
  const dialog = props.dialog;
  if (dialog.kind === 'catalog-folder-create') {
    return <CatalogFolderForm initialValues={defaultCatalogFolderValues} folders={props.folders} loading={props.loading} submitLabelKey="catalog.folders.actions.create" onSubmit={props.onCreateFolder} />;
  }
  if (dialog.kind === 'catalog-folder-edit') {
    return <CatalogFolderForm initialValues={toCatalogFolderValues(dialog.folder)} folders={props.folders.filter((folder) => folder.id !== dialog.folder.id)} loading={props.loading} submitLabelKey="catalog.shared.save" onSubmit={(values) => props.onUpdateFolder(dialog.folder.id, values)} />;
  }
  if (dialog.kind === 'catalog-item-create') {
    return <CatalogItemForm initialValues={defaultCatalogItemValues} folders={props.folders} loading={props.loading} submitLabelKey="catalog.items.actions.create" onSubmit={props.onCreateItem} />;
  }
  if (dialog.kind === 'catalog-item-edit') {
    return <CatalogItemForm initialValues={toCatalogItemValues(dialog.item)} folders={props.folders} loading={props.loading} submitLabelKey="catalog.shared.save" onSubmit={(values) => props.onUpdateItem(dialog.item.id, values)} />;
  }
  if (dialog.kind === 'catalog-tag-create') {
    return <CatalogTagForm initialValues={defaultCatalogTagValues} loading={props.loading} submitLabelKey="catalog.tags.actions.create" onSubmit={props.onCreateTag} />;
  }
  if (dialog.kind === 'catalog-tag-edit') {
    return <CatalogTagForm initialValues={toCatalogTagValues(dialog.tag)} loading={props.loading} submitLabelKey="catalog.shared.save" onSubmit={(values) => props.onUpdateTag(dialog.tag.id, values)} />;
  }
  if (dialog.kind === 'item-tag-assign') {
    return <ItemTagForm items={props.items} tags={props.tags} loading={props.loading} success={props.itemTagSuccess} onSubmit={props.onAssignTag} />;
  }
  if (dialog.kind === 'menu-category-create') {
    return <MenuCategoryForm initialValues={{ ...defaultMenuCategoryValues, restaurant_id: props.restaurantId }} loading={props.loading} submitLabelKey="menu.categories.actions.create" onSubmit={props.onCreateMenuCategory} />;
  }
  if (dialog.kind === 'menu-category-edit') {
    return <MenuCategoryForm initialValues={{ restaurant_id: dialog.category.restaurant_id, name: dialog.category.name, sort_order: dialog.category.sort_order }} loading={props.loading} submitLabelKey="catalog.shared.save" onSubmit={(values) => props.onUpdateMenuCategory(dialog.category.id, values)} />;
  }
  if (dialog.kind === 'menu-item-create') {
    const initial = {
      ...defaultMenuItemValues,
      catalog_item_id: dialog.catalogItem?.id ?? '',
      name: dialog.catalogItem?.name ?? '',
      currency: props.restaurantCurrency,
    };
    return <MenuItemForm initialValues={initial} items={props.items} categories={props.menuCategories} tags={props.tags} loading={props.loading} submitLabelKey="menu.items.actions.create" onSubmit={props.onCreateMenuItem} />;
  }
  return <MenuItemForm initialValues={toMenuItemValues(dialog.item)} items={props.items} categories={props.menuCategories} tags={props.tags} loading={props.loading} submitLabelKey="catalog.shared.save" onSubmit={(values) => props.onUpdateMenuItem(dialog.item.id, values)} />;
}

function CatalogFolderForm({ initialValues, folders, loading, submitLabelKey, onSubmit }: {
  initialValues: CatalogFolderFormValues;
  folders: CatalogFolder[];
  loading: boolean;
  submitLabelKey: string;
  onSubmit: (values: CatalogFolderFormValues) => Promise<void>;
}) {
  const { t } = useI18n();
  const [values, setValues] = useState(initialValues);
  return (
    <BaseForm onSubmit={() => onSubmit({ ...values, name: values.name.trim() })}>
      <Field label={t('catalog.folders.fields.name')}><input value={values.name} onChange={(event) => setValues({ ...values, name: event.target.value })} className={inputClass()} disabled={loading} /></Field>
      <Field label={t('catalog.folders.fields.parent')}><select value={values.parent_id} onChange={(event) => setValues({ ...values, parent_id: event.target.value })} className={inputClass()} disabled={loading}><option value="">{t('catalog.folders.fields.root')}</option>{folders.map((folder) => <option key={folder.id} value={folder.id}>{folder.name}</option>)}</select></Field>
      <Field label={t('catalog.folders.fields.sortOrder')}><input type="number" value={values.sort_order} onChange={(event) => setValues({ ...values, sort_order: Number(event.target.value) || 0 })} className={inputClass()} disabled={loading} /></Field>
      <Field label={t('catalog.folders.fields.status')}><StatusSelect value={values.status} onChange={(status) => setValues({ ...values, status })} disabled={loading} /></Field>
      <SubmitButton disabled={loading || !values.name.trim()}>{t(submitLabelKey)}</SubmitButton>
    </BaseForm>
  );
}

function CatalogItemForm({ initialValues, folders, loading, submitLabelKey, onSubmit }: {
  initialValues: CatalogItemFormValues;
  folders: CatalogFolder[];
  loading: boolean;
  submitLabelKey: string;
  onSubmit: (values: CatalogItemFormValues) => Promise<void>;
}) {
  const { t } = useI18n();
  const [values, setValues] = useState(initialValues);
  return (
    <BaseForm onSubmit={() => onSubmit({ ...values, name: values.name.trim(), sku: values.sku.trim(), base_unit: values.base_unit.trim() })}>
      <div className="grid gap-4 md:grid-cols-2">
        <Field label={t('catalog.items.fields.name')}><input value={values.name} onChange={(event) => setValues({ ...values, name: event.target.value })} className={inputClass()} disabled={loading} /></Field>
        <Field label={t('catalog.items.fields.sku')}><input value={values.sku} onChange={(event) => setValues({ ...values, sku: event.target.value })} className={inputClass()} disabled={loading} /></Field>
      </div>
      <div className="grid gap-4 md:grid-cols-2">
        <Field label={t('catalog.items.fields.kind')}><select value={values.kind} onChange={(event) => setValues({ ...values, kind: event.target.value as CatalogKind })} className={inputClass()} disabled={loading}>{catalogKinds.map((kind) => <option key={kind} value={kind}>{t(`catalog.kinds.${kind}`)}</option>)}</select></Field>
        <Field label={t('catalog.items.fields.status')}><StatusSelect value={values.status} onChange={(status) => setValues({ ...values, status })} disabled={loading} /></Field>
      </div>
      <div className="grid gap-4 md:grid-cols-2">
        <Field label={t('catalog.items.fields.baseUnit')}><input value={values.base_unit} onChange={(event) => setValues({ ...values, base_unit: event.target.value })} className={inputClass()} disabled={loading} /></Field>
        <Field label={t('catalog.items.fields.folder')}><select value={values.folder_id} onChange={(event) => setValues({ ...values, folder_id: event.target.value })} className={inputClass()} disabled={loading}><option value="">{t('catalog.shared.noFolder')}</option>{folders.map((folder) => <option key={folder.id} value={folder.id}>{folder.name}</option>)}</select></Field>
      </div>
      <div className="grid gap-4 md:grid-cols-2">
        <Field label={t('catalog.items.fields.kitchenType')}><input value={values.kitchen_type} onChange={(event) => setValues({ ...values, kitchen_type: event.target.value })} className={inputClass()} disabled={loading} /></Field>
        <Field label={t('catalog.items.fields.accountingCategory')}><input value={values.accounting_category} onChange={(event) => setValues({ ...values, accounting_category: event.target.value })} className={inputClass()} disabled={loading} /></Field>
      </div>
      {values.kind === 'service' ? (
        <label className="flex items-center gap-2 rounded-xl border border-slate-200 bg-slate-50 p-3 text-sm text-slate-700">
          <input type="checkbox" checked={values.qr_confirmation_enabled} onChange={(event) => setValues({ ...values, qr_confirmation_enabled: event.target.checked, validity_mode: event.target.checked ? values.validity_mode : '', validity_expires_at: event.target.checked ? values.validity_expires_at : '' })} disabled={loading} />
          {t('catalog.items.fields.qrConfirmationEnabled')}
        </label>
      ) : null}
      {values.kind === 'service' && values.qr_confirmation_enabled ? (
        <div className="grid gap-4 md:grid-cols-2">
          <Field label={t('catalog.items.fields.validityMode')}><select value={values.validity_mode} onChange={(event) => setValues({ ...values, validity_mode: event.target.value as typeof values.validity_mode })} className={inputClass()} disabled={loading}><option value="">-</option><option value="cash_session">{t('catalog.items.fields.validityModes.cash_session')}</option><option value="business_date">{t('catalog.items.fields.validityModes.business_date')}</option><option value="absolute_date">{t('catalog.items.fields.validityModes.absolute_date')}</option></select></Field>
          {values.validity_mode === 'absolute_date' ? <Field label={t('catalog.items.fields.validityExpiresAt')}><input type="datetime-local" value={values.validity_expires_at} onChange={(event) => setValues({ ...values, validity_expires_at: event.target.value })} className={inputClass()} disabled={loading} /></Field> : null}
        </div>
      ) : null}
      <SubmitButton disabled={loading || !values.name.trim() || !values.sku.trim() || !values.base_unit.trim() || (values.qr_confirmation_enabled && !values.validity_mode) || (values.qr_confirmation_enabled && values.validity_mode === 'absolute_date' && !values.validity_expires_at)}>{t(submitLabelKey)}</SubmitButton>
    </BaseForm>
  );
}

function CatalogTagForm({ initialValues, loading, submitLabelKey, onSubmit }: {
  initialValues: CatalogTagFormValues;
  loading: boolean;
  submitLabelKey: string;
  onSubmit: (values: CatalogTagFormValues) => Promise<void>;
}) {
  const { t } = useI18n();
  const [values, setValues] = useState(initialValues);
  return (
    <BaseForm onSubmit={() => onSubmit({ ...values, name: values.name.trim(), code: values.code.trim() })}>
      <Field label={t('catalog.tags.fields.name')}><input value={values.name} onChange={(event) => setValues({ ...values, name: event.target.value })} className={inputClass()} disabled={loading} /></Field>
      <Field label={t('catalog.tags.fields.code')}><input value={values.code} onChange={(event) => setValues({ ...values, code: event.target.value })} className={inputClass()} disabled={loading} /></Field>
      <Field label={t('catalog.tags.fields.status')}><StatusSelect value={values.status} onChange={(status) => setValues({ ...values, status })} disabled={loading} /></Field>
      <SubmitButton disabled={loading || !values.name.trim() || !values.code.trim()}>{t(submitLabelKey)}</SubmitButton>
    </BaseForm>
  );
}

function ItemTagForm({ items, tags, loading, success, onSubmit }: {
  items: CatalogItem[];
  tags: CatalogTag[];
  loading: boolean;
  success: boolean;
  onSubmit: (values: ItemTagCommandFormValues) => Promise<void>;
}) {
  const { t } = useI18n();
  const [values, setValues] = useState(defaultItemTagCommandValues);
  return (
    <BaseForm onSubmit={() => onSubmit(values)}>
      <Field label={t('catalog.itemTags.fields.item')}><select value={values.catalog_item_id} onChange={(event) => setValues({ ...values, catalog_item_id: event.target.value })} className={inputClass()} disabled={loading}><option value="">{t('catalog.itemTags.fields.selectItem')}</option>{items.filter((item) => item.status !== 'archived').map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}</select></Field>
      <Field label={t('catalog.itemTags.fields.tag')}><select value={values.tag_id} onChange={(event) => setValues({ ...values, tag_id: event.target.value })} className={inputClass()} disabled={loading}><option value="">{t('catalog.itemTags.fields.selectTag')}</option>{tags.filter((tag) => tag.status !== 'archived').map((tag) => <option key={tag.id} value={tag.id}>{tag.name}</option>)}</select></Field>
      {success ? <p className="rounded-xl border border-emerald-200 bg-emerald-50 p-3 text-sm text-emerald-800">{t('catalog.itemTags.success')}</p> : null}
      <SubmitButton disabled={loading || !values.catalog_item_id || !values.tag_id}>{t('catalog.itemTags.actions.assign')}</SubmitButton>
    </BaseForm>
  );
}

function MenuCategoryForm({ initialValues, loading, submitLabelKey, onSubmit }: {
  initialValues: MenuCategoryFormValues;
  loading: boolean;
  submitLabelKey: string;
  onSubmit: (values: MenuCategoryFormValues) => Promise<void>;
}) {
  const { t } = useI18n();
  const [values, setValues] = useState(initialValues);
  return (
    <BaseForm onSubmit={() => onSubmit({ ...values, name: values.name.trim() })}>
      <Field label={t('menu.categories.fields.name')}><input value={values.name} onChange={(event) => setValues({ ...values, name: event.target.value })} className={inputClass()} disabled={loading} /></Field>
      <Field label={t('menu.categories.fields.sortOrder')}><input type="number" value={values.sort_order} onChange={(event) => setValues({ ...values, sort_order: Number(event.target.value) || 0 })} className={inputClass()} disabled={loading} /></Field>
      <SubmitButton disabled={loading || !values.name.trim()}>{t(submitLabelKey)}</SubmitButton>
    </BaseForm>
  );
}

function MenuItemForm({ initialValues, items, categories, tags, loading, submitLabelKey, onSubmit }: {
  initialValues: MenuItemFormValues;
  items: CatalogItem[];
  categories: Category[];
  tags: CatalogTag[];
  loading: boolean;
  submitLabelKey: string;
  onSubmit: (values: MenuItemFormValues) => Promise<void>;
}) {
  const { t } = useI18n();
  const [values, setValues] = useState(initialValues);
  return (
    <BaseForm onSubmit={() => onSubmit(normalizeMenuItemValues(values))}>
      <div className="grid gap-4 md:grid-cols-2">
        <Field label={t('menu.items.fields.catalogItem')}><select value={values.catalog_item_id} onChange={(event) => setValues({ ...values, catalog_item_id: event.target.value })} className={inputClass()} disabled={loading}><option value="">{t('menu.items.fields.selectCatalogItem')}</option>{items.filter((item) => item.status !== 'archived').map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}</select></Field>
        <Field label={t('menu.items.fields.name')}><input value={values.name} onChange={(event) => setValues({ ...values, name: event.target.value })} className={inputClass()} disabled={loading} /></Field>
      </div>
      <div className="grid gap-4 md:grid-cols-3">
        <Field label={t('menu.items.fields.category')}><select value={values.category_id} onChange={(event) => setValues({ ...values, category_id: event.target.value })} className={inputClass()} disabled={loading}><option value="">{t('catalogMenu.menuTree.noCategory')}</option>{categories.filter((category) => category.status !== 'archived').map((category) => <option key={category.id} value={category.id}>{category.name}</option>)}</select></Field>
        <Field label={t('menu.items.fields.tag')}><select value={values.tag_id} onChange={(event) => setValues({ ...values, tag_id: event.target.value })} className={inputClass()} disabled={loading}><option value="">-</option>{tags.filter((tag) => tag.status !== 'archived').map((tag) => <option key={tag.id} value={tag.id}>{tag.name}</option>)}</select></Field>
        <Field label={t('menu.items.fields.taxProfile')}><input value={values.tax_profile_id} onChange={(event) => setValues({ ...values, tax_profile_id: event.target.value })} className={inputClass()} disabled={loading} /></Field>
      </div>
      <div className="grid gap-4 md:grid-cols-4">
        <Field label={t('menu.items.fields.price')}><input type="number" min="0" value={values.price} onChange={(event) => setValues({ ...values, price: Number(event.target.value) })} className={inputClass()} disabled={loading} /></Field>
        <Field label={t('menu.items.fields.currency')}><input value={values.currency} onChange={(event) => setValues({ ...values, currency: event.target.value })} className={inputClass()} disabled={loading} /></Field>
        <Field label={t('menu.items.fields.status')}><StatusSelect value={values.status} onChange={(status) => setValues({ ...values, status })} disabled={loading} /></Field>
        <Field label={t('menu.items.fields.runtimeStatus')}><select value={values.runtime_status} onChange={(event) => setValues({ ...values, runtime_status: event.target.value })} className={inputClass()} disabled={loading}>{runtimeStatuses.map((status) => <option key={status} value={status}>{t(`menu.items.runtimeStatuses.${status}`)}</option>)}</select></Field>
      </div>
      <Field label={t('menu.items.fields.station')}><input value={values.station_routing_key} onChange={(event) => setValues({ ...values, station_routing_key: event.target.value })} className={inputClass()} disabled={loading} /></Field>
      <Field label={t('menu.items.fields.availability')}><textarea value={values.availability_json} onChange={(event) => setValues({ ...values, availability_json: event.target.value })} className={`${inputClass()} min-h-24 font-mono`} disabled={loading} /></Field>
      <SubmitButton disabled={loading || !values.catalog_item_id || !values.name.trim()}>{t(submitLabelKey)}</SubmitButton>
    </BaseForm>
  );
}

function StatusSelect({ value, onChange, disabled }: { value: LifecycleStatus; onChange: (status: LifecycleStatus) => void; disabled: boolean }) {
  const { t } = useI18n();
  return <select value={value} onChange={(event) => onChange(event.target.value as LifecycleStatus)} className={inputClass()} disabled={disabled}>{lifecycleStatuses.map((status) => <option key={status} value={status}>{t(`catalog.statuses.${status}`)}</option>)}</select>;
}

function Field({ label, children }: { label: string; children: ReactNode }) {
  return <label className="block"><span className="mb-1.5 block text-xs font-semibold text-slate-600">{label}</span>{children}</label>;
}

function BaseForm({ children, onSubmit }: { children: ReactNode; onSubmit: () => Promise<void> }) {
  const submit = (event: FormEvent) => {
    event.preventDefault();
    void onSubmit();
  };
  return <form className="space-y-4" onSubmit={submit}>{children}</form>;
}

function SubmitButton({ children, disabled }: { children: ReactNode; disabled: boolean }) {
  return <button type="submit" disabled={disabled} className="rounded-xl bg-slate-900 px-3 py-2.5 text-sm font-semibold text-white hover:bg-slate-700 disabled:cursor-not-allowed disabled:opacity-50">{children}</button>;
}

function MenuButton({ children, disabled, onClick }: { children: ReactNode; disabled?: boolean; onClick?: () => void }) {
  return <button type="button" disabled={disabled} onClick={onClick} className="block w-full rounded-lg px-3 py-2 text-left text-sm font-medium text-slate-700 hover:bg-slate-50 disabled:cursor-not-allowed disabled:text-slate-400">{children}</button>;
}

type ResolvedSelection =
  | { kind: 'catalog-folder'; selection: NodeSelection; title: string; value: CatalogFolder }
  | { kind: 'catalog-item'; selection: NodeSelection; title: string; value: CatalogItem }
  | { kind: 'menu-category'; selection: NodeSelection; title: string; value: Category }
  | { kind: 'menu-item'; selection: NodeSelection; title: string; value: MenuItem };

function resolveSelection(
  selection: NodeSelection | null,
  folders: Map<string, CatalogFolder>,
  items: Map<string, CatalogItem>,
  categories: Map<string, Category>,
  menuItems: Map<string, MenuItem>,
): ResolvedSelection | null {
  if (!selection) return null;
  if (selection.kind === 'catalog-folder') {
    const value = folders.get(selection.id);
    return value ? { kind: selection.kind, selection, title: value.name, value } : null;
  }
  if (selection.kind === 'catalog-item') {
    const value = items.get(selection.id);
    return value ? { kind: selection.kind, selection, title: value.name, value } : null;
  }
  if (selection.kind === 'menu-category') {
    const value = categories.get(selection.id);
    return value ? { kind: selection.kind, selection, title: value.name, value } : null;
  }
  const value = menuItems.get(selection.id);
  return value ? { kind: selection.kind, selection, title: value.name, value } : null;
}

function filterCatalogItems(items: CatalogItem[], search: string) {
  const query = search.trim().toLowerCase();
  if (!query) return items;
  return items.filter((item) => `${item.name} ${item.sku} ${item.kind}`.toLowerCase().includes(query));
}

function filterMenuItems(items: MenuItem[], search: string, catalogItems: Map<string, CatalogItem>) {
  const query = search.trim().toLowerCase();
  if (!query) return items;
  return items.filter((item) => `${item.name} ${catalogItems.get(item.catalog_item_id)?.name ?? ''} ${item.category_id}`.toLowerCase().includes(query));
}

function bySortOrder<T extends { sort_order: number; id: string }>(a: T, b: T) {
  return a.sort_order - b.sort_order || a.id.localeCompare(b.id);
}

function menuState(event: MouseEvent, selection: NodeSelection): ContextMenuState {
  event.preventDefault();
  return { x: event.clientX, y: event.clientY, selection };
}

function inputClass() {
  return 'w-full rounded-xl border border-slate-200 bg-white px-3 py-2.5 text-sm outline-none transition-colors focus:border-blue-500 disabled:cursor-not-allowed disabled:opacity-60';
}

function statusBadgeClass(status: RouteStatus) {
  if (status === 'ready') return 'rounded-full border border-emerald-100 bg-emerald-50 px-3 py-1.5 text-xs font-semibold text-emerald-700';
  if (status === 'loading') return 'rounded-full border border-blue-100 bg-blue-50 px-3 py-1.5 text-xs font-semibold text-blue-700';
  return 'rounded-full border border-amber-100 bg-amber-50 px-3 py-1.5 text-xs font-semibold text-amber-700';
}

function segmentClass(active: boolean, disabled = false) {
  return [
    'rounded-xl border px-3 py-2 text-sm font-semibold transition-colors',
    active ? 'border-slate-900 bg-slate-900 text-white' : 'border-slate-200 bg-white text-slate-700 hover:bg-slate-50',
    disabled ? 'cursor-not-allowed opacity-50' : '',
  ].join(' ');
}

function dialogTitle(dialog: DialogState, t: (key: string) => string) {
  if (dialog.kind === 'catalog-folder-create') return t('catalogMenu.dialogTitles.createFolder');
  if (dialog.kind === 'catalog-folder-edit') return t('catalogMenu.dialogTitles.editFolder');
  if (dialog.kind === 'catalog-item-create') return t('catalogMenu.dialogTitles.createItem');
  if (dialog.kind === 'catalog-item-edit') return t('catalogMenu.dialogTitles.editItem');
  if (dialog.kind === 'catalog-tag-create') return t('catalogMenu.dialogTitles.createTag');
  if (dialog.kind === 'catalog-tag-edit') return t('catalogMenu.dialogTitles.editTag');
  if (dialog.kind === 'item-tag-assign') return t('catalogMenu.dialogTitles.assignTag');
  if (dialog.kind === 'menu-category-create') return t('catalogMenu.dialogTitles.createMenuCategory');
  if (dialog.kind === 'menu-category-edit') return t('catalogMenu.dialogTitles.editMenuCategory');
  if (dialog.kind === 'menu-item-create') return t('catalogMenu.dialogTitles.createMenuItem');
  return t('catalogMenu.dialogTitles.editMenuItem');
}
