import React, { useState } from 'react';
import { Package, Search, Plus, Trash2, ClipboardList, TrendingDown, BookOpen, Truck, HelpCircle, FileText, CheckCircle } from 'lucide-react';

interface WarehouseDoc {
  id: string;
  docNumber: string;
  type: 'incoming' | 'write_off' | 'inventory_act';
  date: string;
  contractor: string; // Supplier / Person in charge
  totalSum: number;
  status: 'draft' | 'processed' | 'pending';
  itemsCount: number;
  warehouseName: string;
}

const INITIAL_DOCS: WarehouseDoc[] = [
  {
    id: 'doc-1',
    docNumber: 'ТТН-04812',
    type: 'incoming',
    date: '2026-06-15',
    contractor: 'ООО Мясной Союз Родина',
    totalSum: 48500,
    status: 'processed',
    itemsCount: 4,
    warehouseName: 'Основная камера заморозки'
  },
  {
    id: 'doc-2',
    docNumber: 'АКТ-00918',
    type: 'write_off',
    date: '2026-06-14',
    contractor: 'Шеф-повар Козлов И.',
    totalSum: 2300,
    status: 'processed',
    itemsCount: 3,
    warehouseName: 'Кухня (Холодный цех)'
  },
  {
    id: 'doc-3',
    docNumber: 'ИНВ-2026-06',
    type: 'inventory_act',
    date: '2026-06-10',
    contractor: 'Управляющая Романова Е.',
    totalSum: -1400, // discrepancy check
    status: 'processed',
    itemsCount: 124,
    warehouseName: 'Сводный склад Сухой бакалеи'
  },
  {
    id: 'doc-4',
    docNumber: 'ТТН-04815',
    type: 'incoming',
    date: '2026-06-16',
    contractor: 'ТД Молочные Продукты',
    totalSum: 12800,
    status: 'pending',
    itemsCount: 2,
    warehouseName: 'Основная холодильная витрина'
  },
  {
    id: 'doc-5',
    docNumber: 'ТТН-04816',
    type: 'incoming',
    date: '2026-06-17',
    contractor: 'ИП Овощи Фрукты Садовод',
    totalSum: 9200,
    status: 'draft',
    itemsCount: 8,
    warehouseName: 'Склад свежих продуктов'
  }
];

export default function WarehousePanel() {
  const [docs, setDocs] = useState<WarehouseDoc[]>(INITIAL_DOCS);
  const [searchQuery, setSearchQuery] = useState('');
  const [docFilter, setDocFilter] = useState<'all' | 'incoming' | 'write_off' | 'inventory_act'>('all');
  const [isModalOpen, setIsModalOpen] = useState(false);

  // Form states
  const [docNumber, setDocNumber] = useState('');
  const [type, setType] = useState<'incoming' | 'write_off' | 'inventory_act'>('incoming');
  const [contractor, setContractor] = useState('');
  const [totalSum, setTotalSum] = useState<number>(0);
  const [warehouseName, setWarehouseName] = useState('Основной склад');
  const [itemsCount, setItemsCount] = useState<number>(1);

  const handleAddDoc = (e: React.FormEvent) => {
    e.preventDefault();
    if (!docNumber.trim()) return;

    const newDoc: WarehouseDoc = {
      id: `doc-${Date.now()}`,
      docNumber,
      type,
      date: new Date().toISOString().split('T')[0],
      contractor,
      totalSum: Number(totalSum),
      status: type === 'incoming' ? 'pending' : 'processed',
      itemsCount,
      warehouseName
    };

    setDocs(prev => [newDoc, ...prev]);
    setIsModalOpen(false);

    // Reset form
    setDocNumber('');
    setContractor('');
    setTotalSum(0);
    setWarehouseName('Основной склад');
    setItemsCount(1);
  };

  const handleDeleteDoc = (id: string) => {
    setDocs(prev => prev.filter(d => d.id !== id));
  };

  const handleProcessDoc = (id: string) => {
    setDocs(prev => prev.map(d => {
      if (d.id === id) {
        return { ...d, status: 'processed' };
      }
      return d;
    }));
  };

  const filteredDocs = docs.filter(doc => {
    const matchesSearch = doc.docNumber.toLowerCase().includes(searchQuery.toLowerCase()) ||
                          doc.contractor.toLowerCase().includes(searchQuery.toLowerCase()) ||
                          doc.warehouseName.toLowerCase().includes(searchQuery.toLowerCase());
    const matchesFilter = docFilter === 'all' || doc.type === docFilter;
    return matchesSearch && matchesFilter;
  });

  const getDocTypeLabel = (t: WarehouseDoc['type']) => {
    switch (t) {
      case 'incoming': return { text: 'Приход (ТТН)', color: 'bg-emerald-50 text-emerald-700 border-emerald-100' };
      case 'write_off': return { text: 'Списание / Акт списания', color: 'bg-rose-50 text-rose-700 border-rose-100' };
      default: return { text: 'Инвентаризация', color: 'bg-blue-50 text-blue-700 border-blue-105' };
    }
  };

  const kpis = {
    incomingSum: docs.filter(d => d.type === 'incoming' && d.status === 'processed').reduce((acc, d) => acc + d.totalSum, 0),
    writeOffSum: docs.filter(d => d.type === 'write_off').reduce((acc, d) => acc + d.totalSum, 0),
    pendingDelivery: docs.filter(d => d.status === 'pending').length
  };

  return (
    <div className="flex-1 overflow-y-auto bg-slate-50/50 p-8 flex flex-col h-full font-sans">
      {/* Header */}
      <div className="flex flex-col sm:flex-row justify-between sm:items-center gap-4 mb-8 shrink-0">
        <div>
          <h2 className="text-2xl font-bold tracking-tight text-slate-900">
            Складской учет и товародвижение
          </h2>
          <p className="text-sm text-slate-500 mt-1">
            Регистрация внешних поставок сырья, автоматическое списание калькуляционных карт техпроцессов и промежуточная инвентаризация остатков.
          </p>
        </div>

        <button
          onClick={() => setIsModalOpen(true)}
          className="flex items-center gap-2 px-5 py-3 rounded-xl bg-blue-600 hover:bg-blue-700 text-white font-bold text-xs shadow-sm transition-all border border-blue-700 select-none transform active:scale-95 cursor-pointer"
        >
          <Plus className="w-4 h-4" />
          <span>Новый документ склада</span>
        </button>
      </div>

      {/* KPI Blocks */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8 shrink-0">
        <div className="bg-white p-5 rounded-2xl border border-slate-200 shadow-sm flex items-center justify-between">
          <div>
            <span className="text-[10px] text-slate-400 font-extrabold uppercase tracking-wider block">Поставки за месяц (ТТН)</span>
            <p className="text-2xl font-black text-slate-800 mt-1">{kpis.incomingSum.toLocaleString()} ₽</p>
          </div>
          <div className="p-3 bg-emerald-50 text-emerald-600 rounded-xl">
            <Truck className="w-5 h-5" />
          </div>
        </div>

        <div className="bg-white p-5 rounded-2xl border border-slate-200 shadow-sm flex items-center justify-between">
          <div>
            <span className="text-[10px] text-slate-400 font-extrabold uppercase tracking-wider block font-sans">Зафиксированный износ / Брак</span>
            <p className="text-2xl font-black text-rose-600 mt-1">{kpis.writeOffSum.toLocaleString()} ₽</p>
          </div>
          <div className="p-3 bg-rose-50 text-rose-500 rounded-xl">
            <TrendingDown className="w-5 h-5" />
          </div>
        </div>

        <div className="bg-white p-5 rounded-2xl border border-slate-200 shadow-sm flex items-center justify-between">
          <div>
            <span className="text-[10px] text-slate-400 font-extrabold uppercase tracking-wider block">Ожидает выгрузки (ТТН)</span>
            <p className="text-2xl font-black text-amber-500 mt-1">{kpis.pendingDelivery} док.</p>
          </div>
          <span className="w-2.5 h-2.5 rounded-full bg-amber-500 shadow-sm animate-pulse"></span>
        </div>
      </div>

      {/* Filter and Search Bar */}
      <div className="flex flex-col md:flex-row justify-between items-stretch md:items-center gap-4 mb-6 select-none shrink-0">
        <div className="flex flex-wrap gap-2">
          <button
            onClick={() => setDocFilter('all')}
            className={`px-4 py-2.5 rounded-xl text-xs font-bold leading-none transition-all ${
              docFilter === 'all'
                ? 'bg-slate-900 text-white'
                : 'bg-white text-slate-600 border border-slate-200 hover:bg-slate-50'
            }`}
          >
            Все типы документов ({docs.length})
          </button>
          <button
            onClick={() => setDocFilter('incoming')}
            className={`px-4 py-2.5 rounded-xl text-xs font-bold leading-none transition-all ${
              docFilter === 'incoming'
                ? 'bg-slate-950 text-white'
                : 'bg-white text-slate-600 border border-slate-200 hover:bg-slate-50'
            }`}
          >
            Только Поставки (ТТН)
          </button>
          <button
            onClick={() => setDocFilter('write_off')}
            className={`px-4 py-2.5 rounded-xl text-xs font-bold leading-none transition-all ${
              docFilter === 'write_off'
                ? 'bg-slate-950 text-white'
                : 'bg-white text-slate-600 border border-slate-200 hover:bg-slate-50'
            }`}
          >
            Акты списания
          </button>
          <button
            onClick={() => setDocFilter('inventory_act')}
            className={`px-4 py-2.5 rounded-xl text-xs font-bold leading-none transition-all ${
              docFilter === 'inventory_act'
                ? 'bg-slate-950 text-white'
                : 'bg-white text-slate-600 border border-slate-200 hover:bg-slate-50'
            }`}
          >
            Инвентаризации
          </button>
        </div>

        {/* Search Input */}
        <div className="relative min-w-[280px]">
          <Search className="w-4 h-4 text-slate-400 absolute left-3.5 top-1/2 -translate-y-1/2" />
          <input
            type="text"
            placeholder="Поиск по ТТН, контрагенту, складу..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-full pl-10 pr-4 py-2.5 bg-white border border-slate-200 rounded-xl text-xs font-semibold focus:outline-none focus:border-blue-500 transition-colors placeholder-slate-400"
          />
        </div>
      </div>

      {/* Main Table with lists */}
      <div className="flex-1 bg-white border border-slate-200 rounded-3xl overflow-hidden shadow-sm flex flex-col min-h-0">
        <div className="flex-1 overflow-y-auto">
          <table className="w-full text-left border-collapse min-w-[650px] text-xs">
            <thead>
              <tr className="border-b border-slate-100 bg-slate-50 text-slate-400 font-mono font-bold uppercase select-none sticky top-0 z-10">
                <th className="py-4 px-6">Номер документа</th>
                <th className="py-4 px-4">Тип акта</th>
                <th className="py-4 px-4">Дата проводки</th>
                <th className="py-4 px-4">Склад назначения</th>
                <th className="py-4 px-4">Поставщик / Ответственный</th>
                <th className="py-4 px-4">Сумма документа</th>
                <th className="py-4 px-4 text-center">Статус</th>
                <th className="py-4 px-6 text-right">Действия</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100 font-sans text-slate-700">
              {filteredDocs.map((doc) => {
                const badge = getDocTypeLabel(doc.type);
                return (
                  <tr key={doc.id} className="hover:bg-slate-50 transition-colors">
                    <td className="py-4 px-6 font-bold text-slate-900 font-mono">
                      {doc.docNumber}
                    </td>
                    <td className="py-4 px-4">
                      <span className={`inline-block border px-2 py-0.5 rounded text-[10px] font-bold ${badge.color}`}>
                        {badge.text}
                      </span>
                    </td>
                    <td className="py-4 px-4 font-mono text-slate-550">
                      {doc.date}
                    </td>
                    <td className="py-4 px-4 font-medium text-slate-700">
                      {doc.warehouseName}
                    </td>
                    <td className="py-4 px-4 text-slate-650">
                      {doc.contractor} ({doc.itemsCount} наим.)
                    </td>
                    <td className={`py-4 px-4 font-semibold font-mono ${doc.totalSum < 0 ? 'text-rose-500' : 'text-slate-900'}`}>
                      {doc.totalSum.toLocaleString()} ₽
                    </td>
                    <td className="py-4 px-4 text-center">
                      {doc.status === 'processed' ? (
                        <span className="bg-emerald-50 text-emerald-600 font-bold px-1.5 py-0.5 rounded text-[10px] tracking-wide uppercase">Проведён</span>
                      ) : doc.status === 'pending' ? (
                        <span className="bg-amber-50 text-amber-600 font-bold px-1.5 py-0.5 rounded text-[10px] tracking-wide uppercase animate-pulse">Ожидание</span>
                      ) : (
                        <span className="bg-slate-100 text-slate-550 font-bold px-1.5 py-0.5 rounded text-[10px] tracking-wide uppercase">Черновик</span>
                      )}
                    </td>
                    <td className="py-4 px-6">
                      <div className="flex items-center justify-end gap-1.5 text-[10px] font-bold">
                        {doc.status === 'pending' && (
                          <button
                            onClick={() => handleProcessDoc(doc.id)}
                            className="bg-blue-600 hover:bg-blue-700 text-white rounded px-2 py-1 shadow-sm transition-all"
                          >
                            Провести ТТН
                          </button>
                        )}
                        <button
                          onClick={() => handleDeleteDoc(doc.id)}
                          className="text-slate-400 hover:text-rose-600 p-1 rounded hover:bg-slate-100"
                          title="Удалить накладную"
                        >
                          <Trash2 className="w-3.5 h-3.5" />
                        </button>
                      </div>
                    </td>
                  </tr>
                );
              })}

              {filteredDocs.length === 0 && (
                <tr>
                  <td colSpan={8} className="py-16 text-center text-slate-400">
                    <p className="font-bold font-sans text-slate-600">Документы не найдены</p>
                    <p className="text-[11px] text-slate-400">Сбросьте поисковый запрос или создайте новую накладную.</p>
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>

        {/* Sync status footer */}
        <div className="px-6 py-4.5 bg-slate-50 border-t border-slate-150 flex items-center justify-between text-xs text-slate-500 font-mono select-none">
          <div className="flex items-center gap-1.5">
            <BookOpen className="w-4 h-4 text-slate-400" />
            <span>Калькуляционный шлюз: Активен (списание по чекам кухни в реальном времени)</span>
          </div>
          <span className="text-[10px] uppercase font-bold text-slate-400">Склад RMS v2</span>
        </div>
      </div>

      {/* Simplified Addition Modal */}
      {isModalOpen && (
        <div className="fixed inset-0 bg-slate-900/40 backdrop-blur-sm z-50 flex items-center justify-center p-4 select-none animate-in fade-in duration-200">
          <div className="bg-white rounded-3xl w-full max-w-md shadow-2xl overflow-hidden p-6 text-slate-800">
            <h3 className="text-base font-extrabold text-slate-900 font-sans border-b pb-3 mb-4">
              Добавление складского документа
            </h3>

            <form onSubmit={handleAddDoc} className="space-y-4">
              <div>
                <label className="text-[10px] uppercase font-bold font-mono text-slate-400 block mb-1">Номер документа (ТТН, Акт)</label>
                <input
                  type="text"
                  required
                  placeholder="ТТН-04817"
                  value={docNumber}
                  onChange={(e) => setDocNumber(e.target.value)}
                  className="w-full p-2.5 text-xs font-semibold bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:border-blue-500"
                />
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="text-[10px] uppercase font-bold font-mono text-slate-400 block mb-1">Тип операции</label>
                  <select
                    value={type}
                    onChange={(e) => setType(e.target.value as any)}
                    className="w-full p-2.5 text-xs font-semibold bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:border-blue-500"
                  >
                    <option value="incoming">ТТН Приход</option>
                    <option value="write_off">Списание брака</option>
                    <option value="inventory_act">Акт инвентаризации</option>
                  </select>
                </div>

                <div>
                  <label className="text-[10px] uppercase font-bold font-mono text-slate-400 block mb-1 font-sans">Позиций (наим.)</label>
                  <input
                    type="number"
                    min={1}
                    value={itemsCount}
                    onChange={(e) => setItemsCount(Number(e.target.value))}
                    className="w-full p-2.5 text-xs font-semibold bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:border-blue-500 font-mono"
                  />
                </div>
              </div>

              <div>
                <label className="text-[10px] uppercase font-bold font-mono text-slate-400 block mb-1">Поставщик / Ответственное лицо</label>
                <input
                  type="text"
                  required
                  placeholder="ООО ФрешМясоМаркет"
                  value={contractor}
                  onChange={(e) => setContractor(e.target.value)}
                  className="w-full p-2.5 text-xs font-semibold bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:border-blue-500"
                />
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="text-[10px] uppercase font-bold font-mono text-slate-400 block mb-1">Сумма акта (₽)</label>
                  <input
                    type="number"
                    value={totalSum}
                    onChange={(e) => setTotalSum(Number(e.target.value))}
                    className="w-full p-2.5 text-xs font-semibold bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:border-blue-500 font-mono"
                  />
                </div>

                <div>
                  <label className="text-[10px] uppercase font-bold font-mono text-slate-400 block mb-1">Склад</label>
                  <input
                    type="text"
                    value={warehouseName}
                    onChange={(e) => setWarehouseName(e.target.value)}
                    className="w-full p-2.5 text-xs font-semibold bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:border-blue-500"
                  />
                </div>
              </div>

              <div className="flex justify-end gap-2.5 pt-4 border-t border-slate-100">
                <button
                  type="button"
                  onClick={() => setIsModalOpen(false)}
                  className="px-4 py-2 rounded-xl text-xs font-semibold text-slate-600 hover:text-slate-800 hover:bg-slate-100 cursor-pointer"
                >
                  Отмена
                </button>
                <button
                  type="submit"
                  className="px-5 py-2 rounded-xl text-xs font-bold text-white bg-blue-600 hover:bg-blue-700"
                >
                  Зарегистрировать
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
