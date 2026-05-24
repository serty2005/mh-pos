import React, { createContext, useContext, useState, useEffect } from 'react';
import { 
  EmployeeShift, 
  CashSession, 
  Table, 
  Order, 
  ClosedOrder, 
  CashDrawerEvent, 
  MenuItem, 
  SelectedModifier, 
  OrderLine, 
  Payment,
  FinancialOperation,
  POSSection
} from '../types';
import { mockHalls, mockTables, mockMenuItems } from '../data';

interface POSContextType {
  // Navigation / Display
  currentSection: POSSection;
  setCurrentSection: (section: POSSection) => void;
  activeHallId: string;
  setActiveHallId: (id: string) => void;
  theme: 'light' | 'dark';
  toggleTheme: () => void;
  isPinLocked: boolean;
  setPinLocked: (locked: boolean) => void;
  
  // Auth / Operators
  currentOperator: EmployeeShift | null;
  employeeShifts: EmployeeShift[];
  pinLogin: (pin: string) => boolean;
  logout: () => void;
  openEmployeeShift: () => void;
  closeEmployeeShift: () => void;
  
  // Cash sessions
  cashSession: CashSession | null;
  openCashSession: (initialAmount: number) => void;
  closeCashSession: () => void;
  cashDrawerEvents: CashDrawerEvent[];
  addCashDrawerEvent: (type: 'in' | 'out', amount: number, reason: string) => void;

  // Tables & Hall grid
  tables: Table[];
  setSelectedTableId: (id: string | null) => void;
  selectedTableId: string | null;
  createOrderForTable: (tableId: string, guestsCount: number) => void;

  // Order Entry (Menu / Catalog checkout)
  activeOrders: Order[];
  currentOrder: Order | null;
  addMenuItemToOrder: (item: MenuItem, selectedModifiers: SelectedModifier[]) => { success: boolean; errorKey?: string };
  removeOrderLine: (lineId: string) => void;
  changeLineQuantity: (lineId: string, newQty: number) => { success: boolean; errorKey?: string };
  updateCommentAndCourse: (lineId: string, comment: string, course: number) => void;
  
  // Precheck
  issuePrecheck: () => void;
  cancelPrecheck: (managerPin: string, reason: string) => boolean;
  reprintPrecheck: () => void;

  // Payment
  payOrder: (method: 'cash' | 'card', inputAmount: number) => { success: boolean; change: number; errorKey?: string };
  
  // Closed Check Activities & Refunds
  closedOrders: ClosedOrder[];
  refundCheck: (checkId: string, reason: string, disposition: 'waste' | 'return') => void;
  partialRefundCheck: (
    checkId: string, 
    lineId: string, 
    qtyToRefund: number, 
    reason: string, 
    disposition: 'waste' | 'return'
  ) => void;
  reprintCheck: (checkId: string) => void;

  // Sync Diagnostics
  outboxCount: number;
  syncOutbox: () => void;
  syncStatus: 'online' | 'offline';
  setSyncStatus: (status: 'online' | 'offline') => void;
  logEvents: Array<{ id: string; time: string; msg: string; type: 'info' | 'warn' | 'success' }>;
  addLogEvent: (msg: string, type?: 'info' | 'warn' | 'success') => void;
}

const POSContext = createContext<POSContextType | undefined>(undefined);

export const POSProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  // Navigation
  const [currentSection, setCurrentSection] = useState<POSSection>('floor');
  const [activeHallId, setActiveHallId] = useState<string>('hall-main');
  const [theme, setTheme] = useState<'light' | 'dark'>('light');
  const [isPinLocked, setPinLocked] = useState<boolean>(true);

  // Auth / Operators
  const [currentOperator, setCurrentOperator] = useState<EmployeeShift | null>(null);
  const [employeeShifts, setEmployeeShifts] = useState<EmployeeShift[]>([]);

  // Cash KKM Sessions
  const [cashSession, setCashSession] = useState<CashSession | null>(null);
  const [cashDrawerEvents, setCashDrawerEvents] = useState<CashDrawerEvent[]>([]);

  // Tables & Orders state
  const [tables, setTables] = useState<Table[]>(mockTables);
  const [activeOrders, setActiveOrders] = useState<Order[]>([]);
  const [selectedTableId, setSelectedTableId] = useState<string | null>(null);

  // Closed Orders & Activities archives
  const [closedOrders, setClosedOrders] = useState<ClosedOrder[]>([]);

  // Sync / Diagnostics Outbox
  const [syncStatus, setSyncStatus] = useState<'online' | 'offline'>('online');
  const [outboxCount, setOutboxCount] = useState<number>(0);
  const [logEvents, setLogEvents] = useState<Array<{ id: string; time: string; msg: string; type: 'info' | 'warn' | 'success' }>>([]);

  // Initialize with standard sample values to represent a live environment
  useEffect(() => {
    // Light dashboard diagnostic events
    addLogEvent('Система MyHoreca POS запущена.', 'success');
    addLogEvent('Локальная реплика Master-Data загружена успешно.', 'info');
    
    // Auto-detect theme from body class
    document.documentElement.classList.remove('dark');
  }, []);

  const addLogEvent = (msg: string, type: 'info' | 'warn' | 'success' = 'info') => {
    setLogEvents(prev => [
      {
        id: Math.random().toString(),
        time: new Date().toLocaleTimeString(),
        msg,
        type
      },
      ...prev.slice(0, 19) // Limit logs size
    ]);
  };

  const toggleTheme = () => {
    setTheme(prev => {
      const next = prev === 'light' ? 'dark' : 'light';
      if (next === 'dark') {
        document.documentElement.classList.add('dark');
      } else {
        document.documentElement.classList.remove('dark');
      }
      addLogEvent(`Выбрана ${next === 'dark' ? 'темная' : 'светлая'} визуальная тема.`, 'info');
      return next;
    });
  };

  // Switch operator login PIN
  const pinLogin = (pin: string): boolean => {
    let operatorRole: 'cashier' | 'waiter' | 'manager' = 'waiter';
    let operatorName = '';

    if (pin === '1111') {
      operatorRole = 'cashier';
      operatorName = 'Анна Румянцева (Старший Кассир)';
    } else if (pin === '2222') {
      operatorRole = 'waiter';
      operatorName = 'Иван Соколов (Официант)';
    } else if (pin === '3333') {
      operatorRole = 'manager';
      operatorName = 'Сергей Петров (Администратор)';
    } else {
      return false;
    }

    const newShift: EmployeeShift = {
      id: `shift-${Math.random().toString().slice(2, 6)}`,
      employeeName: operatorName,
      role: operatorRole,
      openTime: new Date().toLocaleString(),
      status: 'open'
    };

    setCurrentOperator(newShift);
    setEmployeeShifts(prev => [newShift, ...prev]);
    setPinLocked(false);
    
    addLogEvent(`Сотрудник ${operatorName} авторизован. Роль: ${operatorRole}`, 'success');

    // Smart Section Entry fallback
    // tables_count > 1 ? floor : orders
    if (mockTables.length > 1) {
      setCurrentSection('floor');
    } else {
      setCurrentSection('order');
    }

    return true;
  };

  const logout = () => {
    setCurrentOperator(null);
    setPinLocked(true);
    setSelectedTableId(null);
    addLogEvent('Выход из сессии оператора.', 'info');
  };

  const openEmployeeShift = () => {
    if (currentOperator) {
      addLogEvent(`Личная смена сотрудника ${currentOperator.employeeName} уже открыта.`, 'warn');
      return;
    }
  };

  const closeEmployeeShift = () => {
    if (!currentOperator) return;
    
    addLogEvent(`Личная смена сотрудника ${currentOperator.employeeName} закрыта.`, 'info');
    setEmployeeShifts(prev => 
      prev.map(sh => sh.id === currentOperator.id ? { ...sh, status: 'closed', closeTime: new Date().toLocaleString() } : sh)
    );
    logout();
  };

  // Cash flow sessions
  const openCashSession = (initialAmount: number) => {
    const newSession: CashSession = {
      id: `session-${Math.random().toString().slice(2, 6)}`,
      openedAt: new Date().toLocaleString(),
      initialAmount,
      currentAmount: initialAmount,
      status: 'open',
      openedBy: currentOperator?.employeeName || 'Оператор'
    };
    setCashSession(newSession);
    addLogEvent(`Кассовая сессия ККМ зарегистрирована на сумму ${initialAmount} руб.`, 'success');
  };

  const closeCashSession = () => {
    if (!cashSession) return;
    setCashSession(null);
    addLogEvent('Кассовая сессия ККМ успешно закрыта. Выпущен Z-отчет.', 'info');
  };

  const addCashDrawerEvent = (type: 'in' | 'out', amount: number, reason: string) => {
    const event: CashDrawerEvent = {
      id: `cd-${Math.random().toString().substr(2, 5)}`,
      timestamp: new Date().toLocaleString(),
      type,
      amount,
      reason,
      operator: currentOperator?.employeeName || 'Оператор'
    };
    setCashDrawerEvents(prev => [event, ...prev]);
    if (cashSession) {
      setCashSession(prev => {
        if (!prev) return null;
        const delta = type === 'in' ? amount : -amount;
        return {
          ...prev,
          currentAmount: Math.max(0, prev.currentAmount + delta)
        };
      });
    }
    
    // Add event mock to sync queue
    if (syncStatus === 'offline') {
      setOutboxCount(prev => prev + 1);
    }

    addLogEvent(`Операция кассового ящика: ${type === 'in' ? 'Внесение' : 'Извлечение'} на сумму ${amount} руб.`, 'info');
  };

  // Active table orders
  const findActiveOrderByTable = (tableId: string): Order | null => {
    return activeOrders.find(ord => ord.tableId === tableId && ord.status !== 'closed') || null;
  };

  const currentOrder = selectedTableId ? findActiveOrderByTable(selectedTableId) : null;

  const createOrderForTable = (tableId: string, guestsCount: number) => {
    if (!currentOperator) return;

    const table = tables.find(t => t.id === tableId);
    if (!table) return;

    const newOrder: Order = {
      id: `order-${Math.random().toString().slice(2, 6)}`,
      shortId: Math.random().toString().slice(2, 5),
      tableId,
      tableName: `Стол ${table.number}`,
      hallName: mockHalls.find(h => h.id === table.hallId)?.name || 'Основной',
      status: 'open',
      lines: [],
      subtotal: 0,
      tax: 0,
      discount: 0,
      total: 0,
      payments: [],
      openedAt: new Date().toLocaleTimeString(),
      waiterName: currentOperator.employeeName
    };

    setActiveOrders(prev => [...prev, newOrder]);
    setTables(prev => 
      prev.map(t => t.id === tableId ? { ...t, status: 'occupied', currentOrderId: newOrder.id, guestsCount } : t)
    );
    
    addLogEvent(`Создан новый активный заказ #${newOrder.shortId} на Стол ${table.number}`, 'success');
  };

  // Add line item to order helper
  const addMenuItemToOrder = (item: MenuItem, selectedModifiers: SelectedModifier[]): { success: boolean; errorKey?: string } => {
    if (!currentOperator) return { success: false, errorKey: 'errors.noShift' };
    if (!currentOrder) return { success: false, errorKey: 'blocks.noOrderSelected' };
    if (currentOrder.status === 'precheck_issued') return { success: false, errorKey: 'errors.orderLocked' };

    // Stop list handling
    if (!item.isAvailable) {
      addLogEvent(`Ошибка продажи: ${item.name} находится в Стоп-листе!`, 'warn');
      return { success: false, errorKey: 'errors.stopListConflict' };
    }

    // Compose selected modifiers array sum
    const modSum = selectedModifiers.reduce((acc, m) => acc + m.price, 0);

    // Look if item lines exists with EXACT same selected modifiers configuration
    const matchLineIndex = currentOrder.lines.findIndex((line) => {
      if (line.itemId !== item.id) return false;
      if (line.selectedModifiers.length !== selectedModifiers.length) return false;
      
      // Compare option ids
      const currentOpts = line.selectedModifiers.map(m => m.optionId).sort().join(',');
      const incomingOpts = selectedModifiers.map(m => m.optionId).sort().join(',');
      return currentOpts === incomingOpts;
    });

    let updatedLines = [...currentOrder.lines];

    if (matchLineIndex > -1) {
      // Line matched, increase count
      const line = updatedLines[matchLineIndex];
      updatedLines[matchLineIndex] = {
        ...line,
        quantity: line.quantity + 1
      };
    } else {
      // Create new line instead
      const newLine: OrderLine = {
        id: `line-${Math.random().toString().slice(2, 6)}`,
        itemId: item.id,
        name: item.name,
        price: item.price + modSum,
        quantity: 1,
        selectedModifiers,
        course: 1 // 1st Course standard fallback
      };
      updatedLines.push(newLine);
    }

    updateActiveOrderLines(currentOrder.id, updatedLines);
    return { success: true };
  };

  const removeOrderLine = (lineId: string) => {
    if (!currentOrder || currentOrder.status === 'precheck_issued') return;
    const updated = currentOrder.lines.filter(l => l.id !== lineId);
    updateActiveOrderLines(currentOrder.id, updated);
    addLogEvent(`Удалена позиция из заказа #${currentOrder.shortId}`, 'info');
  };

  const changeLineQuantity = (lineId: string, newQty: number): { success: boolean; errorKey?: string } => {
    if (!currentOrder) return { success: false, errorKey: 'blocks.noOrderSelected' };
    if (currentOrder.status === 'precheck_issued') return { success: false, errorKey: 'errors.orderLocked' };

    if (newQty <= 0) {
      removeOrderLine(lineId);
      return { success: true };
    }

    const updated = currentOrder.lines.map(line => {
      if (line.id === lineId) {
        return {
          ...line,
          quantity: newQty
        };
      }
      return line;
    });

    updateActiveOrderLines(currentOrder.id, updated);
    return { success: true };
  };

  const updateCommentAndCourse = (lineId: string, comment: string, course: number) => {
    if (!currentOrder || currentOrder.status === 'precheck_issued') return;
    const updated = currentOrder.lines.map(line => {
      if (line.id === lineId) {
        return {
          ...line,
          comment,
          course
        };
      }
      return line;
    });
    updateActiveOrderLines(currentOrder.id, updated);
    addLogEvent(`Обновлен комментарий/курс для позиции в заказе #${currentOrder.shortId}`, 'info');
  };

  // Update order lines and calculate totals properly
  const updateActiveOrderLines = (orderId: string, lines: OrderLine[]) => {
    setActiveOrders(prev => {
      return prev.map(ord => {
        if (ord.id === orderId) {
          const subtotal = lines.reduce((acc, l) => acc + (l.price * l.quantity), 0);
          // Standard VAT formula 10%
          const tax = Math.round(subtotal * 0.1);
          const total = subtotal; // subtotal includes item base & modifiers

          return {
            ...ord,
            lines,
            subtotal,
            tax,
            total
          };
        }
        return ord;
      });
    });

    // Mirror to table view
    setTables(prev => {
      return prev.map(t => {
        if (t.id === selectedTableId) {
          const subtotal = lines.reduce((acc, l) => acc + (l.price * l.quantity), 0);
          return {
            ...t,
            activeOrderSum: subtotal
          };
        }
        return t;
      });
    });
  };

  // Issue precheck
  const issuePrecheck = () => {
    if (!currentOrder) return;
    
    setActiveOrders(prev => 
      prev.map(o => o.id === currentOrder.id ? { 
        ...o, 
        status: 'precheck_issued',
        precheckTime: new Date().toLocaleTimeString(),
        precheckBy: currentOperator?.employeeName || 'Оператор'
      } : o)
    );

    addLogEvent(`Пречек для заказа #${currentOrder.shortId} успешно выпущен. Заказ заблокирован.`, 'success');
  };

  // Cancel issue precheck through manager PIN override
  const cancelPrecheck = (managerPin: string, reason: string): boolean => {
    if (!currentOrder) return false;
    
    // Verify Manager code override
    if (managerPin !== '3333' && managerPin !== '1111') {
      addLogEvent('Ошибка авторизации при отмене пречека. Неверный PIN.', 'warn');
      return false;
    }

    setActiveOrders(prev => 
      prev.map(o => o.id === currentOrder.id ? { ...o, status: 'open', precheckTime: undefined, precheckBy: undefined } : o)
    );

    addLogEvent(`Пречек заказа #${currentOrder.shortId} отменен. Авторизовал: Сергей Петров. Причина: ${reason}`, 'success');
    return true;
  };

  const reprintPrecheck = () => {
    if (!currentOrder) return;
    addLogEvent(`Печать копии пречека для заказа #${currentOrder.shortId}... готово.`, 'success');
  };

  // Process checkout cash count / credit card manual payments
  const payOrder = (method: 'cash' | 'card', inputAmount: number): { success: boolean; change: number; errorKey?: string } => {
    if (!currentOperator) return { success: false, change: 0, errorKey: 'errors.noShift' };
    if (!cashSession) return { success: false, change: 0, errorKey: 'errors.noCashSession' };
    if (!currentOrder) return { success: false, change: 0, errorKey: 'blocks.noOrderSelected' };
    
    const change = Math.max(0, inputAmount - currentOrder.total);

    // Final check object
    const finalizedCheck: ClosedOrder = {
      id: `check-${Math.random().toString().slice(2, 6)}`,
      shortId: currentOrder.shortId,
      tableName: currentOrder.tableName,
      hallName: currentOrder.hallName,
      total: currentOrder.total,
      paymentMethod: method,
      closedAt: new Date().toLocaleString(),
      operator: currentOperator.employeeName,
      status: 'closed',
      originalShiftId: currentOperator.id,
      lines: currentOrder.lines,
      operations: [
        {
          id: `op-${Math.random().toString().slice(2, 5)}`,
          type: 'payment',
          kind: method === 'cash' ? 'Наличные' : 'Карта',
          amount: currentOrder.total,
          reason: 'Продажа',
          employee: currentOperator.employeeName,
          timestamp: new Date().toLocaleTimeString()
        }
      ]
    };

    // Record check
    setClosedOrders(prev => [finalizedCheck, ...prev]);

    // Record total in KKM cash Session if cash method
    if (method === 'cash') {
      setCashSession(prev => {
        if (!prev) return null;
        return {
          ...prev,
          currentAmount: prev.currentAmount + currentOrder.total
        };
      });
    }

    // Free table
    setTables(prev => 
      prev.map(t => t.id === currentOrder.tableId ? { 
        ...t, 
        status: 'free', 
        currentOrderId: undefined, 
        activeOrderSum: 0,
        guestsCount: 0 
      } : t)
    );

    // Remove from active list
    setActiveOrders(prev => prev.filter(o => o.id !== currentOrder.id));
    setSelectedTableId(null);

    // Unsaved events online fallback sync count
    if (syncStatus === 'offline') {
      setOutboxCount(prev => prev + 1);
    }

    addLogEvent(`Заказ #${finalizedCheck.shortId} успешно оплачен (${method === 'cash' ? 'наличными' : 'картой'}).`, 'success');
    return { success: true, change };
  };

  // Full Refund / Cancellation handler with inventories dispositions ledger
  const refundCheck = (checkId: string, reason: string, disposition: 'waste' | 'return') => {
    if (!currentOperator || currentOperator.role === 'waiter') {
      addLogEvent('Доступ отклонен: У официантов нет прав на возврат.', 'warn');
      return;
    }
    if (!cashSession) {
      addLogEvent('Доступ отклонен: Возврат невозможен при закрытой кассовой смене.', 'warn');
      return;
    }

    setClosedOrders(prev => {
      return prev.map(check => {
        if (check.id === checkId) {
          const refundAction: FinancialOperation = {
            id: `op-${Math.random().toString().slice(2, 5)}`,
            type: 'refund',
            kind: 'Полный возврат',
            amount: check.total,
            reason: reason || 'Возврат по требованию гостя',
            employee: currentOperator.employeeName,
            timestamp: new Date().toLocaleTimeString(),
            disposition
          };

          const cancelAction: FinancialOperation = {
            id: `op-${Math.random().toString().slice(2, 5)}`,
            type: 'cancellation',
            kind: 'Отмена чека',
            amount: check.total,
            reason: 'Отмена чека ФР',
            employee: currentOperator.employeeName,
            timestamp: new Date().toLocaleTimeString()
          };

          // Adjust cashSession current sum if cash method
          if (check.paymentMethod === 'cash') {
            setCashSession(session => {
              if (!session) return null;
              return {
                ...session,
                currentAmount: Math.max(0, session.currentAmount - check.total)
              };
            });
          }

          if (syncStatus === 'offline') {
            setOutboxCount(c => c + 1);
          }

          addLogEvent(`Выполнен ПОЛНЫЙ ВОЗВРАТ по чеку #${check.shortId} (${disposition === 'waste' ? 'Списание утилизацией' : 'На склад'})`, 'success');

          return {
            ...check,
            status: 'refunded',
            refundReason: reason,
            operations: [refundAction, cancelAction, ...check.operations]
          };
        }
        return check;
      });
    });
  };

  // Partial Order line level cancellation/refund with exact quantities
  const partialRefundCheck = (
    checkId: string, 
    lineId: string, 
    qtyToRefund: number, 
    reason: string, 
    disposition: 'waste' | 'return'
  ) => {
    if (!currentOperator || currentOperator.role === 'waiter') return;
    if (!cashSession) return;

    setClosedOrders(prev => {
      return prev.map(check => {
        if (check.id === checkId) {
          const targetLine = check.lines.find(l => l.id === lineId);
          if (!targetLine) return check;

          const refundAmount = Math.round(targetLine.price * qtyToRefund);

          const refundAction: FinancialOperation = {
            id: `op-${Math.random().toString().slice(2, 5)}`,
            type: 'refund',
            kind: `Частичный возврат (линия) x${qtyToRefund}`,
            amount: refundAmount,
            reason: `${reason} (${targetLine.name})`,
            employee: currentOperator.employeeName,
            timestamp: new Date().toLocaleTimeString(),
            disposition
          };

          // Adjust KKM cash session is cash
          if (check.paymentMethod === 'cash') {
            setCashSession(session => {
              if (!session) return null;
              return {
                ...session,
                currentAmount: Math.max(0, session.currentAmount - refundAmount)
              };
            });
          }

          if (syncStatus === 'offline') {
            setOutboxCount(c => c + 1);
          }

          addLogEvent(`Выполнен ЧАСТИЧНЫЙ ВОЗВРАТ (${qtyToRefund} шт. ${targetLine.name}) по чеку #${check.shortId}`, 'success');

          // Check if partial returns represents the entire total sum of check
          const previousRefundedAmount = check.operations
            .filter(op => op.type === 'refund')
            .reduce((sum, op) => sum + op.amount, 0);

          const isNowfullyRefunded = (previousRefundedAmount + refundAmount) >= check.total;

          return {
            ...check,
            status: isNowfullyRefunded ? 'refunded' : 'partially_refunded',
            refundReason: reason,
            operations: [refundAction, ...check.operations]
          };
        }
        return check;
      });
    });
  };

  const reprintCheck = (checkId: string) => {
    const check = closedOrders.find(c => c.id === checkId);
    if (!check) return;
    addLogEvent(`Печать копии Фискального чека #${check.shortId} по ККМ... готово.`, 'success');
  };

  // Sync Diagnostics Outbox trigger
  const syncOutbox = () => {
    if (syncStatus === 'offline') {
      addLogEvent('Невозможно выполнить синхронизацию: терминал находится офлайн!', 'warn');
      return;
    }
    if (outboxCount === 0) {
      addLogEvent('Очередь транзакций (Outbox) пуста. Код ответа: 200 OK.', 'info');
      return;
    }

    addLogEvent(`Локальные события синхронизированы в Cloud. Очередь [${outboxCount}] очищена.`, 'success');
    setOutboxCount(0);
  };

  return (
    <POSContext.Provider value={{
      currentSection,
      setCurrentSection,
      activeHallId,
      setActiveHallId,
      theme,
      toggleTheme,
      isPinLocked,
      setPinLocked,
      
      currentOperator,
      employeeShifts,
      pinLogin,
      logout,
      openEmployeeShift,
      closeEmployeeShift,

      cashSession,
      openCashSession,
      closeCashSession,
      cashDrawerEvents,
      addCashDrawerEvent,

      tables,
      setSelectedTableId,
      selectedTableId,
      createOrderForTable,

      activeOrders,
      currentOrder,
      addMenuItemToOrder,
      removeOrderLine,
      changeLineQuantity,
      updateCommentAndCourse,

      issuePrecheck,
      cancelPrecheck,
      reprintPrecheck,

      payOrder,

      closedOrders,
      refundCheck,
      partialRefundCheck,
      reprintCheck,

      outboxCount,
      syncOutbox,
      syncStatus,
      setSyncStatus,
      logEvents,
      addLogEvent
    }}>
      {children}
    </POSContext.Provider>
  );
};

export const usePOS = () => {
  const context = useContext(POSContext);
  if (context === undefined) {
    throw new Error('usePOS must be used within a POSProvider');
  }
  return context;
};
