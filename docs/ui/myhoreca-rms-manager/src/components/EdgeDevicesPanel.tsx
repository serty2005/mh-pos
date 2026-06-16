import { useState } from 'react';
import { Cpu, Server, Printer, RefreshCw, Signal, AlertTriangle, CheckCircle, Radio, Play, FileText, Ban } from 'lucide-react';

interface EdgeDevice {
  id: string;
  name: string;
  type: 'terminal' | 'printer' | 'kds' | 'fiscal_device';
  ip: string;
  port: string;
  status: 'online' | 'error' | 'offline' | 'warning';
  paperLevel?: number; // for printers
  fnExpiryDate?: string; // for fiscal devices
  firmware: string;
  serialNumber: string;
}

const INITIAL_DEVICES: EdgeDevice[] = [
  {
    id: 'ed-1',
    name: 'Основной фискальный регистратор ПТК (Атол 25Ф)',
    type: 'fiscal_device',
    ip: '192.168.1.180',
    port: '5560',
    status: 'online',
    paperLevel: 85,
    fnExpiryDate: '2027-11-10',
    firmware: 'f-v5.8.1',
    serialNumber: 'SN-778401928'
  },
  {
    id: 'ed-2',
    name: 'Принтер чеков Сервис-Печать Кухня (Star TSP)',
    type: 'printer',
    ip: '192.168.1.200',
    port: '9100',
    status: 'warning',
    paperLevel: 10,
    firmware: 'fw-star-v2',
    serialNumber: 'SN-00941923'
  },
  {
    id: 'ed-3',
    name: 'KDS монитор Шеф-Повара (Kitchen Display System)',
    type: 'kds',
    ip: '192.168.1.110',
    port: '8080',
    status: 'online',
    firmware: 'android-kds-v4.2',
    serialNumber: 'SN-KDS-99014'
  },
  {
    id: 'ed-4',
    name: 'Мобильный терминал Официанта (Pax A930)',
    type: 'terminal',
    ip: '192.168.1.115',
    port: '7000',
    status: 'offline',
    paperLevel: 0,
    firmware: 'pax-os-1.2.9',
    serialNumber: 'SN-PAX-884010'
  },
  {
    id: 'ed-5',
    name: 'Фискальный регистратор Кофейня (Атол Сигма)',
    type: 'fiscal_device',
    ip: '192.168.1.182',
    port: '5560',
    status: 'error',
    paperLevel: 100,
    fnExpiryDate: '2026-06-30', // Urgent! Close to expiry (today is 2026-06-15)
    firmware: 'f-v5.8.0',
    serialNumber: 'SN-99014120'
  }
];

export default function EdgeDevicesPanel() {
  const [devices, setDevices] = useState<EdgeDevice[]>(INITIAL_DEVICES);
  const [activeLogs, setActiveLogs] = useState<string[]>([
    '15:00:12 [Служба Edge] Сканирование портов оборудования...',
    '15:00:13 [Атол 25Ф] Фискальный статус: Смена открыта. Ошибок ФН нет.',
    '15:00:14 [Принтер Кухня] ВНИМАНИЕ: Снижение уровня рулона термобумаги (<15%).'
  ]);
  const [isRefreshing, setIsRefreshing] = useState(false);

  const handlePingHardware = (id: string, name: string) => {
    const timestamp = new Date().toTimeString().split(' ')[0];
    setActiveLogs(prev => [
      `${timestamp} [Тест связи] Пинг запрошен для: ${name}...`,
      ...prev
    ]);

    setDevices(prev => prev.map(d => {
      if (d.id === id) {
        return { ...d, status: 'online' }; // Restore on ping test for great user feedback!
      }
      return d;
    }));

    setTimeout(() => {
      setActiveLogs(prev => [
        `${timestamp} [Тест связи] Ответ от ${name} получен. Задержка: 12мс. Пакеты: 4/4.`,
        ...prev
      ]);
    }, 600);
  };

  const handlePrintTest = (device: EdgeDevice) => {
    const timestamp = new Date().toTimeString().split(' ')[0];
    setActiveLogs(prev => [
      `${timestamp} [Устройство] Печать пробного чека на: ${device.name}...`,
      ...prev
    ]);

    if (device.paperLevel !== undefined) {
      setDevices(prev => prev.map(d => {
        if (d.id === device.id && d.paperLevel) {
          return { ...d, paperLevel: Math.max(0, d.paperLevel - 5) };
        }
        return d;
      }));
    }

    setTimeout(() => {
      setActiveLogs(prev => [
        `${timestamp} [Печать] Задание завершено успешно. Счётчик отрезан.`,
        ...prev
      ]);
    }, 1000);
  };

  const triggerGlobalScan = () => {
    if (isRefreshing) return;
    setIsRefreshing(true);
    const timestamp = new Date().toTimeString().split(' ')[0];
    setActiveLogs(prev => [
      `${timestamp} [Начало] Полномасштабная ревизия периферии RMS...`,
      ...prev
    ]);

    setTimeout(() => {
      setDevices(prev => prev.map(d => {
        if (d.id === 'ed-4') return { ...d, status: 'online', paperLevel: 98 }; // Simulate turning on PAX!
        return d;
      }));
      setIsRefreshing(false);
      setActiveLogs(prev => [
        `${timestamp} [Edge Ревизия] Сканирование завершено. 5 из 5 устройств зарегистрированы.`,
        ...prev
      ]);
    }, 1200);
  };

  const getStatusIcon = (status: EdgeDevice['status']) => {
    switch (status) {
      case 'online':
        return <CheckCircle className="w-4 h-4 text-emerald-500 animate-pulse" />;
      case 'warning':
        return <AlertTriangle className="w-4 h-4 text-amber-500" />;
      case 'error':
        return <AlertTriangle className="w-4 h-4 text-rose-500" />;
      default:
        return <Ban className="w-4 h-4 text-slate-400" />;
    }
  };

  return (
    <div className="flex-1 overflow-y-auto bg-slate-50/50 p-8 flex flex-col h-full">
      {/* Header */}
      <div className="flex flex-col sm:flex-row justify-between sm:items-center gap-4 mb-8 shrink-0">
        <div>
          <h2 className="text-2xl font-bold tracking-tight text-slate-900 font-sans">
            Edge-терминалы, периферия и ККТ
          </h2>
          <p className="text-sm text-slate-500 mt-1">
            Мониторинг физических периферийных устройств: сенсорные моноблоки клиентов, КСО, кухонные KDS экраны и термопринтеры маркировочных листов.
          </p>
        </div>

        <button
          onClick={triggerGlobalScan}
          disabled={isRefreshing}
          className="flex items-center gap-2 px-5 py-3 rounded-xl bg-blue-600 hover:bg-blue-700 text-white font-semibold shadow-sm transition-all text-xs border border-blue-700 select-none"
        >
          <RefreshCw className={`w-3.5 h-3.5 ${isRefreshing ? 'animate-spin' : ''}`} />
          <span>Сканировать шину портов</span>
        </button>
      </div>

      {/* Grid listing */}
      <div className="flex-1 grid grid-cols-1 lg:grid-cols-3 gap-6 min-h-0">
        {/* Devices list (Left scrollable) */}
        <div className="lg:col-span-2 space-y-4 overflow-y-auto pr-1">
          {devices.map((device) => (
            <div
              key={device.id}
              className={`bg-white border rounded-2xl p-5 shadow-sm transition-all flex items-start gap-4 hover:border-slate-300 relative ${
                device.status === 'error'
                  ? 'border-rose-100 shadow shadow-rose-50/40'
                  : 'border-slate-200'
              }`}
            >
              {/* Type Badge Icon */}
              <div className={`p-3 rounded-xl ${
                device.type === 'fiscal_device'
                  ? 'bg-blue-50 text-blue-600'
                  : device.type === 'printer'
                  ? 'bg-purple-50 text-purple-600'
                  : device.type === 'kds'
                  ? 'bg-orange-50 text-orange-600'
                  : 'bg-emerald-50 text-emerald-600'
              }`}>
                {device.type === 'printer' || device.type === 'fiscal_device' ? (
                  <Printer className="w-5 h-5" />
                ) : (
                  <Cpu className="w-5 h-5" />
                )}
              </div>

              {/* Specs info content */}
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2">
                  <h4 className="font-sans font-bold text-slate-850 truncate">{device.name}</h4>
                  <span className="shrink-0">{getStatusIcon(device.status)}</span>
                </div>

                <div className="grid grid-cols-2 sm:grid-cols-3 gap-y-2 gap-x-4 mt-3 text-[11px] font-mono text-slate-500">
                  <p>
                    <span className="text-slate-400 font-bold block uppercase text-[10px]">Тип узла:</span>
                    <span className="font-sans font-bold text-slate-700">
                      {device.type === 'fiscal_device' && 'Фискальный регистратор Код 54-ФЗ'}
                      {device.type === 'printer' && 'Термопринтер Кухня / Сервис'}
                      {device.type === 'kds' && 'Монитор шеф-повара (KDS)'}
                      {device.type === 'terminal' && 'Официантский POS терминал'}
                    </span>
                  </p>
                  <p>
                    <span className="text-slate-400 font-bold block uppercase text-[10px]">Сетевой адрес:</span>
                    <span className="font-bold text-slate-700">{device.ip}:{device.port}</span>
                  </p>
                  <p>
                    <span className="text-slate-400 font-bold block uppercase text-[10px]">Серийный номер:</span>
                    <span className="text-slate-700">{device.serialNumber}</span>
                  </p>

                  {device.paperLevel !== undefined && (
                    <div className="col-span-2 sm:col-span-1">
                      <span className="text-slate-400 font-bold block uppercase text-[10px]">Термолента:</span>
                      <div className="flex items-center gap-1.5 mt-0.5">
                        <div className="flex-1 h-1.5 bg-slate-100 rounded-full overflow-hidden">
                          <div
                            style={{ width: `${device.paperLevel}%` }}
                            className={`h-full rounded-full ${
                              device.paperLevel <= 15 ? 'bg-rose-500' : 'bg-emerald-500'
                            }`}
                          />
                        </div>
                        <span className={`font-bold shrink-0 ${device.paperLevel <= 15 ? 'text-rose-600 animate-pulse' : 'text-slate-700'}`}>
                          {device.paperLevel}%
                        </span>
                      </div>
                    </div>
                  )}

                  {device.fnExpiryDate && (
                    <p className="col-span-2">
                      <span className="text-slate-400 font-bold block uppercase text-[10px]">Фискальный накопитель ФН срок:</span>
                      <span className={`font-bold ${
                        new Date(device.fnExpiryDate).getTime() - Date.now() < 30 * 24 * 60 * 60 * 1000
                          ? 'text-rose-600 underline'
                          : 'text-slate-700'
                      }`}>
                        Действует до {device.fnExpiryDate} &nbsp;
                        {new Date(device.fnExpiryDate).getTime() - Date.now() < 30 * 24 * 60 * 60 * 1000 && '(СРОЧНАЯ ЗАМЕНА!)'}
                      </span>
                    </p>
                  )}
                </div>

                {/* Operations bar in card */}
                <div className="flex items-center gap-2 mt-4 pt-4 border-t border-slate-100 text-[10px] uppercase font-bold select-none">
                  <button
                    onClick={() => handlePingHardware(device.id, device.name)}
                    className="flex items-center gap-1 hover:bg-slate-50 border border-slate-200 hover:border-slate-350 text-slate-700 px-3 py-1.5 rounded-lg transition-all"
                  >
                    <Signal className="w-3.5 h-3.5 text-blue-500" />
                    <span>Тест связи / Ping</span>
                  </button>
                  
                  {(device.type === 'printer' || device.type === 'fiscal_device') && (
                    <button
                      onClick={() => handlePrintTest(device)}
                      className="flex items-center gap-1 hover:bg-slate-50 border border-slate-200 hover:border-slate-350 text-slate-700 px-3 py-1.5 rounded-lg transition-all"
                    >
                      <Printer className="w-3.5 h-3.5 text-indigo-500" />
                      <span>Печать чека X</span>
                    </button>
                  )}
                </div>
              </div>
            </div>
          ))}
        </div>

        {/* Action Logger Console (Right sidebar of Edge view) */}
        <div className="bg-slate-900 text-slate-300 border border-slate-800 rounded-3xl p-5 flex flex-col shadow-inner select-none h-full overflow-hidden">
          <div className="flex items-center justify-between border-b border-slate-850 pb-3 mb-4">
            <div className="flex items-center gap-2 mb-1">
              <Server className="w-4 h-4 text-blue-400 animate-pulse" />
              <h3 className="font-sans font-bold text-sm text-white">Журнал портов ККТ</h3>
            </div>
            <button
              onClick={() => setActiveLogs([])}
              className="text-[9px] font-mono tracking-wider border border-slate-800 hover:border-slate-600 px-2.5 py-1 rounded hover:text-white"
            >
              Очистить
            </button>
          </div>

          <div className="flex-1 font-mono text-[10px] leading-relaxed space-y-2 overflow-y-auto max-h-[400px] lg:max-h-none text-slate-400">
            {activeLogs.map((log, index) => (
              <p
                key={index}
                className={
                  log.includes('ВНИМАНИЕ') || log.includes('Expiry')
                    ? 'text-amber-400 font-semibold'
                    : log.includes('Тест')
                    ? 'text-blue-400'
                    : log.includes('Печать')
                    ? 'text-purple-400'
                    : 'text-slate-300'
                }
              >
                {log}
              </p>
            ))}
            {activeLogs.length === 0 && (
              <div className="py-24 text-center text-slate-600 text-xs">
                Лог Edge пуст. Выполните пинг или диагностику оборудования.
              </div>
            )}
          </div>

          <div className="mt-4 pt-3 border-t border-slate-850 text-[10px] text-slate-550 font-mono flex items-center justify-between">
            <span>Шина: TCP/IP & ComPort Emul</span>
            <span className="text-emerald-500">Авто-опрос: 10с</span>
          </div>
        </div>
      </div>
    </div>
  );
}
