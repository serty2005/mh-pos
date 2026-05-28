export const ru = {
  app: {
    title: 'MyHoReCa Cloud Manager',
    subtitle: 'React/Vite shell для поэтапной миграции на production UI',
    environment: 'Окружение',
    apiBase: 'API base',
    status: 'Статус подключения',
  },
  status: {
    loading: 'Проверка доступности маршрута',
    ready: 'Маршрут доступен',
    blocked: 'Маршрут недоступен',
  },
  readiness: {
    title: 'Готовность backend маршрутов',
    description: 'Экран не имитирует CRUD и показывает только подтвержденное состояние API.',
    route: 'Проверяемый route',
    lastCheck: 'Последняя проверка',
    retry: 'Повторить проверку',
    emptyTitle: 'Функциональные разделы подключаются по мере миграции',
    emptyBody: 'Данные и формы будут включаться только после подтверждения route, schema и error contract.',
  },
  sections: {
    analytics: 'Аналитика',
    menu: 'Меню',
    staff: 'Персонал',
    sync: 'Синхронизация',
    blocked: 'Ожидает подтвержденных backend контрактов',
  },
  errors: {
    unavailable: 'Сервис временно недоступен. Повторите проверку позже.',
    invalidResponse: 'Получен неожиданный формат ответа от сервиса.',
  },
};
