export type CloudRouteId =
  | 'dashboard'
  | 'restaurants'
  | 'edge-sync'
  | 'catalog'
  | 'menu'
  | 'modifiers'
  | 'pricing-taxes'
  | 'staff-permissions'
  | 'floor'
  | 'publications'
  | 'inventory'
  | 'reports';

export type CloudRouteScope = 'global' | 'restaurant';

export type CloudRoute = {
  id: CloudRouteId;
  scope: CloudRouteScope;
};

export const defaultRoute: CloudRoute = {
  id: 'dashboard',
  scope: 'global',
};
