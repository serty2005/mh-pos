import React, { useState, useEffect } from 'react';
import { POSProvider, usePOS } from './context/POSContext';
import { PinLogin } from './components/PinLogin';
import { POSFloorSection } from './components/floor/POSFloorSection';
import { POSOrderSection } from './components/menu/POSOrderSection';
import { POSActivitySection } from './components/activity/POSActivitySection';
import { POSReportsSection } from './components/reports/POSReportsSection';
import { POSCashSection } from './components/cash/POSCashSection';
import { t } from './shared/i18n';
import { 
  Armchair, 
  ReceiptText, 
  Clock, 
  Wallet, 
  BarChart3, 
  Lock, 
  Sun, 
  Moon, 
  Menu, 
  X, 
  ShieldAlert,
  Wifi,
  WifiOff
} from 'lucide-react';
import { PosButton } from './shared/ui';

function POSAppShellContent() {
  const {
    currentSection,
    setCurrentSection,
    theme,
    toggleTheme,
    currentOperator,
    logout,
    activeOrders,
    selectedTableId,
    tables,
    syncStatus,
    outboxCount,
    appVersion
  } = usePOS();

  const [isSideMenuOpen, setSideMenuOpen] = useState<boolean>(false);
  const [currentTime, setCurrentTime] = useState<string>('');

  // Clock snapshot runner
  useEffect(() => {
    const updateTime = () => {
      const now = new Date();
      setCurrentTime(now.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit', second: '2-digit' }));
    };
    updateTime();
    const timer = setInterval(updateTime, 1000);
    return () => clearInterval(timer);
  }, []);

  // Listen to Escape to shut side drawers
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        setSideMenuOpen(false);
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, []);

  const handleSectionSelect = (section: 'floor' | 'order' | 'activity' | 'reports' | 'cash') => {
    setCurrentSection(section);
    setSideMenuOpen(false);
  };

  const getActiveTableDescriptor = (): string => {
    if (!selectedTableId) return '';
    const table = tables.find(t => t.id === selectedTableId);
    return table ? `Стол ${table.number}` : '';
  };

  const renderCurrentSection = () => {
    switch (currentSection) {
      case 'floor':
        return <POSFloorSection />;
      case 'order':
        return <POSOrderSection />;
      case 'activity':
        return <POSActivitySection />;
      case 'reports':
        return <POSReportsSection />;
      case 'cash':
        return <POSCashSection />;
      default:
        return <POSFloorSection />;
    }
  };

  const navItems = [
    { id: 'floor', label: t.sections.floor, icon: Armchair },
    { id: 'order', label: t.sections.order, icon: UtensilsIcon },
    { id: 'activity', label: t.sections.activity, icon: ReceiptText },
    { id: 'reports', label: t.sections.reports, icon: BarChart3 },
    { id: 'cash', label: t.sections.cash, icon: Wallet }
  ] as const;

  function UtensilsIcon(props: any) {
    return <ReceiptText {...props} />; // fallback for dishes
  }

  return (
    <div className="h-screen w-screen flex flex-col overflow-hidden bg-[var(--pos-bg)] text-[var(--pos-text-primary)]">
      
      {/* 1. Header Context & Quick stats bar */}
      <header className="h-14 border-b border-[var(--pos-border)] bg-[var(--pos-surface)] px-4 flex items-center justify-between select-none shrink-0">
        
        {/* Left operator info */}
        <div className="flex items-center gap-3">
          <button 
            id="sidemenu-trigger-btn"
            onClick={() => setSideMenuOpen(true)}
            className="w-10 h-10 border border-[var(--pos-border)] flex items-center justify-center bg-[var(--pos-surface-raised)] hover:bg-[var(--pos-border)] cursor-pointer rounded-none active:bg-[var(--pos-border-strong)]"
            aria-label="Toggle Side Menu"
          >
            <Menu className="w-5 h-5" />
          </button>
          
          <h1 className="text-sm md:text-lg font-sans font-semibold tracking-normal text-[var(--pos-text-secondary)] mr-1 shrink-0">MyHoreca POS</h1>
          <div className="h-6 w-px bg-[var(--pos-border)] hidden sm:block"></div>
          
          <div className="hidden sm:flex flex-col">
            <span className="font-mono text-[9px] font-bold text-[var(--pos-text-muted)] uppercase tracking-widest leading-none">
              {currentOperator?.role === 'manager' ? 'Администратор' : currentOperator?.role === 'cashier' ? 'Кассир' : 'Официант'}
            </span>
            <span className="font-sans text-xs font-bold text-[var(--pos-text-secondary)] mt-0.5 leading-tight">
              {currentOperator ? currentOperator.employeeName.split(' ')[0] : 'Сотрудник'}
            </span>
          </div>
        </div>

        {/* Center Room / table Context descriptor */}
        <div className="flex items-center gap-3 font-mono text-xs select-none">
          {getActiveTableDescriptor() && (
            <div className="bg-[var(--pos-action-primary)] text-[var(--pos-surface)] px-3 py-1 font-bold animate-fade-in uppercase tracking-wider">
              {getActiveTableDescriptor()}
            </div>
          )}
          {activeOrders.length > 0 && (
            <div className="hidden md:flex items-center gap-1 bg-[var(--pos-action-secondary)] border border-[var(--pos-border)] px-3 py-1 text-[var(--pos-text-secondary)] font-semibold uppercase tracking-wider">
              <span>Заказы: {activeOrders.length}</span>
            </div>
          )}
        </div>

        {/* Right tools (Sync, Lock, Time, Light/Dark) */}
        <div className="flex items-center gap-3 font-mono text-xs select-none shrink-0">
          
          {/* UTC Clock */}
          <span className="hidden sm:block font-bold text-[var(--pos-text-secondary)] bg-[var(--pos-action-secondary)] px-3 py-1.5 border border-[var(--pos-border)] tracking-widest text-[11px]">
            {currentTime}
          </span>

          {/* Sync indicator */}
          <div 
            className={`w-10 h-10 border border-[var(--pos-border)] flex items-center justify-center rounded-none
              ${syncStatus === 'online' ? 'text-[var(--pos-status-success)]' : 'text-[var(--pos-status-danger)] animate-pulse'}`}
            title={syncStatus === 'online' ? 'Терминал подключен к Cloud' : 'Автономный режим'}
          >
            {syncStatus === 'online' ? <Wifi className="w-4 h-4" /> : <WifiOff className="w-4 h-4" />}
          </div>

          {/* Theme Switcher Toggle */}
          <button
            id="theme-toggler-btn"
            onClick={toggleTheme}
            className="w-10 h-10 border border-[var(--pos-border)] hover:bg-[var(--pos-border)] flex items-center justify-center cursor-pointer rounded-none outline-none text-[var(--pos-text-secondary)]"
            aria-label="Toggle Theme"
          >
            {theme === 'light' ? <Moon className="w-4 h-4" /> : <Sun className="w-4 h-4" />}
          </button>

          {/* Lock Screen */}
          <button
            id="lock-terminal-btn"
            onClick={logout}
            className="h-10 px-4 border border-[var(--pos-status-danger)] bg-red-500/10 hover:bg-red-500 hover:text-white text-[var(--pos-status-danger)] flex items-center gap-1.5 font-bold cursor-pointer uppercase text-[10px] tracking-wider transition-colors rounded-none"
          >
            <Lock className="w-3.5 h-3.5 shrink-0" />
            <span className="hidden sm:inline">Замок</span>
          </button>

        </div>

      </header>

      {/* 2. Primary section workspace container */}
      <main className="flex-1 flex min-h-0 min-w-0 relative">
        {renderCurrentSection()}
      </main>

      {/* 3. Bottom quick access navigation bar */}
      <footer className="h-14 border-t border-[var(--pos-border)] bg-[var(--pos-surface)] flex items-center justify-between select-none shrink-0 select-none">
        <div className="flex w-full divide-x divide-[var(--pos-border)] h-full overflow-hidden">
          {navItems.map((item) => {
            const isActive = currentSection === item.id;
            const Icon = item.icon;
            return (
              <button
                key={item.id}
                id={`nav-${item.id}`}
                onClick={() => handleSectionSelect(item.id)}
                className={`flex-1 h-full font-mono text-center flex flex-col md:flex-row items-center justify-center gap-1.5 border-t-2 select-none whitespace-nowrap cursor-pointer transition-colors
                  ${isActive 
                    ? 'bg-[var(--pos-surface-raised)] border-t-[var(--pos-action-primary)] text-[var(--pos-text-primary)] font-black' 
                    : 'bg-transparent border-t-transparent text-[var(--pos-text-muted)] hover:text-[var(--pos-text-primary)]'
                  }`}
              >
                <Icon className={`w-4 h-4 md:w-5 md:h-5 shrink-0 ${isActive ? 'text-[var(--pos-text-primary)]' : 'text-[var(--pos-text-muted)]'}`} />
                <span className="text-[10px] md:text-xs font-bold uppercase tracking-widest">{item.label}</span>
              </button>
            );
          })}
        </div>
      </footer>

      {/* 4. Collapsible side section menu drawers overlay */}
      {isSideMenuOpen && (
        <div className="fixed inset-0 z-50 flex">
          {/* Clickable Backdrop overlay drawer */}
          <div 
            className="fixed inset-0 bg-neutral-950/60 transition-opacity duration-200"
            onClick={() => setSideMenuOpen(false)}
          />

          {/* Sliding box Container */}
          <div className="relative w-full max-w-[280px] sm:max-w-[320px] bg-[var(--pos-surface)] border-r border-[var(--pos-border)] h-full flex flex-col justify-between shadow-2xl z-10 animate-slide-in select-none">
            
            {/* Drawer top list */}
            <div className="flex flex-col min-h-0">
              {/* Profile card summary */}
              <div className="p-6 border-b border-[var(--pos-border)] bg-[var(--pos-surface-raised)] flex items-center justify-between">
                <div className="flex items-center gap-3 min-w-0">
                  <div className="w-10 h-10 bg-[var(--pos-action-primary)] flex items-center justify-center p-2 rounded-none shrink-0 text-white font-mono font-bold">
                    {currentOperator ? currentOperator.employeeName.charAt(0) : 'U'}
                  </div>
                  <div className="min-w-0">
                    <h4 className="font-sans text-xs md:text-sm font-bold text-[var(--pos-text-primary)] leading-tight truncate">
                      {currentOperator ? currentOperator.employeeName : 'Гость'}
                    </h4>
                    <span className="font-mono text-[9px] font-bold text-[var(--pos-text-muted)] uppercase tracking-wider leading-none mt-1 block">
                      Смена: {currentOperator ? currentOperator.id : 'Закрыта'}
                    </span>
                  </div>
                </div>

                <button 
                  id="close-drawer-btn"
                  onClick={() => setSideMenuOpen(false)}
                  className="w-8 h-8 hover:bg-[var(--pos-border)] border border-[var(--pos-border)] flex items-center justify-center cursor-pointer rounded-none"
                  aria-label="Close"
                >
                  <X className="w-4 h-4" />
                </button>
              </div>

              {/* Navigator options */}
              <div className="py-4 space-y-1">
                {navItems.map((item) => {
                  const isActive = currentSection === item.id;
                  const Icon = item.icon;
                  return (
                    <button
                      key={item.id}
                      id={`drawer-nav-${item.id}`}
                      onClick={() => handleSectionSelect(item.id)}
                      className={`w-full h-12 px-6 flex items-center gap-4 text-left font-mono font-semibold transition-colors border-none cursor-pointer
                        ${isActive 
                          ? 'bg-[var(--pos-action-secondary)] text-[var(--pos-text-primary)] font-bold border-l-4 border-l-[var(--pos-action-primary)]' 
                          : 'bg-transparent text-[var(--pos-text-secondary)] hover:bg-[var(--pos-surface-raised)]'
                        }`}
                    >
                      <Icon className="w-4.5 h-4.5 shrink-0" />
                      <span className="text-xs uppercase tracking-wider">{item.label}</span>
                    </button>
                  );
                })}
              </div>
            </div>

            {/* Shift and Outbox diagnostics summary in bottom drawers */}
            <div className="p-6 border-t border-[var(--pos-border)] bg-[var(--pos-surface-raised)]/30 space-y-4">
              
              {/* Safeguard warnings */}
              {currentOperator?.role === 'waiter' && (
                <div className="p-3 border border-red-200 bg-red-50/10 text-red-500 rounded-none flex items-start gap-1.5 text-[10px]">
                  <ShieldAlert className="w-3.5 h-3.5 text-red-500 shrink-0 mt-0.5" />
                  <span>Режим официанта: права на прием платежей и возврат ограничены.</span>
                </div>
              )}

              {/* Mini Diagnostic strip */}
              <div className="space-y-1 font-mono text-[9px] text-[var(--pos-text-muted)] uppercase tracking-widest leading-none select-none">
                <div>Версия БД: {appVersion}</div>
                <div>Буфер Outbox: {outboxCount} событий</div>
                <div className="flex items-center gap-1 mt-1 font-bold">
                  <div className={`w-1.5 h-1.5 rounded-full ${syncStatus === 'online' ? 'bg-emerald-500' : 'bg-red-500 animate-pulse'}`} />
                  <span>{syncStatus === 'online' ? 'Связь с Cloud стабильна' : 'Работа в офлаин-режиме'}</span>
                </div>
              </div>

              {/* Relock entry */}
              <PosButton
                id="drawer-lock-btn"
                variant="danger"
                size="sm"
                fullWidth
                onClick={() => {
                  setSideMenuOpen(false);
                  logout();
                }}
                icon={<Lock className="w-4 h-4" />}
              >
                {t.auth.logout}
              </PosButton>
            </div>

          </div>
        </div>
      )}

    </div>
  );
}

// Wrap in Auth Lock logic
function POSAppShell() {
  const { isPinLocked } = usePOS();
  
  if (isPinLocked) {
    return <PinLogin />;
  }

  return <POSAppShellContent />;
}

export default function App() {
  return (
    <POSProvider>
      <POSAppShell />
    </POSProvider>
  );
}
