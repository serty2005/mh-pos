import React, { useState } from 'react';
import { Users, UserPlus, Shield, Check, X, ShieldAlert, Phone, Mail, Edit2, Trash2, Key, Info, HelpCircle, ToggleLeft, ToggleRight, CheckSquare, Square, Lock } from 'lucide-react';
import { StaffMember, Role } from '../types';

interface StaffPanelProps {
  staffList: StaffMember[];
  onAddStaff: (newStaff: StaffMember) => void;
  onUpdateStaff: (updatedStaff: StaffMember) => void;
  onDeleteStaff: (id: string) => void;
}

interface PermissionDefinition {
  id: string;
  category: string;
  label: string;
  description: string;
}

const MATRIX_PERMISSIONS: PermissionDefinition[] = [
  { id: 'pos_open_shift', category: 'Кассовые операции (POS)', label: 'Открытие и закрытие кассовых смен', description: 'Разрешает запускать торговый день на кассовой ноде и инкассировать выручку.' },
  { id: 'pos_delete_checks', category: 'Кассовые операции (POS)', label: 'Удаление чеков без оплаты', description: 'Дает право аннулировать позиции или весь чек после его пробития.' },
  { id: 'pos_apply_discounts', category: 'Кассовые операции (POS)', label: 'Назначение ручных скидок', description: 'Позволяет применять произвольные проценты скидок к заказу на кассе.' },
  
  { id: 'menu_edit_items', category: 'Управление номенклатурой', label: 'Редактирование блюд и цен', description: 'Позволяет менять отпускные цены в меню для торгового зала.' },
  { id: 'menu_edit_tech_cards', category: 'Управление номенклатурой', label: 'Изменение рецептур и техкарт', description: 'Открывает доступ к калькуляционным ведомостям сырья и истории версий.' },
  { id: 'menu_edit_stop_list', category: 'Управление номенклатурой', label: 'Ведение кассового стоплиста', description: 'Быстрое отключение блюд из продажи при отсутствии продуктов.' },
  
  { id: 'warehouse_view_docs', category: 'Склад и логистика', label: 'Просмотр складских накладных', description: 'Отображение журнала ТТН, актов списания и инвентаризации.' },
  { id: 'warehouse_process_ttn', category: 'Склад и логистика', label: 'Проведение ТТН на приход', description: 'Фактическая постановка продуктов на баланс склада с пересчетом себестоимости.' },
  { id: 'warehouse_write_offs', category: 'Склад и логистика', label: 'Списание испорченного товара', description: 'Регистрация порчи, боя или дегустаций на балансовых счетах.' },
  
  { id: 'analytics_view_margin', category: 'Безопасность и отчеты', label: 'Просмотр маржинальности и прибыли', description: 'Доступ к реальным цифрам прибыли, выгоды и наценки в отчетах.' },
  { id: 'staff_edit_matrix', category: 'Безопасность и отчеты', label: 'Редактирование матрицы прав', description: 'Главный допуск, позволяющий изменять права ролей и должностей.' }
];

const INITIAL_ROLE_MATRIX: Record<Role, Record<string, boolean>> = {
  admin: {
    pos_open_shift: true, pos_delete_checks: true, pos_apply_discounts: true,
    menu_edit_items: true, menu_edit_tech_cards: true, menu_edit_stop_list: true,
    warehouse_view_docs: true, warehouse_process_ttn: true, warehouse_write_offs: true,
    analytics_view_margin: true, staff_edit_matrix: true
  },
  manager: {
    pos_open_shift: true, pos_delete_checks: true, pos_apply_discounts: true,
    menu_edit_items: true, menu_edit_tech_cards: false, menu_edit_stop_list: true,
    warehouse_view_docs: true, warehouse_process_ttn: true, warehouse_write_offs: true,
    analytics_view_margin: true, staff_edit_matrix: false
  },
  chef: {
    pos_open_shift: false, pos_delete_checks: false, pos_apply_discounts: false,
    menu_edit_items: true, menu_edit_tech_cards: true, menu_edit_stop_list: true,
    warehouse_view_docs: true, warehouse_process_ttn: false, warehouse_write_offs: true,
    analytics_view_margin: false, staff_edit_matrix: false
  },
  waiter: {
    pos_open_shift: true, pos_delete_checks: false, pos_apply_discounts: false,
    menu_edit_items: false, menu_edit_tech_cards: false, menu_edit_stop_list: false,
    warehouse_view_docs: false, warehouse_process_ttn: false, warehouse_write_offs: false,
    analytics_view_margin: false, staff_edit_matrix: false
  },
  cashier: {
    pos_open_shift: true, pos_delete_checks: false, pos_apply_discounts: true,
    menu_edit_items: false, menu_edit_tech_cards: false, menu_edit_stop_list: false,
    warehouse_view_docs: false, warehouse_process_ttn: false, warehouse_write_offs: false,
    analytics_view_margin: false, staff_edit_matrix: false
  }
};

export default function StaffPanel({
  staffList,
  onAddStaff,
  onUpdateStaff,
  onDeleteStaff,
}: StaffPanelProps) {
  // Panel Navigation Tab
  const [activeTab, setActiveTab] = useState<'employees' | 'matrix'>('employees');

  const [roleFilter, setRoleFilter] = useState<'all' | Role>('all');
  const [searchQuery, setSearchQuery] = useState('');
  const [editingStaff, setEditingStaff] = useState<StaffMember | null>(null);
  const [isModalOpen, setIsModalOpen] = useState(false);

  // iiko Rights matrix state
  const [roleMatrix, setRoleMatrix] = useState<Record<Role, Record<string, boolean>>>(INITIAL_ROLE_MATRIX);
  const [hoveredPerm, setHoveredPerm] = useState<PermissionDefinition | null>(null);

  // Form states
  const [name, setName] = useState('');
  const [role, setRole] = useState<Role>('waiter');
  const [email, setEmail] = useState('');
  const [phone, setPhone] = useState('');
  const [status, setStatus] = useState<'active' | 'inactive' | 'on_shift'>('active');

  const resetForm = () => {
    setName('');
    setRole('waiter');
    setEmail('');
    setPhone('');
    setStatus('active');
    setEditingStaff(null);
  };

  const initEdit = (member: StaffMember) => {
    setEditingStaff(member);
    setName(member.name);
    setRole(member.role);
    setEmail(member.email);
    setPhone(member.phone);
    setStatus(member.status);
    setIsModalOpen(true);
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) return;

    // Permissions are compiled from the current iiko role matrix
    const compiledPermissions = {
      editMenu: roleMatrix[role]?.menu_edit_items || false,
      viewAnalytics: roleMatrix[role]?.analytics_view_margin || false,
      manageStaff: roleMatrix[role]?.staff_edit_matrix || false,
      posSync: roleMatrix[role]?.pos_open_shift || false,
    };

    const updatedMember: StaffMember = {
      id: editingStaff ? editingStaff.id : `staff-${Date.now()}`,
      name,
      role,
      email,
      phone,
      status,
      shiftStart: status === 'on_shift' ? (editingStaff?.shiftStart || '11:00') : undefined,
      permissions: compiledPermissions,
    };

    if (editingStaff) {
      onUpdateStaff(updatedMember);
    } else {
      onAddStaff(updatedMember);
    }
    setIsModalOpen(false);
    resetForm();
  };

  const handleDelete = (id: string, name: string) => {
    if (confirm(`Вы уверены, что хотите удалить сотрудника "${name}"? Учетная запись будет навсегда стерта.`)) {
      onDeleteStaff(id);
    }
  };

  const toggleStatus = (member: StaffMember) => {
    const nextStatusMap: Record<string, 'active' | 'inactive' | 'on_shift'> = {
      active: 'on_shift',
      on_shift: 'inactive',
      inactive: 'active',
    };
    const nextStatus = nextStatusMap[member.status];
    onUpdateStaff({
      ...member,
      status: nextStatus,
      shiftStart: nextStatus === 'on_shift' ? '12:00' : undefined,
    });
  };

  // Toggle Matrix Cell
  const handleToggleMatrix = (targetRole: Role, permId: string) => {
    setRoleMatrix(prev => {
      const updatedRolePerms = {
        ...prev[targetRole],
        [permId]: !prev[targetRole][permId]
      };

      const nextMatrix = {
        ...prev,
        [targetRole]: updatedRolePerms
      };

      // Propagate permissions to active staff with this role
      staffList.forEach(m => {
        if (m.role === targetRole) {
          onUpdateStaff({
            ...m,
            permissions: {
              editMenu: nextMatrix[targetRole]?.menu_edit_items || false,
              viewAnalytics: nextMatrix[targetRole]?.analytics_view_margin || false,
              manageStaff: nextMatrix[targetRole]?.staff_edit_matrix || false,
              posSync: nextMatrix[targetRole]?.pos_open_shift || false,
            }
          });
        }
      });

      return nextMatrix;
    });
  };

  const filteredStaff = staffList.filter((m) => {
    const matchesRole = roleFilter === 'all' || m.role === roleFilter;
    const matchesSearch = m.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
                          m.email.toLowerCase().includes(searchQuery.toLowerCase()) ||
                          m.phone.includes(searchQuery);
    return matchesRole && matchesSearch;
  });

  const getRoleBadgeColor = (r: Role) => {
    const map = {
      admin: 'bg-rose-50 border-rose-150 text-rose-600',
      manager: 'bg-indigo-50 border-indigo-150 text-indigo-700',
      chef: 'bg-amber-50 border-amber-150 text-amber-700',
      waiter: 'bg-emerald-50 border-emerald-150 text-emerald-700',
      cashier: 'bg-teal-50 border-teal-150 text-teal-700',
    };
    return map[r] || 'bg-slate-50 border-slate-150 text-slate-700';
  };

  const roleLabels: Record<Role, string> = {
    admin: 'Администратор',
    manager: 'Управляющий',
    chef: 'Шеф-повар',
    waiter: 'Официант',
    cashier: 'Кассир',
  };

  return (
    <div className="flex-1 overflow-y-auto bg-slate-50 p-8 flex flex-col h-full font-sans">
      {/* View Header */}
      <div className="flex flex-col md:flex-row justify-between items-start md:items-center gap-4 mb-8 shrink-0">
        <div>
          <h2 className="text-2xl font-bold tracking-tight text-slate-900 font-sans">
            Управление персоналом и уровнями доступа
          </h2>
          <p className="text-sm text-slate-500 mt-1">
            Добавляйте сотрудников, открывайте кассовые смены и гибко формируйте политики безопасности на базе ролевой матрицы iiko RMS.
          </p>
        </div>

        {activeTab === 'employees' ? (
          <button
            onClick={() => {
              resetForm();
              setIsModalOpen(true);
            }}
            className="flex items-center gap-2 px-5 py-3 rounded-xl bg-blue-600 border border-blue-700 hover:bg-blue-700 text-white font-semibold shadow-sm transition-all duration-300 transform active:scale-95 text-xs select-none cursor-pointer"
          >
            <UserPlus className="w-4 h-4" />
            <span>Новый сотрудник</span>
          </button>
        ) : (
          <div className="p-3 bg-indigo-50 border border-indigo-100 rounded-xl text-indigo-700 font-mono text-[10px] font-bold select-none">
            🔒 Изменения матрицы применяются ко всем сотрудникам роли на лету
          </div>
        )}
      </div>

      {/* Top Secondary Tab Navigator */}
      <div className="flex border-b border-slate-200 mb-6 bg-white rounded-t-2xl p-2 pb-0 shrink-0 border select-none">
        <button
          onClick={() => setActiveTab('employees')}
          className={`flex items-center gap-2 px-5 py-3 border-b-2 font-bold text-xs transition-all ${
            activeTab === 'employees'
              ? 'border-blue-600 text-blue-600'
              : 'border-transparent text-slate-400 hover:text-slate-600'
          }`}
        >
          <Users className="w-4 h-4" />
          <span>Список сотрудников ({staffList.length})</span>
        </button>
        <button
          onClick={() => setActiveTab('matrix')}
          className={`flex items-center gap-2 px-5 py-3 border-b-2 font-bold text-xs transition-all ${
            activeTab === 'matrix'
              ? 'border-blue-600 text-blue-600'
              : 'border-transparent text-slate-400 hover:text-slate-600'
          }`}
        >
          <Shield className="w-4 h-4" />
          <span>Матрица прав и привилегий (iiko Сетка)</span>
        </button>
      </div>

      {/* TAB 1: EMPLOYEES ROSTER */}
      {activeTab === 'employees' && (
        <div className="flex-1 flex flex-col min-h-0">
          {/* Staff directory toolbar */}
          <div className="bg-white rounded-2xl border border-slate-200 p-4 mb-6 flex flex-col xl:flex-row xl:items-center xl:justify-between gap-4 select-none shrink-0">
            {/* Role Filters */}
            <div className="flex flex-wrap gap-1.5">
              <button
                onClick={() => setRoleFilter('all')}
                className={`px-3.5 py-2.5 rounded-xl text-xs font-semibold leading-none border transition-all ${
                  roleFilter === 'all'
                    ? 'bg-slate-900 text-white border-slate-900'
                    : 'bg-white text-slate-600 border-slate-200 hover:bg-slate-50'
                }`}
              >
                Все должности
              </button>
              {(Object.keys(roleLabels) as Role[]).map((r) => (
                <button
                  key={r}
                  onClick={() => setRoleFilter(r)}
                  className={`px-3.5 py-2.5 rounded-xl text-xs font-semibold leading-none border transition-all ${
                    roleFilter === r
                      ? 'bg-slate-900 text-white border-slate-900'
                      : 'bg-white text-slate-600 border-slate-200 hover:bg-slate-50'
                  }`}
                >
                  {roleLabels[r]}
                </button>
              ))}
            </div>

            <div className="relative min-w-[280px]">
              <input
                type="text"
                placeholder="Поиск по имени, почте или телефону..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="w-full pl-4 pr-10 py-2.5 bg-slate-50 border border-slate-200 rounded-xl text-xs font-semibold focus:outline-none focus:bg-white focus:border-blue-500 transition-colors"
              />
              <Users className="w-4 h-4 text-slate-400 absolute right-3.5 top-1/2 -translate-y-1/2" />
            </div>
          </div>

          {/* Grid list of employees */}
          <div className="flex-1 overflow-y-auto">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6 pb-8">
              {filteredStaff.map((member) => (
                <div
                  key={member.id}
                  className="bg-white rounded-2xl border border-slate-200 p-5 flex flex-col justify-between gap-4 shadow-sm hover:shadow transition-shadow"
                >
                  {/* Header */}
                  <div className="flex justify-between items-start gap-3">
                    <div className="flex items-center gap-3">
                      <div className="w-11 h-11 rounded-2xl bg-[#eff6ff] flex items-center justify-center font-extrabold text-[#1e40af] select-none text-sm border border-blue-50 shadow-inner shrink-0">
                        {member.name.slice(0, 2).toUpperCase()}
                      </div>
                      <div>
                        <h3 className="font-sans font-bold text-slate-900 leading-tight">
                          {member.name}
                        </h3>
                        <div className="flex items-center gap-1.5 mt-1.5 flex-wrap">
                          <span className={`px-2 py-0.5 rounded text-[10px] font-mono tracking-wide uppercase font-bold border ${getRoleBadgeColor(member.role)}`}>
                            {roleLabels[member.role]}
                          </span>
                          <button
                            onClick={() => toggleStatus(member)}
                            className={`text-[9px] font-mono tracking-wide uppercase font-bold py-0.5 px-2 rounded border select-none transition-all ${
                              member.status === 'on_shift'
                                ? 'bg-emerald-50 border-emerald-100 text-emerald-600'
                                : member.status === 'active'
                                ? 'bg-slate-50 border-slate-200 text-slate-500'
                                : 'bg-stone-50 border-stone-200 text-stone-400'
                            }`}
                            title="Сменить рабочий статус (Смена)"
                          >
                            {member.status === 'on_shift'
                              ? '🟢 На смене'
                              : member.status === 'active'
                              ? '⚪ Вне смены'
                              : '🔴 Offline'}
                          </button>
                        </div>
                      </div>
                    </div>

                    <div className="flex items-center gap-1.5">
                      <button
                        onClick={() => initEdit(member)}
                        className="p-2 text-slate-500 hover:text-blue-600 bg-slate-50 hover:bg-blue-50/50 border border-slate-100 rounded-xl transition-all cursor-pointer"
                        title="Редактировать учетную запись"
                      >
                        <Edit2 className="w-3.5 h-3.5" />
                      </button>
                      <button
                        onClick={() => handleDelete(member.id, member.name)}
                        className="p-2 text-slate-400 hover:text-rose-600 bg-slate-50 hover:bg-rose-50 border border-slate-100 rounded-xl transition-all cursor-pointer"
                        title="Удалить"
                      >
                        <Trash2 className="w-3.5 h-3.5" />
                      </button>
                    </div>
                  </div>

                  {/* Contact Meta */}
                  <div className="grid grid-cols-2 gap-2.5 py-3 border-y border-slate-100 text-xs font-mono font-medium text-slate-550">
                    <div className="flex items-center gap-1.5">
                      <Mail className="w-3.5 h-3.5 text-slate-300 shrink-0" />
                      <span className="truncate">{member.email}</span>
                    </div>
                    <div className="flex items-center gap-1.5">
                      <Phone className="w-3.5 h-3.5 text-slate-300 shrink-0" />
                      <span>{member.phone}</span>
                    </div>
                  </div>

                  {/* Active Privileges compiled automatically preview */}
                  <div className="bg-slate-50/65 rounded-xl border border-slate-100 p-2.5">
                    <span className="text-[9px] font-black uppercase text-slate-400 font-mono tracking-wider block mb-1.5">Активно прав по матрице:</span>
                    <div className="flex flex-wrap gap-1">
                      {Object.entries(member.permissions).map(([k, active]) => {
                        if (!active) return null;
                        const labelMap: Record<string, string> = {
                          editMenu: 'Меню',
                          viewAnalytics: 'Финансы',
                          manageStaff: 'Персонал',
                          posSync: 'POS кассы'
                        };
                        return (
                          <span key={k} className="text-[10px] font-semibold bg-white border border-slate-150 text-slate-600 px-1.5 py-0.5 rounded flex items-center gap-1 font-sans">
                            <Check className="w-3 h-3 text-blue-500" />
                            {labelMap[k] || k}
                          </span>
                        );
                      })}
                      {Object.values(member.permissions).filter(Boolean).length === 0 && (
                        <span className="text-[10px] font-mono text-slate-400 font-bold italic">Нет ролевых допусков</span>
                      )}
                    </div>
                  </div>

                </div>
              ))}

              {filteredStaff.length === 0 && (
                <div className="col-span-1 md:col-span-2 bg-white py-16 text-center text-slate-400 border border-slate-200 rounded-2xl flex flex-col items-center justify-center gap-2">
                  <ShieldAlert className="w-10 h-10 text-slate-300" />
                  <p className="font-sans font-bold text-slate-700">Никто не зафиксирован в этой должности</p>
                  <p className="text-xs text-slate-400">Сбросьте критерии поиска или зарегистрируйте нового сотрудника.</p>
                </div>
              )}
            </div>
          </div>
        </div>
      )}

      {/* TAB 2: iiko ROLE ACCESS MATRIX */}
      {activeTab === 'matrix' && (
        <div className="flex-1 flex flex-col md:flex-row gap-6 min-h-0 select-none pb-8">
          {/* Main Matrix Grid */}
          <div className="flex-1 bg-white border border-slate-200 rounded-2xl shadow-sm overflow-hidden flex flex-col">
            <div className="flex-1 overflow-auto">
              <table className="w-full text-left border-collapse text-xs min-w-[750px]">
                <thead>
                  <tr className="border-b border-slate-100 bg-slate-50 text-slate-400 font-mono font-bold uppercase sticky top-0 z-10">
                    <th className="py-4.5 px-5">Допуск к операции RMS</th>
                    {(Object.keys(roleLabels) as Role[]).map((r) => (
                      <th key={r} className="py-4.5 px-4 text-center font-sans tracking-tight">
                        <div className="flex flex-col items-center">
                          <span>{roleLabels[r]}</span>
                          <span className="text-[9px] text-slate-400 lowercase font-mono">({r})</span>
                        </div>
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-100">
                  {MATRIX_PERMISSIONS.map((perm) => (
                    <tr
                      key={perm.id}
                      onMouseEnter={() => setHoveredPerm(perm)}
                      onMouseLeave={() => setHoveredPerm(null)}
                      className={`hover:bg-slate-50/80 transition-colors ${
                        hoveredPerm?.id === perm.id ? 'bg-slate-50/50' : ''
                      }`}
                    >
                      {/* Permission Metadata */}
                      <td className="py-3.5 px-5 select-none">
                        <div className="flex items-center gap-1.5 mb-0.5">
                          <span className="text-[9px] font-extrabold uppercase font-mono px-1.5 py-0.5 rounded bg-slate-100 text-slate-500">
                            {perm.category}
                          </span>
                        </div>
                        <h4 className="font-bold text-slate-800 font-sans">{perm.label}</h4>
                      </td>

                      {/* Interactive intersects (roles checkboxes) */}
                      {(Object.keys(roleLabels) as Role[]).map((r) => {
                        const isChecked = roleMatrix[r]?.[perm.id] || false;
                        const isRootAdmin = r === 'admin'; // lock admin rights to prevent accidental lockout
                        return (
                          <td key={r} className="py-3.5 px-4 text-center">
                            <button
                              type="button"
                              onClick={() => !isRootAdmin && handleToggleMatrix(r, perm.id)}
                              disabled={isRootAdmin}
                              className={`mx-auto w-8 h-8 rounded-xl flex items-center justify-center border transition-all ${
                                isRootAdmin
                                  ? 'bg-rose-50 text-rose-500 border-rose-100 cursor-not-allowed'
                                  : isChecked
                                  ? 'bg-blue-600 text-white border-blue-700 hover:bg-blue-700 hover:scale-105 active:scale-95'
                                  : 'bg-slate-50 border-slate-200 text-slate-350 hover:bg-slate-100 hover:text-slate-500'
                              }`}
                              title={isRootAdmin ? "Администратор обладает полным доступом по умолчанию" : `Кликните чтобы изменить привилегию для '${roleLabels[r]}'`}
                            >
                              {isChecked ? (
                                <CheckSquare className="w-5 h-5 shrink-0" />
                              ) : isRootAdmin ? (
                                <Lock className="w-3.5 h-3.5 shrink-0 text-rose-500" />
                              ) : (
                                <Square className="w-5 h-5 shrink-0" />
                              )}
                            </button>
                          </td>
                        );
                      })}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            {/* Matrix Help bottom label */}
            <div className="px-5 py-4 bg-slate-50 border-t border-slate-150 flex items-center justify-between text-xs text-slate-400 font-mono">
              <div className="flex items-center gap-1.5 text-slate-500">
                <Info className="w-4 h-4 text-blue-500" />
                <span>Наведите курсор на строчку для подробного разбора кассового сценария.</span>
              </div>
              <span className="font-bold text-[10px] text-slate-400 uppercase">MyHoreca Enterprise</span>
            </div>
          </div>

          {/* Explanation sidebar */}
          <div className="w-full md:w-80 bg-white border border-slate-200 rounded-2xl p-5 shadow-sm shrink-0 flex flex-col justify-between">
            <div className="space-y-4">
              <div className="flex items-center gap-2 pb-3 border-b text-slate-800">
                <Key className="text-blue-500 w-5 h-5" />
                <h3 className="font-sans font-extrabold text-sm">Пояснения привилегии</h3>
              </div>

              {hoveredPerm ? (
                <div className="space-y-3.5">
                  <div>
                    <span className="text-[10px] tracking-wider uppercase font-extrabold text-blue-500 font-mono bg-blue-50 px-2 py-0.5 rounded">
                      {hoveredPerm.category}
                    </span>
                    <h4 className="font-bold text-slate-900 mt-2 text-xs leading-snug">{hoveredPerm.label}</h4>
                  </div>
                  <p className="text-xs text-slate-500 leading-relaxed bg-slate-50 p-3 rounded-xl border border-slate-100">
                    {hoveredPerm.description}
                  </p>
                  
                  <div className="pt-2">
                    <span className="text-[9px] font-mono uppercase text-slate-400 block mb-1">Ключевые сценарии iiKo API:</span>
                    <div className="text-[10px] font-mono text-slate-500 space-y-1 bg-slate-900 text-slate-300 p-2.5 rounded-lg">
                      <div>// Triggered on core POS event:</div>
                      <div className="text-emerald-400">check_privilege("{hoveredPerm.id}");</div>
                      <div>if (denied) throw AuthReject();</div>
                    </div>
                  </div>
                </div>
              ) : (
                <div className="text-center py-16 text-slate-400 flex flex-col items-center justify-center gap-2 select-none">
                  <HelpCircle className="w-10 h-10 text-slate-200" />
                  <p className="text-xs font-bold font-sans text-slate-650">Привилегия не выбрана</p>
                  <p className="text-[11px] text-slate-400 leading-relaxed max-w-[180px]">
                    Наведите на любую строку таблицы доступа для отображения детального описания и кода API.
                  </p>
                </div>
              )}
            </div>

            <div className="bg-slate-50 p-3.5 rounded-xl border text-[10px] text-slate-500 font-mono leading-relaxed select-none">
              <span className="font-black text-slate-700 block uppercase mb-1">Синхронизация POS-прав:</span>
              Кассовые планшеты автоматически забирают обновленные токеризованные права пользователей при каждом закрытии кассовой смены.
            </div>
          </div>
        </div>
      )}

      {/* Roster addition Modal */}
      {isModalOpen && (
        <div className="fixed inset-0 bg-slate-900/40 backdrop-blur-sm z-50 flex items-center justify-center p-4">
          <div className="bg-white rounded-3xl w-full max-w-md shadow-2xl border border-slate-100 overflow-hidden text-slate-800">
            {/* Modal Header */}
            <div className="px-6 py-4.5 border-b border-slate-100 flex justify-between items-center bg-slate-50">
              <h3 className="text-xs font-extrabold text-slate-900 font-sans border-b-none p-0 m-0">
                {editingStaff ? 'Редактировать сотрудника' : 'Регистрация сотрудника в системе'}
              </h3>
              <button
                onClick={() => setIsModalOpen(false)}
                className="px-3 py-1 text-xs text-slate-400 hover:text-slate-600 font-semibold border rounded-lg hover:bg-slate-100 select-none cursor-pointer animate-none"
              >
                Закрыть
              </button>
            </div>

            {/* Modal Form Content */}
            <form onSubmit={handleSubmit} className="p-6 space-y-4">
              {/* Full Name */}
              <div>
                <label className="text-[10px] uppercase tracking-wider font-bold font-mono text-slate-400 block mb-1">
                  ФИО Сотрудника *
                </label>
                <input
                  type="text"
                  required
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="Константин Константинопольский"
                  className="w-full p-2.5 text-xs font-semibold bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:border-blue-500"
                />
              </div>

              {/* Role & status */}
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="text-[10px] uppercase tracking-wider font-bold font-mono text-slate-400 block mb-1">
                    Должность в ресторане
                  </label>
                  <select
                    value={role}
                    onChange={(e) => setRole(e.target.value as Role)}
                    className="w-full p-2.5 text-xs font-semibold bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:border-blue-500"
                  >
                    <option value="waiter">Официант</option>
                    <option value="chef">Шеф-повар</option>
                    <option value="manager">Управляющий</option>
                    <option value="cashier">Кассир смены</option>
                    <option value="admin">Администратор</option>
                  </select>
                </div>

                <div>
                  <label className="text-[10px] uppercase tracking-wider font-bold font-mono text-slate-400 block mb-1">
                    Рабочий статус
                  </label>
                  <select
                    value={status}
                    onChange={(e) => setStatus(e.target.value as any)}
                    className="w-full p-2.5 text-xs font-semibold bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:border-blue-500"
                  >
                    <option value="active">Активен (Вне смены)</option>
                    <option value="on_shift">На смене (🟢 В зале)</option>
                    <option value="inactive">В отпуске / Выключен</option>
                  </select>
                </div>
              </div>

              {/* Email & Phone */}
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="text-[10px] uppercase tracking-wider font-bold font-mono text-slate-400 block mb-1">
                    Электронная почта
                  </label>
                  <input
                    type="email"
                    required
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    placeholder="k.konst@myhoreca.ru"
                    className="w-full p-2.5 text-xs font-semibold bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:border-blue-500"
                  />
                </div>

                <div>
                  <label className="text-[10px] uppercase tracking-wider font-bold font-mono text-slate-400 block mb-1">
                    Номер телефона
                  </label>
                  <input
                    type="text"
                    required
                    value={phone}
                    onChange={(e) => setPhone(e.target.value)}
                    placeholder="+7 (900) 123-45-67"
                    className="w-full p-2.5 text-xs font-semibold bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:border-blue-500 font-mono"
                  />
                </div>
              </div>

              {/* Notice */}
              <div className="p-3.5 bg-indigo-50 border border-indigo-100 rounded-xl text-indigo-700 text-[10px] leading-relaxed">
                <span className="font-extrabold uppercase font-mono block mb-0.5">Косвенные ролевые полномочия:</span>
                Зарегистрированный сотрудник унаследует разрешения выбранной роли из общей сетки iiko RMS. Вы сможете подкорректировать сетку в соседней вкладке в любое время.
              </div>

              {/* Modal controls */}
              <div className="flex justify-end gap-3 pt-4 border-t border-slate-100">
                <button
                  type="button"
                  onClick={() => setIsModalOpen(false)}
                  className="px-4.5 py-2 text-xs font-semibold text-slate-650 hover:text-slate-800 border rounded-xl hover:bg-slate-100 select-none cursor-pointer"
                >
                  Отмена
                </button>
                <button
                  type="submit"
                  className="px-5 py-2 rounded-xl text-xs font-bold text-white bg-blue-600 hover:bg-blue-700 select-none border border-blue-700"
                >
                  {editingStaff ? 'Сохранить изменения' : 'Зарегистрировать'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
