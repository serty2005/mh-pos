import type { CloudRoute, CloudRouteId } from './routes';

export type NavigationItem = {
  route: CloudRoute;
  labelKey: string;
};

export const navigationItems: NavigationItem[] = [
  { route: { id: 'dashboard', scope: 'global' }, labelKey: 'nav.dashboard' },
  { route: { id: 'restaurants', scope: 'global' }, labelKey: 'nav.restaurants' },
  { route: { id: 'edge-sync', scope: 'restaurant' }, labelKey: 'nav.edgeSync' },
  { route: { id: 'catalog', scope: 'restaurant' }, labelKey: 'nav.catalog' },
  { route: { id: 'menu', scope: 'restaurant' }, labelKey: 'nav.menu' },
  { route: { id: 'modifiers', scope: 'restaurant' }, labelKey: 'nav.modifiers' },
  { route: { id: 'pricing-taxes', scope: 'restaurant' }, labelKey: 'nav.pricingTaxes' },
  { route: { id: 'staff-permissions', scope: 'restaurant' }, labelKey: 'nav.staffPermissions' },
  { route: { id: 'floor', scope: 'restaurant' }, labelKey: 'nav.floor' },
  { route: { id: 'publications', scope: 'restaurant' }, labelKey: 'nav.publications' },
  { route: { id: 'inventory', scope: 'restaurant' }, labelKey: 'nav.inventory' },
  { route: { id: 'reports', scope: 'restaurant' }, labelKey: 'nav.reports' },
];

export const navigationById = new Map<CloudRouteId, NavigationItem>(
  navigationItems.map((item) => [item.route.id, item]),
);
