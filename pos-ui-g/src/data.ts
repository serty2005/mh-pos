import { Hall, MenuItem, Table } from './types';

export const mockHalls: Hall[] = [
  { id: 'hall-main', name: 'Основной зал' },
  { id: 'hall-terrace', name: 'Терраса' },
  { id: 'hall-bar', name: 'Барная зона' }
];

export const mockTables: Table[] = [
  // Основной зал
  { id: 'table-1', number: 1, hallId: 'hall-main', status: 'free', guestsCount: 0 },
  { id: 'table-2', number: 2, hallId: 'hall-main', status: 'free', guestsCount: 0 },
  { id: 'table-3', number: 3, hallId: 'hall-main', status: 'free', guestsCount: 0 },
  { id: 'table-4', number: 4, hallId: 'hall-main', status: 'free', guestsCount: 0 },
  { id: 'table-5', number: 5, hallId: 'hall-main', status: 'free', guestsCount: 0 },
  { id: 'table-6', number: 6, hallId: 'hall-main', status: 'free', guestsCount: 0 },

  // Терраса
  { id: 'table-10', number: 10, hallId: 'hall-terrace', status: 'free', guestsCount: 0 },
  { id: 'table-11', number: 11, hallId: 'hall-terrace', status: 'free', guestsCount: 0 },
  { id: 'table-12', number: 12, hallId: 'hall-terrace', status: 'free', guestsCount: 0 },
  { id: 'table-13', number: 13, hallId: 'hall-terrace', status: 'free', guestsCount: 0 },

  // Бар
  { id: 'table-20', number: 20, hallId: 'hall-bar', status: 'free', guestsCount: 0 },
  { id: 'table-21', number: 21, hallId: 'hall-bar', status: 'free', guestsCount: 0 },
  { id: 'table-22', number: 22, hallId: 'hall-bar', status: 'free', guestsCount: 0 }
];

export const mockMenuItems: MenuItem[] = [
  // Закуски (Starters)
  {
    id: 'starter-olivier',
    name: 'Салат Оливье с языком',
    price: 340,
    category: 'starters',
    isAvailable: true
  },
  {
    id: 'starter-borsch',
    name: 'Борщ домашний со смальцем',
    price: 380,
    category: 'starters',
    isAvailable: true
  },
  {
    id: 'starter-pelmeni',
    name: 'Пельмени ручной лепки',
    price: 420,
    category: 'starters',
    isAvailable: true
  },
  {
    id: 'starter-croutons',
    name: 'Гренки чесночные из бородинского',
    price: 240,
    category: 'starters',
    isAvailable: true
  },

  // Горячее (Mains)
  {
    id: 'main-ribeye',
    name: 'Рибай стейк из мраморного мяса',
    price: 1450,
    category: 'mains',
    isAvailable: true,
    modifierGroups: [
      {
        id: 'mod-doneness',
        name: 'Прожарка стейка',
        minRequired: 1,
        maxAllowed: 1,
        options: [
          { id: 'opt-rare', name: 'Rare (С кровью)', price: 0 },
          { id: 'opt-medium', name: 'Medium (Средняя)', price: 0 },
          { id: 'opt-well', name: 'Well Done (Полная)', price: 0 }
        ]
      }
    ]
  },
  {
    id: 'main-salmon',
    name: 'Филе лосося на гриле',
    price: 1250,
    category: 'mains',
    isAvailable: true,
    modifierGroups: [
      {
        id: 'mod-sauce',
        name: 'Соус к рыбе',
        minRequired: 1,
        maxAllowed: 1,
        options: [
          { id: 'opt-tartar', name: 'Сливочный Тартар', price: 0 },
          { id: 'opt-pesto', name: 'Песто', price: 80 },
          { id: 'opt-lemon', name: 'Лимонный сок', price: 0 }
        ]
      }
    ]
  },
  {
    id: 'main-kiev',
    name: 'Котлета по-Киевски с пюре',
    price: 490,
    category: 'mains',
    isAvailable: true
  },
  {
    id: 'main-carbonara',
    name: 'Каноничная паста Карбонара',
    price: 520,
    category: 'mains',
    isAvailable: true,
    modifierGroups: [
      {
        id: 'mod-extra-carb',
        name: 'Дополнительные топпинги',
        minRequired: 0,
        maxAllowed: 3,
        options: [
          { id: 'opt-bacon', name: 'Двойной бекон', price: 120 },
          { id: 'opt-cheese', name: 'Пармезан', price: 80 },
          { id: 'opt-mushrooms', name: 'Шампиньоны', price: 60 }
        ]
      }
    ]
  },

  // Десерты (Desserts)
  {
    id: 'dessert-napoleon',
    name: 'Пирожное Наполеон',
    price: 320,
    category: 'desserts',
    isAvailable: true
  },
  {
    id: 'dessert-cheesecake',
    name: 'Чизкейк Сан-Себастьян',
    price: 380,
    category: 'desserts',
    isAvailable: true
  },
  {
    id: 'dessert-honey',
    name: 'Медовик с кедровыми орехами',
    price: 290,
    category: 'desserts',
    isAvailable: true
  },

  // Напитки (Drinks)
  {
    id: 'drink-cappuccino',
    name: 'Капучино на выбор',
    price: 180,
    category: 'drinks',
    isAvailable: true,
    modifierGroups: [
      {
        id: 'mod-milk',
        name: 'Тип молока',
        minRequired: 1,
        maxAllowed: 1,
        options: [
          { id: 'opt-milk-std', name: 'Коровье классическое', price: 0 },
          { id: 'opt-milk-coconut', name: 'Кокосовое молоко', price: 70 },
          { id: 'opt-milk-soy', name: 'Соевое молоко', price: 50 },
          { id: 'opt-milk-almond', name: 'Миндальное молоко', price: 80 }
        ]
      }
    ]
  },
  {
    id: 'drink-espresso',
    name: 'Эспрессо',
    price: 130,
    category: 'drinks',
    isAvailable: true
  },
  {
    id: 'drink-morse',
    name: 'Ягодный морс домашний',
    price: 140,
    category: 'drinks',
    isAvailable: true
  },
  {
    id: 'drink-beer',
    name: 'Пиво крафтовое IPA',
    price: 360,
    category: 'drinks',
    isAvailable: true
  },
  {
    id: 'drink-stoplisted-juice',
    name: 'Апельсиновый фреш (Стоп)',
    price: 280,
    category: 'drinks',
    isAvailable: false // This item is stop-listed, to test availability validation
  }
];
