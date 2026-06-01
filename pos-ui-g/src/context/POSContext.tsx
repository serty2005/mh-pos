import React, { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from 'react';

import { createApiClient, ApiError, type AuthSnapshot } from '../shared/api';
import {
  dispositionToBackend,
  mapCashDrawerEvent,
  mapCashSession,
  mapClosedOrder,
  mapHall,
  mapMenuItem,
  mapOperator,
  mapOrder,
  mapPricingPolicy,
  mapSyncStatus,
  mapTable,
  outboxCount,
  selectedModifiersToPayload,
} from '../shared/backendMappers';
import { getClientDeviceId } from '../shared/clientIdentity';
import { t } from '../shared/i18n';
import type {
  BackendActorContext,
  BackendCashDrawerEvent,
  BackendCashSession,
  BackendClosedOrder,
  BackendHall,
  BackendMenuItem,
  BackendOrder,
  BackendPrecheck,
  BackendPricingCalculation,
  BackendProvisioningStatus,
  BackendPricingPolicy,
  BackendShift,
  BackendStorageStatus,
  BackendSyncStatus,
  BackendTable,
} from '../shared/schemas';
import {
  CashDrawerEvent,
  CashSession,
  ClosedOrder,
  EmployeeShift,
  MenuItem,
  Order,
  POSSection,
  PricingPolicy,
  SelectedModifier,
  Table,
} from '../types';
import { activeIssuedPrecheck, paymentChange } from './posContextHelpers';
import {
  applyPOSTheme,
  defaultThemeSettings,
  posThemeSchemes,
  readStoredThemeSettings,
  writeStoredThemeSettings,
  type POSThemeMode,
  type POSThemeScheme,
  type POSThemeSchemeId,
} from './theme';

type LogEvent = { id: string; time: string; msg: string; type: 'info' | 'warn' | 'success' };

const CLOSED_ORDERS_PAGE_SIZE = 25;

type PaymentResult = { success: boolean; change: number; errorKey?: string };
type CommandResult = { success: boolean; errorKey?: string };

interface POSContextType {
  currentSection: POSSection;
  setCurrentSection: (section: POSSection) => void;
  activeHallId: string;
  setActiveHallId: (id: string) => void;
  theme: POSThemeMode;
  themeScheme: POSThemeSchemeId;
  themeSchemes: POSThemeScheme[];
  toggleTheme: () => void;
  setThemeMode: (mode: POSThemeMode) => void;
  setThemeScheme: (scheme: POSThemeSchemeId) => void;
  isPinLocked: boolean;
  setPinLocked: (locked: boolean) => void;
  currentOperator: EmployeeShift | null;
  employeeShifts: EmployeeShift[];
  pinLogin: (pin: string) => Promise<boolean>;
  logout: () => Promise<void>;
  openEmployeeShift: () => Promise<void>;
  closeEmployeeShift: () => Promise<void>;
  cashSession: CashSession | null;
  openCashSession: (initialAmount: number) => Promise<void>;
  closeCashSession: () => Promise<void>;
  cashDrawerEvents: CashDrawerEvent[];
  addCashDrawerEvent: (type: 'in' | 'out', amount: number, reason: string) => Promise<void>;
  halls: ReturnType<typeof mapHall>[];
  menuItems: MenuItem[];
  tables: Table[];
  setSelectedTableId: (id: string | null) => void;
  selectedTableId: string | null;
  createOrderForTable: (tableId: string, guestsCount: number) => Promise<void>;
  activeOrders: Order[];
  currentOrder: Order | null;
  addMenuItemToOrder: (item: MenuItem, selectedModifiers: SelectedModifier[]) => Promise<CommandResult>;
  editOrderLineModifiers: (lineId: string, selectedModifiers: SelectedModifier[]) => Promise<CommandResult>;
  removeOrderLine: (lineId: string) => Promise<void>;
  changeLineQuantity: (lineId: string, newQty: number) => Promise<CommandResult>;
  updateCommentAndCourse: (lineId: string, comment: string, course: number) => Promise<void>;
  issuePrecheck: () => Promise<void>;
  cancelPrecheck: (managerPin: string, reason: string) => Promise<boolean>;
  reprintPrecheck: () => Promise<void>;
  payOrder: (method: 'cash' | 'card', inputAmount: number) => Promise<PaymentResult>;
  closedOrders: ClosedOrder[];
  closedOrdersPage: number;
  closedOrdersPageSize: number;
  closedOrdersHasNextPage: boolean;
  closedOrdersLoading: boolean;
  closedOrdersError: string;
  closedOrdersBusinessDate: string;
  loadClosedOrdersPage: (page: number, businessDate?: string) => Promise<void>;
  refundCheck: (checkId: string, reason: string, disposition: 'waste' | 'return') => Promise<void>;
  partialRefundCheck: (checkId: string, lineId: string, qtyToRefund: number, reason: string, disposition: 'waste' | 'return') => Promise<void>;
  reprintCheck: (checkId: string) => Promise<void>;
  pricingPolicies: PricingPolicy[];
  applyPricingPolicy: (policyId: string, orderLineId?: string, reason?: string) => Promise<CommandResult>;
  outboxCount: number;
  syncOutbox: () => Promise<void>;
  syncStatus: 'online' | 'offline';
  syncRevision: number;
  appVersion: string;
  logEvents: LogEvent[];
  addLogEvent: (msg: string, type?: 'info' | 'warn' | 'success') => void;
  authSnapshot: AuthSnapshot;
  isEdgePaired: boolean;
  provisioningStatus: BackendProvisioningStatus | null;
  provisioningLoading: boolean;
  provisioningError: string;
  refreshProvisioningStatus: () => Promise<void>;
  registerCloudProvisioning: (cloudUrl?: string) => Promise<void>;
  pairViaLicense: (pairingCode: string) => Promise<void>;
}

const POSContext = createContext<POSContextType | undefined>(undefined);

const storageKeys = {
  nodeDeviceId: 'mh-pos.node_device_id',
  restaurantId: 'mh-pos.restaurant_id',
  sessionId: 'mh-pos.session_id',
};

function initialAuth(): AuthSnapshot {
  return {
    clientDeviceId: getClientDeviceId(),
    nodeDeviceId: localStorage.getItem(storageKeys.nodeDeviceId) ?? '',
    restaurantId: localStorage.getItem(storageKeys.restaurantId) ?? '',
    sessionId: localStorage.getItem(storageKeys.sessionId) ?? '',
    actorEmployeeId: '',
  };
}

export const POSProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [currentSection, setCurrentSection] = useState<POSSection>('floor');
  const [activeHallId, setActiveHallIdState] = useState<string>('');
  const [themeSettings, setThemeSettings] = useState(() => {
    if (typeof window === 'undefined') return defaultThemeSettings;
    return readStoredThemeSettings();
  });
  const [isPinLocked, setPinLocked] = useState<boolean>(true);
  const [selectedTableId, setSelectedTableId] = useState<string | null>(null);

  useEffect(() => {
    applyPOSTheme(themeSettings);
    writeStoredThemeSettings(themeSettings);
  }, [themeSettings]);

  const [auth, setAuthState] = useState<AuthSnapshot>(() => initialAuth());
  const authRef = useRef<AuthSnapshot>(auth);
  const api = useMemo(() => createApiClient(() => authRef.current), []);

  const [actor, setActor] = useState<BackendActorContext | null>(null);
  const [shift, setShift] = useState<BackendShift | null>(null);
  const [recentShifts, setRecentShifts] = useState<BackendShift[]>([]);
  const [cashSessionDto, setCashSessionDto] = useState<BackendCashSession | null>(null);
  const [cashDrawerEventsDto, setCashDrawerEventsDto] = useState<BackendCashDrawerEvent[]>([]);
  const [hallsDto, setHallsDto] = useState<BackendHall[]>([]);
  const [tablesDto, setTablesDto] = useState<BackendTable[]>([]);
  const [activeOrdersDto, setActiveOrdersDto] = useState<BackendOrder[]>([]);
  const [menuDto, setMenuDto] = useState<BackendMenuItem[]>([]);
  const [pricingPoliciesDto, setPricingPoliciesDto] = useState<BackendPricingPolicy[]>([]);
  const [prechecksDto, setPrechecksDto] = useState<BackendPrecheck[]>([]);
  const [currentPricingDto, setCurrentPricingDto] = useState<BackendPricingCalculation | null>(null);
  const [closedOrdersDto, setClosedOrdersDto] = useState<BackendClosedOrder[]>([]);
  const [closedOrdersPage, setClosedOrdersPage] = useState<number>(1);
  const [closedOrdersHasNextPage, setClosedOrdersHasNextPage] = useState<boolean>(false);
  const [closedOrdersLoading, setClosedOrdersLoading] = useState<boolean>(false);
  const [closedOrdersError, setClosedOrdersError] = useState<string>('');
  const [closedOrdersBusinessDate, setClosedOrdersBusinessDate] = useState<string>('');
  const [syncStatusDto, setSyncStatusDto] = useState<BackendSyncStatus | null>(null);
  const [storageStatusDto, setStorageStatusDto] = useState<BackendStorageStatus | null>(null);
  const [provisioningStatusDto, setProvisioningStatusDto] = useState<BackendProvisioningStatus | null>(null);
  const [provisioningLoading, setProvisioningLoading] = useState<boolean>(false);
  const [provisioningError, setProvisioningError] = useState<string>('');
  const [logEvents, setLogEvents] = useState<LogEvent[]>([]);

  const isEdgePaired = useMemo(
    () => Boolean(auth.nodeDeviceId && auth.restaurantId),
    [auth.nodeDeviceId, auth.restaurantId],
  );

  const setAuth = useCallback((next: AuthSnapshot) => {
    authRef.current = next;
    setAuthState(next);
    if (next.nodeDeviceId) localStorage.setItem(storageKeys.nodeDeviceId, next.nodeDeviceId);
    else localStorage.removeItem(storageKeys.nodeDeviceId);
    if (next.restaurantId) localStorage.setItem(storageKeys.restaurantId, next.restaurantId);
    else localStorage.removeItem(storageKeys.restaurantId);
    if (next.sessionId) localStorage.setItem(storageKeys.sessionId, next.sessionId);
    else localStorage.removeItem(storageKeys.sessionId);
  }, []);

  const applyProvisioningStatus = useCallback((status: BackendProvisioningStatus) => {
    setProvisioningStatusDto(status);
    if (!status.paired) {
      setActor(null);
    }
    setAuth({
      ...authRef.current,
      nodeDeviceId: status.node_device_id || authRef.current.nodeDeviceId,
      restaurantId: status.paired && status.restaurant_id ? status.restaurant_id : '',
      sessionId: status.paired ? authRef.current.sessionId : '',
      actorEmployeeId: status.paired ? authRef.current.actorEmployeeId : '',
    });
  }, [setAuth]);

  const addLogEvent = useCallback((msg: string, type: 'info' | 'warn' | 'success' = 'info') => {
    setLogEvents((prev) => [
      {
        id: `${Date.now()}-${Math.random()}`,
        time: new Date().toLocaleTimeString('ru-RU'),
        msg,
        type,
      },
      ...prev.slice(0, 19),
    ]);
  }, []);

  const handleError = useCallback((error: unknown, fallback: string) => {
    const message = error instanceof ApiError ? `${fallback}: ${error.code}` : fallback;
    addLogEvent(message, 'warn');
  }, [addLogEvent]);

  const refreshIdentity = useCallback(async () => {
    try {
      const status = await api.getPairingStatus();
      const nodeDeviceId = status.node_device_id ?? status.identity?.node_device_id ?? authRef.current.nodeDeviceId;
      const restaurantId = status.restaurant_id ?? status.identity?.restaurant_id ?? authRef.current.restaurantId;
      if (status.paired && nodeDeviceId && restaurantId) {
        setAuth({ ...authRef.current, nodeDeviceId, restaurantId });
        return;
      }
      const provisioning = await api.getProvisioningStatus();
      applyProvisioningStatus(provisioning);
    } catch (error) {
      handleError(error, t.ops.pairingStatusFailed);
    }
  }, [api, applyProvisioningStatus, handleError, setAuth]);

  const refreshProvisioningStatus = useCallback(async () => {
    setProvisioningLoading(true);
    setProvisioningError('');
    try {
      const status = await api.getProvisioningStatus();
      applyProvisioningStatus(status);
    } catch (error) {
      setProvisioningError(t.pair.error);
      handleError(error, t.ops.provisioningStatusFailed);
    } finally {
      setProvisioningLoading(false);
    }
  }, [api, applyProvisioningStatus, handleError]);

  const registerCloudProvisioning = useCallback(async (cloudUrl = '') => {
    setProvisioningLoading(true);
    setProvisioningError('');
    try {
      const status = await api.registerCloudProvisioning(cloudUrl);
      applyProvisioningStatus(status);
      addLogEvent(t.ops.edgeRegistrationSent, 'success');
    } catch (error) {
      setProvisioningError(t.pair.error);
      handleError(error, t.ops.edgeRegistrationFailed);
    } finally {
      setProvisioningLoading(false);
    }
  }, [addLogEvent, api, applyProvisioningStatus, handleError]);

  const pairViaLicense = useCallback(async (pairingCode: string) => {
    setProvisioningLoading(true);
    setProvisioningError('');
    try {
      const status = await api.pairViaLicense(pairingCode.trim().toUpperCase());
      applyProvisioningStatus(status);
      addLogEvent(t.ops.licensePaired, 'success');
    } catch (error) {
      setProvisioningError(t.pair.error);
      handleError(error, t.ops.licensePairFailed);
    } finally {
      setProvisioningLoading(false);
    }
  }, [addLogEvent, api, applyProvisioningStatus, handleError]);

  const refreshOps = useCallback(async () => {
    if (!authRef.current.sessionId || !authRef.current.nodeDeviceId) return;
    const [currentShift, shifts] = await Promise.all([
      api.getCurrentShift(),
      api.listRecentShifts().catch(() => []),
    ]);
    setShift(currentShift);
    setRecentShifts(shifts);
    if (!currentShift) {
      setCashSessionDto(null);
      return;
    }
    const currentCash = await api.getCurrentCashSession().catch(() => null);
    setCashSessionDto(currentCash);
  }, [api]);

  const refreshDirectory = useCallback(async () => {
    if (!authRef.current.sessionId || !authRef.current.restaurantId || !shift) return;
    const [halls, menu, pricingPolicies] = await Promise.all([
      api.listHalls(),
      api.listMenuItems(),
      api.listActivePricingPolicies().catch(() => []),
    ]);
    const activeHalls = halls.filter((hall) => hall.active);
    setHallsDto(activeHalls);
    setMenuDto(menu);
    setPricingPoliciesDto(pricingPolicies.filter((policy) => policy.active && !policy.manual));
    if (!activeHallId && activeHalls[0]) {
      setActiveHallIdState(activeHalls[0].id);
    }
  }, [activeHallId, api, shift]);

  const refreshFloor = useCallback(async () => {
    if (!authRef.current.sessionId || !activeHallId || !shift) return;
    const [tables, activeOrders] = await Promise.all([
      api.listTables(activeHallId),
      api.listActiveOrdersByHall(activeHallId),
    ]);
    setTablesDto(tables.filter((table) => table.active));
    setActiveOrdersDto(activeOrders);
  }, [activeHallId, api, shift]);

  const refreshCurrentPrechecks = useCallback(async (orderId?: string) => {
    const id = orderId ?? activeOrdersDto.find((order) => order.table_id === selectedTableId)?.id;
    if (!id || !authRef.current.sessionId) {
      setPrechecksDto([]);
      return;
    }
    setPrechecksDto(await api.listPrechecksByOrder(id));
  }, [activeOrdersDto, api, selectedTableId]);

  const refreshCurrentPricing = useCallback(async (orderId?: string) => {
    const id = orderId ?? activeOrdersDto.find((order) => order.table_id === selectedTableId)?.id;
    if (!id || !authRef.current.sessionId) {
      setCurrentPricingDto(null);
      return;
    }
    setCurrentPricingDto(await api.getOrderPricing(id).catch(() => null));
  }, [activeOrdersDto, api, selectedTableId]);

  const applyClosedOrdersPage = useCallback(async (page: number, businessDate: string) => {
    const nextPage = Math.max(1, page);
    const rows = await api.listClosedOrders({
      businessDateLocal: businessDate,
      limit: CLOSED_ORDERS_PAGE_SIZE + 1,
      offset: (nextPage - 1) * CLOSED_ORDERS_PAGE_SIZE,
    });
    setClosedOrdersDto(rows.slice(0, CLOSED_ORDERS_PAGE_SIZE));
    setClosedOrdersHasNextPage(rows.length > CLOSED_ORDERS_PAGE_SIZE);
    setClosedOrdersPage(nextPage);
    setClosedOrdersBusinessDate(businessDate);
    setClosedOrdersError('');
  }, [api]);

  const loadClosedOrdersPage = useCallback(async (page: number, businessDate = closedOrdersBusinessDate) => {
    if (!authRef.current.sessionId) return;
    setClosedOrdersLoading(true);
    try {
      await applyClosedOrdersPage(page, businessDate);
    } catch (error) {
      setClosedOrdersDto([]);
      setClosedOrdersHasNextPage(false);
      setClosedOrdersError(t.errors.unknown);
      handleError(error, t.ops.closedOrdersLoadFailed);
    } finally {
      setClosedOrdersLoading(false);
    }
  }, [applyClosedOrdersPage, closedOrdersBusinessDate, handleError]);

  const refreshActivity = useCallback(async () => {
    if (!authRef.current.sessionId) return;
    const canViewSync = actor?.permissions.includes('pos.sync.view') ?? false;
    const [closedOrders, sync, storage] = await Promise.all([
      api.listClosedOrders({
        businessDateLocal: closedOrdersBusinessDate,
        limit: CLOSED_ORDERS_PAGE_SIZE + 1,
        offset: (closedOrdersPage - 1) * CLOSED_ORDERS_PAGE_SIZE,
      }).catch(() => []),
      canViewSync ? api.getSyncStatus().catch(() => null) : Promise.resolve(null),
      canViewSync ? api.getStorageStatus().catch(() => null) : Promise.resolve(null),
    ]);
    setClosedOrdersDto(closedOrders.slice(0, CLOSED_ORDERS_PAGE_SIZE));
    setClosedOrdersHasNextPage(closedOrders.length > CLOSED_ORDERS_PAGE_SIZE);
    setSyncStatusDto(sync);
    setStorageStatusDto(storage);
  }, [actor, api, closedOrdersBusinessDate, closedOrdersPage]);

  const refreshAll = useCallback(async () => {
    try {
      await refreshOps();
      await refreshDirectory();
      await refreshFloor();
      await refreshCurrentPrechecks();
      await refreshCurrentPricing();
      await refreshActivity();
    } catch (error) {
      handleError(error, t.ops.posRefreshFailed);
    }
  }, [handleError, refreshActivity, refreshCurrentPrechecks, refreshCurrentPricing, refreshDirectory, refreshFloor, refreshOps]);

  useEffect(() => {
    void refreshIdentity();
  }, [refreshIdentity]);

  useEffect(() => {
    if (isEdgePaired) return undefined;
    const timer = window.setInterval(() => {
      void refreshProvisioningStatus();
    }, 2500);
    return () => window.clearInterval(timer);
  }, [isEdgePaired, refreshProvisioningStatus]);

  useEffect(() => {
    const restore = async () => {
      if (!auth.sessionId || !auth.nodeDeviceId) return;
      try {
        const session = await api.getAuthSession();
        setActor(session.actor);
        setAuth({ ...authRef.current, actorEmployeeId: session.actor.employee_id });
        setPinLocked(false);
        await refreshOps();
      } catch {
        setPinLocked(true);
      }
    };
    void restore();
  }, [api, auth.nodeDeviceId, auth.sessionId, refreshOps, setAuth]);

  useEffect(() => {
    void refreshDirectory();
  }, [refreshDirectory, shift]);

  useEffect(() => {
    void refreshFloor();
  }, [refreshFloor]);

  useEffect(() => {
    void refreshCurrentPrechecks();
  }, [refreshCurrentPrechecks]);

  useEffect(() => {
    void refreshCurrentPricing();
  }, [refreshCurrentPricing]);

  useEffect(() => {
    if (!isPinLocked && actor && !shift) {
      setSelectedTableId(null);
      setCurrentSection('cash');
    }
  }, [actor, isPinLocked, shift]);

  const activePrecheck = useMemo(() => activeIssuedPrecheck(prechecksDto), [prechecksDto]);
  const backendCurrentOrder = useMemo(
    () => activeOrdersDto.find((order) => order.table_id === selectedTableId) ?? null,
    [activeOrdersDto, selectedTableId],
  );
  const currentOrder = useMemo(
    () => backendCurrentOrder ? applyPricingToOrder(mapOrder(backendCurrentOrder, activePrecheck), currentPricingDto) : null,
    [activePrecheck, backendCurrentOrder, currentPricingDto],
  );
  const activeOrders = useMemo(() => activeOrdersDto.map((order) => mapOrder(order, null)), [activeOrdersDto]);
  const tables = useMemo(
    () => tablesDto.map((table) => mapTable(table, activeOrdersDto.find((order) => order.table_id === table.id))),
    [activeOrdersDto, tablesDto],
  );
  const halls = useMemo(() => hallsDto.map(mapHall), [hallsDto]);
  const cashSession = useMemo(() => mapCashSession(cashSessionDto, actor), [actor, cashSessionDto]);
  const currentOperator = useMemo(() => mapOperator(actor, shift), [actor, shift]);
  const employeeShifts = useMemo(() => recentShifts.map((item) => mapOperator(actor, item)).filter(Boolean) as EmployeeShift[], [actor, recentShifts]);
  const closedOrders = useMemo(() => closedOrdersDto.map((item) => mapClosedOrder(item)), [closedOrdersDto]);
  const menuItems = useMemo(() => menuDto.map(mapMenuItem), [menuDto]);
  const pricingPolicies = useMemo(() => pricingPoliciesDto.map(mapPricingPolicy), [pricingPoliciesDto]);
  const syncStatus = syncStatusDto ? mapSyncStatus(syncStatusDto) : isEdgePaired ? 'online' : 'offline';
  const syncRevision = syncStatusDto?.last_cloud_version ?? 0;
  const appVersion = useMemo(() => {
    const runtime = storageStatusDto?.runtime_versions.find((item) => item.module_name === 'pos-backend') ?? storageStatusDto?.runtime_versions[0];
    const schemaVersion = runtime?.schema_version || storageStatusDto?.schema_migrations[0]?.version || '';
    const moduleVersion = runtime?.module_version || '';
    if (moduleVersion && schemaVersion) return `${moduleVersion} / ${schemaVersion}`;
    return moduleVersion || schemaVersion || t.common.notAvailable;
  }, [storageStatusDto]);

  const setActiveHallId = useCallback((id: string) => {
    setActiveHallIdState(id);
    setSelectedTableId(null);
  }, []);

  const pinLogin = useCallback(async (pin: string) => {
    try {
      await refreshIdentity();
      const result = await api.pinLogin(pin);
      setActor(result.actor);
      setAuth({
        ...authRef.current,
        sessionId: result.session.id,
        nodeDeviceId: result.session.node_device_id,
        restaurantId: result.session.restaurant_id,
        actorEmployeeId: result.actor.employee_id,
      });
      setPinLocked(false);
      addLogEvent(t.ops.employeeAuthorized(result.actor.name), 'success');
      await refreshAll();
      return true;
    } catch (error) {
      handleError(error, t.ops.authFailed);
      return false;
    }
  }, [addLogEvent, api, handleError, refreshAll, refreshIdentity, setAuth]);

  const logout = useCallback(async () => {
    try {
      if (authRef.current.sessionId) await api.logout();
    } catch (error) {
      handleError(error, t.ops.logoutFailed);
    } finally {
      localStorage.removeItem(storageKeys.sessionId);
      setAuth({ ...authRef.current, sessionId: '', actorEmployeeId: '' });
      setActor(null);
      setShift(null);
      setCashSessionDto(null);
      setSelectedTableId(null);
      setPinLocked(true);
      addLogEvent(t.ops.terminalLocked, 'info');
    }
  }, [addLogEvent, api, handleError, setAuth]);

  const openEmployeeShift = useCallback(async () => {
    try {
      await api.openShift();
      addLogEvent(t.ops.employeeShiftOpened, 'success');
      await refreshAll();
    } catch (error) {
      handleError(error, t.ops.employeeShiftOpenFailed);
    }
  }, [addLogEvent, api, handleError, refreshAll]);

  const closeEmployeeShift = useCallback(async () => {
    if (!shift) return;
    try {
      await api.closeShift(shift.id);
      addLogEvent(t.ops.employeeShiftClosed, 'success');
      await logout();
    } catch (error) {
      handleError(error, t.ops.employeeShiftCloseFailed);
    }
  }, [addLogEvent, api, handleError, logout, shift]);

  const openCashSession = useCallback(async (initialAmount: number) => {
    try {
      await api.openCashSession(initialAmount);
      addLogEvent(t.ops.cashSessionOpened, 'success');
      await refreshAll();
    } catch (error) {
      handleError(error, t.ops.cashSessionOpenFailed);
    }
  }, [addLogEvent, api, handleError, refreshAll]);

  const closeCashSession = useCallback(async () => {
    if (!cashSessionDto) return;
    try {
      await api.closeCashSession(cashSessionDto.id, cashSessionDto.opening_cash_amount);
      addLogEvent(t.ops.cashSessionClosed, 'success');
      await refreshAll();
    } catch (error) {
      handleError(error, t.ops.cashSessionCloseFailed);
    }
  }, [addLogEvent, api, cashSessionDto, handleError, refreshAll]);

  const addCashDrawerEvent = useCallback(async (type: 'in' | 'out', amount: number, reason: string) => {
    if (!cashSessionDto) return;
    try {
      await api.recordCashDrawerEvent(cashSessionDto.id, type === 'in' ? 'cash_in' : 'cash_out', amount, reason);
      setCashDrawerEventsDto((prev) => [
        {
          id: `${Date.now()}`,
          edge_cash_drawer_event_id: `${Date.now()}`,
          cash_session_id: cashSessionDto.id,
          restaurant_id: cashSessionDto.restaurant_id,
          device_id: cashSessionDto.device_id,
          shift_id: cashSessionDto.shift_id,
          created_by_employee_id: authRef.current.actorEmployeeId,
          event_type: type === 'in' ? 'cash_in' : 'cash_out',
          amount,
          reason,
          note: '',
          occurred_at: new Date().toISOString(),
          created_at: new Date().toISOString(),
        },
        ...prev,
      ]);
      addLogEvent(t.ops.cashDrawerEventRecorded, 'success');
      await refreshActivity();
    } catch (error) {
      handleError(error, t.ops.cashDrawerEventFailed);
    }
  }, [addLogEvent, api, cashSessionDto, handleError, refreshActivity]);

  const createOrderForTable = useCallback(async (tableId: string, guestsCount: number) => {
    const table = tablesDto.find((item) => item.id === tableId);
    if (!table) return;
    try {
      const created = await api.createOrder(tableId, table.name, guestsCount);
      setSelectedTableId(tableId);
      setCurrentSection('order');
      addLogEvent(t.ops.orderCreated(created.edge_order_id), 'success');
      await refreshFloor();
    } catch (error) {
      handleError(error, t.ops.orderCreateFailed);
    }
  }, [addLogEvent, api, handleError, refreshFloor, tablesDto]);

  const addMenuItemToOrder = useCallback(async (item: MenuItem, selectedModifiers: SelectedModifier[]): Promise<CommandResult> => {
    if (!backendCurrentOrder) return { success: false, errorKey: 'blocks.noOrderSelected' };
    try {
      await api.addOrderLine(backendCurrentOrder.id, item.id, 1, selectedModifiersToPayload(selectedModifiers));
      await refreshFloor();
      await refreshCurrentPrechecks(backendCurrentOrder.id);
      await refreshCurrentPricing(backendCurrentOrder.id);
      return { success: true };
    } catch (error) {
      handleError(error, t.ops.orderLineAddFailed);
      return { success: false, errorKey: 'errors.conflict' };
    }
  }, [api, backendCurrentOrder, handleError, refreshCurrentPrechecks, refreshCurrentPricing, refreshFloor]);

  const editOrderLineModifiers = useCallback(async (lineId: string, selectedModifiers: SelectedModifier[]): Promise<CommandResult> => {
    if (!backendCurrentOrder) return { success: false, errorKey: 'blocks.noOrderSelected' };
    try {
      await api.updateOrderLineModifiers(backendCurrentOrder.id, lineId, selectedModifiersToPayload(selectedModifiers));
      await refreshFloor();
      await refreshCurrentPrechecks(backendCurrentOrder.id);
      await refreshCurrentPricing(backendCurrentOrder.id);
      return { success: true };
    } catch (error) {
      handleError(error, t.ops.modifiersUpdateFailed);
      return { success: false, errorKey: 'errors.conflict' };
    }
  }, [api, backendCurrentOrder, handleError, refreshCurrentPrechecks, refreshCurrentPricing, refreshFloor]);

  const applyPricingPolicy = useCallback(async (policyId: string, orderLineId = '', reason = ''): Promise<CommandResult> => {
    if (!backendCurrentOrder) return { success: false, errorKey: 'blocks.noOrderSelected' };
    const policy = pricingPoliciesDto.find((item) => item.id === policyId);
    if (!policy) return { success: false, errorKey: 'errors.conflict' };
    try {
      if (policy.kind === 'discount') {
        await api.applyDiscountPolicy(backendCurrentOrder.id, policy.id, policy.scope === 'line' ? orderLineId : '', reason);
      } else {
        await api.applySurchargePolicy(backendCurrentOrder.id, policy.id, reason);
      }
      addLogEvent(t.ops.pricingPolicyApplied(policy.name), 'success');
      await refreshFloor();
      await refreshCurrentPrechecks(backendCurrentOrder.id);
      await refreshCurrentPricing(backendCurrentOrder.id);
      await refreshActivity();
      return { success: true };
    } catch (error) {
      handleError(error, t.ops.pricingPolicyFailed);
      return { success: false, errorKey: 'errors.conflict' };
    }
  }, [addLogEvent, api, backendCurrentOrder, handleError, pricingPoliciesDto, refreshActivity, refreshCurrentPrechecks, refreshCurrentPricing, refreshFloor]);

  const removeOrderLine = useCallback(async (lineId: string) => {
    if (!backendCurrentOrder) return;
    try {
      await api.voidOrderLine(backendCurrentOrder.id, lineId, 'Удаление позиции на POS');
      await refreshFloor();
      await refreshCurrentPricing(backendCurrentOrder.id);
    } catch (error) {
      handleError(error, t.ops.orderLineRemoveFailed);
    }
  }, [api, backendCurrentOrder, handleError, refreshCurrentPricing, refreshFloor]);

  const changeLineQuantity = useCallback(async (lineId: string, newQty: number): Promise<CommandResult> => {
    if (!backendCurrentOrder) return { success: false, errorKey: 'blocks.noOrderSelected' };
    try {
      if (newQty <= 0) {
        await removeOrderLine(lineId);
      } else {
        await api.changeOrderLineQuantity(backendCurrentOrder.id, lineId, newQty);
        await refreshFloor();
        await refreshCurrentPricing(backendCurrentOrder.id);
      }
      return { success: true };
    } catch (error) {
      handleError(error, t.ops.quantityChangeFailed);
      return { success: false, errorKey: 'errors.conflict' };
    }
  }, [api, backendCurrentOrder, handleError, refreshCurrentPricing, refreshFloor, removeOrderLine]);

  const updateCommentAndCourse = useCallback(async (lineId: string, comment: string, course: number) => {
    if (!backendCurrentOrder) return;
    try {
      await api.updateOrderLineDetails(backendCurrentOrder.id, lineId, String(course), comment);
      await refreshFloor();
      await refreshCurrentPricing(backendCurrentOrder.id);
    } catch (error) {
      handleError(error, t.ops.orderLineUpdateFailed);
    }
  }, [api, backendCurrentOrder, handleError, refreshCurrentPricing, refreshFloor]);

  const issuePrecheck = useCallback(async () => {
    if (!backendCurrentOrder) return;
    try {
      await api.issuePrecheck(backendCurrentOrder.id);
      addLogEvent(t.ops.precheckIssued, 'success');
      await refreshFloor();
      await refreshCurrentPrechecks(backendCurrentOrder.id);
    } catch (error) {
      handleError(error, t.ops.precheckIssueFailed);
    }
  }, [addLogEvent, api, backendCurrentOrder, handleError, refreshCurrentPrechecks, refreshFloor]);

  const cancelPrecheck = useCallback(async (managerPin: string, reason: string) => {
    if (!activePrecheck) return false;
    try {
      await api.cancelPrecheck(activePrecheck.id, managerPin, reason);
      addLogEvent(t.ops.precheckCancelled, 'success');
      await refreshFloor();
      await refreshCurrentPrechecks(activePrecheck.order_id);
      return true;
    } catch (error) {
      handleError(error, t.ops.precheckCancelFailed);
      return false;
    }
  }, [activePrecheck, addLogEvent, api, handleError, refreshCurrentPrechecks, refreshFloor]);

  const reprintPrecheck = useCallback(async () => {
    if (!activePrecheck) return;
    try {
      await api.reprintPrecheck(activePrecheck.id);
      addLogEvent(t.ops.precheckReprintSent, 'success');
    } catch (error) {
      handleError(error, t.ops.precheckReprintFailed);
    }
  }, [activePrecheck, addLogEvent, api, handleError]);

  const payOrder = useCallback(async (method: 'cash' | 'card', inputAmount: number): Promise<PaymentResult> => {
    if (!activePrecheck) return { success: false, change: 0, errorKey: 'errors.orderLocked' };
    try {
      await api.capturePrecheckPayment(activePrecheck.id, method, inputAmount, activePrecheck.currency_code);
      const change = paymentChange(activePrecheck.remaining_total, inputAmount);
      addLogEvent(t.ops.paymentCaptured, 'success');
      setSelectedTableId(null);
      await refreshFloor();
      await refreshActivity();
      return { success: true, change };
    } catch (error) {
      handleError(error, t.ops.paymentCaptureFailed);
      return { success: false, change: 0, errorKey: 'errors.paymentFailed' };
    }
  }, [activePrecheck, addLogEvent, api, handleError, refreshActivity, refreshFloor]);

  const refundCheck = useCallback(async (checkId: string, reason: string, disposition: 'waste' | 'return') => {
    try {
      await api.recordCheckRefund(checkId, {
        reason,
        operationKind: 'full',
        inventoryDisposition: dispositionToBackend(disposition),
      });
      addLogEvent(t.ops.refundRecorded, 'success');
      await refreshActivity();
    } catch (error) {
      handleError(error, t.ops.refundFailed);
    }
  }, [addLogEvent, api, handleError, refreshActivity]);

  const partialRefundCheck = useCallback(async (checkId: string, lineId: string, qtyToRefund: number, reason: string, disposition: 'waste' | 'return') => {
    const order = closedOrders.find((item) => item.id === checkId);
    const line = order?.lines.find((item) => item.id === lineId);
    if (!line) return;
    try {
      await api.recordCheckRefund(checkId, {
        reason,
        operationKind: 'partial',
        inventoryDisposition: dispositionToBackend(disposition),
        items: [{
          scope: 'order_line',
          orderLineId: line.id,
          quantity: qtyToRefund,
          amount: line.price * qtyToRefund,
          currency: 'RUB',
          taxAmount: 0,
        }],
      });
      addLogEvent(t.ops.partialRefundRecorded, 'success');
      await refreshActivity();
    } catch (error) {
      handleError(error, t.ops.partialRefundFailed);
    }
  }, [addLogEvent, api, closedOrders, handleError, refreshActivity]);

  const reprintCheck = useCallback(async (checkId: string) => {
    try {
      await api.reprintCheck(checkId);
      addLogEvent(t.ops.checkReprintSent, 'success');
    } catch (error) {
      handleError(error, t.ops.checkReprintFailed);
    }
  }, [addLogEvent, api, handleError]);

  const syncOutbox = useCallback(async () => {
    try {
      await api.retryFailedOutbox();
      addLogEvent(t.ops.outboxRetryStarted, 'success');
      await refreshActivity();
    } catch (error) {
      handleError(error, t.ops.outboxRetryFailed);
    }
  }, [addLogEvent, api, handleError, refreshActivity]);

  const toggleTheme = useCallback(() => {
    setThemeSettings((prev) => ({
      ...prev,
      mode: prev.mode === 'light' ? 'dark' : 'light',
    }));
  }, []);

  const setThemeMode = useCallback((mode: POSThemeMode) => {
    setThemeSettings((prev) => ({ ...prev, mode }));
  }, []);

  const setThemeScheme = useCallback((scheme: POSThemeSchemeId) => {
    setThemeSettings((prev) => ({ ...prev, scheme }));
  }, []);

  const value = useMemo<POSContextType>(() => ({
    currentSection,
    setCurrentSection,
    activeHallId,
    setActiveHallId,
    theme: themeSettings.mode,
    themeScheme: themeSettings.scheme,
    themeSchemes: posThemeSchemes,
    toggleTheme,
    setThemeMode,
    setThemeScheme,
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
    cashDrawerEvents: cashDrawerEventsDto.map((event) => mapCashDrawerEvent(event, actor)),
    addCashDrawerEvent,
    halls,
    menuItems,
    tables,
    setSelectedTableId,
    selectedTableId,
    createOrderForTable,
    activeOrders,
    currentOrder,
    addMenuItemToOrder,
    editOrderLineModifiers,
    removeOrderLine,
    changeLineQuantity,
    updateCommentAndCourse,
    issuePrecheck,
    cancelPrecheck,
    reprintPrecheck,
    payOrder,
    closedOrders,
    closedOrdersPage,
    closedOrdersPageSize: CLOSED_ORDERS_PAGE_SIZE,
    closedOrdersHasNextPage,
    closedOrdersLoading,
    closedOrdersError,
    closedOrdersBusinessDate,
    loadClosedOrdersPage,
    refundCheck,
    partialRefundCheck,
    reprintCheck,
    pricingPolicies,
    applyPricingPolicy,
    outboxCount: outboxCount(syncStatusDto),
    syncOutbox,
    syncStatus,
    syncRevision,
    appVersion,
    logEvents,
    addLogEvent,
    authSnapshot: auth,
    isEdgePaired,
    provisioningStatus: provisioningStatusDto,
    provisioningLoading,
    provisioningError,
    refreshProvisioningStatus,
    registerCloudProvisioning,
    pairViaLicense,
  }), [
    activeHallId,
    activeOrders,
    actor,
    auth,
    addCashDrawerEvent,
    addLogEvent,
    addMenuItemToOrder,
    appVersion,
    applyPricingPolicy,
    cancelPrecheck,
    cashDrawerEventsDto,
    cashSession,
    changeLineQuantity,
    closeCashSession,
    closeEmployeeShift,
    closedOrders,
    closedOrdersBusinessDate,
    closedOrdersError,
    closedOrdersHasNextPage,
    closedOrdersLoading,
    closedOrdersPage,
    createOrderForTable,
    currentOperator,
    currentOrder,
    currentSection,
    editOrderLineModifiers,
    employeeShifts,
    halls,
    isPinLocked,
    isEdgePaired,
    logEvents,
    loadClosedOrdersPage,
    logout,
    menuItems,
    openCashSession,
    openEmployeeShift,
    partialRefundCheck,
    payOrder,
    pinLogin,
    pairViaLicense,
    provisioningError,
    provisioningLoading,
    provisioningStatusDto,
    pricingPolicies,
    refreshProvisioningStatus,
    registerCloudProvisioning,
    refundCheck,
    removeOrderLine,
    reprintCheck,
    reprintPrecheck,
    selectedTableId,
    setActiveHallId,
    syncOutbox,
    syncStatus,
    syncStatusDto,
    syncRevision,
    tables,
    themeSettings,
    toggleTheme,
    setThemeMode,
    setThemeScheme,
    updateCommentAndCourse,
    issuePrecheck,
  ]);

  return <POSContext.Provider value={value}>{children}</POSContext.Provider>;
};

export const usePOS = () => {
  const context = useContext(POSContext);
  if (context === undefined) {
    throw new Error('usePOS must be used within a POSProvider');
  }
  return context;
};

export const usePOSMenuItems = () => {
  const context = useContext(POSContext);
  if (context === undefined) {
    throw new Error('usePOSMenuItems must be used within a POSProvider');
  }
  return context;
};

function applyPricingToOrder(order: Order, pricing: BackendPricingCalculation | null): Order {
  if (!pricing) return order;
  const lineTotals = new Map(pricing.lines.map((line) => [line.order_line_id, line.total_minor]));
  return {
    ...order,
    subtotal: pricing.subtotal_minor,
    tax: pricing.tax_total_minor,
    discount: pricing.discount_total_minor,
    total: pricing.grand_total_minor,
    lines: order.lines.map((line) => ({
      ...line,
      totalPrice: lineTotals.get(line.id) ?? line.totalPrice,
    })),
  };
}
