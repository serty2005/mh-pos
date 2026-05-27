/**
 * @license
 * SPDX-License-Identifier: Apache-2.0
 */

import { LayoutDashboard, Utensils, Users, RefreshCw, LogOut, Clock, Landmark } from 'lucide-react';

interface SidebarProps {
  currentTab: string;
  setCurrentTab: (tab: string) => void;
  staffName: string;
  staffRole: string;
}

export default function Sidebar({ currentTab, setCurrentTab, staffName, staffRole }: SidebarProps) {
  const menuItems = [
    { id: 'analytics', label: 'Аналитика продаж', icon: LayoutDashboard },
    { id: 'menu', label: 'Управление меню', icon: Utensils },
    { id: 'staff', label: 'Персонал & Доступ', icon: Users },
    { id: 'sync', label: 'Синхронизация POS', icon: RefreshCw },
  ];

  const roleLabels: Record<string, string> = {
    admin: 'Администратор',
    manager: 'Управляющий',
    chef: 'Шеф-повар',
    waiter: 'Официант',
    cashier: 'Кассир',
  };

  return (
    <aside className="w-72 bg-[#0f172a] border-r border-slate-800 text-white flex flex-col h-full shrink-0">
      {/* Brand Section */}
      <div className="p-6 border-b border-slate-700/60 flex items-center gap-3">
        <div className="w-9 h-9 bg-blue-500 rounded-lg flex items-center justify-center font-bold text-lg text-white shadow-md">
          M
        </div>
        <div>
          <h1 className="font-sans font-bold text-lg tracking-tight text-white">
            MyHoreca <span className="text-blue-400 font-extrabold text-xl font-mono">RMS</span>
          </h1>
          <p className="font-mono text-[9px] tracking-widest text-slate-500 font-semibold uppercase">
            Management Suite
          </p>
        </div>
      </div>

      {/* Navigation Links */}
      <nav className="flex-1 py-6 space-y-1">
        <div className="px-6 py-2 text-[10px] font-bold text-slate-500 uppercase tracking-wider mb-2">
          Управление системой
        </div>
        {menuItems.map((item) => {
          const IconComponent = item.icon;
          const isActive = currentTab === item.id;
          return (
            <button
              key={item.id}
              onClick={() => setCurrentTab(item.id)}
              className={`w-full flex items-center gap-3 px-6 py-3.5 text-sm transition-all duration-200 border-l-4 ${
                isActive
                  ? 'bg-blue-600/10 text-blue-400 border-blue-500 font-semibold'
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
      <div className="p-4 border-t border-slate-800 bg-slate-950/60">
        <div className="flex items-center gap-3 p-3 bg-slate-900/50 rounded-xl border border-slate-800">
          <div className="w-9 h-9 rounded-lg bg-blue-600 flex items-center justify-center text-white font-bold text-sm shadow-inner">
            {staffName ? staffName.slice(0, 2).toUpperCase() : 'УС'}
          </div>
          <div className="flex-1 min-w-0">
            <h4 className="font-sans text-xs font-bold text-slate-100 truncate">
              {staffName}
            </h4>
            <p className="font-mono text-[9px] text-slate-500 uppercase tracking-wider font-semibold">
              {roleLabels[staffRole] || staffRole}
            </p>
          </div>
        </div>

        <div className="mt-3 px-1 flex items-center justify-between text-slate-500 text-[10px] font-mono">
          <div className="flex items-center gap-1.5">
            <span className="w-1.5 h-1.5 rounded-full bg-green-500 animate-pulse"></span>
            <span className="text-green-500 uppercase font-bold tracking-wider">Sync: Online</span>
          </div>
          <span>v2.4.1</span>
        </div>
      </div>
    </aside>
  );
}
