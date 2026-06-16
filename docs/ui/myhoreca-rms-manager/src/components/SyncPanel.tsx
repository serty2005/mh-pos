/**
 * @license
 * SPDX-License-Identifier: Apache-2.0
 */

import { useState } from 'react';
import { RefreshCw, Radio, HardDrive, Wifi, WifiOff, FileText, CheckCircle2, AlertTriangle, Play, HelpCircle, Server } from 'lucide-react';
import { POSTerminal, SyncLog } from '../types';

interface SyncPanelProps {
  terminals: POSTerminal[];
  syncLogs: SyncLog[];
  triggerForceSync: () => void;
  isSyncingAll: boolean;
  onSimulateOutage: (terminalId: string) => void;
  onClearLogs: () => void;
}

export default function SyncPanel({
  terminals,
  syncLogs,
  triggerForceSync,
  isSyncingAll,
  onSimulateOutage,
  onClearLogs,
}: SyncPanelProps) {
  const [selectedTerminal, setSelectedTerminal] = useState<POSTerminal | null>(null);

  const getStatusIcon = (status: POSTerminal['status']) => {
    if (status === 'online') {
      return (
        <span className="flex items-center gap-1 text-emerald-600 font-semibold text-xs py-1 px-2.5 rounded-lg bg-emerald-50 border border-emerald-100">
          <Wifi className="w-3.5 h-3.5 shrink-0" />
          <span>Онлайн</span>
        </span>
      );
    } else if (status === 'syncing') {
      return (
        <span className="flex items-center gap-1 text-blue-600 font-semibold text-xs py-1 px-2.5 rounded-lg bg-blue-50 border border-blue-150 animate-pulse">
          <RefreshCw className="w-3.5 h-3.5 animate-spin" />
          <span>Синхронизация</span>
        </span>
      );
    } else {
      return (
        <span className="flex items-center gap-1 text-rose-500 font-semibold text-xs py-1 px-2.5 rounded-lg bg-rose-50 border border-rose-100">
          <WifiOff className="w-3.5 h-3.5 shrink-0" />
          <span>Оффлайн</span>
        </span>
      );
    }
  };

  const getLogStatusClass = (status: SyncLog['status']) => {
    if (status === 'success') return 'border-emerald-100 bg-emerald-50/40 text-emerald-800';
    if (status === 'warning') return 'border-amber-100 bg-amber-50/40 text-amber-800';
    return 'border-rose-100 bg-rose-50/40 text-rose-800';
  };

  const getLogIcon = (status: SyncLog['status']) => {
    if (status === 'success') return <CheckCircle2 className="w-4 h-4 text-emerald-500" />;
    if (status === 'warning') return <AlertTriangle className="w-4 h-4 text-amber-500" />;
    return <AlertTriangle className="w-4 h-4 text-rose-500" />;
  };

  const getLogTypeLabel = (type: SyncLog['type']) => {
    const map = {
      menu_push: 'МЕНЮ ➔ POS',
      sales_pull: 'ЧЕКИ ➔ RMS',
      staff_update: 'ШТАТ ➔ POS',
      heartbeat: 'ПИНГ',
    };
    return map[type] || type;
  };

  // Stats for terminals
  const onlineCount = terminals.filter((t) => t.status === 'online').length;
  const offlineCount = terminals.filter((t) => t.status === 'offline').length;
  const totalPendingTx = terminals.reduce((sum, t) => sum + t.pendingTransactions, 0);

  return (
    <div className="flex-1 overflow-y-auto bg-slate-50 p-8">
      {/* Header */}
      <div className="flex flex-col md:flex-row justify-between items-start md:items-center gap-4 mb-8">
        <div>
          <h2 className="text-2xl font-bold tracking-tight text-slate-900 font-sans">
            Служба синхронизации POS терминалов
          </h2>
          <p className="text-sm text-slate-500 mt-1">
            Мониторинг кассовых аппаратов, выгрузка офлайн-чеков, сверка баз данных и рассылка обновлений меню.
          </p>
        </div>

        <button
          onClick={triggerForceSync}
          disabled={isSyncingAll}
          className={`flex items-center gap-2.5 px-5 py-3 rounded-xl text-sm font-semibold transition-all duration-300 shadow-sm border ${
            isSyncingAll
              ? 'bg-slate-200 border-slate-300 text-slate-400 cursor-not-allowed'
              : 'bg-blue-600 hover:bg-blue-700 text-white border-blue-700 hover:shadow-md'
          }`}
        >
          <RefreshCw className={`w-4 h-4 ${isSyncingAll ? 'animate-spin' : ''}`} />
          {isSyncingAll ? 'Идет синхронизация...' : 'Выгрузить & Обновить POS-сетку'}
        </button>
      </div>

      {/* Network Metrics row */}
      <div className="grid grid-cols-1 sm:grid-cols-4 gap-6 mb-8">
        <div className="bg-white p-5 rounded-2xl border border-slate-200 shadow-sm flex items-center justify-between">
          <div>
            <span className="text-xs text-slate-400 font-mono font-bold uppercase tracking-wider">Терминалы</span>
            <p className="text-2xl font-bold text-slate-900 mt-1">{terminals.length} устр.</p>
          </div>
          <div className="p-3.5 bg-blue-50 rounded-xl text-blue-600">
            <Server className="w-5 h-5 animate-pulse" />
          </div>
        </div>

        <div className="bg-white p-5 rounded-2xl border border-slate-200 shadow-sm flex items-center justify-between">
          <div>
            <span className="text-xs text-slate-400 font-mono font-bold uppercase tracking-wider">Активная связь</span>
            <p className="text-2xl font-bold text-emerald-600 mt-1">{onlineCount} POS</p>
          </div>
          <div className="p-3.5 bg-emerald-50 rounded-xl text-emerald-600">
            <Radio className="w-5 h-5" />
          </div>
        </div>

        <div className="bg-white p-5 rounded-2xl border border-slate-200 shadow-sm flex items-center justify-between">
          <div>
            <span className="text-xs text-slate-400 font-mono font-bold uppercase tracking-wider">Отключены</span>
            <p className="text-2xl font-bold text-rose-500 mt-1">{offlineCount} POS</p>
          </div>
          <div className="p-3.5 bg-rose-50 rounded-xl text-rose-500">
            <WifiOff className="w-5 h-5" />
          </div>
        </div>

        <div className="bg-white p-5 rounded-2xl border border-slate-200 shadow-sm flex items-center justify-between">
          <div>
            <span className="text-xs text-slate-400 font-mono font-bold uppercase tracking-wider">Чеки в очереди</span>
            <p className="text-2xl font-bold text-amber-600 mt-1">{totalPendingTx} чеков</p>
          </div>
          <div className="p-3.5 bg-amber-50 rounded-xl text-amber-600">
            <HardDrive className="w-5 h-5" />
          </div>
        </div>
      </div>

      {/* Main split grid */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
        {/* Terminal Directory view */}
        <div className="lg:col-span-2 bg-white p-6 rounded-2xl border border-slate-200 shadow-sm">
          <div className="flex justify-between items-center mb-5">
            <div>
              <h3 className="font-sans font-bold text-base text-slate-900">
                Кассовые узлы в реестре системы
              </h3>
              <p className="text-xs text-slate-400 mt-0.5">Все активные RMS-POS терминалы вашей сети</p>
            </div>
            <span className="bg-emerald-50 text-emerald-700 text-[10px] font-mono font-semibold px-2 py-1 rounded-md border border-emerald-150">
              Локальный сегмент: 192.168.1.*
            </span>
          </div>

          <div className="space-y-4">
            {terminals.map((terminal) => {
              const matchesSelected = selectedTerminal?.id === terminal.id;
              return (
                <div
                  key={terminal.id}
                  onClick={() => setSelectedTerminal(terminal)}
                  className={`p-4 border rounded-2xl flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 transition-all duration-300 cursor-pointer select-none ${
                    matchesSelected
                      ? 'border-blue-500 bg-blue-50/10 shadow-md shadow-blue-100/10'
                      : 'border-slate-150 hover:border-slate-300 bg-slate-50/40 hover:bg-slate-50'
                  }`}
                >
                  <div className="flex items-center gap-3.5">
                    <div className="p-3 bg-white border border-slate-150 rounded-xl text-slate-700 shadow-sm shrink-0">
                      🖥️
                    </div>
                    <div>
                      <div className="flex items-center gap-2">
                        <h4 className="text-sm font-bold text-slate-900">
                          {terminal.name}
                        </h4>
                        <span className="text-[10px] bg-slate-100 text-slate-500 font-mono px-1.5 py-0.5 rounded leading-none">
                          {terminal.version}
                        </span>
                      </div>
                      <div className="flex flex-wrap items-center gap-x-3 gap-y-1 mt-1 text-[11px] text-slate-400 font-mono">
                        <span>IP: {terminal.ipAddress}</span>
                        <span className="text-slate-200">|</span>
                        <span>Локация: {terminal.location}</span>
                      </div>
                    </div>
                  </div>

                  <div className="flex items-center justify-between sm:justify-end gap-4 border-t sm:border-t-0 pt-3 sm:pt-0 border-slate-100">
                    <div className="text-right">
                      {terminal.pendingTransactions > 0 && (
                        <span className="inline-block text-[10px] font-mono font-bold text-amber-700 bg-amber-50 border border-amber-150 px-2 py-0.5 rounded-lg mb-1">
                          {terminal.pendingTransactions} чека оффлайн
                        </span>
                      )}
                      <p className="text-[10px] font-mono text-slate-400">
                        Синхр.: {terminal.lastSyncTime}
                      </p>
                    </div>

                    <div className="flex items-center gap-2">
                      {getStatusIcon(terminal.status)}
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          onSimulateOutage(terminal.id);
                        }}
                        className={`p-1.5 border rounded-lg hover:shadow-inner transition-colors ${
                          terminal.status === 'offline'
                            ? 'bg-emerald-50 text-emerald-600 border-emerald-150 hover:bg-emerald-100'
                            : 'bg-rose-50 text-rose-500 border-rose-150 hover:bg-rose-100'
                        }`}
                        title={terminal.status === 'offline' ? 'Подключить терминал назад' : 'Симулировать обрыв связи'}
                      >
                        {terminal.status === 'offline' ? <Play className="w-3.5 h-3.5" /> : <WifiOff className="w-3.5 h-3.5" />}
                      </button>
                    </div>
                  </div>
                </div>
              );
            })}
          </div>

          {/* Quick Troubleshooting Guide */}
          <div className="mt-6 p-4 bg-slate-50 border border-slate-150 rounded-2xl text-xs text-slate-500 flex items-start gap-3">
            <HelpCircle className="w-5 h-5 text-blue-500 shrink-0 mt-0.5" />
            <div>
              <p className="font-bold text-slate-700 mb-0.5">Вспомогательное правило синхронизации:</p>
              <p className="leading-relaxed">
                Если терминал находится в режиме <span className="font-semibold text-rose-600">Оффлайн</span>, он продолжает собирать транзакции локально во внутреннюю оффлайн БД. Как только соединение будет восстановлено, RMS-POS автоматически выгрузит накопившиеся чеки в центральную панель для пересчета аналитики.
              </p>
            </div>
          </div>
        </div>

        {/* Real-time System Sync telemetry Console logger */}
        <div className="bg-slate-900 rounded-3xl border border-slate-800 p-6 text-slate-100 flex flex-col justify-between h-[520px] max-h-[520px]">
          <div className="flex flex-col flex-1 h-full min-h-0">
            <div className="flex justify-between items-center mb-4 pb-3 border-b border-slate-800 select-none">
              <div className="flex items-center gap-2">
                <FileText className="w-4 h-4 text-blue-400 animate-pulse" />
                <h3 className="font-sans font-bold text-sm text-slate-200">
                  Журнал событий ядра синхронизации
                </h3>
              </div>
              <button
                onClick={onClearLogs}
                className="text-[10px] font-mono uppercase tracking-wider text-blue-400 hover:text-blue-200 border border-slate-800 hover:border-blue-400 px-2.5 py-1 rounded"
              >
                Очистить лог
              </button>
            </div>

            {/* Simulated Live Terminal scroll content */}
            <div className="flex-1 overflow-y-auto space-y-3 font-mono text-[11px] pr-1.5 min-h-0">
              {syncLogs.map((log) => (
                <div
                  key={log.id}
                  className={`p-3 rounded-xl border flex flex-col gap-1.5 transition-all ${getLogStatusClass(log.status)}`}
                >
                  <div className="flex justify-between items-center text-[10px] font-semibold">
                    <div className="flex items-center gap-1.5">
                      {getLogIcon(log.status)}
                      <span className="tracking-wide uppercase bg-slate-950/20 px-1.5 py-0.5 rounded leading-none text-slate-300">
                        {getLogTypeLabel(log.type)}
                      </span>
                    </div>
                    <span className="text-slate-400">{log.timestamp}</span>
                  </div>

                  <p className="font-bold text-slate-100 leading-snug">
                    {log.terminalName}:
                  </p>
                  <p className="text-slate-300 leading-relaxed font-semibold">
                    {log.details}
                  </p>
                </div>
              ))}
              {syncLogs.length === 0 && (
                <div className="h-full flex items-center justify-center text-center text-slate-500 py-12">
                  Нет зафиксированных логов.
                </div>
              )}
            </div>
          </div>

          <div className="pt-4 border-t border-slate-800 mt-4 flex items-center justify-between text-[11px] font-mono text-slate-500">
            <span>Протокол: HorecaSync BLE/HTTP v2</span>
            <span className="flex items-center gap-1">
              <span className="w-2.5 h-2.5 rounded-full bg-blue-500 animate-ping"></span>
              <span className="text-blue-400">Служба активна</span>
            </span>
          </div>
        </div>
      </div>
    </div>
  );
}
