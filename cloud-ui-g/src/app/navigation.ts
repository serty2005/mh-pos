import type { CloudRoute, CloudRouteId } from './routes';
import { hasEntitlement } from '../shared/licenseModules';

export type NavigationItem = {
  route: CloudRoute;
  labelKey: string;
};

export const navigationItems: NavigationItem[] = [
  { route: { id: 'dashboard', scope: 'global' }, labelKey: 'nav.dashboard' },
  { route: { id: 'catalog', scope: 'global' }, labelKey: 'nav.catalog' },
  { route: { id: 'staff-permissions', scope: 'global' }, labelKey: 'nav.staffPermissions' },
  { route: { id: 'licenses', scope: 'global' }, labelKey: 'nav.licenses' },
  { route: { id: 'restaurants', scope: 'global' }, labelKey: 'nav.restaurants' },
  { route: { id: 'edge-sync', scope: 'restaurant' }, labelKey: 'nav.edgeSync' },
  { route: { id: 'modifiers', scope: 'restaurant' }, labelKey: 'nav.modifiers' },
  { route: { id: 'pricing-taxes', scope: 'restaurant' }, labelKey: 'nav.pricingTaxes' },
  { route: { id: 'floor', scope: 'restaurant' }, labelKey: 'nav.floor' },
  { route: { id: 'publications', scope: 'restaurant' }, labelKey: 'nav.publications' },
  { route: { id: 'inventory', scope: 'restaurant' }, labelKey: 'nav.inventory' },
  { route: { id: 'reports', scope: 'restaurant' }, labelKey: 'nav.reports' },
  { route: { id: 'receipt-templates', scope: 'global' }, labelKey: 'nav.receiptTemplates' },
  { route: { id: 'printers', scope: 'restaurant' }, labelKey: 'nav.printers' },
];

export const navigationById = new Map<CloudRouteId, NavigationItem>(
  navigationItems.map((item) => [item.route.id, item]),
);

export function navigationForEntitlements(entitlements: Record<string, boolean>) {
  return navigationItems.filter((item) => {
    if (item.route.id === 'licenses') return true;
    if (item.route.id === 'floor') return hasEntitlement(entitlements, 'table-mode');
    if (item.route.id === 'inventory') return hasEntitlement(entitlements, 'warehouse-mode');
    if (item.route.id !== 'dashboard') return hasEntitlement(entitlements, 'cloud-subscription');
    return true;
  });
}
