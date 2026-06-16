import React, { useState, useRef, useEffect } from 'react';
import { Armchair, Plus, Trash2, Shield, Eye, HelpCircle, Sparkles, Check, Move } from 'lucide-react';
import { HallTable, StaffMember } from '../types';

interface TablesPanelProps {
  staffList: StaffMember[];
}

const INITIAL_TABLES: HallTable[] = [
  { id: 'tbl-1', number: '1', seats: 2, shape: 'square', x: 20, y: 25, status: 'busy', waiterId: 'staff-4' },
  { id: 'tbl-2', number: '2', seats: 4, shape: 'circle', x: 50, y: 25, status: 'free', waiterId: 'staff-4' },
  { id: 'tbl-3', number: '3', seats: 4, shape: 'square', x: 80, y: 25, status: 'reserved', waiterId: 'staff-5' },
  { id: 'tbl-4', number: '4', seats: 6, shape: 'rectangle', x: 20, y: 70, status: 'free', waiterId: 'staff-4' },
  { id: 'tbl-5', number: '5', seats: 8, shape: 'rectangle', x: 50, y: 70, status: 'busy', waiterId: 'staff-5' },
  { id: 'tbl-6', number: '6', seats: 2, shape: 'circle', x: 80, y: 70, status: 'free' }
];

export default function TablesPanel({ staffList }: TablesPanelProps) {
  const [tables, setTables] = useState<HallTable[]>(INITIAL_TABLES);
  const [selectedTable, setSelectedTable] = useState<HallTable | null>(null);
  const [activeZone, setActiveZone] = useState<'main' | 'terrace' | 'vip'>('main');
  const [isEditMode, setIsEditMode] = useState<boolean>(true); // Layout editor mode
  const boardRef = useRef<HTMLDivElement>(null);
  const [draggingId, setDraggingId] = useState<string | null>(null);
  const [dragStart, setDragStart] = useState<{ x: number; y: number }>({ x: 0, y: 0 });

  const activeWaiters = staffList.filter(s => s.role === 'waiter' || s.role === 'manager');

  // Trigger click-to-add custom table
  const handleAddTable = (shape: HallTable['shape']) => {
    const nextNum = (Math.max(...tables.map(t => parseInt(t.number) || 0)) + 1).toString();
    const newTbl: HallTable = {
      id: `tbl-${Date.now()}`,
      number: nextNum,
      seats: shape === 'square' ? 4 : shape === 'circle' ? 2 : 8,
      shape: shape,
      x: 35 + Math.random() * 20,
      y: 35 + Math.random() * 20,
      status: 'free'
    };
    setTables(prev => [...prev, newTbl]);
    setSelectedTable(newTbl);
  };

  const handleDeleteTable = (id: string) => {
    setTables(prev => prev.filter(t => t.id !== id));
    if (selectedTable?.id === id) {
      setSelectedTable(null);
    }
  };

  // Drag handlers
  const handleTableMouseDown = (e: React.MouseEvent, table: HallTable) => {
    if (!isEditMode) return;
    e.stopPropagation();
    setSelectedTable(table);
    setDraggingId(table.id);

    // Get current click coords relative to viewport
    setDragStart({
      x: e.clientX,
      y: e.clientY
    });
  };

  const handleBoardMouseMove = (e: React.MouseEvent) => {
    if (!draggingId || !boardRef.current) return;

    const boardRect = boardRef.current.getBoundingClientRect();
    const table = tables.find(t => t.id === draggingId);
    if (!table) return;

    // Calculate delta movement in percentage points of the board size
    const deltaX = ((e.clientX - dragStart.x) / boardRect.width) * 100;
    const deltaY = ((e.clientY - dragStart.y) / boardRect.height) * 100;

    // New coords bound between 2% and 90%
    const nextX = Math.min(Math.max(2, table.x + deltaX), 90);
    const nextY = Math.min(Math.max(2, table.y + deltaY), 90);

    setTables(prev => prev.map(t => {
      if (t.id === draggingId) {
        return { ...t, x: Math.round(nextX), y: Math.round(nextY) };
      }
      return t;
    }));

    setDragStart({
      x: e.clientX,
      y: e.clientY
    });
  };

  const handleMouseUp = () => {
    setDraggingId(null);
  };

  useEffect(() => {
    const handleGlobalMouseUp = () => {
      setDraggingId(null);
    };
    window.addEventListener('mouseup', handleGlobalMouseUp);
    return () => window.removeEventListener('mouseup', handleGlobalMouseUp);
  }, []);

  const handleUpdateField = (field: keyof HallTable, val: any) => {
    if (!selectedTable) return;
    setTables(prev => prev.map(t => {
      if (t.id === selectedTable.id) {
        const next = { ...t, [field]: val };
        // Sync selected state
        setTimeout(() => setSelectedTable(next), 0);
        return next;
      }
      return t;
    }));
  };

  const getStatusColor = (status: HallTable['status']) => {
    switch (status) {
      case 'busy': return 'bg-rose-500 text-white border-rose-600 ring-rose-200';
      case 'reserved': return 'bg-amber-500 text-white border-amber-600 ring-amber-200';
      default: return 'bg-emerald-500 text-white border-emerald-600 ring-emerald-200';
    }
  };

  const stats = {
    total: tables.length,
    busy: tables.filter(t => t.status === 'busy').length,
    reserved: tables.filter(t => t.status === 'reserved').length,
    free: tables.filter(t => t.status === 'free').length,
    seats: tables.reduce((acc, t) => acc + t.seats, 0),
    busySeats: tables.reduce((acc, t) => acc + (t.status === 'busy' ? t.seats : 0), 0)
  };

  return (
    <div className="flex-1 overflow-y-auto bg-slate-50/50 p-8 flex flex-col h-full">
      {/* Header */}
      <div className="flex flex-col sm:flex-row justify-between sm:items-center gap-4 mb-8 shrink-0">
        <div>
          <h2 className="text-2xl font-bold tracking-tight text-slate-900 font-sans">
            Редактор схемы зала и столов
          </h2>
          <p className="text-sm text-slate-500 mt-1">
            Проектируйте рассадку заведения, связывайте столы с официантами и управляйте заказами в реальном времени.
          </p>
        </div>

        <div className="flex items-center gap-2 bg-white p-1 rounded-xl border border-slate-200 shadow-sm">
          <button
            onClick={() => setIsEditMode(false)}
            className={`flex items-center gap-1.5 px-4 py-2 rounded-lg text-xs font-bold transition-all ${
              !isEditMode
                ? 'bg-blue-600 text-white'
                : 'text-slate-600 hover:text-slate-900'
            }`}
          >
            <Eye className="w-3.5 h-3.5" />
            <span>Мониторинг залов</span>
          </button>
          <button
            onClick={() => setIsEditMode(true)}
            className={`flex items-center gap-1.5 px-4 py-2 rounded-lg text-xs font-bold transition-all ${
              isEditMode
                ? 'bg-blue-600 text-white'
                : 'text-slate-600 hover:text-slate-900'
            }`}
          >
            <Move className="w-3.5 h-3.5" />
            <span>Конструктор (Drag-and-Drop)</span>
          </button>
        </div>
      </div>

      {/* KPI Row */}
      <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-5 gap-4 mb-6 shrink-0">
        <div className="bg-white p-4 rounded-xl border border-slate-200 flex items-center justify-between">
          <div>
            <span className="text-[10px] text-slate-400 block font-bold uppercase tracking-wider">Всего столов</span>
            <span className="text-xl font-extrabold text-slate-800">{stats.total}</span>
          </div>
          <div className="p-2 bg-slate-100 rounded-lg text-slate-500">
            <Armchair className="w-4 h-4" />
          </div>
        </div>
        <div className="bg-white p-4 rounded-xl border border-slate-200 flex items-center justify-between">
          <div>
            <span className="text-[10px] text-slate-400 block font-bold uppercase tracking-wider">Занято</span>
            <span className="text-xl font-extrabold text-rose-500">{stats.busy}</span>
          </div>
          <span className="w-2.5 h-2.5 rounded-full bg-rose-500 shadow-sm shadow-rose-200 animate-pulse"></span>
        </div>
        <div className="bg-white p-4 rounded-xl border border-slate-200 flex items-center justify-between">
          <div>
            <span className="text-[10px] text-slate-400 block font-bold uppercase tracking-wider">Бронь</span>
            <span className="text-xl font-extrabold text-amber-500">{stats.reserved}</span>
          </div>
          <span className="w-2.5 h-2.5 rounded-full bg-amber-500 shadow-sm shadow-amber-200"></span>
        </div>
        <div className="bg-white p-4 rounded-xl border border-slate-200 flex items-center justify-between">
          <div>
            <span className="text-[10px] text-slate-400 block font-bold uppercase tracking-wider">Свободно</span>
            <span className="text-xl font-extrabold text-emerald-500">{stats.free}</span>
          </div>
          <span className="w-2.5 h-2.5 rounded-full bg-emerald-500 shadow-sm"></span>
        </div>
        <div className="bg-white p-4 rounded-xl border border-slate-200 flex items-center justify-between col-span-2 md:col-span-1">
          <div>
            <span className="text-[10px] text-slate-400 block font-bold uppercase tracking-wider font-sans">Загрузка мест</span>
            <span className="text-sm font-semibold text-slate-800">
              {stats.busySeats} из {stats.seats} чел.
            </span>
          </div>
          <span className="text-[10px] font-mono font-bold text-slate-400 bg-slate-100 p-1.5 rounded-lg">
            {stats.seats > 0 ? Math.round((stats.busySeats / stats.seats) * 100) : 0}%
          </span>
        </div>
      </div>

      {/* Main floor board and Side controller */}
      <div className="flex-1 flex flex-col lg:flex-row gap-6 min-h-0">
        
        {/* Left Interactive Hall Panel */}
        <div className="flex-1 bg-white border border-slate-200 rounded-3xl flex flex-col overflow-hidden shadow-sm relative min-h-[450px]">
          {/* Tab zones controller */}
          <div className="px-6 py-4 border-b border-slate-200 bg-slate-50 flex flex-wrap items-center justify-between gap-3 select-none">
            <div className="flex items-center gap-1.5">
              <button
                onClick={() => setActiveZone('main')}
                className={`px-3 py-1.5 text-xs font-bold rounded-lg transition-all ${
                  activeZone === 'main' ? 'bg-slate-900 text-white shadow-sm' : 'text-slate-500 hover:text-slate-800'
                }`}
              >
                Основной зал
              </button>
              <button
                onClick={() => setActiveZone('terrace')}
                className={`px-3 py-1.5 text-xs font-bold rounded-lg transition-all ${
                  activeZone === 'terrace' ? 'bg-slate-900 text-white shadow-sm' : 'text-slate-500 hover:text-slate-800'
                }`}
              >
                Летняя веранда
              </button>
              <button
                onClick={() => setActiveZone('vip')}
                className={`px-3 py-1.5 text-xs font-bold rounded-lg transition-all ${
                  activeZone === 'vip' ? 'bg-slate-900 text-white shadow-sm' : 'text-slate-500 hover:text-slate-800'
                }`}
              >
                VIP Кабинеты
              </button>
            </div>

            {isEditMode && (
              <div className="flex items-center gap-1">
                <span className="text-[10px] uppercase font-bold text-slate-400 font-mono mr-1.5">Конструктор:</span>
                <button
                  onClick={() => handleAddTable('square')}
                  className="flex items-center gap-1 bg-white border border-slate-200 text-slate-700 font-semibold px-2.5 py-1.5 rounded-lg text-xs hover:bg-slate-50 transition-all hover:border-slate-350"
                  title="Добавить квадратный стол"
                >
                  <Plus className="w-3.5 h-3.5 text-blue-500" />
                  <span>Квадрат</span>
                </button>
                <button
                  onClick={() => handleAddTable('circle')}
                  className="flex items-center gap-1 bg-white border border-slate-200 text-slate-700 font-semibold px-2.5 py-1.5 rounded-lg text-xs hover:bg-slate-50 transition-all hover:border-slate-350"
                  title="Добавить круглый стол"
                >
                  <Plus className="w-3.5 h-3.5 text-blue-500" />
                  <span>Круг</span>
                </button>
                <button
                  onClick={() => handleAddTable('rectangle')}
                  className="flex items-center gap-1 bg-white border border-slate-200 text-slate-700 font-semibold px-2.5 py-1.5 rounded-lg text-xs hover:bg-slate-50 transition-all hover:border-slate-350"
                  title="Добавить банкетный прямоугольный стол"
                >
                  <Plus className="w-3.5 h-3.5 text-blue-500" />
                  <span>Банкет</span>
                </button>
              </div>
            )}
          </div>

          {/* 2D Grid floor plan board */}
          <div
            ref={boardRef}
            onMouseMove={handleBoardMouseMove}
            onMouseUp={handleMouseUp}
            className="flex-1 relative bg-slate-900/5 overflow-hidden border-b border-slate-200 bg-[radial-gradient(#e2e8f0_1px,transparent_1px)] [background-size:16px_16px]"
          >
            {/* Stage walls or directions labels */}
            <div className="absolute top-2 left-1/2 -translate-x-1/2 text-[9px] uppercase font-mono font-extrabold tracking-widest text-slate-400/70 select-none">
              Вход / Ресепшн заведения
            </div>
            <div className="absolute bottom-2 left-1/2 -translate-x-1/2 text-[9px] uppercase font-mono font-extrabold tracking-widest text-slate-400/70 select-none">
              Служебный проход / Кухня POS
            </div>
            <div className="absolute left-2 top-1/2 -translate-y-1/2 -rotate-90 text-[9px] uppercase font-mono font-extrabold tracking-widest text-slate-400/70 select-none">
              Окна на улицу
            </div>
            <div className="absolute right-2 top-1/2 -translate-y-1/2 rotate-90 text-[9px] uppercase font-mono font-extrabold tracking-widest text-slate-400/70 select-none">
              Барная зона
            </div>

            {/* Simulated tables */}
            {tables.map((table) => {
              const isSelected = selectedTable?.id === table.id;
              const shapeClass =
                table.shape === 'circle'
                  ? 'rounded-full'
                  : table.shape === 'rectangle'
                  ? 'rounded-2xl'
                  : 'rounded-xl';

              // Visual dimensions depending on shape and seats
              const widthStyle = table.shape === 'rectangle' ? 'w-28' : 'w-20';
              const heightStyle = table.shape === 'rectangle' ? 'h-16' : 'h-20';

              return (
                <div
                  key={table.id}
                  onMouseDown={(e) => handleTableMouseDown(e, table)}
                  onClick={() => setSelectedTable(table)}
                  style={{
                    left: `${table.x}%`,
                    top: `${table.y}%`,
                  }}
                  className={`absolute -translate-x-1/2 -translate-y-1/2 p-1.5 transition-all outline-none leading-none ${widthStyle} ${heightStyle} ${
                    isEditMode ? 'cursor-move' : 'cursor-pointer'
                  }`}
                >
                  <div
                    className={`w-full h-full ${shapeClass} border-2 relative flex flex-col items-center justify-center font-sans transition-all shadow-md group ${
                      isSelected
                        ? 'border-blue-500 ring-4 ring-blue-500/20 scale-105 scale-105 shadow-lg z-30'
                        : 'border-slate-300 shadow hover:scale-102 z-10'
                    } ${
                      table.status === 'busy'
                        ? 'bg-rose-50 border-rose-400'
                        : table.status === 'reserved'
                        ? 'bg-amber-50 border-amber-350'
                        : 'bg-white hover:bg-slate-50'
                    }`}
                  >
                    {/* Seat node circles visual overlay */}
                    <div className="absolute -top-1.5 inset-x-0 flex justify-center gap-1">
                      {Array.from({ length: Math.ceil(table.seats / 2) }).map((_, i) => (
                        <span key={i} className={`w-2 h-2 rounded-full border border-slate-300 ${table.status === 'busy' ? 'bg-rose-400' : 'bg-slate-200'}`} />
                      ))}
                    </div>
                    <div className="absolute -bottom-1.5 inset-x-0 flex justify-center gap-1">
                      {Array.from({ length: Math.floor(table.seats / 2) }).map((_, i) => (
                        <span key={i} className={`w-2 h-2 rounded-full border border-slate-300 ${table.status === 'busy' ? 'bg-rose-400' : 'bg-slate-200'}`} />
                      ))}
                    </div>

                    {/* Table description */}
                    <span className="font-mono text-[10px] text-slate-400 font-bold">СТОЛ</span>
                    <span className="text-xl font-extrabold text-slate-800 leading-none">{table.number}</span>
                    <span className="text-[9px] font-semibold font-mono text-slate-500 uppercase mt-0.5 whitespace-nowrap">
                      {table.seats} мест
                    </span>

                    {/* Bottom Status indicator */}
                    <div className={`absolute -right-1 -bottom-1 w-4 h-4 rounded-full border border-white flex items-center justify-center text-[8px] font-bold shadow ${getStatusColor(table.status)}`}>
                      {table.status === 'busy' ? '!' : table.status === 'reserved' ? 'P' : 'ok'}
                    </div>

                    {/* Assigned Waiter name short label */}
                    {table.waiterId && (
                      <div className="absolute -top-5 bg-slate-800 text-white text-[8px] font-bold uppercase rounded px-1.5 py-0.5 tracking-wider select-none transform scale-90 group-hover:scale-100 transition-all shadow pointer-events-none">
                        {activeWaiters.find(w => w.id === table.waiterId)?.name.split(' ')[0] || 'Оф.'}
                      </div>
                    )}
                  </div>
                </div>
              );
            })}
          </div>

          {/* Guidelines info bar */}
          <div className="px-6 py-3 bg-slate-50 border-t border-slate-200 text-xs text-slate-500 flex items-center gap-2 select-none shrink-0 font-medium">
            <HelpCircle className="w-4 h-4 text-slate-400 shrink-0" />
            <span>
              {isEditMode
                ? 'Режим редактирования: зажмите левую кнопку мыши на столе для перемещения, выберите стол для детальной настройки.'
                : 'Режим мониторинга: вы можете нажать на любой стол, чтобы просмотреть его привязанный персонал или сменить статус обслуживания гостей.'}
            </span>
          </div>
        </div>

        {/* Right Details & Edit parameters card */}
        <div className="w-full lg:w-80 bg-white border border-slate-200 rounded-3xl p-6 shadow-sm flex flex-col justify-between shrink-0 h-full overflow-y-auto">
          <div>
            <div className="border-b border-slate-100 pb-4 mb-5">
              <h3 className="font-sans font-bold text-slate-900 flex items-center gap-2">
                <Sparkles className="w-4 h-4 text-indigo-500" />
                <span>Спецификация стола</span>
              </h3>
              <p className="text-xs text-slate-400 mt-0.5">Выберите любой объект на схеме зала</p>
            </div>

            {selectedTable ? (
              <div className="space-y-4">
                {/* Table Number & Shape parameters */}
                <div className="grid grid-cols-2 gap-3.5">
                  <div>
                    <label className="text-[10px] uppercase tracking-wider font-semibold font-mono text-slate-400 block mb-1">
                      Номер стола
                    </label>
                    <input
                      type="text"
                      value={selectedTable.number}
                      onChange={(e) => handleUpdateField('number', e.target.value)}
                      className="w-full p-2.5 text-xs font-bold text-slate-850 bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:border-blue-500"
                    />
                  </div>

                  <div>
                    <label className="text-[10px] uppercase tracking-wider font-semibold font-mono text-slate-400 block mb-1">
                      Форма стола
                    </label>
                    <select
                      value={selectedTable.shape}
                      onChange={(e) => handleUpdateField('shape', e.target.value)}
                      className="w-full p-2.5 text-xs font-semibold bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:border-blue-500"
                    >
                      <option value="square">Квадрат</option>
                      <option value="circle">Круглый</option>
                      <option value="rectangle">Длинный банкет</option>
                    </select>
                  </div>
                </div>

                {/* Capacity Guests */}
                <div>
                  <label className="text-[10px] uppercase tracking-wider font-semibold font-mono text-slate-400 block mb-1">
                    Вместимость гостей (мест)
                  </label>
                  <input
                    type="number"
                    min={1}
                    max={16}
                    value={selectedTable.seats}
                    onChange={(e) => handleUpdateField('seats', Number(e.target.value))}
                    className="w-full p-2.5 text-xs font-semibold bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:border-blue-500 font-mono"
                  />
                </div>

                {/* Table Status */}
                <div>
                  <label className="text-[10px] uppercase tracking-wider font-semibold font-mono text-slate-400 block mb-1">
                    Статус обслуживания
                  </label>
                  <div className="grid grid-cols-3 gap-1.5 mt-1.5">
                    {(['free', 'busy', 'reserved'] as const).map((st) => {
                      const isActive = selectedTable.status === st;
                      const labels = { free: 'Свободен', busy: 'Занят', reserved: 'Бронь' };
                      const colors = {
                        free: isActive ? 'bg-emerald-500 text-white border-emerald-600' : 'bg-slate-50 text-slate-500',
                        busy: isActive ? 'bg-rose-500 text-white border-rose-600' : 'bg-slate-50 text-slate-500',
                        reserved: isActive ? 'bg-amber-500 text-white border-amber-600' : 'bg-slate-50 text-slate-500',
                      };
                      return (
                        <button
                          key={st}
                          type="button"
                          onClick={() => handleUpdateField('status', st)}
                          className={`py-2 text-[10px] font-bold border rounded-xl leading-none transition-all ${colors[st]}`}
                        >
                          {labels[st]}
                        </button>
                      );
                    })}
                  </div>
                </div>

                {/* Assigned Waiter */}
                <div>
                  <label className="text-[10px] uppercase tracking-wider font-semibold font-mono text-slate-400 block mb-1">
                    Ответственный официант
                  </label>
                  <select
                    value={selectedTable.waiterId || ''}
                    onChange={(e) => handleUpdateField('waiterId', e.target.value ? e.target.value : undefined)}
                    className="w-full p-2.5 text-xs font-semibold bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:border-blue-500"
                  >
                    <option value="">Не назначен (Свободная очередь)</option>
                    {activeWaiters.map((waiter) => (
                      <option key={waiter.id} value={waiter.id}>
                        {waiter.name} ({waiter.role === 'manager' ? 'Менеджер' : 'Официант'})
                      </option>
                    ))}
                  </select>
                </div>

                {/* Local Positions helper info */}
                <div className="p-3 bg-slate-50 border border-slate-150 rounded-2xl text-[11px] text-slate-500 space-y-1 font-sans">
                  <span className="font-extrabold text-slate-600 uppercase block tracking-wide">Параметры сетки зала:</span>
                  <p>Координата по оси X: {selectedTable.x}%</p>
                  <p>Координата по оси Y: {selectedTable.y}%</p>
                  <p className="italic text-[10px] text-slate-400 mt-1">
                    * Вы можете плавно перемещать стол зажав за него в рабочей области. Сетка имеет авто-блокировку границ.
                  </p>
                </div>
              </div>
            ) : (
              <div className="py-16 text-center text-slate-400 border border-slate-150 border-dashed rounded-3xl flex flex-col items-center justify-center gap-2 select-none">
                <span className="text-3xl text-slate-300">🛋️</span>
                <p className="font-bold text-xs text-slate-500">Стол не выбран</p>
                <p className="text-[10px] leading-relaxed max-w-[180px] text-slate-400">
                  Кликните по любому столу в зале, чтобы настроить его параметры.
                </p>
              </div>
            )}
          </div>

          {selectedTable && isEditMode && (
            <div className="pt-6 border-t border-slate-100 mt-6 select-none">
              <button
                onClick={() => handleDeleteTable(selectedTable.id)}
                className="w-full flex items-center justify-center gap-2 px-4 py-3 rounded-xl border border-rose-200 text-rose-500 hover:bg-rose-50 text-xs font-bold transition-all"
              >
                <Trash2 className="w-3.5 h-3.5" />
                <span>Удалить этот стол</span>
              </button>
            </div>
          )}
        </div>

      </div>
    </div>
  );
}
