import { createI18n } from 'vue-i18n';

export const i18n = createI18n({
  legacy: false,
  locale: 'ru',
  fallbackLocale: 'en',
  messages: {
    ru: {
      actions: {
        backToLogin: 'К входу',
        lock: 'Заблокировать',
        login: 'Войти',
        logout: 'Выйти',
        pair: 'Привязать',
        retry: 'Повторить',
      },
      common: {
        client: 'Клиент',
        empty: 'Нет данных',
        error: 'Не удалось загрузить данные',
        loading: 'Загрузка',
        node: 'Node',
      },
      pair: {
        title: 'Enter Pairing Code',
        code: 'Pairing code',
        hint: 'MVP формат: MHPOS:<restaurant_id>:<node_device_id>',
        paired: 'Edge Node paired',
      },
      login: {
        title: 'PIN login',
        pin: 'PIN',
        pinHint: 'Введите PIN сотрудника',
      },
      lock: {
        title: 'Session locked',
        body: 'Backend session revoked. Для продолжения нужен новый PIN login.',
      },
      pos: {
        title: 'POS',
        actor: 'Сотрудник',
        session: 'Session',
        halls: 'Залы',
        tables: 'Столы',
        noHalls: 'Залы еще не созданы',
        noTables: 'В этом зале нет активных столов',
      },
    },
    en: {
      actions: {
        backToLogin: 'Back to login',
        lock: 'Lock',
        login: 'Login',
        logout: 'Logout',
        pair: 'Pair',
        retry: 'Retry',
      },
      common: {
        client: 'Client',
        empty: 'No data',
        error: 'Could not load data',
        loading: 'Loading',
        node: 'Node',
      },
      pair: {
        title: 'Enter Pairing Code',
        code: 'Pairing code',
        hint: 'MVP format: MHPOS:<restaurant_id>:<node_device_id>',
        paired: 'Edge Node paired',
      },
      login: {
        title: 'PIN login',
        pin: 'PIN',
        pinHint: 'Enter employee PIN',
      },
      lock: {
        title: 'Session locked',
        body: 'Backend session revoked. A new PIN login is required.',
      },
      pos: {
        title: 'POS',
        actor: 'Actor',
        session: 'Session',
        halls: 'Halls',
        tables: 'Tables',
        noHalls: 'No halls yet',
        noTables: 'No active tables in this hall',
      },
    },
  },
});
