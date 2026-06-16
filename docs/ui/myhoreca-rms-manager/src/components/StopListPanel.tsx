import { useState } from 'react';
import { Ban, Search, RefreshCw, AlertOctagon, CheckCircle2, ShieldAlert, Sparkles, Filter } from 'lucide-react';
import { MenuItem } from '../types';

interface StopListPanelProps {
  menuItems: MenuItem[];
  onUpdateMenuItem: (item: MenuItem) => void;
}

export default function StopListPanel({ menuItems, onUpdateMenuItem }: StopListPanelProps) {
  const [searchQuery, setSearchQuery] = useState('');
  const [filterType, setFilterType] = useState<'all' | 'stopped' | 'low-stock'>('all');

  const stoppedItems = menuItems.filter(item => !item.isAvailable || item.stock === 0);
  const lowStockThreshold = 5;
  const lowStockItems = menuItems.filter(item => item.stock > 0 && item.stock <= lowStockThreshold && item.isAvailable);

  const displayItems = menuItems.filter(item => {
    const matchesSearch = item.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
                          item.description.toLowerCase().includes(searchQuery.toLowerCase());
    
    if (!matchesSearch) return false;
    if (filterType === 'stopped') return !item.isAvailable || item.stock === 0;
    if (filterType === 'low-stock') return item.stock > 0 && item.stock <= lowStockThreshold && item.isAvailable;
    return true;
  });

  const handleToggleStop = (item: MenuItem) => {
    onUpdateMenuItem({
      ...item,
      isAvailable: !item.isAvailable,
      // If releasing, make sure stock is either unlimited or reset to 10
      stock: !item.isAvailable ? (item.stock === 0 ? 15 : item.stock) : item.stock
    });
  };

  const handleSetStockToZero = (item: MenuItem) => {
    onUpdateMenuItem({
      ...item,
      stock: 0,
      isAvailable: false
    });
  };

  const handleResetStock = (item: MenuItem) => {
    onUpdateMenuItem({
      ...item,
      stock: -1,
      isAvailable: true
    });
  };

  return (
    <div className="flex-1 overflow-y-auto bg-slate-50/50 p-8 flex flex-col h-full">
      {/* Header */}
      <div className="flex flex-col sm:flex-row justify-between sm:items-center gap-4 mb-8 shrink-0">
        <div>
          <h2 className="text-2xl font-bold tracking-tight text-slate-900 font-sans">
            Управление активным Стоп-листом
          </h2>
          <p className="text-sm text-slate-500 mt-1">
            Координационная панель блокировки блюд. Позиции в стоп-листе мгновенно блокируются для официантов, защищая от перепродаж.
          </p>
        </div>
      </div>

      {/* KPI Blocks */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-6 mb-8 shrink-0">
        <div className="bg-white p-5 rounded-2xl border border-slate-200 shadow-sm flex items-center justify-between">
          <div>
            <span className="text-[10px] text-slate-400 font-extrabold uppercase tracking-wider block">Блокировано (Стоп-лист)</span>
            <p className="text-2xl font-black text-rose-600 mt-1">{stoppedItems.length} поз.</p>
          </div>
          <div className="p-3 bg-rose-50 text-rose-500 rounded-xl">
            <Ban className="w-5 h-5 animate-pulse" />
          </div>
        </div>

        <div className="bg-white p-5 rounded-2xl border border-slate-200 shadow-sm flex items-center justify-between">
          <div>
            <span className="text-[10px] text-slate-400 font-extrabold uppercase tracking-wider block">Критический остаток</span>
            <p className="text-2xl font-black text-amber-600 mt-1">{lowStockItems.length} поз.</p>
          </div>
          <div className="p-3 bg-amber-50 text-amber-500 rounded-xl">
            <AlertOctagon className="w-5 h-5" />
          </div>
        </div>

        <div className="bg-white p-5 rounded-2xl border border-slate-200 shadow-sm flex items-center justify-between">
          <div>
            <span className="text-[10px] text-slate-400 font-extrabold uppercase tracking-wider block">Всего в меню</span>
            <p className="text-2xl font-black text-slate-800 mt-1">{menuItems.length} поз.</p>
          </div>
          <div className="p-3 bg-blue-50 text-blue-600 rounded-xl">
            <CheckCircle2 className="w-5 h-5" />
          </div>
        </div>
      </div>

      {/* Filters bar */}
      <div className="flex flex-col md:flex-row justify-between items-stretch md:items-center gap-4 mb-6 select-none shrink-0">
        <div className="flex flex-wrap gap-2">
          <button
            onClick={() => setFilterType('all')}
            className={`px-4 py-2.5 rounded-xl text-xs font-bold leading-none transition-all ${
              filterType === 'all'
                ? 'bg-slate-900 text-white'
                : 'bg-white text-slate-600 border border-slate-200 hover:bg-slate-50'
            }`}
          >
            Все блюда ({menuItems.length})
          </button>
          <button
            onClick={() => setFilterType('stopped')}
            className={`px-4 py-2.5 rounded-xl text-xs font-bold leading-none transition-all flex items-center gap-1.5 ${
              filterType === 'stopped'
                ? 'bg-rose-600 text-white border border-rose-700 shadow-sm'
                : 'bg-white text-slate-600 border border-slate-200 hover:bg-rose-50/50'
            }`}
          >
            <Ban className="w-3.5 h-3.5" />
            <span>В стоп-листе ({stoppedItems.length})</span>
          </button>
          <button
            onClick={() => setFilterType('low-stock')}
            className={`px-4 py-2.5 rounded-xl text-xs font-bold leading-none transition-all flex items-center gap-1.5 ${
              filterType === 'low-stock'
                ? 'bg-amber-505 bg-amber-500 text-white border border-amber-600 shadow-sm'
                : 'bg-white text-slate-600 border border-slate-200 hover:bg-amber-50/40'
            }`}
          >
            <AlertOctagon className="w-3.5 h-3.5" />
            <span>Заканчиваются ({lowStockItems.length})</span>
          </button>
        </div>

        {/* Search */}
        <div className="relative min-w-[280px]">
          <Search className="w-4 h-4 text-slate-400 absolute left-3.5 top-1/2 -translate-y-1/2" />
          <input
            type="text"
            placeholder="Поиск по меню стоп-листов..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-full pl-10 pr-4 py-2.5 bg-white border border-slate-200 rounded-xl text-xs font-semibold focus:outline-none focus:border-blue-500 transition-colors placeholder-slate-400"
          />
        </div>
      </div>

      {/* Grid container */}
      <div className="flex-1 bg-white border border-slate-200 rounded-3xl overflow-hidden shadow-sm flex flex-col min-h-0">
        <div className="flex-1 overflow-y-auto">
          <table className="w-full text-left border-collapse min-w-[600px] text-xs">
            <thead>
              <tr className="border-b border-slate-100 bg-slate-50 text-slate-400 font-mono font-bold uppercase select-none sticky top-0 z-10">
                <th className="py-4 px-6">Позиция</th>
                <th className="py-4 px-4 text-center">Статус наличия</th>
                <th className="py-4 px-4 text-center">Остаток кухни</th>
                <th className="py-4 px-4 text-center">Цена продажи</th>
                <th className="py-4 px-6 text-right">Управление стоп-листом</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100 font-sans text-slate-700">
              {displayItems.map((item) => {
                const isBlocked = !item.isAvailable || item.stock === 0;
                const isLow = item.stock > 0 && item.stock <= lowStockThreshold;

                return (
                  <tr key={item.id} className={`hover:bg-slate-55 bg-white transition-colors duration-150 ${isBlocked ? 'bg-slate-50/40' : ''}`}>
                    <td className="py-4 px-6">
                      <div className="flex items-center gap-3">
                        <span className="text-2xl p-1.5 bg-slate-50 border border-slate-100 rounded-lg shrink-0 leading-none">
                          {item.emoji}
                        </span>
                        <div>
                          <p className="font-bold text-slate-900">{item.name}</p>
                          <p className="text-[10px] text-slate-400 line-clamp-1">{item.description}</p>
                        </div>
                      </div>
                    </td>
                    <td className="py-4 px-4 text-center">
                      {isBlocked ? (
                        <span className="inline-flex items-center gap-1 bg-rose-50 border border-rose-100 text-rose-600 font-extrabold text-[10px] tracking-wide uppercase px-2.5 py-1 rounded-full">
                          <Ban className="w-3 h-3" />
                          <span>СТОП</span>
                        </span>
                      ) : isLow ? (
                        <span className="inline-flex items-center gap-1 bg-amber-50 border border-amber-100 text-amber-700 font-extrabold text-[10px] tracking-wide uppercase px-2.5 py-1 rounded-full">
                          <AlertOctagon className="w-3 h-3" />
                          <span>Лимит</span>
                        </span>
                      ) : (
                        <span className="inline-flex items-center gap-1 bg-emerald-50 border border-emerald-100 text-emerald-700 font-extrabold text-[10px] tracking-wide uppercase px-2.5 py-1 rounded-full">
                          <CheckCircle2 className="w-3 h-3" />
                          <span>Активно</span>
                        </span>
                      )}
                    </td>
                    <td className="py-4 px-4 text-center font-mono font-bold text-slate-800">
                      {item.stock === -1 ? (
                        <span className="text-emerald-500">Без лимита</span>
                      ) : item.stock === 0 ? (
                        <span className="text-rose-500">Закончилось (0)</span>
                      ) : (
                        <span className={item.stock <= lowStockThreshold ? 'text-amber-550 text-amber-600 font-extrabold' : 'text-slate-700'}>
                          {item.stock} порц.
                        </span>
                      )}
                    </td>
                    <td className="py-4 px-4 text-center font-semibold text-slate-900 font-mono">
                      {item.price} ₽
                    </td>
                    <td className="py-4 px-6">
                      <div className="flex items-center justify-end gap-2 text-[10px] font-bold">
                        {isBlocked ? (
                          <button
                            onClick={() => handleToggleStop(item)}
                            className="bg-emerald-600 border border-emerald-700 hover:bg-emerald-700 text-white rounded-lg px-3 py-1.5 shadow-sm transition-all"
                          >
                            Вернуть в продажу
                          </button>
                        ) : (
                          <>
                            <button
                              onClick={() => handleSetStockToZero(item)}
                              className="bg-white border border-rose-200 text-rose-600 hover:bg-rose-50 rounded-lg px-3 py-1.5 shadow-sm transition-all"
                            >
                              В стоп-лист (0)
                            </button>
                            <button
                              onClick={() => handleToggleStop(item)}
                              className="bg-rose-600 border border-rose-700 hover:bg-rose-700 text-white rounded-lg px-3 py-1.5 shadow-sm transition-all"
                            >
                              Заблокировать
                            </button>
                          </>
                        )}
                        {item.stock !== -1 && (
                          <button
                            onClick={() => handleResetStock(item)}
                            className="bg-white border border-slate-200 text-slate-600 hover:bg-slate-100 rounded-lg px-2.5 py-1.5 shadow-sm"
                            title="Снять лимиты кухни"
                          >
                            Снять лимит
                          </button>
                        )}
                      </div>
                    </td>
                  </tr>
                );
              })}

              {displayItems.length === 0 && (
                <tr>
                  <td colSpan={5} className="py-16 text-center text-slate-400 bg-slate-50/20">
                    <div className="flex flex-col items-center justify-center gap-2">
                      <ShieldAlert className="w-8 h-8 text-slate-300" />
                      <p className="font-sans font-bold text-slate-600">Соответствующие блюда не найдены</p>
                      <p className="text-[11px] text-slate-400">Сбросьте поисковый фильтр или вернитесь ко всему меню.</p>
                    </div>
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>

        {/* Sync status footer */}
        <div className="px-6 py-4.5 bg-slate-50 border-t border-slate-150 flex flex-col sm:flex-row items-center justify-between gap-3 text-xs text-slate-500 font-mono select-none">
          <div className="flex items-center gap-2">
            <span className="w-2 h-2 rounded-full bg-blue-500 animate-ping"></span>
            <span className="font-bold text-slate-600">Шлюз стоп-листов: Активен</span>
          </div>
          <span className="text-[10px] uppercase font-bold text-slate-400 bg-white border px-2 py-0.5 rounded shadow-sm">
            iiko API & HTTP Sink v2.4-ready
          </span>
        </div>
      </div>
    </div>
  );
}
