import { useState, useCallback } from 'react';
import Sidebar from './components/Sidebar';
import AnalyticsPanel from './components/AnalyticsPanel';
import MenuPanel from './components/MenuPanel';
import TablesPanel from './components/TablesPanel';
import StopListPanel from './components/StopListPanel';
import StaffPanel from './components/StaffPanel';
import EdgeDevicesPanel from './components/EdgeDevicesPanel';
import WarehousePanel from './components/WarehousePanel';
import SyncPanel from './components/SyncPanel';

import {
  INITIAL_CATEGORIES,
  INITIAL_MENU_ITEMS,
  INITIAL_STAFF,
  INITIAL_TERMINALS,
  INITIAL_SYNC_LOGS,
  INITIAL_TRANSACTIONS
} from './data/mockData';
import { MenuItem, MenuCategory, StaffMember, POSTerminal, SyncLog, SalesTransaction } from './types';

export default function App() {
  const [currentTab, setCurrentTab] = useState<string>('analytics');
  const [selectedRestaurant, setSelectedRestaurant] = useState<string>('all');

  // Core application states (fully synchronized)
  const [menuItems, setMenuItems] = useState<MenuItem[]>(INITIAL_MENU_ITEMS);
  const [categories] = useState<MenuCategory[]>(INITIAL_CATEGORIES);
  const [staffList, setStaffList] = useState<StaffMember[]>(INITIAL_STAFF);
  const [terminals, setTerminals] = useState<POSTerminal[]>(INITIAL_TERMINALS);
  const [syncLogs, setSyncLogs] = useState<SyncLog[]>(INITIAL_SYNC_LOGS);
  const [transactions, setTransactions] = useState<SalesTransaction[]>(INITIAL_TRANSACTIONS);

  // Simulation indicators
  const [isSimulatingSale, setIsSimulatingSale] = useState(false);
  const [isSyncingAll, setIsSyncingAll] = useState(false);

  // Current logged in manager mock user (first admin)
  const currentMockUser = staffList.find((s) => s.role === 'admin' || s.role === 'manager') || staffList[0];

  // Helper to push a new real-time log to the console
  const appendLog = useCallback((
    terminalId: string,
    terminalName: string,
    type: SyncLog['type'],
    status: SyncLog['status'],
    details: string
  ) => {
    const freshLog: SyncLog = {
      id: `log-${Date.now()}`,
      timestamp: new Date().toTimeString().split(' ')[0],
      terminalId,
      terminalName,
      type,
      status,
      details,
    };
    setSyncLogs((prev) => [freshLog, ...prev]);
  }, []);

  // 1. MENU POSITION MANAGEMENT ACTIONS
  const handleAddMenuItem = (item: MenuItem) => {
    setMenuItems((prev) => [item, ...prev]);
    appendLog(
      'term-main',
      'Основная консоль / Менеджер',
      'menu_push',
      'success',
      `Добавлено новое блюдо: "${item.name}" за ${item.price} ₽. Распространено в очередь синхронизации.`
    );
  };

  const handleUpdateMenuItem = (item: MenuItem) => {
    setMenuItems((prev) => prev.map((mi) => (mi.id === item.id ? item : mi)));
    appendLog(
      'term-main',
      'Основная консоль / Менеджер',
      'menu_push',
      'success',
      `Внесены изменения в позицию: "${item.name}". Статус наличия: ${item.isAvailable ? 'доступно' : 'ограничено'}.`
    );
  };

  const handleDeleteMenuItem = (id: string) => {
    const targetItem = menuItems.find((m) => m.id === id);
    setMenuItems((prev) => prev.filter((mi) => mi.id !== id));
    if (targetItem) {
      appendLog(
        'term-main',
        'Основная консоль / Менеджер',
        'menu_push',
        'warning',
        `Изъято из меню: "${targetItem.name}". Терминалы будут оповещены на следующем цикле пинга.`
      );
    }
  };

  // 2. STAFF DIRECTORY ACTIONS
  const handleAddStaff = (newStaff: StaffMember) => {
    setStaffList((prev) => [...prev, newStaff]);
    appendLog(
      'sys',
      'Ядро системы доступа',
      'staff_update',
      'success',
      `Добавлен новый пользователь: "${newStaff.name}" (${newStaff.role}). Свежий токен доступа сгенерирован.`
    );
  };

  const handleUpdateStaff = (updatedStaff: StaffMember) => {
    setStaffList((prev) => prev.map((sm) => (sm.id === updatedStaff.id ? updatedStaff : sm)));
    appendLog(
      'sys',
      'Ядро системы доступа',
      'staff_update',
      'success',
      `Изменен статус сотрудника "${updatedStaff.name}". Рабочее состояние обновлено.`
    );
  };

  const handleDeleteStaff = (id: string) => {
    const targetMember = staffList.find((s) => s.id === id);
    setStaffList((prev) => prev.filter((sm) => sm.id !== id));
    if (targetMember) {
      appendLog(
        'sys',
        'Ядро системы доступа',
        'staff_update',
        'warning',
        `Сотрудник "${targetMember.name}" уволен/удален из реестра. Кодовые терминальные пропуска отозваны.`
      );
    }
  };

  // 3. FORCE MANUAL SYNCHRONIZATION ALGORITHM
  const triggerForceSync = useCallback(() => {
    if (isSyncingAll) return;
    setIsSyncingAll(true);

    appendLog('system', 'Центральный сервер RMS', 'heartbeat', 'success', 'Запущена волна принудительной синхронизации терминальной сети...');

    setTerminals((prev) =>
      prev.map((term) => (term.status === 'online' ? { ...term, status: 'syncing' } : term))
    );

    setTimeout(() => {
      appendLog(
        'system',
        'Служба дистрибуции меню',
        'menu_push',
        'success',
        `Разосланы спецификации меню (${menuItems.length} позиций) на терминалы Бар, Веранда.`
      );

      setTimeout(() => {
        setTerminals((prev) =>
          prev.map((term) => {
            if (term.id === 'term-hall' && term.pendingTransactions > 0) {
              appendLog(
                term.id,
                term.name,
                'sales_pull',
                'success',
                `Выгружено ${term.pendingTransactions} чека из локальной оффлайн БД. Проведена сверка хэшей транзакций.`
              );
              return { ...term, pendingTransactions: 0, lastSyncTime: 'Только что' };
            }
            return term.status === 'syncing' ? { ...term, lastSyncTime: 'Только что' } : term;
          })
        );

        setTimeout(() => {
          setTerminals((prev) =>
            prev.map((term) => (term.status === 'syncing' ? { ...term, status: 'online' } : term))
          );
          setIsSyncingAll(false);
          appendLog('system', 'Центральный сервер RMS', 'heartbeat', 'success', 'Волна синхронизации успешно завершена. Все POS-ноды в актуальном состоянии.');
        }, 800);
      }, 900);
    }, 800);
  }, [isSyncingAll, appendLog, menuItems]);

  // 4. RANDOM LIVE CHIEF/WAITSTAFF TRANSMISSION SIMULATOR
  const simulateSale = useCallback(() => {
    if (isSimulatingSale) return;
    setIsSimulatingSale(true);

    const availableDishes = menuItems.filter((m) => m.isAvailable);
    if (availableDishes.length === 0) {
      appendLog('term-main', 'Терминал Бар-Касса', 'heartbeat', 'warning', 'Попытка заказать: нет доступных блюд в меню.');
      setIsSimulatingSale(false);
      return;
    }

    const activeTerminals = terminals.filter((t) => t.status === 'online');
    const terminal = activeTerminals.length > 0
      ? activeTerminals[Math.floor(Math.random() * activeTerminals.length)]
      : terminals[0];

    const activeWaiters = staffList.filter((s) => s.role === 'waiter' && s.status === 'on_shift');
    const waiter = activeWaiters.length > 0
      ? activeWaiters[Math.floor(Math.random() * activeWaiters.length)]
      : staffList[staffList.length - 1];

    const itemsCount = Math.floor(Math.random() * 3) + 1;
    const itemsSelectedForTicket: { itemId: string; itemName: string; quantity: number; price: number; category: string }[] = [];
    let totalPrice = 0;

    for (let i = 0; i < itemsCount; i++) {
      const randDish = availableDishes[Math.floor(Math.random() * availableDishes.length)];
      const quantity = Math.floor(Math.random() * 2) + 1;
      itemsSelectedForTicket.push({
        itemId: randDish.id,
        itemName: randDish.name,
        quantity,
        price: randDish.price,
        category: randDish.category,
      });
      totalPrice += randDish.price * quantity;

      if (randDish.stock !== -1) {
        setMenuItems((prev) =>
          prev.map((mi) => {
            if (mi.id === randDish.id) {
              const freshStock = Math.max(0, mi.stock - quantity);
              return { ...mi, stock: freshStock, isAvailable: freshStock > 0 };
            }
            return mi;
          })
        );
      }
    }

    const payMethods: ('cash' | 'card' | 'mobile_pay')[] = ['card', 'cash', 'mobile_pay'];
    const pMethod = payMethods[Math.floor(Math.random() * payMethods.length)];
    const tableNum = (Math.floor(Math.random() * 18) + 1).toString();

    const newTx: SalesTransaction = {
      id: `tx-${Math.floor(Math.random() * 90000) + 10000}`,
      timestamp: new Date().toTimeString().split(' ')[0],
      terminalId: terminal.id,
      items: itemsSelectedForTicket,
      totalAmount: totalPrice,
      paymentMethod: pMethod,
      waiterId: waiter.id,
      tableNumber: tableNum,
    };

    setTimeout(() => {
      setTransactions((prev) => [...prev, newTx]);
      appendLog(
        terminal.id,
        terminal.name,
        'sales_pull',
        'success',
        `С кассы выгружен чек ${newTx.id} (Стол #${newTx.tableNumber}, Официант: ${waiter.name.split(' ')[0]}). На сумму: ${newTx.totalAmount} ₽.`
      );
      setIsSimulatingSale(false);
    }, 600);
  }, [isSimulatingSale, menuItems, terminals, staffList, appendLog]);

  // Terminal Outage Simulation (Internet off)
  const handleSimulateOutage = (terminalId: string) => {
    setTerminals((prev) =>
      prev.map((t) => {
        if (t.id === terminalId) {
          const isOfflineNow = t.status === 'offline';
          const nextStatus = isOfflineNow ? 'online' : 'offline';
          appendLog(
            t.id,
            t.name,
            'heartbeat',
            isOfflineNow ? 'success' : 'error',
            isOfflineNow
              ? 'Канал связи восстановлен! Пинг 45мс. Авторизован в координационной сети.'
              : 'Соединение разорвано: Превышено время ожидания шлюза терминала.'
          );

          const accumulatedTx = isOfflineNow ? Math.floor(Math.random() * 3) + 1 : t.pendingTransactions;

          return {
            ...t,
            status: nextStatus,
            pendingTransactions: accumulatedTx,
            lastSyncTime: isOfflineNow ? 'Только что' : t.lastSyncTime,
          };
        }
        return t;
      })
    );
  };

  const handleClearLogs = () => {
    setSyncLogs([]);
  };

  return (
    <div className="flex h-screen w-screen overflow-hidden bg-slate-100 font-sans text-slate-800">
      {/* Visual Sidebar */}
      <Sidebar
        currentTab={currentTab}
        setCurrentTab={setCurrentTab}
        staffName={currentMockUser?.name || 'Администратор'}
        staffRole={currentMockUser?.role || 'admin'}
        selectedRestaurant={selectedRestaurant}
        onRestaurantChange={setSelectedRestaurant}
      />

      {/* Primary Panel Wrapper */}
      <main className="flex-1 flex flex-col h-full min-w-0 border-l border-slate-200">
        {currentTab === 'analytics' && (
          <AnalyticsPanel
            transactions={transactions}
            menuItems={menuItems}
            categories={categories}
            simulateSale={simulateSale}
            isSimulating={isSimulatingSale}
          />
        )}
        {currentTab === 'menu' && (
          <MenuPanel
            menuItems={menuItems}
            categories={categories}
            onAddMenuItem={handleAddMenuItem}
            onUpdateMenuItem={handleUpdateMenuItem}
            onDeleteMenuItem={handleDeleteMenuItem}
            selectedRestaurant={selectedRestaurant}
          />
        )}
        {currentTab === 'tables-editor' && (
          <TablesPanel staffList={staffList} />
        )}
        {currentTab === 'stop-list' && (
          <StopListPanel menuItems={menuItems} onUpdateMenuItem={handleUpdateMenuItem} />
        )}
        {currentTab === 'staff' && (
          <StaffPanel
            staffList={staffList}
            onAddStaff={handleAddStaff}
            onUpdateStaff={handleUpdateStaff}
            onDeleteStaff={handleDeleteStaff}
          />
        )}
        {currentTab === 'edge-devices' && (
          <EdgeDevicesPanel />
        )}
        {currentTab === 'warehouse' && (
          <WarehousePanel />
        )}
        {currentTab === 'sync' && (
          <SyncPanel
            terminals={terminals}
            syncLogs={syncLogs}
            triggerForceSync={triggerForceSync}
            isSyncingAll={isSyncingAll}
            onSimulateOutage={handleSimulateOutage}
            onClearLogs={handleClearLogs}
          />
        )}
      </main>
    </div>
  );
}
