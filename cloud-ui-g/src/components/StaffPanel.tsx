/**
 * @license
 * SPDX-License-Identifier: Apache-2.0
 */

import React, { useState } from 'react';
import { Users, UserPlus, Shield, Check, X, ShieldAlert, Phone, Mail, Edit2, Trash2 } from 'lucide-react';
import { StaffMember, Role } from '../types';

interface StaffPanelProps {
  staffList: StaffMember[];
  onAddStaff: (newStaff: StaffMember) => void;
  onUpdateStaff: (updatedStaff: StaffMember) => void;
  onDeleteStaff: (id: string) => void;
}

export default function StaffPanel({
  staffList,
  onAddStaff,
  onUpdateStaff,
  onDeleteStaff,
}: StaffPanelProps) {
  const [roleFilter, setRoleFilter] = useState<'all' | Role>('all');
  const [searchQuery, setSearchQuery] = useState('');
  const [editingStaff, setEditingStaff] = useState<StaffMember | null>(null);
  const [isModalOpen, setIsModalOpen] = useState(false);

  // Form states
  const [name, setName] = useState('');
  const [role, setRole] = useState<Role>('waiter');
  const [email, setEmail] = useState('');
  const [phone, setPhone] = useState('');
  const [status, setStatus] = useState<'active' | 'inactive' | 'on_shift'>('active');
  const [editMenuPerm, setEditMenuPerm] = useState(false);
  const [viewAnalyticsPerm, setViewAnalyticsPerm] = useState(false);
  const [manageStaffPerm, setManageStaffPerm] = useState(false);
  const [posSyncPerm, setPosSyncPerm] = useState(false);

  const resetForm = () => {
    setName('');
    setRole('waiter');
    setEmail('');
    setPhone('');
    setStatus('active');
    setEditMenuPerm(false);
    setViewAnalyticsPerm(false);
    setManageStaffPerm(false);
    setPosSyncPerm(false);
    setEditingStaff(null);
  };

  const initEdit = (member: StaffMember) => {
    setEditingStaff(member);
    setName(member.name);
    setRole(member.role);
    setEmail(member.email);
    setPhone(member.phone);
    setStatus(member.status);
    setEditMenuPerm(member.permissions.editMenu);
    setViewAnalyticsPerm(member.permissions.viewAnalytics);
    setManageStaffPerm(member.permissions.manageStaff);
    setPosSyncPerm(member.permissions.posSync);
    setIsModalOpen(true);
  };

  const handleRoleChange = (selectedRole: Role) => {
    setRole(selectedRole);
    // Autofill logical default permissions based on role to save time
    if (selectedRole === 'admin') {
      setEditMenuPerm(true);
      setViewAnalyticsPerm(true);
      setManageStaffPerm(true);
      setPosSyncPerm(true);
    } else if (selectedRole === 'manager') {
      setEditMenuPerm(true);
      setViewAnalyticsPerm(true);
      setManageStaffPerm(false);
      setPosSyncPerm(true);
    } else if (selectedRole === 'chef') {
      setEditMenuPerm(true);
      setViewAnalyticsPerm(false);
      setManageStaffPerm(false);
      setPosSyncPerm(false);
    } else {
      setEditMenuPerm(false);
      setViewAnalyticsPerm(false);
      setManageStaffPerm(false);
      setPosSyncPerm(false);
    }
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) return;

    const updatedMember: StaffMember = {
      id: editingStaff ? editingStaff.id : `staff-${Date.now()}`,
      name,
      role,
      email,
      phone,
      status,
      shiftStart: status === 'on_shift' ? (editingStaff?.shiftStart || '11:00') : undefined,
      permissions: {
        editMenu: editMenuPerm,
        viewAnalytics: viewAnalyticsPerm,
        manageStaff: manageStaffPerm,
        posSync: posSyncPerm,
      },
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

  // Inline toggle helper for permissions (highly responsive layout)
  const togglePermission = (member: StaffMember, key: 'editMenu' | 'viewAnalytics' | 'manageStaff' | 'posSync') => {
    onUpdateStaff({
      ...member,
      permissions: {
        ...member.permissions,
        [key]: !member.permissions[key],
      },
    });
  };

  // Filtered members list
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
    <div className="flex-1 overflow-y-auto bg-slate-50 p-8">
      {/* View Header */}
      <div className="flex flex-col md:flex-row justify-between items-start md:items-center gap-4 mb-8">
        <div>
          <h2 className="text-2xl font-bold tracking-tight text-slate-900 font-sans">
            Управление персоналом и уровнями доступа
          </h2>
          <p className="text-sm text-slate-500 mt-1">
            Добавляйте сотрудников, назначайте роли, открывайте смены и регулируйте права доступа к RMS/POS.
          </p>
        </div>

        <button
          onClick={() => {
            resetForm();
            setIsModalOpen(true);
          }}
          className="flex items-center gap-2 px-5 py-3 rounded-xl bg-blue-600 border border-blue-700 hover:bg-blue-700 text-white font-semibold shadow-sm transition-all duration-300 transform active:scale-95"
        >
          <UserPlus className="w-4 h-4" />
          <span>Новый сотрудник</span>
        </button>
      </div>

      {/* Staff directory toolbar */}
      <div className="bg-white rounded-2xl border border-slate-200 p-5 mb-8 flex flex-col xl:flex-row xl:items-center xl:justify-between gap-4">
        {/* Role Filters */}
        <div className="flex flex-wrap gap-2">
          <button
            onClick={() => setRoleFilter('all')}
            className={`px-4 py-2 rounded-xl text-xs font-semibold leading-none border transition-all ${
              roleFilter === 'all'
                ? 'bg-slate-900 text-white border-slate-900'
                : 'bg-white text-slate-600 border-slate-200 hover:bg-slate-50'
            }`}
          >
            Все
          </button>
          {(Object.keys(roleLabels) as Role[]).map((r) => (
            <button
              key={r}
              onClick={() => setRoleFilter(r)}
              className={`px-4 py-2 rounded-xl text-xs font-semibold leading-none border transition-all ${
                roleFilter === r
                  ? 'bg-slate-900 text-white border-slate-900'
                  : 'bg-white text-slate-600 border-slate-200 hover:bg-slate-50'
              }`}
            >
              {roleLabels[r]}
            </button>
          ))}
        </div>

        <div className="relative min-w-[300px]">
          <input
            type="text"
            placeholder="Поиск по имени, email или номеру телефона..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-full pl-4 pr-10 py-2.5 bg-slate-50 border border-slate-200 rounded-xl text-xs font-semibold focus:outline-none focus:bg-white focus:border-blue-500 transition-colors"
          />
          <Users className="w-4 h-4 text-slate-400 absolute right-3 top-1/2 -translate-y-1/2" />
        </div>
      </div>

      {/* Staff Grid/Table */}
      <div className="grid grid-cols-1 xl:grid-cols-2 gap-6">
        {filteredStaff.map((member) => (
          <div
            key={member.id}
            className="bg-white rounded-2xl border border-slate-200 p-6 flex flex-col justify-between gap-6 shadow-sm hover:shadow transition-shadow"
          >
            {/* Header: Avatar, Name, Role, Shift */}
            <div className="flex justify-between items-start gap-4">
              <div className="flex items-center gap-4">
                <div className="w-12 h-12 rounded-2xl bg-gradient-to-tr from-slate-200 to-slate-300 flex items-center justify-center font-extrabold text-[#312e81] select-none text-base border border-slate-100 shadow-inner">
                  {member.name.slice(0, 2).toUpperCase()}
                </div>
                <div>
                  <h3 className="font-sans font-bold text-sm text-slate-900">{member.name}</h3>
                  <div className="flex items-center gap-2 mt-1">
                    <span className={`px-2 py-0.5 rounded-lg border text-[10px] font-mono tracking-wide uppercase font-bold ${getRoleBadgeColor(member.role)}`}>
                      {roleLabels[member.role]}
                    </span>
                    <button
                      onClick={() => toggleStatus(member)}
                      className={`text-[9px] font-mono tracking-wide uppercase font-bold py-0.5 px-2 rounded-lg border select-none transition-all ${
                        member.status === 'on_shift'
                          ? 'bg-emerald-50 border-emerald-150 text-emerald-600 font-semibold'
                          : member.status === 'active'
                          ? 'bg-slate-50 border-slate-200 text-slate-500'
                          : 'bg-stone-50 border-stone-150 text-stone-400'
                      }`}
                      title="Кликните чтобы изменить рабочий статус"
                    >
                      {member.status === 'on_shift'
                        ? '🟢 На смене'
                        : member.status === 'active'
                        ? '⚪️ Активен'
                        : '🔴 Оффлайн'}
                    </button>
                  </div>
                </div>
              </div>

              {/* Action buttons */}
              <div className="flex items-center gap-1.5">
                <button
                  onClick={() => initEdit(member)}
                  className="p-2 text-slate-500 hover:text-blue-600 bg-slate-50 hover:bg-blue-50/50 border border-slate-100 rounded-lg transition-all"
                  title="Права и данные"
                >
                  <Edit2 className="w-3.5 h-3.5" />
                </button>
                <button
                  onClick={() => handleDelete(member.id, member.name)}
                  className="p-2 text-slate-400 hover:text-rose-600 bg-slate-50 hover:bg-rose-50 border border-slate-100 rounded-lg transition-all"
                  title="Стереть учетную запись"
                >
                  <Trash2 className="w-3.5 h-3.5" />
                </button>
              </div>
            </div>

            {/* Contacts area */}
            <div className="grid grid-cols-2 gap-3.5 py-4 border-y border-slate-100 text-xs font-mono font-medium text-slate-500">
              <div className="flex items-center gap-2">
                <Mail className="w-4 h-4 text-slate-300" />
                <span className="truncate">{member.email}</span>
              </div>
              <div className="flex items-center gap-2">
                <Phone className="w-4 h-4 text-slate-300" />
                <span>{member.phone}</span>
              </div>
            </div>

            {/* Access Permissions Area */}
            <div>
              <div className="flex items-center justify-between mb-3 text-xs text-slate-400 font-bold uppercase tracking-wider">
                <span className="flex items-center gap-1">
                  <Shield className="w-3.5 h-3.5" />
                  Разрешения доступа
                </span>
                <span className="text-[10px] lowercase font-semibold font-mono text-slate-400 normal-case">
                  (нажмите чтобы переключить)
                </span>
              </div>

              <div className="grid grid-cols-2 gap-2.5">
                {[
                  { key: 'editMenu', label: 'Редактировать меню' },
                  { key: 'viewAnalytics', label: 'Просмотр аналитики' },
                  { key: 'manageStaff', label: 'Администрировать команду' },
                  { key: 'posSync', label: 'Синхронизация POS' },
                ].map((perm) => {
                  const isGranted = member.permissions[perm.key as keyof typeof member.permissions];
                  return (
                    <button
                      key={perm.key}
                      onClick={() => togglePermission(member, perm.key as any)}
                      className={`flex items-center justify-between p-2.5 rounded-xl border text-[11px] font-semibold text-left transition-all duration-200 select-none ${
                        isGranted
                          ? 'bg-blue-50/50 border-blue-100/60 text-blue-820 hover:bg-blue-100/30'
                          : 'bg-slate-50 border-slate-150 text-slate-400 hover:bg-slate-100/40'
                      }`}
                    >
                      <span>{perm.label}</span>
                      {isGranted ? (
                        <Check className="w-3.5 h-3.5 text-blue-600 shrink-0 ml-1.5" />
                      ) : (
                        <X className="w-3.5 h-3.5 text-slate-300 shrink-0 ml-1.5" />
                      )}
                    </button>
                  );
                })}
              </div>
            </div>
          </div>
        ))}

        {filteredStaff.length === 0 && (
          <div className="col-span-1 xl:col-span-2 bg-white py-16 text-center text-slate-400 border border-slate-200 rounded-2xl flex flex-col items-center justify-center gap-2">
            <ShieldAlert className="w-10 h-10 text-slate-300" />
            <p className="font-sans font-bold text-slate-700">Никто не найден</p>
            <p className="text-xs text-slate-400">Проверьте поисковое слово или фильтр ролей.</p>
          </div>
        )}
      </div>

      {/* Staff Editor overlay Modal */}
      {isModalOpen && (
        <div className="fixed inset-0 bg-slate-900/40 backdrop-blur-sm z-50 flex items-center justify-center p-4">
          <div className="bg-white rounded-3xl w-full max-w-lg shadow-2xl border border-slate-100 overflow-hidden text-slate-800 transform transition-all animate-in fade-in zoom-in-95 duration-200">
            {/* Modal Header */}
            <div className="px-6 py-5 border-b border-slate-100 flex justify-between items-center bg-slate-50">
              <h3 className="text-base font-bold text-slate-900 font-sans">
                {editingStaff ? 'Редактировать данные сотрудника' : 'Регистрация нового сотрудника'}
              </h3>
              <button
                onClick={() => setIsModalOpen(false)}
                className="p-1 px-3 text-sm text-slate-400 hover:text-slate-600 font-semibold border rounded-lg hover:bg-slate-100"
              >
                Закрыть
              </button>
            </div>

            {/* Modal Form Content */}
            <form onSubmit={handleSubmit} className="p-6 space-y-4">
              {/* Full Name */}
              <div>
                <label className="text-[10px] uppercase tracking-wider font-semibold font-mono text-slate-400 block mb-1">
                  ФИО Сотрудника *
                </label>
                <input
                  type="text"
                  required
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="Константин Хабенский"
                  className="w-full p-2.5 text-xs font-semibold bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:border-blue-500"
                />
              </div>

              {/* Role & status */}
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="text-[10px] uppercase tracking-wider font-semibold font-mono text-slate-400 block mb-1">
                    Должность в ресторане
                  </label>
                  <select
                    value={role}
                    onChange={(e) => handleRoleChange(e.target.value as Role)}
                    className="w-full p-2.5 text-xs font-semibold bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:border-blue-500"
                  >
                    <option value="waiter">Официант</option>
                    <option value="chef">Шеф-повар</option>
                    <option value="manager">Управляющий (Manager)</option>
                    <option value="cashier">Кассир смены</option>
                    <option value="admin">Учетная запись: Администратор</option>
                  </select>
                </div>

                <div>
                  <label className="text-[10px] uppercase tracking-wider font-semibold font-mono text-slate-400 block mb-1">
                    Рабочий статус
                  </label>
                  <select
                    value={status}
                    onChange={(e) => setStatus(e.target.value as any)}
                    className="w-full p-2.5 text-xs font-semibold bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:border-blue-500"
                  >
                    <option value="active">Активен (Вне смены)</option>
                    <option value="on_shift">На смене (🟢 Работает сейчас)</option>
                    <option value="inactive">В отпуске / Выключен</option>
                  </select>
                </div>
              </div>

              {/* Email & Phone */}
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="text-[10px] uppercase tracking-wider font-semibold font-mono text-slate-400 block mb-1">
                    Электронная почта
                  </label>
                  <input
                    type="email"
                    required
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    placeholder="k.hab@myhoreca.ru"
                    className="w-full p-2.5 text-xs font-semibold bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:border-blue-500"
                  />
                </div>

                <div>
                  <label className="text-[10px] uppercase tracking-wider font-semibold font-mono text-slate-400 block mb-1">
                    Номер телефона
                  </label>
                  <input
                    type="text"
                    required
                    value={phone}
                    onChange={(e) => setPhone(e.target.value)}
                    placeholder="+7 (900) 123-45-67"
                    className="w-full p-2.5 text-xs font-semibold bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:border-blue-500"
                  />
                </div>
              </div>

              {/* Set custom access level rights manually */}
              <div>
                <label className="text-[10px] uppercase tracking-wider font-semibold font-mono text-slate-400 block mb-2.5">
                  Уровни ручного управления привилегиями
                </label>
                <div className="grid grid-cols-2 gap-3.5 bg-slate-50 p-4 border border-slate-100 rounded-2xl">
                  <div className="flex items-center gap-2">
                    <input
                      type="checkbox"
                      id="perm-menu"
                      checked={editMenuPerm}
                      onChange={(e) => setEditMenuPerm(e.target.checked)}
                      className="w-4 h-4 text-blue-600 border-slate-300 rounded"
                    />
                    <label htmlFor="perm-menu" className="text-xs font-semibold text-slate-600 select-none">
                      Редактор меню
                    </label>
                  </div>
                  <div className="flex items-center gap-2">
                    <input
                      type="checkbox"
                      id="perm-analytics"
                      checked={viewAnalyticsPerm}
                      onChange={(e) => setViewAnalyticsPerm(e.target.checked)}
                      className="w-4 h-4 text-blue-600 border-slate-300 rounded"
                    />
                    <label htmlFor="perm-analytics" className="text-xs font-semibold text-slate-600 select-none">
                      Видеть продажи и маржу
                    </label>
                  </div>
                  <div className="flex items-center gap-2">
                    <input
                      type="checkbox"
                      id="perm-staff"
                      checked={manageStaffPerm}
                      onChange={(e) => setManageStaffPerm(e.target.checked)}
                      className="w-4 h-4 text-blue-600 border-slate-300 rounded"
                    />
                    <label htmlFor="perm-staff" className="text-xs font-semibold text-slate-600 select-none">
                      Редактировать персонал
                    </label>
                  </div>
                  <div className="flex items-center gap-2">
                    <input
                      type="checkbox"
                      id="perm-sync"
                      checked={posSyncPerm}
                      onChange={(e) => setPosSyncPerm(e.target.checked)}
                      className="w-4 h-4 text-blue-600 border-slate-300 rounded"
                    />
                    <label htmlFor="perm-sync" className="text-xs font-semibold text-slate-600 select-none">
                      Выполнять POS-выгрузки
                    </label>
                  </div>
                </div>
              </div>

              {/* Modal controls */}
              <div className="flex justify-end gap-3 pt-4 border-t border-slate-100">
                <button
                  type="button"
                  onClick={() => setIsModalOpen(false)}
                  className="px-4.5 py-2.5 text-xs font-semibold text-slate-600 hover:text-slate-800 border rounded-xl hover:bg-slate-100"
                >
                  Отмена
                </button>
                <button
                  type="submit"
                  className="px-5 py-2.5 text-xs font-bold text-white bg-blue-600 border border-blue-700 hover:bg-blue-700 rounded-xl"
                >
                  {editingStaff ? 'Сохранить сотрудника' : 'Зарегистрировать'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
