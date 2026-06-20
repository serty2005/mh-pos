import React, { useState, useEffect, useMemo } from 'react';
import { POSProvider, usePOS } from './context/POSContext';
import { PairingScreen } from './components/PairingScreen';
import { PinLogin } from './components/PinLogin';
import { POSFloorSection } from './components/floor/POSFloorSection';
import { POSOrderSection } from './components/menu/POSOrderSection';
import { POSActivitySection } from './components/activity/POSActivitySection';
import { POSReportsSection } from './components/reports/POSReportsSection';
import { POSCashSection } from './components/cash/POSCashSection';
import { POSKitchenSection } from './components/kitchen/POSKitchenSection';
import { POSWaiterSection } from './components/waiter/POSWaiterSection';
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
  Palette,
  Menu, 
  X, 
  ShieldAlert,
  Wifi,
  WifiOff,
  ChefHat,
  Smartphone,
  Truck,
  Monitor
} from 'lucide-react';
import { PosBottomNav, PosButton, PosIconButton, PosInlineStatusBadge, PosSelectableChip, PosSelectableTile } from './shared/ui';
import type { POSSection } from './types';
import { createApiClient } from './shared/api';

function POSAppShellContent() {
  const {
    currentSection,
    setCurrentSection,
    theme,
    themeScheme,
    themeSchemes,
    setThemeMode,
    setThemeScheme,
    currentOperator,
    logout,
    activeOrders,
    selectedTableId,
    tables,
    syncStatus,
    syncRevision,
    outboxCount,
    appVersion,
    authSnapshot,
  } = usePOS();

  const [isSideMenuOpen, setSideMenuOpen] = useState<boolean>(false);
  const [currentTime, setCurrentTime] = useState<string>('');
  const [currentMode, setCurrentMode] = useState<TerminalMode>(() => isMobileViewport() ? 'waiter' : 'pos');
  const [currentKdsSection, setCurrentKdsSection] = useState<'orders' | 'stock' | 'kitchen'>('orders');
  const [entitlements, setEntitlements] = useState<Record<string, boolean>>({});
  const licenseApi = useMemo(() => createApiClient(() => authSnapshot), [authSnapshot]);

  useEffect(() => {
    void licenseApi.getEntitlements().then((snapshot) => {
      if (snapshot.status === 'active' && new Date(snapshot.expires_at).getTime() > Date.now()) setEntitlements(snapshot.entitlements);
    }).catch(() => setEntitlements({}));
  }, [licenseApi]);

  useEffect(() => {
    if (currentSection === 'floor' && entitlements['table-mode'] !== true) setCurrentSection('order');
    if (currentMode === 'kds' && entitlements['kitchen-space'] !== true) setCurrentMode('pos');
    if (currentMode === 'waiter' && entitlements['waiter-space'] !== true) setCurrentMode('pos');
    if (currentKdsSection === 'stock' && entitlements['warehouse-mode'] !== true) setCurrentKdsSection('orders');
  }, [currentKdsSection, currentMode, currentSection, entitlements, setCurrentSection]);

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

  useEffect(() => {
    const mediaQuery = window.matchMedia('(max-width: 639px)');
    const handleMobileHandoff = () => {
      if (mediaQuery.matches) {
        setCurrentMode((mode) => mode === 'pos' || mode === 'kds' ? 'waiter' : mode);
      }
    };
    handleMobileHandoff();
    mediaQuery.addEventListener('change', handleMobileHandoff);
    return () => mediaQuery.removeEventListener('change', handleMobileHandoff);
  }, []);

  const handleSectionSelect = (section: POSSection) => {
    setCurrentSection(section);
    setSideMenuOpen(false);
  };

  const handleModeSelect = (mode: TerminalMode) => {
    setCurrentMode(mode);
    setSideMenuOpen(false);
  };

  const getActiveTableDescriptor = (): string => {
    if (!selectedTableId) return '';
    const table = tables.find((item) => item.id === selectedTableId);
    return table ? `${t.common.table} ${table.number}` : '';
  };

  const getOperatorRoleLabel = (): string => {
    switch (currentOperator?.role) {
      case 'manager':
        return t.shell.roleManager;
      case 'cashier':
        return t.shell.roleCashier;
      case 'kitchen':
        return t.shell.roleKitchen;
      case 'support':
        return t.shell.roleSupport;
      case 'waiter':
      default:
        return t.shell.roleWaiter;
    }
  };

  const renderCurrentSection = () => {
    if (currentMode === 'kds') {
      return <POSKitchenSection section={currentKdsSection} />;
    }
    if (currentMode === 'waiter') {
      return <POSWaiterSection />;
    }
    if (currentMode === 'delivery') {
      return <ModePlaceholder title={t.modes.deliveryBacklog} tone="warning" />;
    }
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
  ].filter((item) => item.id !== 'floor' || entitlements['table-mode'] === true);

  const modeItems = [
    { id: 'pos', label: t.modes.pos, icon: Monitor },
    { id: 'kds', label: t.modes.kds, icon: ChefHat },
    { id: 'waiter', label: t.modes.waiter, icon: Smartphone },
    { id: 'delivery', label: t.modes.delivery, icon: Truck, badge: t.status.notAgreed },
  ].filter((item) => (item.id !== 'kds' || entitlements['kitchen-space'] === true) && (item.id !== 'waiter' || entitlements['waiter-space'] === true));

  const kdsNavItems = [
    { id: 'orders', label: t.kitchen.navOrders, icon: ReceiptText },
    { id: 'stock', label: t.kitchen.navStock, icon: Truck },
    { id: 'kitchen', label: t.kitchen.navKitchen, icon: ChefHat },
  ].filter((item) => item.id !== 'stock' || entitlements['warehouse-mode'] === true);

  const renderBottomNavigation = () => {
    if (currentMode === 'pos') {
      return (
        <PosBottomNav
          items={[...navItems]}
          activeId={currentSection}
          idPrefix="nav"
          onChange={(id) => handleSectionSelect(id as POSSection)}
        />
      );
    }

    if (currentMode === 'kds') {
      return (
        <PosBottomNav
          items={[...kdsNavItems]}
          activeId={currentKdsSection}
          idPrefix="nav-kds"
          horizontal
          onChange={(id) => setCurrentKdsSection(id as 'orders' | 'stock' | 'kitchen')}
        />
      );
    }

    return (
      <div className="flex-1 h-full flex items-center justify-center font-mono text-xs font-bold uppercase tracking-widest text-[var(--pos-text-muted)]">
        {modeItems.find((item) => item.id === currentMode)?.label}
      </div>
    );
  };

  function UtensilsIcon(props: any) {
    return <ReceiptText {...props} />; // fallback for dishes
  }

  return (
    <div className="h-screen w-screen flex flex-col overflow-hidden bg-[var(--pos-bg)] text-[var(--pos-text-primary)]">
      
      {/* 1. Header Context & Quick stats bar */}
      <header className="h-14 border-b border-[var(--pos-border)] bg-[var(--pos-surface)] px-4 flex items-center justify-between select-none shrink-0">
        
        {/* Left operator info */}
        <div className="flex items-center gap-3">
          <PosIconButton
            id="sidemenu-trigger-btn"
            onClick={() => setSideMenuOpen(true)}
            label={t.shell.openMenu}
            icon={<Menu className="w-5 h-5" />}
          />
          
          <h1 className="text-sm md:text-lg font-sans font-semibold tracking-normal text-[var(--pos-text-secondary)] mr-1 shrink-0">{t.app.title}</h1>
          <div className="h-6 w-px bg-[var(--pos-border)] hidden sm:block"></div>
          
          <div className="hidden sm:flex flex-col">
            <span className="font-mono text-[9px] font-bold text-[var(--pos-text-muted)] uppercase tracking-widest leading-none">
              {getOperatorRoleLabel()}
            </span>
            <span className="font-sans text-xs font-bold text-[var(--pos-text-secondary)] mt-0.5 leading-tight">
              {currentOperator ? currentOperator.employeeName.split(' ')[0] : t.shell.employeeFallback}
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
              <span>{t.shell.orders}: {activeOrders.length}</span>
            </div>
          )}
        </div>

        {/* Right tools (Sync, Lock, Time) */}
        <div className="flex items-center gap-3 font-mono text-xs select-none shrink-0">
          
          {/* UTC Clock */}
          <span className="hidden sm:block font-bold text-[var(--pos-text-secondary)] bg-[var(--pos-action-secondary)] px-3 py-1.5 border border-[var(--pos-border)] tracking-widest text-[11px]">
            {currentTime}
          </span>

          {/* Sync indicator */}
          <div 
            className={`w-10 h-10 border border-[var(--pos-border)] flex items-center justify-center rounded-none
              ${syncStatus === 'online' ? 'text-[var(--pos-status-success)]' : 'text-[var(--pos-status-danger)] animate-pulse'}`}
            title={syncStatus === 'online' ? t.modes.terminalOnlineTitle : t.modes.terminalOfflineTitle}
          >
            {syncStatus === 'online' ? <Wifi className="w-4 h-4" /> : <WifiOff className="w-4 h-4" />}
          </div>

          {/* Lock Screen */}
          <PosButton
            id="lock-terminal-btn"
            onClick={logout}
            variant="danger"
            size="sm"
            className="h-10 text-[10px]"
            icon={<Lock className="w-3.5 h-3.5 shrink-0" />}
          >
            <span className="hidden sm:inline">{t.shell.lock}</span>
          </PosButton>

        </div>

      </header>

      {/* 2. Primary section workspace container */}
      <main className="flex-1 flex min-h-0 min-w-0 relative">
        {renderCurrentSection()}
      </main>

      {/* 3. Bottom quick access navigation bar */}
      <footer className="h-14 border-t border-[var(--pos-border)] bg-[var(--pos-surface)] flex items-center justify-between select-none shrink-0 select-none">
        {renderBottomNavigation()}
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
                      {currentOperator ? currentOperator.employeeName : t.shell.guest}
                    </h4>
                    <span className="font-mono text-[9px] font-bold text-[var(--pos-text-muted)] uppercase tracking-wider leading-none mt-1 block">
                      {t.shell.shift}: {currentOperator ? currentOperator.id : t.status.closed}
                    </span>
                  </div>
                </div>

                <PosIconButton
                  id="close-drawer-btn"
                  onClick={() => setSideMenuOpen(false)}
                  size="sm"
                  label={t.common.close}
                  icon={<X className="w-4 h-4" />}
                />
              </div>

              {/* Mode navigator options */}
              <div className="py-4 space-y-1">
                {modeItems.map((item) => {
                  const isActive = currentMode === item.id;
                  const Icon = item.icon;
                  return (
                    <ShellDrawerButton
                      key={item.id}
                      id={`drawer-mode-${item.id}`}
                      label={item.label}
                      icon={Icon}
                      active={isActive}
                      badge={'badge' in item ? item.badge : undefined}
                      onClick={() => handleModeSelect(item.id as TerminalMode)}
                    />
                  );
                })}
              </div>

              <div className="mx-6 mb-4 border border-[var(--pos-border)] bg-[var(--pos-surface-raised)]/45">
                <div className="px-4 py-3 border-b border-[var(--pos-border)] flex items-center justify-between gap-3">
                  <div className="flex items-center gap-2 min-w-0">
                    <Palette className="w-4 h-4 text-[var(--pos-text-secondary)] shrink-0" />
                    <span className="font-mono text-[10px] font-black uppercase tracking-widest text-[var(--pos-text-secondary)]">
                      {t.theme.title}
                    </span>
                  </div>
                  <div className="flex border border-[var(--pos-border)] bg-[var(--pos-surface)]">
                    <PosSelectableChip
                      id="drawer-theme-dark-btn"
                      active={theme === 'dark'}
                      onClick={() => setThemeMode('dark')}
                      className="h-8 px-2.5 flex items-center gap-1.5 text-[10px] border-0"
                    >
                      <Moon className="w-3.5 h-3.5" />
                      <span className="hidden sm:inline">{t.theme.modeDark}</span>
                    </PosSelectableChip>
                    <PosSelectableChip
                      id="drawer-theme-light-btn"
                      active={theme === 'light'}
                      onClick={() => setThemeMode('light')}
                      className="h-8 px-2.5 flex items-center gap-1.5 text-[10px] border-0"
                    >
                      <Sun className="w-3.5 h-3.5" />
                      <span className="hidden sm:inline">{t.theme.modeLight}</span>
                    </PosSelectableChip>
                  </div>
                </div>

                <div className="p-3">
                  <div className="font-mono text-[9px] text-[var(--pos-text-muted)] uppercase tracking-widest mb-2">
                    {t.theme.schemeTitle}
                  </div>
                  <div className="grid grid-cols-2 gap-2">
                    {themeSchemes.map((scheme) => {
                      const isActive = themeScheme === scheme.id;
                      return (
                        <PosSelectableChip
                          key={scheme.id}
                          id={`drawer-theme-scheme-${scheme.id}`}
                          active={isActive}
                          onClick={() => setThemeScheme(scheme.id)}
                          className="h-10 px-2 flex items-center gap-2 text-left justify-start"
                        >
                          <span
                            className="w-3 h-3 shrink-0 border border-white/30"
                            style={{ backgroundColor: scheme.accent }}
                            aria-hidden="true"
                          />
                          <span className="font-mono text-[9px] font-bold uppercase tracking-wider truncate">
                            {t.theme.schemes[scheme.id]}
                          </span>
                        </PosSelectableChip>
                      );
                    })}
                  </div>
                </div>
              </div>
            </div>

            {/* Shift and Outbox diagnostics summary in bottom drawers */}
            <div className="p-6 border-t border-[var(--pos-border)] bg-[var(--pos-surface-raised)]/30 space-y-4">
              
              {/* Safeguard warnings */}
              {currentOperator?.role === 'waiter' && (
                <div className="p-3 border border-red-200 bg-red-50/10 text-red-500 rounded-none flex items-start gap-1.5 text-[10px]">
                  <ShieldAlert className="w-3.5 h-3.5 text-red-500 shrink-0 mt-0.5" />
                  <span>{t.blocks.waiterNotice}</span>
                </div>
              )}

              {/* Mini Diagnostic strip */}
              <div className="space-y-1.5 font-mono text-[9px] text-[var(--pos-text-muted)] uppercase tracking-widest leading-none select-none">
                <div className="font-bold text-[var(--pos-text-secondary)]">{t.modes.serviceData}</div>
                <div>{t.modes.cloudRevision}: {syncRevision || t.common.notAvailable}</div>
                <div>{t.modes.edgeDbVersion}: {appVersion}</div>
                <div>{t.modes.eventBuffer}: {outboxCount}</div>
                <div className="flex items-center gap-1 mt-1 font-bold">
                  <div className={`w-1.5 h-1.5 rounded-full ${syncStatus === 'online' ? 'bg-emerald-500' : 'bg-red-500 animate-pulse'}`} />
                  <span>{syncStatus === 'online' ? t.modes.connectionOnline : t.modes.connectionOffline}</span>
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

type TerminalMode = 'pos' | 'kds' | 'waiter' | 'delivery';

function isMobileViewport() {
  return typeof window !== 'undefined' && window.matchMedia('(max-width: 639px)').matches;
}

type ShellIcon = React.ComponentType<{ className?: string }>;
interface ShellDrawerButtonProps {
  id: string;
  label: string;
  icon: ShellIcon;
  active: boolean;
  badge?: string;
  onClick: () => void;
  key?: React.Key;
}

function ShellDrawerButton({
  id,
  label,
  icon: Icon,
  active,
  badge,
  onClick,
}: ShellDrawerButtonProps) {
  return (
    <PosSelectableTile
      id={id}
      active={active}
      onClick={onClick}
      className={`w-full h-12 px-6 flex items-center gap-4 text-left font-mono font-semibold transition-colors border-none cursor-pointer ${
        active
          ? 'bg-[var(--pos-action-secondary)] text-[var(--pos-text-primary)] font-bold border-l-4 border-l-[var(--pos-action-primary)]'
          : 'bg-transparent text-[var(--pos-text-secondary)] hover:bg-[var(--pos-surface-raised)]'
      }`}
    >
      <Icon className="w-4.5 h-4.5 shrink-0" />
      <span className="text-xs uppercase tracking-wider">{label}</span>
      {badge && (
        <PosInlineStatusBadge variant="warning" className="ml-auto text-[8px] px-1 py-0">
          {badge}
        </PosInlineStatusBadge>
      )}
    </PosSelectableTile>
  );
}

function ModePlaceholder({ title, tone = 'neutral' }: { title: string; tone?: 'neutral' | 'warning' }) {
  return (
    <div className="flex-1 min-h-0 flex items-center justify-center bg-[var(--pos-bg)] p-8">
      <div className={`max-w-md w-full border ${tone === 'warning' ? 'border-amber-300 text-amber-600' : 'border-[var(--pos-border)] text-[var(--pos-text-secondary)]'} bg-[var(--pos-surface)] p-8 text-center`}>
        <span className="font-mono text-xs font-black uppercase tracking-widest">{title}</span>
      </div>
    </div>
  );
}

// Wrap in Auth Lock logic
function POSAppShell() {
  const { isEdgePaired, isPinLocked } = usePOS();
  
  if (!isEdgePaired) {
    return <PairingScreen />;
  }
  
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
