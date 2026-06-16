import React, { useState, useEffect } from 'react';
import { Search, Plus, Edit2, Trash2, CheckCircle, XCircle, Clock, AlertCircle, Sparkles, HelpCircle, Layers, Check, Award, FileText } from 'lucide-react';
import { MenuItem, MenuCategory, MenuItemType, MenuItemScope, TechCardVersion, TechCardIngredient } from '../types';
import { RESTAURANTS } from './Sidebar';

interface MenuPanelProps {
  menuItems: MenuItem[];
  categories: MenuCategory[];
  onAddMenuItem: (item: MenuItem) => void;
  onUpdateMenuItem: (item: MenuItem) => void;
  onDeleteMenuItem: (id: string) => void;
  selectedRestaurant: string;
}

export default function MenuPanel({
  menuItems,
  categories,
  onAddMenuItem,
  onUpdateMenuItem,
  onDeleteMenuItem,
  selectedRestaurant,
}: MenuPanelProps) {
  const [selectedCategory, setSelectedCategory] = useState<string>('all');
  const [searchQuery, setSearchQuery] = useState<string>('');
  const [editingItem, setEditingItem] = useState<MenuItem | null>(null);
  const [isFormOpen, setIsFormOpen] = useState(false);

  // Modal active sub-tab
  const [modalTab, setModalTab] = useState<'general' | 'recipes'>('general');

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

  // New states for scope, type and recipes
  const [type, setType] = useState<MenuItemType>('dish');
  const [scope, setScope] = useState<MenuItemScope>('global');
  const [techCards, setTechCards] = useState<TechCardVersion[]>([]);
  const [selectedVersionId, setSelectedVersionId] = useState<string>('');

  // Initial template for a new techcard version
  const createBlankVersion = (nameStr = 'Базовая рецептура'): TechCardVersion => ({
    id: `tc-${Date.now()}`,
    versionName: nameStr,
    dateFrom: new Date().toISOString().split('T')[0],
    isActive: true,
    ingredients: [],
    instructions: 'Опишите технологические шаги приготовления...'
  });

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
    setType('dish');
    setScope('global');
    setTechCards([]);
    setSelectedVersionId('');
    setModalTab('general');
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
    setType(item.type || 'dish');
    setScope(item.scope || 'global');
    
    const existingCards = item.techCards || [];
    setTechCards(existingCards);
    
    // Choose active card or first card, or empty
    const activeCard = existingCards.find(c => c.isActive) || existingCards[0];
    setSelectedVersionId(activeCard ? activeCard.id : '');
    
    setModalTab('general');
    setIsFormOpen(true);
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) return;

    // Compile techCards and sync cost if active card has ingredients
    let computedCost = cost;
    const activeCard = techCards.find(c => c.id === selectedVersionId || c.isActive);
    if (activeCard && activeCard.ingredients && activeCard.ingredients.length > 0) {
      computedCost = activeCard.ingredients.reduce((acc, ing) => acc + ing.costPerUnit, 0);
    }

    const newItem: MenuItem = {
      id: editingItem ? editingItem.id : `item-${Date.now()}`,
      name,
      description,
      price: Number(price),
      cost: Number(computedCost),
      category,
      emoji,
      prepTime: Number(prepTime),
      stock: Number(stock),
      isAvailable,
      type,
      scope,
      techCards: techCards
    };

    if (editingItem) {
      onUpdateMenuItem(newItem);
    } else {
      onAddMenuItem(newItem);
    }
    setIsFormOpen(false);
    resetForm();
  };

  // Techcard helper modifications
  const handleAddVersion = () => {
    // Set all other versions to inactive
    const updatedCards = techCards.map(c => ({ ...c, isActive: false }));
    const newVer = createBlankVersion(`Рецептура v${techCards.length + 1}`);
    const finalCards = [...updatedCards, newVer];
    setTechCards(finalCards);
    setSelectedVersionId(newVer.id);
  };

  const handleSelectVersion = (id: string) => {
    setSelectedVersionId(id);
    setTechCards(prev => prev.map(c => ({
      ...c,
      isActive: c.id === id
    })));
  };

  const handleUpdateVersionField = (field: keyof TechCardVersion, value: any) => {
    setTechCards(prev => prev.map(c => {
      if (c.id === selectedVersionId) {
        return { ...c, [field]: value };
      }
      return c;
    }));
  };

  const handleAddIngredient = () => {
    const newIngredient: TechCardIngredient = {
      name: 'Сырье / Специя',
      amount: '100г',
      costPerUnit: 50
    };
    setTechCards(prev => prev.map(c => {
      if (c.id === selectedVersionId) {
        return {
          ...c,
          ingredients: [...(c.ingredients || []), newIngredient]
        };
      }
      return c;
    }));
  };

  const handleUpdateIngredient = (index: number, field: keyof TechCardIngredient, val: any) => {
    setTechCards(prev => prev.map(c => {
      if (c.id === selectedVersionId) {
        const ingredients = [...(c.ingredients || [])];
        ingredients[index] = {
          ...ingredients[index],
          [field]: field === 'costPerUnit' ? Number(val) : val
        };
        return { ...c, ingredients };
      }
      return c;
    }));
  };

  const handleRemoveIngredient = (index: number) => {
    setTechCards(prev => prev.map(c => {
      if (c.id === selectedVersionId) {
        return {
          ...c,
          ingredients: (c.ingredients || []).filter((_, idx) => idx !== index)
        };
      }
      return c;
    }));
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

  // Filtered Menu depending on sidebar restaurant scope and tab categories
  const filteredItems = menuItems.filter((item) => {
    // 1. Restaurant Scope Match
    const matchesScope = selectedRestaurant === 'all' || 
                         !item.scope || 
                         item.scope === 'global' || 
                         item.scope === selectedRestaurant;

    // 2. Category Match
    const matchesCategory = selectedCategory === 'all' || item.category === selectedCategory;

    // 3. Search text Match
    const matchesSearch = item.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
                          item.description.toLowerCase().includes(searchQuery.toLowerCase());

    return matchesScope && matchesCategory && matchesSearch;
  });

  // Calculate Average food cost margin or stats
  const menuStats = {
    totalCount: filteredItems.length,
    activeCount: filteredItems.filter((i) => i.isAvailable).length,
    avgPrice: filteredItems.length > 0 ? Math.round(filteredItems.reduce((acc, c) => acc + c.price, 0) / filteredItems.length) : 0,
  };

  // Active techcard selected to display ingredients inside form
  const selectedVersion = techCards.find(c => c.id === selectedVersionId);
  const costSum = selectedVersion?.ingredients?.reduce((sum, ing) => sum + ing.costPerUnit, 0) || 0;

  // Track and update the dynamic cost on change
  useEffect(() => {
    if (costSum > 0) {
      setCost(costSum);
    }
  }, [costSum]);

  return (
    <div className="flex-1 overflow-y-auto bg-slate-50 p-8">
      {/* Top Banner and stats */}
      <div className="flex flex-col md:flex-row justify-between items-start md:items-center gap-4 mb-8">
        <div>
          <h2 className="text-2xl font-bold tracking-tight text-slate-900 font-sans flex items-center gap-2">
            <span>Каталог блюд и услуг</span>
            <span className="text-xs bg-blue-100 text-blue-600 font-extrabold uppercase px-2 py-0.5 rounded-full font-mono">
              {RESTAURANTS.find(r => r.id === selectedRestaurant)?.name || 'Все рестораны'}
            </span>
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
          className="flex items-center gap-2 px-5 py-3 rounded-xl bg-blue-600 border border-blue-700 hover:bg-blue-700 text-white font-semibold shadow-sm transition-all duration-300 transform active:scale-95 cursor-pointer text-xs"
        >
          <Plus className="w-4 h-4" />
          <span>Добавить позицию</span>
        </button>
      </div>

      {/* Stats row */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-6 mb-8 select-none">
        <div className="bg-white p-4 rounded-xl border border-slate-200 flex items-center gap-4">
          <div className="p-3 bg-slate-100 rounded-lg text-slate-600 font-extrabold font-mono text-sm leading-none shrink-0">
            {menuStats.totalCount}
          </div>
          <div>
            <p className="text-xs text-slate-400 font-medium">Позиций в выборке</p>
            <p className="text-sm font-bold text-slate-800">Блюда, комплекты, товары</p>
          </div>
        </div>
        <div className="bg-white p-4 rounded-xl border border-slate-200 flex items-center gap-4">
          <div className="p-3 bg-emerald-50 rounded-lg text-emerald-600 font-extrabold font-mono text-sm leading-none shrink-0">
            {menuStats.activeCount}
          </div>
          <div>
            <p className="text-xs text-slate-400 font-medium">Активных в зале</p>
            <p className="text-sm font-bold text-slate-800">Доступно на планшетах</p>
          </div>
        </div>
        <div className="bg-white p-4 rounded-xl border border-slate-200 flex items-center gap-4">
          <div className="p-3 bg-blue-50 rounded-lg text-blue-600 font-extrabold font-mono text-sm leading-none shrink-0">
            {menuStats.avgPrice} ₽
          </div>
          <div>
            <p className="text-xs text-slate-400 font-medium">Средняя цена за блюдо</p>
            <p className="text-sm font-bold text-slate-800">Рентабельность back-office</p>
          </div>
        </div>
      </div>

      {/* Filter and search controllers */}
      <div className="flex flex-col md:flex-row justify-between items-stretch md:items-center gap-4 mb-6 select-none shrink-0">
        {/* Horizontal Category Toggles */}
        <div className="flex flex-wrap gap-2">
          <button
            onClick={() => setSelectedCategory('all')}
            className={`px-4 py-2.5 rounded-xl text-xs font-semibold leading-none transition-all ${
              selectedCategory === 'all'
                ? 'bg-slate-900 text-white shadow-sm'
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
                  ? 'bg-slate-900 text-white shadow-sm'
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
          const marginPercent = item.price > 0 ? ((item.price - item.cost) / item.price) * 105 : 0;
          const isCriticalMargin = marginPercent < 45; // Low profitability warning
          const itemTypeLabel =
            item.type === 'modifier'
              ? 'Модификатор'
              : item.type === 'goods'
              ? 'Складской товар'
              : 'Блюдо кухни';

          const activeScope = RESTAURANTS.find(r => r.id === item.scope);

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
                    <span className="text-3xl p-2.5 bg-slate-50 border border-slate-100/80 rounded-2xl shadow-inner leading-none shrink-0">
                      {item.emoji}
                    </span>
                    <div>
                      <div className="flex items-center gap-1.5 flex-wrap">
                        <span className="text-[10px] font-extrabold uppercase px-1.5 py-0.5 rounded bg-slate-100 text-slate-500 font-mono tracking-wide leading-none">
                          {itemTypeLabel}
                        </span>
                        <span className={`text-[10px] font-extrabold uppercase px-1.5 py-0.5 rounded font-mono tracking-wide leading-none ${
                          !item.scope || item.scope === 'global'
                            ? 'bg-blue-50 text-blue-600 border border-blue-100'
                            : 'bg-emerald-50 text-emerald-600 border border-emerald-100'
                        }`}>
                          {activeScope ? activeScope.name.split(' ')[0] : 'Весь Сервер'}
                        </span>
                      </div>
                      <h3 className="font-sans font-bold text-sm text-slate-900 line-clamp-1 mt-1">
                        {item.name}
                      </h3>
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

                {/* Sub-meta (prep time, stock, recipes indicator) */}
                <div className="flex flex-wrap gap-x-4 gap-y-1 mt-4 py-2 border-y border-slate-50 text-[11px] text-slate-400 font-mono">
                  <div className="flex items-center gap-1">
                    <Clock className="w-3.5 h-3.5 text-slate-300" />
                    <span>~{item.prepTime} мин</span>
                  </div>
                  <div className="flex items-center gap-1">
                    <span className="w-1.5 h-1.5 rounded-full bg-blue-400"></span>
                    <span>Остаток: {item.stock === -1 ? <span className="text-emerald-500 font-bold">Без лимита</span> : <span className="text-slate-800 font-semibold">{item.stock} порц</span>}</span>
                  </div>
                  {item.techCards && item.techCards.length > 0 && (
                    <div className="flex items-center gap-1 text-slate-500 font-bold">
                      <Award className="w-3.5 h-3.5 text-indigo-500 shrink-0" />
                      <span>{item.techCards.length} техкарты</span>
                    </div>
                  )}
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
                  <div className="flex items-center gap-1 mt-0.5 select-none">
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
                    className="p-2.5 bg-white border border-slate-200 text-slate-600 hover:text-blue-600 rounded-xl hover:border-blue-150 shadow-sm hover:shadow transition-all cursor-pointer"
                    title="Редактировать техкарту и данные"
                  >
                    <Edit2 className="w-3.5 h-3.5" />
                  </button>
                  <button
                    onClick={() => handleDelete(item.id)}
                    className="p-2.5 bg-white border border-slate-200 text-slate-400 hover:text-rose-600 rounded-xl hover:border-rose-150 shadow-sm hover:shadow transition-all cursor-pointer"
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
            <span className="text-4xl text-slate-300">🔍</span>
            <p className="font-sans font-bold text-slate-700">Ничего не найдено в этом ресторане</p>
            <p className="text-xs text-slate-400 max-w-[280px] mx-auto select-none leading-relaxed">
              Попробуйте изменить категорию, сбросить поисковый запрос или переключить ресторан на серверный уровень.
            </p>
          </div>
        )}
      </div>

      {/* Advanced Interactive Modal for add & edit dishes, modifiers & techcards */}
      {isFormOpen && (
        <div className="fixed inset-0 bg-slate-900/40 backdrop-blur-sm z-50 flex items-center justify-center p-4">
          <div className="bg-white rounded-3xl w-full max-w-2xl shadow-2xl border border-slate-100 overflow-hidden text-slate-800 flex flex-col max-h-[90vh]">
            
            {/* Modal Header */}
            <div className="px-6 py-4.5 border-b border-slate-100 flex justify-between items-center bg-slate-50 shrink-0">
              <div className="flex items-center gap-2 select-none">
                <Sparkles className="text-blue-500 w-5 h-5 shrink-0" />
                <h3 className="text-sm font-extrabold text-slate-900 font-sans">
                  {editingItem ? `Редактирование: ${editingItem.name}` : 'Создание позиции RMS'}
                </h3>
              </div>
              <button
                onClick={() => setIsFormOpen(false)}
                className="px-3 py-1.5 text-xs text-slate-400 hover:text-slate-600 font-bold border rounded-lg hover:bg-slate-100 select-none cursor-pointer"
              >
                Закрыть
              </button>
            </div>

            {/* Selection Tabs Inside Modal for Techcard editor */}
            <div className="bg-white px-6 border-b border-slate-100 flex gap-4 text-xs select-none shrink-0 font-bold">
              <button
                onClick={() => setModalTab('general')}
                className={`py-3.5 border-b-2 transition-all ${
                  modalTab === 'general'
                    ? 'border-blue-600 text-blue-600'
                    : 'border-transparent text-slate-400 hover:text-slate-700'
                }`}
              >
                1. Основные параметры
              </button>
              <button
                onClick={() => {
                  setModalTab('recipes');
                  if (techCards.length === 0) {
                    handleAddVersion(); // Initialize blank version automatically for editing
                  }
                }}
                className={`py-3.5 border-b-2 transition-all flex items-center gap-1.5 ${
                  modalTab === 'recipes'
                    ? 'border-blue-600 text-blue-600'
                    : 'border-transparent text-slate-400 hover:text-slate-700'
                }`}
              >
                <FileText className="w-3.5 h-3.5" />
                <span>2. Версии техкарт & Рецептуры ({techCards.length})</span>
              </button>
            </div>

            {/* Modal Scrollable Contents */}
            <div className="flex-1 overflow-y-auto p-6">
              <form onSubmit={handleSubmit} id="rms-menu-form" className="space-y-4">
                
                {/* TAB 1: GENERAL INFO */}
                {modalTab === 'general' && (
                  <div className="space-y-4">
                    {/* Symbol + Name */}
                    <div className="grid grid-cols-4 gap-4">
                      <div className="col-span-1">
                        <label className="text-[10px] uppercase tracking-wider font-bold font-mono text-slate-400 block mb-1 select-none">
                          Иконка (Эмодзи)
                        </label>
                        <input
                          type="text"
                          value={emoji}
                          onChange={(e) => setEmoji(e.target.value)}
                          maxLength={2}
                          className="w-full text-center p-2.5 text-2xl bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:border-blue-500"
                        />
                      </div>

                      <div className="col-span-3">
                        <label className="text-[10px] uppercase tracking-wider font-bold font-mono text-slate-400 block mb-1 select-none">
                          Название позиции *
                        </label>
                        <input
                          type="text"
                          required
                          value={name}
                          onChange={(e) => setName(e.target.value)}
                          placeholder="Стейк Томагавк с соусом Чимичурри"
                          className="w-full p-2.5 text-xs font-bold text-slate-800 bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:border-blue-500"
                        />
                      </div>
                    </div>

                    {/* Class Specification: Dish, Modifier or Goods */}
                    <div className="grid grid-cols-2 gap-4">
                      <div>
                        <label className="text-[10px] uppercase tracking-wider font-bold font-mono text-slate-400 block mb-1 select-none">
                          Тип номенклатуры (Класс)
                        </label>
                        <select
                          value={type}
                          onChange={(e) => setType(e.target.value as MenuItemType)}
                          className="w-full p-2.5 text-xs font-bold bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:border-blue-500"
                        >
                          <option value="dish">Блюдо кухни (из продуктов)</option>
                          <option value="modifier">Модификатор (топпинг, доп. соус)</option>
                          <option value="goods">Товар склада (штучный, баночный)</option>
                        </select>
                      </div>

                      <div>
                        <label className="text-[10px] uppercase tracking-wider font-bold font-mono text-slate-400 block mb-1 select-none">
                          Область видимости (Ресторан)
                        </label>
                        <select
                          value={scope}
                          onChange={(e) => setScope(e.target.value as MenuItemScope)}
                          className="w-full p-2.5 text-xs font-bold bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:border-blue-500"
                        >
                          {RESTAURANTS.map((res) => (
                            <option key={res.id} value={res.id}>
                              {res.name} ({res.id === 'all' ? 'Общий сервер' : 'Локальный'})
                            </option>
                          ))}
                        </select>
                      </div>
                    </div>

                    {/* Description */}
                    <div>
                      <label className="text-[10px] uppercase tracking-wider font-bold font-mono text-slate-400 block mb-1 select-none">
                        Краткое техническое описание
                      </label>
                      <textarea
                        value={description}
                        onChange={(e) => setDescription(e.target.value)}
                        placeholder="Краткое описание для официантов, аллергены, вес готового блюда..."
                        rows={2}
                        className="w-full p-2.5 text-xs bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:border-blue-500"
                      />
                    </div>

                    {/* Category & Cook time */}
                    <div className="grid grid-cols-2 gap-4">
                      <div>
                        <label className="text-[10px] uppercase tracking-wider font-bold font-mono text-slate-400 block mb-1 select-none">
                          Раздел меню (Категория)
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
                        <label className="text-[10px] uppercase tracking-wider font-bold font-mono text-slate-400 block mb-1 select-none">
                          Время приготовления кухни (мин)
                        </label>
                        <input
                          type="number"
                          min={1}
                          value={prepTime}
                          onChange={(e) => setPrepTime(Number(e.target.value))}
                          className="w-full p-2.5 text-xs font-semibold bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:border-blue-500 font-mono"
                        />
                      </div>
                    </div>

                    {/* Costing values */}
                    <div className="grid grid-cols-2 gap-4">
                      <div>
                        <label className="text-[10px] uppercase tracking-wider font-bold font-mono text-slate-400 block mb-1 select-none">
                          Розничная цена продажи (₽)
                        </label>
                        <input
                          type="number"
                          min={0}
                          required
                          value={price}
                          onChange={(e) => setPrice(Number(e.target.value))}
                          className="w-full p-2.5 text-xs font-bold bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:border-blue-500 font-mono"
                        />
                      </div>

                      <div>
                        <label className="text-[10px] uppercase tracking-wider font-bold font-mono text-slate-400 block mb-1 select-none">
                          Себестоимость / Фуд кост (₽)
                        </label>
                        <input
                          type="number"
                          min={0}
                          value={cost}
                          onChange={(e) => setCost(Number(e.target.value))}
                          disabled={techCards.length > 0} // Lock if recipes exist
                          title={techCards.length > 0 ? "Сумма заблокирована, так как активны калькуляционные карты" : ""}
                          className={`w-full p-2.5 text-xs font-semibold border rounded-xl focus:outline-none font-mono ${
                            techCards.length > 0 ? 'bg-slate-100 text-slate-500 cursor-not-allowed border-slate-200' : 'bg-slate-50 border-slate-200 focus:border-blue-500 text-slate-800'
                          }`}
                        />
                      </div>
                    </div>

                    <div className="p-3 bg-blue-50/50 rounded-xl border border-blue-100 flex items-center justify-between text-xs text-slate-650 font-medium">
                      <span>Рассчитываемая выгода бэк-офиса:</span>
                      <span className="font-mono text-blue-700 font-bold">
                        {price - cost > 0 ? `${price - cost} ₽` : '0 ₽'}{' '}
                        <span className="text-[10px] text-blue-400">
                          ({price > 0 ? Math.round(((price - cost) / price) * 100) : 0}%)
                        </span>
                      </span>
                    </div>

                    {/* Stock limits */}
                    <div className="grid grid-cols-2 gap-4">
                      <div>
                        <label className="text-[10px] uppercase tracking-wider font-bold font-mono text-slate-400 block mb-1 select-none">
                          Лимитированный остаток
                        </label>
                        <select
                          value={stock === -1 ? 'unlimited' : 'limited'}
                          onChange={(e) => setStock(e.target.value === 'unlimited' ? -1 : 15)}
                          className="w-full p-2.5 text-xs font-semibold bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:border-blue-500"
                        >
                          <option value="unlimited">Игнорировать (Бесконечно)</option>
                          <option value="limited">Счетчик остатка (Порции)</option>
                        </select>
                      </div>

                      {stock !== -1 && (
                        <div>
                          <label className="text-[10px] uppercase tracking-wider font-bold font-mono text-slate-400 block mb-1 select-none">
                            Остаток на порции
                          </label>
                          <input
                            type="number"
                            min={0}
                            value={stock}
                            onChange={(e) => setStock(Number(e.target.value))}
                            className="w-full p-2.5 text-xs font-semibold bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:border-blue-500 font-mono"
                          />
                        </div>
                      )}
                    </div>

                    {/* Checkbox for active availability */}
                    <div className="flex items-center gap-2.5 py-1 select-none">
                      <input
                        type="checkbox"
                        id="availField"
                        checked={isAvailable}
                        onChange={(e) => setIsAvailable(e.target.checked)}
                        className="w-4 h-4 text-blue-600 focus:ring-blue-500 border-slate-300 rounded"
                      />
                      <label htmlFor="availField" className="text-xs font-bold text-slate-700 cursor-pointer">
                        Позиция проверена и допущена к выгрузке на кассы
                      </label>
                    </div>
                  </div>
                )}

                {/* TAB 2: TECHNICAL CARD VERSIONING */}
                {modalTab === 'recipes' && (
                  <div className="space-y-5">
                    {/* Active techcard version tabs summary */}
                    <div className="flex items-center justify-between border-b pb-3 border-slate-100 select-none">
                      <div>
                        <span className="text-[10px] font-mono tracking-widest text-slate-400 uppercase font-black">Рецептуры & акты разбора</span>
                        <h4 className="font-sans font-bold text-slate-850 mt-0.5">Линейка версий техпроцесса</h4>
                      </div>

                      <button
                        type="button"
                        onClick={handleAddVersion}
                        className="flex items-center gap-1.5 px-3 py-1.5 bg-slate-100 border hover:bg-slate-200 rounded-lg text-[10px] font-bold text-slate-700 transition"
                      >
                        <Plus className="w-3.5 h-3.5" />
                        <span>Новая техкарта (Версия)</span>
                      </button>
                    </div>

                    {/* Version Selector Carousel */}
                    {techCards.length > 0 ? (
                      <div className="grid grid-cols-4 gap-4">
                        {/* Selector items column */}
                        <div className="col-span-1 border-r border-slate-100 pr-3 space-y-2 select-none">
                          <label className="text-[9px] uppercase font-bold font-mono tracking-wider text-slate-400 block mb-1">Выбрать версию:</label>
                          {techCards.map((card) => (
                            <button
                              key={card.id}
                              type="button"
                              onClick={() => handleSelectVersion(card.id)}
                              className={`w-full text-left p-2.5 rounded-xl border text-[11px] font-bold transition flex flex-col gap-0.5 leading-tight ${
                                card.id === selectedVersionId
                                  ? 'border-blue-500 bg-blue-50/20 text-blue-700 '
                                  : 'border-slate-150 hover:bg-slate-50 text-slate-600'
                              }`}
                            >
                              <span className="truncate">{card.versionName}</span>
                              <span className="text-[9px] font-mono text-slate-400">{card.dateFrom}</span>
                              {card.isActive && (
                                <span className="bg-emerald-500 text-white font-mono text-[8px] px-1 py-0.5 rounded uppercase mt-1 inline-block uppercase text-center font-black">Активна</span>
                              )}
                            </button>
                          ))}
                        </div>

                        {/* Editor form fields column */}
                        {selectedVersion ? (
                          <div className="col-span-3 space-y-4">
                            <div className="grid grid-cols-2 gap-3">
                              <div>
                                <label className="text-[10px] uppercase font-bold font-mono text-slate-400 block mb-1">Название техкарты</label>
                                <input
                                  type="text"
                                  value={selectedVersion.versionName}
                                  onChange={(e) => handleUpdateVersionField('versionName', e.target.value)}
                                  className="w-full p-2 bg-slate-50 border border-slate-200 rounded-lg text-xs font-semibold"
                                />
                              </div>
                              <div>
                                <label className="text-[10px] uppercase font-bold font-mono text-slate-400 block mb-1">Срок действия (С даты)</label>
                                <input
                                  type="date"
                                  value={selectedVersion.dateFrom}
                                  onChange={(e) => handleUpdateVersionField('dateFrom', e.target.value)}
                                  className="w-full p-2 bg-slate-50 border border-slate-200 rounded-lg text-xs font-mono font-bold"
                                />
                              </div>
                            </div>

                            {/* Ingredients grid list */}
                            <div>
                              <div className="flex justify-between items-center mb-1.5">
                                <label className="text-[10px] uppercase font-bold font-mono text-slate-400 block">Ведомость сырья & Ингредиентов</label>
                                <button
                                  type="button"
                                  onClick={handleAddIngredient}
                                  className="flex items-center gap-1 text-[9px] font-black uppercase text-blue-600 hover:text-blue-700"
                                >
                                  <Plus className="w-3 h-3" />
                                  <span>Добавить сырьё</span>
                                </button>
                              </div>

                              <div className="space-y-1.5 max-h-[160px] overflow-y-auto border border-slate-150 p-2 rounded-xl bg-slate-50/50">
                                {selectedVersion.ingredients && selectedVersion.ingredients.length > 0 ? (
                                  selectedVersion.ingredients.map((ing, idx) => (
                                    <div key={idx} className="flex items-center gap-2 bg-white p-1.5 border border-slate-150 rounded-lg text-xs">
                                      <input
                                        type="text"
                                        value={ing.name}
                                        onChange={(e) => handleUpdateIngredient(idx, 'name', e.target.value)}
                                        placeholder="Название продукта"
                                        className="flex-1 px-2 py-1 bg-slate-50 rounded text-xs font-bold"
                                      />
                                      <input
                                        type="text"
                                        value={ing.amount}
                                        onChange={(e) => handleUpdateIngredient(idx, 'amount', e.target.value)}
                                        placeholder="150г"
                                        className="w-16 px-1.5 py-1 bg-slate-50 rounded font-mono text-center text-xs"
                                      />
                                      <div className="flex items-center gap-1 shrink-0">
                                        <input
                                          type="number"
                                          value={ing.costPerUnit}
                                          onChange={(e) => handleUpdateIngredient(idx, 'costPerUnit', e.target.value)}
                                          className="w-16 px-1.5 py-1 bg-slate-50 rounded font-mono text-right text-xs"
                                        />
                                        <span className="text-slate-400 text-[10px]">₽</span>
                                      </div>
                                      <button
                                        type="button"
                                        onClick={() => handleRemoveIngredient(idx)}
                                        className="text-slate-350 hover:text-rose-500 p-1 rounded hover:bg-slate-50"
                                      >
                                        <Trash2 className="w-3.5 h-3.5" />
                                      </button>
                                    </div>
                                  ))
                                ) : (
                                  <div className="py-8 text-center text-slate-450 border border-dashed border-slate-200 rounded-lg text-[11px]">
                                    Нет продуктов в этой версии калькуляционной карты. Нажмите "+ Добавить сырьё".
                                  </div>
                                )}
                              </div>

                              {/* Recipe Costing Dynamic Info */}
                              <div className="mt-2.5 p-3 rounded-xl border border-slate-200 bg-slate-50 flex justify-between items-center text-[11px] font-sans">
                                <div>
                                  <span className="font-bold text-slate-700 block">Авто-себестоимость версии:</span>
                                  <span className="text-[10px] text-slate-400 leading-none block">Суммируется по спецификации продуктов</span>
                                </div>
                                <span className="font-mono text-xs font-black text-slate-800 bg-white border border-slate-200 px-2.5 py-1 rounded-lg">
                                  {costSum} ₽
                                </span>
                              </div>
                            </div>

                            {/* Preparation steps */}
                            <div>
                              <label className="text-[10px] uppercase font-bold font-mono text-slate-400 block mb-1">Технологическая инструкция / Примечание повара</label>
                              <textarea
                                value={selectedVersion.instructions}
                                onChange={(e) => handleUpdateVersionField('instructions', e.target.value)}
                                rows={2}
                                className="w-full p-2.5 bg-slate-50 border border-slate-200 rounded-xl text-xs"
                              />
                            </div>
                          </div>
                        ) : null}
                      </div>
                    ) : (
                      <div className="py-24 text-center border border-dashed rounded-3xl border-slate-200 flex flex-col items-center justify-center gap-3">
                        <Award className="w-10 h-10 text-slate-300" />
                        <h4 className="font-bold text-slate-700 font-sans">Спецификация техкарт отсутствует</h4>
                        <p className="text-xs text-slate-400 max-w-sm mx-auto leading-relaxed">
                          Для блюда еще не составлено ни одной активной рецептуры. Пожалуйста, инициализируйте новую ведомость.
                        </p>
                        <button
                          type="button"
                          onClick={handleAddVersion}
                          className="bg-blue-600 hover:bg-blue-700 text-white text-xs font-bold px-4 py-2 rounded-xl transition"
                        >
                          Инициализировать карту
                        </button>
                      </div>
                    )}
                  </div>
                )}
              </form>
            </div>

            {/* Modal Actions Footer */}
            <div className="px-6 py-4.5 bg-slate-50 border-t border-slate-100 flex justify-end gap-3 shrink-0">
              <button
                type="button"
                onClick={() => setIsFormOpen(false)}
                className="px-5 py-2.5 text-xs font-bold text-slate-600 hover:text-slate-800 border bg-white rounded-xl select-none cursor-pointer"
              >
                Отмена
              </button>
              <button
                type="submit"
                form="rms-menu-form"
                className="px-5 py-2.5 text-xs font-black text-white bg-blue-600 border border-blue-700 hover:bg-blue-700 rounded-xl transition-all"
              >
                {editingItem ? 'Сохранить изменения' : 'Создать позицию'}
              </button>
            </div>

          </div>
        </div>
      )}
    </div>
  );
}
