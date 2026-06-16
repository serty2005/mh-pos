import { LayoutDashboard, Utensils, Users, RefreshCw, Landmark, Store, Map, Ban, Cpu, Package, ChevronDown } from 'lucide-react';

export const RESTAURANTS = [
  { id: 'all', name: 'Все заведения / Сервер', code: 'RMS-GLOBAL', color: 'bg-blue-500' },
  { id: 'rest-1', name: 'Флагман на Тверской', code: 'MSK-01 / TVE', color: 'bg-emerald-500' },
  { id: 'rest-2', name: 'Суши-бар на Садовой', code: 'MSK-02 / SAD', color: 'bg-rose-500' },
  { id: 'rest-3', name: 'Кофейня на Таганке', code: 'MSK-03 / TAG', color: 'bg-amber-500' },
];

interface SidebarProps {
  currentTab: string;
  setCurrentTab: (tab: string) => void;
  staffName: string;
  staffRole: string;
  selectedRestaurant: string;
  onRestaurantChange: (id: string) => void;
}

export default function Sidebar({
  currentTab,
  setCurrentTab,
  staffName,
  staffRole,
  selectedRestaurant,
  onRestaurantChange
}: SidebarProps) {
  const menuItems = [
    { id: 'analytics', label: 'Аналитика продаж', icon: LayoutDashboard },
    { id: 'menu', label: 'Управление меню', icon: Utensils },
    { id: 'tables-editor', label: 'Схема зала & Столы', icon: Map },
    { id: 'stop-list', label: 'Активные стоп-листы', icon: Ban },
    { id: 'staff', label: 'Персонал & Доступ', icon: Users },
    { id: 'edge-devices', label: 'Edge-терминалы & ККТ', icon: Cpu },
    { id: 'warehouse', label: 'Складской учет & ТТН', icon: Package },
    { id: 'sync', label: 'Синхронизация POS', icon: RefreshCw },
  ];

  const roleLabels: Record<string, string> = {
    admin: 'Администратор',
    manager: 'Управляющий',
    chef: 'Шеф-повар',
    waiter: 'Официант',
    cashier: 'Кассир',
  };

  const activeRest = RESTAURANTS.find(r => r.id === selectedRestaurant) || RESTAURANTS[0];

  return (
    <aside className="w-72 bg-[#0f172a] border-r border-slate-800 text-white flex flex-col h-full shrink-0 select-none">
      {/* Brand Section */}
      <div className="p-5 border-b border-slate-800 flex items-center gap-3">
        <div className="w-9 h-9 bg-blue-600 rounded-xl flex items-center justify-center font-black text-lg text-white shadow-md">
          M
        </div>
        <div>
          <h1 className="font-sans font-bold text-base tracking-tight text-white leading-none">
            MyHoreca <span className="text-blue-400 font-extrabold text-xs font-mono">RMS</span>
          </h1>
          <p className="font-mono text-[8px] tracking-wider text-slate-500 font-semibold uppercase mt-0.5">
            Back-Office Suite v2
          </p>
        </div>
      </div>

      {/* Restaurant Selector Control */}
      <div className="px-4 pt-4 pb-2 border-b border-slate-800">
        <label className="text-[9px] font-bold text-slate-500 uppercase tracking-widest block mb-1.5 font-mono">
          Активная торговая точка / Нода
        </label>
        <div className="relative">
          <Store className="w-4 h-4 text-slate-400 absolute left-3 top-1/2 -translate-y-1/2 pointer-events-none" />
          <select
            value={selectedRestaurant}
            onChange={(e) => onRestaurantChange(e.target.value)}
            className="w-full pl-9 pr-8 py-2.5 bg-slate-900 border border-slate-800 rounded-xl text-xs font-bold text-white focus:outline-none focus:border-blue-500 transition-colors appearance-none cursor-pointer"
          >
            {RESTAURANTS.map((r) => (
              <option key={r.id} value={r.id} className="bg-slate-900 text-white font-sans text-xs">
                {r.name}
              </option>
            ))}
          </select>
          <div className="absolute right-3 top-1/2 -translate-y-1/2 flex items-center gap-1.5 pointer-events-none">
            <span className={`w-2 h-2 rounded-full ${activeRest.color}`}></span>
            <ChevronDown className="w-3.5 h-3.5 text-slate-400" />
          </div>
        </div>
        
        {/* Short Code Tracker */}
        <div className="mt-2 px-1 flex justify-between text-[9px] font-mono text-slate-500">
          <span>Регион: РФ-ЦЕНТР</span>
          <span className="font-bold text-slate-400 uppercase">{activeRest.code}</span>
        </div>
      </div>

      {/* Navigation Links */}
      <nav className="flex-1 py-4 space-y-0.5 overflow-y-auto">
        <div className="px-5 py-1 text-[9px] font-extrabold text-slate-550 uppercase tracking-widest mb-1 font-mono">
          Разделы управления
        </div>
        {menuItems.map((item) => {
          const IconComponent = item.icon;
          const isActive = currentTab === item.id;
          return (
            <button
              key={item.id}
              onClick={() => setCurrentTab(item.id)}
              className={`w-full flex items-center gap-3 px-5 py-2.5 text-xs transition-all duration-200 border-l-4 ${
                isActive
                  ? 'bg-blue-600/10 text-blue-400 border-blue-500 font-extrabold'
                  : 'text-slate-400 hover:text-white hover:bg-slate-800/40 border-transparent'
              }`}
            >
              <IconComponent className={`w-4 h-4 shrink-0 transition-transform ${isActive ? 'scale-110' : ''}`} />
              <span>{item.label}</span>
            </button>
          );
        })}
      </nav>

      {/* User Section */}
      <div className="p-4 border-t border-slate-800 bg-slate-950/60 font-sans">
        <div className="flex items-center gap-3 p-3 bg-slate-900/50 rounded-xl border border-slate-800">
          <div className="w-7 h-7 rounded-lg bg-blue-600 flex items-center justify-center text-white font-bold text-xs shadow-inner shrink-0 leading-none">
            {staffName ? staffName.slice(0, 2).toUpperCase() : 'УС'}
          </div>
          <div className="flex-1 min-w-0">
            <h4 className="text-xs font-bold text-slate-100 truncate">
              {staffName}
            </h4>
            <p className="font-mono text-[8px] text-slate-500 uppercase tracking-wider font-semibold">
              {roleLabels[staffRole] || staffRole}
            </p>
          </div>
        </div>

        <div className="mt-3 px-1 flex items-center justify-between text-slate-500 text-[10px] font-mono">
          <div className="flex items-center gap-1.5">
            <span className="w-1.5 h-1.5 rounded-full bg-green-500 animate-pulse"></span>
            <span className="text-green-500 uppercase font-bold tracking-wider text-[9px]">Sync Active</span>
          </div>
          <span>v2.4.1</span>
        </div>
      </div>
    </aside>
  );
}
