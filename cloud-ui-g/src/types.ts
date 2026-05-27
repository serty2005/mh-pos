/**
 * @license
 * SPDX-License-Identifier: Apache-2.0
 */

export type Role = 'admin' | 'manager' | 'chef' | 'waiter' | 'cashier';

export interface StaffMember {
  id: string;
  name: string;
  role: Role;
  email: string;
  phone: string;
  status: 'active' | 'inactive' | 'on_shift';
  shiftStart?: string;
  avatarUrl?: string;
  permissions: {
    editMenu: boolean;
    viewAnalytics: boolean;
    manageStaff: boolean;
    posSync: boolean;
  };
}

export interface MenuItem {
  id: string;
  name: string;
  description: string;
  price: number;
  cost: number; // For margin calculation
  category: string;
  isAvailable: boolean;
  stock: number; // -1 for unlimited
  imageUrl?: string;
  emoji: string; // fallback icon/emoji for rich cards
  prepTime: number; // in minutes
}

export interface MenuCategory {
  id: string;
  name: string;
  slug: string;
  icon: string; // lucide icon name
}

export interface POSTerminal {
  id: string;
  name: string;
  ipAddress: string;
  version: string;
  status: 'online' | 'offline' | 'syncing';
  lastSyncTime: string;
  pendingTransactions: number;
  location: string;
}

export interface SyncLog {
  id: string;
  timestamp: string;
  terminalId: string;
  terminalName: string;
  type: 'menu_push' | 'sales_pull' | 'staff_update' | 'heartbeat';
  status: 'success' | 'warning' | 'error';
  details: string;
}

export interface SalesTransaction {
  id: string;
  timestamp: string;
  terminalId: string;
  items: {
    itemId: string;
    itemName: string;
    quantity: number;
    price: number;
    category: string;
  }[];
  totalAmount: number;
  paymentMethod: 'cash' | 'card' | 'mobile_pay';
  waiterId: string;
  tableNumber: string;
}

export interface AnalyticsStats {
  totalRevenue: number;
  previousRevenue: number; // for trend
  orderCount: number;
  previousOrderCount: number;
  averageCheck: number;
  previousAverageCheck: number;
  popularItems: {
    itemId: string;
    name: string;
    count: number;
    revenue: number;
    emoji: string;
  }[];
  categoryBreakdown: {
    category: string;
    revenue: number;
    percentage: number;
  }[];
  hourlySales: {
    hour: string;
    revenue: number;
    orders: number;
  }[];
}
