export const ru = {
  app: {
    title: 'MyHoReCa Cloud Manager',
    subtitle: 'React foundation для production Cloud UI',
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
    description: 'Экран показывает только route-backed состояние без имитации CRUD.',
    route: 'Проверяемый route',
    lastCheck: 'Последняя проверка',
    retry: 'Повторить проверку',
    emptyTitle: 'Функциональные разделы подключаются по мере миграции',
    emptyBody: 'Данные и формы включаются после подтверждения route, schema и error contract.',
  },
  sections: {
    analytics: 'Аналитика',
    menu: 'Меню',
    staff: 'Персонал',
    sync: 'Синхронизация',
    blocked: 'Ожидает подтвержденных backend контрактов',
  },
  ui: {
    loading: 'Загрузка данных...',
    retry: 'Повторить',
    noDataTitle: 'Пока нет данных',
    noDataBody: 'Данные появятся после подключения и успешного ответа API.',
  },
  errors: {
    unknown: 'Произошла непредвиденная ошибка.',
    validation: 'Данные запроса не прошли валидацию.',
    notFound: 'Запрошенный ресурс не найден.',
    conflict: 'Конфликт состояния. Обновите данные и повторите попытку.',
    server: 'Сервис временно недоступен. Повторите позже.',
    detailRedacted: '[скрыто из соображений безопасности]',
    network: {
      unavailable: 'Сетевое подключение недоступно.',
      timeout: 'Превышено время ожидания ответа сервера.',
    },
    response: {
      invalid: 'Получен неожиданный формат ответа от сервиса.',
    },
  },
};
