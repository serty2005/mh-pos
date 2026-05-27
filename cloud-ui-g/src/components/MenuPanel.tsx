/**
 * @license
 * SPDX-License-Identifier: Apache-2.0
 */

import React, { useState } from 'react';
import { Search, Plus, Edit2, Trash2, CheckCircle, XCircle, DollarSign, Clock, AlertCircle } from 'lucide-react';
import { MenuItem, MenuCategory } from '../types';

interface MenuPanelProps {
  menuItems: MenuItem[];
  categories: MenuCategory[];
  onAddMenuItem: (item: MenuItem) => void;
  onUpdateMenuItem: (item: MenuItem) => void;
  onDeleteMenuItem: (id: string) => void;
}

export default function MenuPanel({
  menuItems,
  categories,
  onAddMenuItem,
  onUpdateMenuItem,
  onDeleteMenuItem,
}: MenuPanelProps) {
  const [selectedCategory, setSelectedCategory] = useState<string>('all');
  const [searchQuery, setSearchQuery] = useState<string>('');
  const [editingItem, setEditingItem] = useState<MenuItem | null>(null);
  const [isFormOpen, setIsFormOpen] = useState(false);

  // Form states
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [price, setPrice] = useState<number>(0);
  const [cost, setCost] = useState<number>(0);
  const [category, setCategory] = useState('');
  const [emoji, setEmoji] = useState('🍔');
  const [prepTime, setPrepTime] = useState<number>(10);
  const [stock, setStock] = useState<number>(-1);
  const [isAvailable, setIsAvailable] = useState(true);

  const resetForm = () => {
    setName('');
    setDescription('');
    setPrice(0);
    setCost(0);
    setCategory(categories[0]?.slug || 'mains');
    setEmoji('🍔');
    setPrepTime(10);
    setStock(-1);
    setIsAvailable(true);
    setEditingItem(null);
  };

  const handleEdit = (item: MenuItem) => {
    setEditingItem(item);
    setName(item.name);
    setDescription(item.description);
    setPrice(item.price);
    setCost(item.cost);
    setCategory(item.category);
    setEmoji(item.emoji);
    setPrepTime(item.prepTime);
    setStock(item.stock);
    setIsAvailable(item.isAvailable);
    setIsFormOpen(true);
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) return;

    const newItem: MenuItem = {
      id: editingItem ? editingItem.id : `item-${Date.now()}`,
      name,
      description,
      price: Number(price),
      cost: Number(cost),
      category,
      emoji,
      prepTime: Number(prepTime),
      stock: Number(stock),
      isAvailable,
    };

    if (editingItem) {
      onUpdateMenuItem(newItem);
    } else {
      onAddMenuItem(newItem);
    }
    setIsFormOpen(false);
    resetForm();
  };

  const handleDelete = (id: string) => {
    if (confirm('Действительно удалить эту позицию меню? Она исчезнет из касс и отчетов.')) {
      onDeleteMenuItem(id);
    }
  };

  const toggleAvailability = (item: MenuItem) => {
    onUpdateMenuItem({
      ...item,
      isAvailable: !item.isAvailable,
    });
  };

  // Filtered Menu
  const filteredItems = menuItems.filter((item) => {
    const matchesCategory = selectedCategory === 'all' || item.category === selectedCategory;
    const matchesSearch = item.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
                          item.description.toLowerCase().includes(searchQuery.toLowerCase());
    return matchesCategory && matchesSearch;
  });

  // Calculate Average food cost margin or stats
  const menuStats = {
    totalCount: menuItems.length,
    activeCount: menuItems.filter((i) => i.isAvailable).length,
    avgPrice: menuItems.length > 0 ? Math.round(menuItems.reduce((acc, c) => acc + c.price, 0) / menuItems.length) : 0,
  };

  return (
    <div className="flex-1 overflow-y-auto bg-slate-50 p-8">
      {/* Top Banner and stats */}
      <div className="flex flex-col md:flex-row justify-between items-start md:items-center gap-4 mb-8">
        <div>
          <h2 className="text-2xl font-bold tracking-tight text-slate-900 font-sans">
            Управление меню ресторана
          </h2>
          <p className="text-sm text-slate-500 mt-1">
            Добавляйте блюда, настраивайте категории, фуд кост наценки и отслеживайте наличие на складе.
          </p>
        </div>

        <button
          onClick={() => {
            resetForm();
            setIsFormOpen(true);
          }}
          className="flex items-center gap-2 px-5 py-3 rounded-xl bg-blue-600 border border-blue-700 hover:bg-blue-700 text-white font-semibold shadow-sm transition-all duration-300 transform active:scale-95"
        >
          <Plus className="w-4 h-4" />
          <span>Добавить блюдо</span>
        </button>
      </div>

      {/* Stats row */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-6 mb-8">
        <div className="bg-white p-4 rounded-xl border border-slate-200 flex items-center gap-4">
          <div className="p-3 bg-slate-100 rounded-lg text-slate-600 font-bold font-sans">
            {menuStats.totalCount}
          </div>
          <div>
            <p className="text-xs text-slate-400 font-medium">Всего позиций</p>
            <p className="text-sm font-bold text-slate-800">Блюд и напитков</p>
          </div>
        </div>
        <div className="bg-white p-4 rounded-xl border border-slate-200 flex items-center gap-4">
          <div className="p-3 bg-emerald-50 rounded-lg text-emerald-600 font-bold font-sans">
            {menuStats.activeCount}
          </div>
          <div>
            <p className="text-xs text-slate-400 font-medium">В наличии</p>
            <p className="text-sm font-bold text-slate-800">Доступно к заказу</p>
          </div>
        </div>
        <div className="bg-white p-4 rounded-xl border border-slate-200 flex items-center gap-4">
          <div className="p-3 bg-blue-50 rounded-lg text-blue-600 font-bold font-sans">
            {menuStats.avgPrice} ₽
          </div>
          <div>
            <p className="text-xs text-slate-400 font-medium">Средняя цена</p>
            <p className="text-sm font-bold text-slate-800">Одного чека-позиции</p>
          </div>
        </div>
      </div>

      {/* Filter and search controllers */}
      <div className="flex flex-col md:flex-row justify-between items-stretch md:items-center gap-4 mb-6">
        {/* Horizontal Category Toggles */}
        <div className="flex flex-wrap gap-2.5">
          <button
            onClick={() => setSelectedCategory('all')}
            className={`px-4 py-2.5 rounded-xl text-xs font-semibold leading-none transition-all ${
              selectedCategory === 'all'
                ? 'bg-slate-900 text-white'
                : 'bg-white text-slate-600 border border-slate-200 hover:bg-slate-50'
            }`}
          >
            Все категории
          </button>
          {categories.map((cat) => (
            <button
              key={cat.id}
              onClick={() => setSelectedCategory(cat.slug)}
              className={`px-4 py-2.5 rounded-xl text-xs font-semibold leading-none transition-all ${
                selectedCategory === cat.slug
                  ? 'bg-slate-900 text-white'
                  : 'bg-white text-slate-600 border border-slate-200 hover:bg-slate-50'
              }`}
            >
              {cat.name}
            </button>
          ))}
        </div>

        {/* Search */}
        <div className="relative min-w-[240px]">
          <Search className="w-4 h-4 text-slate-400 absolute left-3 top-1/2 -translate-y-1/2" />
          <input
            type="text"
            placeholder="Поиск блюда..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-full pl-9 pr-4 py-2 bg-white border border-slate-200 rounded-xl text-xs font-semibold focus:outline-none focus:border-blue-500 transition-colors placeholder-slate-400"
          />
        </div>
      </div>

      {/* Grid List of menu items */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {filteredItems.map((item) => {
          const marginPercent = item.price > 0 ? ((item.price - item.cost) / item.price) * 100 : 0;
          const isCriticalMargin = marginPercent < 45; // Low profitability warning

          return (
            <div
              key={item.id}
              className={`bg-white rounded-2xl border border-slate-200 overflow-hidden shadow-sm hover:shadow-md transition-all duration-300 flex flex-col justify-between ${
                !item.isAvailable ? 'opacity-70 grayscale-[30%]' : ''
              }`}
            >
              {/* Card Body */}
              <div className="p-5 flex-1 select-none">
                <div className="flex justify-between items-start gap-3">
                  <div className="flex items-center gap-3">
                    <span className="text-3xl p-2.5 bg-slate-50 border border-slate-100/80 rounded-2xl shadow-inner leading-none">
                      {item.emoji}
                    </span>
                    <div>
                      <h3 className="font-sans font-bold text-sm text-slate-900 line-clamp-1">
                        {item.name}
                      </h3>
                      <p className="text-[10px] font-mono font-medium text-slate-400 uppercase tracking-wide">
                        {categories.find((c) => c.slug === item.category)?.name || item.category}
                      </p>
                    </div>
                  </div>

                  {/* Available indicator badge click trigger */}
                  <button
                    onClick={() => toggleAvailability(item)}
                    className={`p-1.5 rounded-lg transition-colors border ${
                      item.isAvailable
                        ? 'bg-emerald-50 border-emerald-100 text-emerald-600 hover:bg-emerald-100'
                        : 'bg-rose-50 border-rose-100 text-rose-500 hover:bg-rose-100'
                    }`}
                    title={item.isAvailable ? 'Активно на кассах. Нажмите чтоб отключить' : 'Отключено. Нажмите чтоб активировать'}
                  >
                    {item.isAvailable ? <CheckCircle className="w-3.5 h-3.5" /> : <XCircle className="w-3.5 h-3.5" />}
                  </button>
                </div>

                <p className="text-xs text-slate-500 mt-4 line-clamp-2 min-h-[32px]">
                  {item.description || 'Описание не указано.'}
                </p>

                {/* Sub-meta (prep time, stock) */}
                <div className="flex gap-4 mt-4 py-2 border-y border-slate-50 text-[11px] text-slate-400 font-mono">
                  <div className="flex items-center gap-1">
                    <Clock className="w-3.5 h-3.5 text-slate-300" />
                    <span>~{item.prepTime} мин</span>
                  </div>
                  <div className="flex items-center gap-1">
                    <span className="w-1.5 h-1.5 rounded-full bg-blue-400"></span>
                    <span>
                      Остаток:{' '}
                      {item.stock === -1 ? (
                        <span className="text-emerald-500 font-bold">Без лимита</span>
                      ) : (
                        <span className={`font-semibold ${item.stock <= 5 ? 'text-rose-500 font-bold' : 'text-slate-800'}`}>
                          {item.stock} порц
                        </span>
                      )}
                    </span>
                  </div>
                </div>
              </div>

              {/* Price and Action Footer */}
              <div className="p-4 bg-slate-50/60 border-t border-slate-100 flex items-center justify-between">
                <div>
                  <span className="text-xs text-slate-400 block font-medium">Менеджерские цены:</span>
                  <div className="flex items-baseline gap-2">
                    <span className="text-base font-extrabold text-slate-900">
                      {item.price} ₽
                    </span>
                    <span className="text-[10px] font-mono text-slate-400">
                      (Себ: {item.cost} ₽)
                    </span>
                  </div>
                  <div className="flex items-center gap-1 mt-0.5">
                    <span
                      className={`text-[9px] font-mono uppercase font-bold px-1.5 py-0.5 rounded ${
                        isCriticalMargin
                          ? 'bg-rose-50 border border-rose-100 text-rose-600 flex items-center gap-0.5'
                          : 'bg-emerald-50 text-emerald-700'
                      }`}
                    >
                      {isCriticalMargin && <AlertCircle className="w-2.5 h-2.5" />}
                      Наценка: {Math.round(marginPercent)}%
                    </span>
                  </div>
                </div>

                {/* Edit and Trash handlers */}
                <div className="flex items-center gap-1.5">
                  <button
                    onClick={() => handleEdit(item)}
                    className="p-2.5 bg-white border border-slate-200 text-slate-600 hover:text-blue-600 rounded-xl hover:border-blue-150 shadow-sm hover:shadow transition-all"
                    title="Редактировать блюдо"
                  >
                    <Edit2 className="w-3.5 h-3.5" />
                  </button>
                  <button
                    onClick={() => handleDelete(item.id)}
                    className="p-2.5 bg-white border border-slate-200 text-slate-400 hover:text-rose-600 rounded-xl hover:border-rose-150 shadow-sm hover:shadow transition-all"
                    title="Удалить"
                  >
                    <Trash2 className="w-3.5 h-3.5" />
                  </button>
                </div>
              </div>
            </div>
          );
        })}

        {filteredItems.length === 0 && (
          <div className="col-span-1 md:col-span-2 lg:col-span-3 bg-white py-16 text-center text-slate-400 border border-slate-200 rounded-2xl flex flex-col items-center justify-center gap-2">
            <span className="text-4xl">🔍</span>
            <p className="font-sans font-bold text-slate-700">Ничего не найдено</p>
            <p className="text-xs text-slate-400">Попробуйте изменить категорию или очистить поисковый запрос.</p>
          </div>
        )}
      </div>

      {/* Modular Form Modal for add & edit dishes */}
      {isFormOpen && (
        <div className="fixed inset-0 bg-slate-900/40 backdrop-blur-sm z-50 flex items-center justify-center p-4">
          <div className="bg-white rounded-3xl w-full max-w-lg shadow-2xl border border-slate-100 overflow-hidden text-slate-800 transform transition-all animate-in fade-in zoom-in-95 duration-200">
            {/* Modal Header */}
            <div className="px-6 py-5 border-b border-slate-100 flex justify-between items-center bg-slate-50">
              <h3 className="text-base font-bold text-slate-900 font-sans">
                {editingItem ? 'Редактировать блюдо' : 'Создать новую позицию меню'}
              </h3>
              <button
                onClick={() => setIsFormOpen(false)}
                className="p-1 px-3 text-sm text-slate-400 hover:text-slate-600 font-semibold border rounded-lg hover:bg-slate-100 select-none cursor-pointer"
              >
                Закрыть
              </button>
            </div>

            {/* Modal Form Content */}
            <form onSubmit={handleSubmit} className="p-6 space-y-4">
              <div className="grid grid-cols-4 gap-4">
                {/* Emoji / Icon Selector */}
                <div className="col-span-1">
                  <label className="text-[10px] uppercase tracking-wider font-semibold font-mono text-slate-400 block mb-1">
                    Символ / Иконка
                  </label>
                  <input
                    type="text"
                    value={emoji}
                    onChange={(e) => setEmoji(e.target.value)}
                    maxLength={2}
                    className="w-full text-center p-2.5 text-2xl bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:border-blue-500"
                    title="Введите эмодзи для блюда"
                  />
                </div>

                {/* Name */}
                <div className="col-span-3">
                  <label className="text-[10px] uppercase tracking-wider font-semibold font-mono text-slate-400 block mb-1">
                    Название блюда *
                  </label>
                  <input
                    type="text"
                    required
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    placeholder="Борщ Черниговский с пампушками"
                    className="w-full p-2.5 text-xs font-semibold bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:border-blue-500"
                  />
                </div>
              </div>

              {/* Description */}
              <div>
                <label className="text-[10px] uppercase tracking-wider font-semibold font-mono text-slate-400 block mb-1">
                  Ингредиенты / Краткое описание
                </label>
                <textarea
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  placeholder="Маринованная свекла, нежная говяжья грудинка, подается со сметаной и..."
                  rows={2}
                  className="w-full p-2.5 text-xs bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:border-blue-500"
                />
              </div>

              {/* Category selector & Preparation Time */}
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="text-[10px] uppercase tracking-wider font-semibold font-mono text-slate-400 block mb-1">
                    Категория меню
                  </label>
                  <select
                    value={category}
                    onChange={(e) => setCategory(e.target.value)}
                    className="w-full p-2.5 text-xs font-semibold bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:border-blue-500"
                  >
                    {categories.map((c) => (
                      <option key={c.id} value={c.slug}>
                        {c.name}
                      </option>
                    ))}
                  </select>
                </div>

                <div>
                  <label className="text-[10px] uppercase tracking-wider font-semibold font-mono text-slate-400 block mb-1">
                    Время приготовления (мин)
                  </label>
                  <input
                    type="number"
                    min={1}
                    value={prepTime}
                    onChange={(e) => setPrepTime(Number(e.target.value))}
                    className="w-full p-2.5 text-xs font-semibold bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:border-blue-500"
                  />
                </div>
              </div>

              {/* Price & Cost Margin Calculations */}
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="text-[10px] uppercase tracking-wider font-semibold font-mono text-slate-400 block mb-1">
                    Цена продажи (₽)
                  </label>
                  <input
                    type="number"
                    min={0}
                    required
                    value={price}
                    onChange={(e) => setPrice(Number(e.target.value))}
                    className="w-full p-2.5 text-xs font-semibold bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:border-blue-500"
                  />
                </div>

                <div>
                  <label className="text-[10px] uppercase tracking-wider font-semibold font-mono text-slate-400 block mb-1">
                    Себестоимость / Фуд кост (₽)
                  </label>
                  <input
                    type="number"
                    min={0}
                    required
                    value={cost}
                    onChange={(e) => setCost(Number(e.target.value))}
                    className="w-full p-2.5 text-xs font-semibold bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:border-blue-500"
                  />
                </div>
              </div>

              {/* Dynamic Cost helper info */}
              <div className="p-3.5 bg-blue-50/50 rounded-xl border border-blue-100 flex items-center justify-between text-xs font-medium text-slate-600">
                <span>Прогнозируемая чистая выгода:</span>
                <span className="font-mono text-blue-700 font-bold">
                  {price - cost > 0 ? `${price - cost} ₽` : '0 ₽'}{' '}
                  <span className="text-[10px] text-blue-400">
                    ({price > 0 ? Math.round(((price - cost) / price) * 100) : 0}%)
                  </span>
                </span>
              </div>

              {/* Stock and availability */}
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="text-[10px] uppercase tracking-wider font-semibold font-mono text-slate-400 block mb-1">
                    Лимитированный остаток
                  </label>
                  <select
                    value={stock === -1 ? 'unlimited' : 'limited'}
                    onChange={(e) => setStock(e.target.value === 'unlimited' ? -1 : 15)}
                    className="w-full p-2.5 text-xs font-semibold bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:border-blue-500"
                  >
                    <option value="unlimited">Игнорировать (Unlimited)</option>
                    <option value="limited">Счетчик остатка (Порции)</option>
                  </select>
                </div>

                {stock !== -1 && (
                  <div>
                    <label className="text-[10px] uppercase tracking-wider font-semibold font-mono text-slate-400 block mb-1">
                      Остаток на порции
                    </label>
                    <input
                      type="number"
                      min={0}
                      value={stock}
                      onChange={(e) => setStock(Number(e.target.value))}
                      className="w-full p-2.5 text-xs font-semibold bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:border-blue-500"
                    />
                  </div>
                )}
              </div>

              {/* Checkbox for visual status */}
              <div className="flex items-center gap-2.5 py-1">
                <input
                  type="checkbox"
                  id="availField"
                  checked={isAvailable}
                  onChange={(e) => setIsAvailable(e.target.checked)}
                  className="w-4 h-4 text-blue-600 focus:ring-blue-500 border-slate-300 rounded"
                />
                <label htmlFor="availField" className="text-xs font-semibold text-slate-700 select-none">
                  Активно к продаже на всех кассах и планшетах (В наличии)
                </label>
              </div>

              {/* Button controllers */}
              <div className="flex justify-end gap-3 pt-4 border-t border-slate-100">
                <button
                  type="button"
                  onClick={() => setIsFormOpen(false)}
                  className="px-4.5 py-2.5 text-xs font-semibold text-slate-600 hover:text-slate-800 border rounded-xl hover:bg-slate-100 select-none cursor-pointer"
                >
                  Отмена
                </button>
                <button
                  type="submit"
                  className="px-5 py-2.5 text-xs font-bold text-white bg-blue-600 border border-blue-700 hover:bg-blue-700 rounded-xl transition-all"
                >
                  {editingItem ? 'Сохранить изменения' : 'Создать блюдо'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
