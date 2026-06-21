export type CloudRouteId =
  | 'dashboard'
  | 'catalog'
  | 'staff-permissions'
  | 'licenses'
  | 'restaurants'
  | 'edge-sync'
  | 'menu'
  | 'modifiers'
  | 'pricing-taxes'
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
