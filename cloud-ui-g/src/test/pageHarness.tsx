import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import type { ReactElement } from 'react';
import { expect, vi } from 'vitest';
import { I18nProvider } from '../shared/i18n/I18nProvider';
import { ApiError } from '../shared/api/errors';
import type {
  CatalogFolder,
  CatalogItem,
  Employee,
  EdgeEvent,
  Hall,
  MasterDataPackage,
  MenuItem,
  PricingPolicy,
  PublicationSummary,
  Restaurant,
  RestaurantEdgeNode,
  RestaurantTable,
  Role,
  UnassignedEdgeNode,
} from '../shared/api/schemas';

export const restaurantId = 'restaurant-alpha';

export function restaurant(overrides: Partial<Restaurant> = {}): Restaurant {
  return {
    id: restaurantId,
    name: 'Alpha',
    timezone: 'Europe/Moscow',
    currency: 'RUB',
    business_day_mode: 'standard',
    business_day_boundary_local_time: '05:00',
    status: 'active',
    cloud_version: 1,
    created_at: '2026-06-01T10:00:00Z',
    updated_at: '2026-06-01T10:00:00Z',
    ...overrides,
  };
}
export const nodeDeviceId = 'edge-node-1';
(globalThis as typeof globalThis & { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true;
export const rawMarkers = [
  'raw sql marker',
  'stack trace marker',
  'secret token marker',
  'raw payload marker',
  'pin hash marker',
] as const;

type ApiResponder = (request: ApiRequest) => unknown | Promise<unknown>;

type ApiRoute = {
  method?: string;
  path: string | RegExp;
  responder: ApiResponder;
};

export type ApiRequest = {
  method: string;
  path: string;
  body: unknown;
  init: RequestInit;
};

export type RenderedPage = {
  container: HTMLDivElement;
  root: Root;
  cleanup: () => Promise<void>;
};

export function safeApiError(details: Record<string, string> = {}) {
  return new ApiError({
    status: 500,
    code: 'INTERNAL_ERROR',
    messageKey: 'errors.server',
    category: 'server',
    details: {
      sql: rawMarkers[0],
      stack: rawMarkers[1],
      token: rawMarkers[2],
      payload: rawMarkers[3],
      pin_hash: rawMarkers[4],
      ...details,
    },
    correlationId: 'corr-safe',
    retryable: true,
  });
}

export function deferred<T>() {
  let resolve!: (value: T | PromiseLike<T>) => void;
  let reject!: (reason?: unknown) => void;
  const promise = new Promise<T>((nextResolve, nextReject) => {
    resolve = nextResolve;
    reject = nextReject;
  });
  return { promise, resolve, reject };
}

export async function renderPage(element: ReactElement): Promise<RenderedPage> {
  const container = document.createElement('div');
  document.body.appendChild(container);
  const root = createRoot(container);

  await act(async () => {
    root.render(<I18nProvider>{element}</I18nProvider>);
  });

  return {
    container,
    root,
    cleanup: async () => {
      await act(async () => {
        root.unmount();
      });
      container.remove();
    },
  };
}

export async function waitFor(assertion: () => void | Promise<void>) {
  let lastError: unknown;
  for (let attempt = 0; attempt < 60; attempt += 1) {
    try {
      await assertion();
      return;
    } catch (error) {
      lastError = error;
      await act(async () => {
        await new Promise((resolve) => setTimeout(resolve, 0));
      });
    }
  }
  throw lastError;
}

export async function click(element: Element) {
  if (element instanceof HTMLButtonElement && element.disabled) return;
  await act(async () => {
    if (element instanceof HTMLElement) {
      element.click();
    } else {
      element.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
    }
  });
}

export async function change(element: Element, value: string) {
  await act(async () => {
    if (element instanceof HTMLInputElement) {
      Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value')?.set?.call(element, value);
    } else if (element instanceof HTMLTextAreaElement) {
      Object.getOwnPropertyDescriptor(HTMLTextAreaElement.prototype, 'value')?.set?.call(element, value);
    } else if (element instanceof HTMLSelectElement) {
      Object.getOwnPropertyDescriptor(HTMLSelectElement.prototype, 'value')?.set?.call(element, value);
    }
    element.dispatchEvent(new Event('input', { bubbles: true }));
    element.dispatchEvent(new Event('change', { bubbles: true }));
  });
}

export function text(container: ParentNode) {
  return container.textContent ?? '';
}

export function expectNoRawMarkers(container: ParentNode) {
  const rendered = text(container);
  rawMarkers.forEach((marker) => {
    expect(rendered).not.toContain(marker);
  });
}

export function buttonByText(container: ParentNode, label: string) {
  const button = Array.from(container.querySelectorAll('button')).find((item) => item.textContent?.includes(label));
  if (!button) throw new Error(`Button not found: ${label}`);
  return button;
}

export function buttonByAriaLabel(container: ParentNode, label: string) {
  const button = container.querySelector(`button[aria-label="${cssEscape(label)}"]`);
  if (!button) throw new Error(`Button not found by aria-label: ${label}`);
  return button;
}

export function inputByPlaceholder(container: ParentNode, placeholder: string) {
  const input = container.querySelector(`input[placeholder="${cssEscape(placeholder)}"]`);
  if (!input) throw new Error(`Input not found: ${placeholder}`);
  return input;
}

export function installFetchMock(routes: ApiRoute[]) {
  const calls: ApiRequest[] = [];
  const fetchMock = vi.fn(async (input: RequestInfo | URL, init: RequestInit = {}) => {
    const url = new URL(String(input));
    const method = (init.method ?? 'GET').toUpperCase();
    const normalizedPath = `${url.pathname}${url.search}`.replace(/^\/api\/v1(?=\/)/, '');
    const request: ApiRequest = {
      method,
      path: normalizedPath,
      body: init.body ? JSON.parse(String(init.body)) : null,
      init,
    };
    calls.push(request);

    const route = routes.find((candidate) => {
      const methodMatches = !candidate.method || candidate.method.toUpperCase() === method;
      const pathMatches = typeof candidate.path === 'string'
        ? candidate.path === request.path
        : candidate.path.test(request.path);
      return methodMatches && pathMatches;
    });

    if (!route) {
      return jsonResponse({ error: { message_key: 'errors.notFound' } }, 404);
    }

    try {
      const data = await route.responder(request);
      return jsonResponse(data, 200);
    } catch (error) {
      if (error instanceof ApiError) {
        return jsonResponse({
          error: {
            code: error.code,
            message_key: error.messageKey,
            details: error.details,
            correlation_id: error.correlationId,
          },
        }, error.status);
      }
      throw error;
    }
  });

  vi.stubGlobal('fetch', fetchMock);
  return {
    calls,
    fetchMock,
    callsFor: (path: string | RegExp, method?: string) => calls.filter((call) => {
      const pathMatches = typeof path === 'string' ? call.path === path : path.test(call.path);
      const methodMatches = !method || call.method === method.toUpperCase();
      return pathMatches && methodMatches;
    }),
  };
}

export function role(overrides: Partial<Role> = {}): Role {
  return {
    id: 'role-manager',
    name: 'Manager',
    permissions_json: '{"pos.catalog.view":true,"pos.sync.view":true}',
    active: true,
    cloud_version: 1,
    created_at: '2026-06-01T10:00:00Z',
    updated_at: '2026-06-01T10:00:00Z',
    ...overrides,
  };
}

export function employee(overrides: Partial<Employee> = {}): Employee {
  return {
    id: 'employee-1',
    role_id: 'role-manager',
    restaurant_ids: [restaurantId],
    all_restaurants: false,
    name: 'Alice Manager',
    status: 'active',
    pin_configured: true,
    pin_credential_version: 2,
    permission_snapshot_json: '{"pos.catalog.view":true,"pos.sync.view":true}',
    cloud_version: 1,
    created_at: '2026-06-01T10:00:00Z',
    updated_at: '2026-06-01T10:00:00Z',
    ...overrides,
  };
}

export function catalogFolder(overrides: Partial<CatalogFolder> = {}): CatalogFolder {
  return {
    id: 'folder-main',
    restaurant_id: restaurantId,
    parent_id: '',
    name: 'Kitchen',
    sort_order: 10,
    status: 'published',
    cloud_version: 1,
    created_at: '2026-06-01T10:00:00Z',
    updated_at: '2026-06-01T10:00:00Z',
    ...overrides,
  };
}

export function catalogItem(overrides: Partial<CatalogItem> = {}): CatalogItem {
  return {
    id: 'catalog-tea',
    restaurant_id: restaurantId,
    kind: 'dish',
    folder_id: 'folder-main',
    name: 'Tea',
    sku: 'TEA',
    base_unit: 'portion',
    kitchen_type: '',
    accounting_category: '',
    status: 'published',
    cloud_version: 1,
    created_at: '2026-06-01T10:00:00Z',
    updated_at: '2026-06-01T10:00:00Z',
    ...overrides,
  };
}

export function menuItem(overrides: Partial<MenuItem> = {}): MenuItem {
  return {
    id: 'menu-tea',
    restaurant_id: restaurantId,
    catalog_item_id: 'catalog-tea',
    category_id: '',
    name: 'Tea service',
    price: 190,
    currency: 'RUB',
    status: 'published',
    availability_json: '{}',
    station_routing_key: 'bar',
    cloud_version: 1,
    created_at: '2026-06-01T10:00:00Z',
    updated_at: '2026-06-01T10:00:00Z',
    ...overrides,
  };
}

export function pricingPolicy(overrides: Partial<PricingPolicy> = {}): PricingPolicy {
  return {
    id: 'policy-service',
    restaurant_id: restaurantId,
    name: 'Service charge',
    kind: 'surcharge',
    scope: 'order',
    amount_kind: 'percentage',
    amount_minor: 0,
    value_basis_points: 1000,
    application_index: 20,
    manual: true,
    requires_permission: 'pricing.surcharge.apply',
    status: 'published',
    cloud_version: 1,
    created_at: '2026-06-01T10:00:00Z',
    updated_at: '2026-06-01T10:00:00Z',
    ...overrides,
  };
}

export function hall(overrides: Partial<Hall> = {}): Hall {
  return {
    id: 'hall-main',
    restaurant_id: restaurantId,
    name: 'Main hall',
    status: 'published',
    cloud_version: 1,
    created_at: '2026-06-01T10:00:00Z',
    updated_at: '2026-06-01T10:00:00Z',
    ...overrides,
  };
}

export function table(overrides: Partial<RestaurantTable> = {}): RestaurantTable {
  return {
    id: 'table-1',
    restaurant_id: restaurantId,
    hall_id: 'hall-main',
    name: 'T1',
    seats: 4,
    status: 'published',
    cloud_version: 1,
    created_at: '2026-06-01T10:00:00Z',
    updated_at: '2026-06-01T10:00:00Z',
    ...overrides,
  };
}

export function unassignedDevice(overrides: Partial<UnassignedEdgeNode> = {}): UnassignedEdgeNode {
  return {
    id: 'device-registration-1',
    node_device_id: nodeDeviceId,
    claimed_cloud_url: 'https://cloud.example.test',
    display_name: 'Front POS',
    app_version: '1.0.0',
    status: 'pending',
    first_seen_at: '2026-06-01T10:00:00Z',
    last_seen_at: '2026-06-01T10:00:00Z',
    assigned_restaurant_id: '',
    created_at: '2026-06-01T10:00:00Z',
    updated_at: '2026-06-01T10:00:00Z',
    ...overrides,
  };
}

export function restaurantDevice(overrides: Partial<RestaurantEdgeNode> = {}): RestaurantEdgeNode {
  return {
    id: 'restaurant-device-1',
    restaurant_id: restaurantId,
    node_device_id: nodeDeviceId,
    display_name: 'Front POS',
    status: 'assigned',
    last_seen_at: '2026-06-01T10:00:00Z',
    assigned_at: '2026-06-01T10:00:00Z',
    created_at: '2026-06-01T10:00:00Z',
    updated_at: '2026-06-01T10:00:00Z',
    ...overrides,
  };
}

export function edgeEvent(overrides: Partial<EdgeEvent> = {}): EdgeEvent {
  return {
    cloud_receipt_id: 'receipt-1',
    idempotency_key: 'idem-1',
    restaurant_id: restaurantId,
    device_id: nodeDeviceId,
    command_id: 'command-1',
    event_id: 'event-1',
    edge_event_id: 'edge-event-1',
    event_type: 'order.closed',
    aggregate_type: 'order',
    aggregate_id: 'order-1',
    envelope_version: '1',
    occurred_at: '2026-06-01T10:00:00Z',
    cloud_received_at: '2026-06-01T10:00:01Z',
    raw_payload_sha256_hex: 'hash-only-no-payload',
    ...overrides,
  };
}

export function masterDataPackage(overrides: Partial<MasterDataPackage> = {}): MasterDataPackage {
  return {
    stream_name: 'catalog',
    node_device_id: nodeDeviceId,
    restaurant_id: restaurantId,
    sync_mode: 'snapshot',
    full_snapshot_reason: '',
    cloud_version: 3,
    checkpoint_token: 'checkpoint-safe',
    cloud_updated_at: '2026-06-01T10:00:00Z',
    payload_json: { items: [{ id: 'catalog-tea' }] },
    created_at: '2026-06-01T10:00:00Z',
    updated_at: '2026-06-01T10:00:00Z',
    ...overrides,
  };
}

export function publication(overrides: Partial<PublicationSummary> = {}): PublicationSummary {
  return {
    id: 'publication-1',
    restaurant_id: restaurantId,
    version: 7,
    status: 'published',
    cloud_version: 7,
    published_at: '2026-06-01T10:00:00Z',
    published_by: 'cloud-ui-test',
    package_sha256: 'sha-safe',
    counts: { catalog: 1, menu: 1 },
    ...overrides,
  };
}

function jsonResponse(data: unknown, status: number) {
  return new Response(JSON.stringify(data), {
    status,
    headers: { 'Content-Type': 'application/json', 'X-Request-ID': 'req-test' },
  });
}

function cssEscape(value: string) {
  if (globalThis.CSS?.escape) return globalThis.CSS.escape(value);
  return value.replace(/["\\]/g, '\\$&');
}
