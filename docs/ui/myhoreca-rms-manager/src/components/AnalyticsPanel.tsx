/**
 * @license
 * SPDX-License-Identifier: Apache-2.0
 */

import { useState, useMemo } from 'react';
import { DollarSign, ShoppingBag, CreditCard, Activity, TrendingUp, TrendingDown, RefreshCw, Trophy } from 'lucide-react';
import { MenuItem, MenuCategory, SalesTransaction } from '../types';

interface AnalyticsPanelProps {
  transactions: SalesTransaction[];
  menuItems: MenuItem[];
  categories: MenuCategory[];
  simulateSale: () => void;
  isSimulating: boolean;
}

export default function AnalyticsPanel({
  transactions,
  menuItems,
  categories,
  simulateSale,
  isSimulating,
}: AnalyticsPanelProps) {
  const [hoveredIndex, setHoveredIndex] = useState<number | null>(null);

  // 1. Calculate major stats
  const stats = useMemo(() => {
    const totalRevenue = transactions.reduce((acc, tx) => acc + tx.totalAmount, 0);
    const orderCount = transactions.length;
    const averageCheck = orderCount > 0 ? totalRevenue / orderCount : 0;

    // Margin estimation
    let totalCost = 0;
    transactions.forEach((tx) => {
      tx.items.forEach((it) => {
        const menuItem = menuItems.find((mi) => mi.id === it.itemId);
        if (menuItem) {
          totalCost += menuItem.cost * it.quantity;
        } else {
          // Fallback margin (approx 30%)
          totalCost += it.price * 0.3 * it.quantity;
        }
      });
    });
    const margin = totalRevenue > 0 ? ((totalRevenue - totalCost) / totalRevenue) * 100 : 0;

    return {
      totalRevenue,
      orderCount,
      averageCheck,
      margin,
    };
  }, [transactions, menuItems]);

  // Previous mock values for realistic trends
  const trends = {
    revenue: 12.4,
    orders: 8.2,
    avgCheck: 3.9,
    margin: 1.5,
  };

  // 2. Aggregate sales hourly dynamics (for line chart)
  const hourlyData = useMemo(() => {
    // Bucket by hours: 12:00, 14:00, 16:00, 18:00, 20:00, 22:00, 00:00
    const timeBuckets = ['12:00', '14:00', '16:00', '18:00', '20:00', '22:00', '24:00'];
    const bucketsRevenue = Array(timeBuckets.length).fill(0);
    const bucketsOrders = Array(timeBuckets.length).fill(0);

    transactions.forEach((tx) => {
      const [hourStr] = tx.timestamp.split(':');
      const hour = parseInt(hourStr || '0', 10);

      let idx = 0;
      if (hour >= 23) idx = 6;
      else if (hour >= 21) idx = 5;
      else if (hour >= 19) idx = 4;
      else if (hour >= 17) idx = 3;
      else if (hour >= 15) idx = 2;
      else if (hour >= 13) idx = 1;
      else idx = 0;

      bucketsRevenue[idx] += tx.totalAmount;
      bucketsOrders[idx] += 1;
    });

    return timeBuckets.map((time, idx) => ({
      hour: time,
      revenue: bucketsRevenue[idx],
      orders: bucketsOrders[idx],
    }));
  }, [transactions]);

  // 3. Category Breakdown (for horizontal bar representation)
  const categoryStats = useMemo(() => {
    const breakdown: Record<string, number> = {};
    let totalRevenue = 0;

    transactions.forEach((tx) => {
      tx.items.forEach((it) => {
        const catSlug = it.category;
        breakdown[catSlug] = (breakdown[catSlug] || 0) + (it.price * it.quantity);
        totalRevenue += it.price * it.quantity;
      });
    });

    return categories.map((cat) => {
      const rev = breakdown[cat.slug] || 0;
      return {
        slug: cat.slug,
        name: cat.name,
        revenue: rev,
        percentage: totalRevenue > 0 ? (rev / totalRevenue) * 100 : 0,
      };
    }).sort((a, b) => b.revenue - a.revenue);
  }, [transactions, categories]);

  // 4. Popular Dishes (top selling items)
  const topDishes = useMemo(() => {
    const itemSales: Record<string, { count: number; revenue: number }> = {};

    transactions.forEach((tx) => {
      tx.items.forEach((it) => {
        if (!itemSales[it.itemId]) {
          itemSales[it.itemId] = { count: 0, revenue: 0 };
        }
        itemSales[it.itemId].count += it.quantity;
        itemSales[it.itemId].revenue += it.price * it.quantity;
      });
    });

    return menuItems.map((item) => {
      const sales = itemSales[item.id] || { count: 0, revenue: 0 };
      return {
        ...item,
        sellCount: sales.count,
        totalRevenue: sales.revenue,
      };
    })
    .filter((it) => it.sellCount > 0)
    .sort((a, b) => b.sellCount - a.sellCount)
    .slice(0, 5);
  }, [transactions, menuItems]);

  // SVG Line Chart Constants & Calculations
  const chartWidth = 600;
  const chartHeight = 260;
  const chartPadding = { top: 20, right: 30, bottom: 40, left: 60 };

  const { linePath, areaPath, points } = useMemo(() => {
    const maxVal = Math.max(...hourlyData.map(d => d.revenue), 2000) * 1.15;
    const minVal = 0;
    const rangeY = maxVal - minVal;

    const pts = hourlyData.map((d, i) => {
      const x = chartPadding.left + (i / (hourlyData.length - 1)) * (chartWidth - chartPadding.left - chartPadding.right);
      const y = chartHeight - chartPadding.bottom - ((d.revenue - minVal) / rangeY) * (chartHeight - chartPadding.top - chartPadding.bottom);
      return { x, y, ...d };
    });

    // Generate smoothed cubic bezier line
    let path = '';
    let area = `M ${chartPadding.left} ${chartHeight - chartPadding.bottom} `;

    if (pts.length > 0) {
      path = `M ${pts[0].x} ${pts[0].y} `;
      area += `L ${pts[0].x} ${pts[0].y} `;

      for (let i = 0; i < pts.length - 1; i++) {
        const cp1x = pts[i].x + (pts[i+1].x - pts[i].x) / 3;
        const cp1y = pts[i].y;
        const cp2x = pts[i].x + 2 * (pts[i+1].x - pts[i].x) / 3;
        const cp2y = pts[i+1].y;

        path += `C ${cp1x} ${cp1y}, ${cp2x} ${cp2y}, ${pts[i+1].x} ${pts[i+1].y} `;
      }

      for (let i = 1; i < pts.length; i++) {
        const cp1x = pts[i-1].x + (pts[i].x - pts[i-1].x) / 3;
        const cp1y = pts[i-1].y;
        const cp2x = pts[i-1].x + 2 * (pts[i].x - pts[i-1].x) / 3;
        const cp2y = pts[i].y;

        area += `C ${cp1x} ${cp1y}, ${cp2x} ${cp2y}, ${pts[i].x} ${pts[i].y} `;
      }

      area += `L ${pts[pts.length - 1].x} ${chartHeight - chartPadding.bottom} Z`;
    }

    return { linePath: path, areaPath: area, points: pts };
  }, [hourlyData]);

  // Payment methods calculation
  const payments = useMemo(() => {
    let card = 0, cash = 0, mobile = 0;
    transactions.forEach((tx) => {
      if (tx.paymentMethod === 'card') card += tx.totalAmount;
      else if (tx.paymentMethod === 'cash') cash += tx.totalAmount;
      else if (tx.paymentMethod === 'mobile_pay') mobile += tx.totalAmount;
    });
    const total = card + cash + mobile || 1;
    return [
      { type: 'Карта', amount: card, pct: (card / total) * 100, color: 'bg-indigo-500' },
      { type: 'Наличные', amount: cash, pct: (cash / total) * 100, color: 'bg-emerald-500' },
      { type: 'СБП / Свежий Pay', amount: mobile, pct: (mobile / total) * 100, color: 'bg-purple-500' },
    ];
  }, [transactions]);

  return (
    <div className="flex-1 overflow-y-auto bg-slate-50/50 p-8">
      {/* Header */}
      <div className="flex flex-col sm:flex-row justify-between sm:items-center gap-4 mb-8">
        <div>
          <h2 className="text-2xl font-bold tracking-tight text-slate-900 font-sans">
            Аналитика в реальном времени
          </h2>
          <p className="text-sm text-slate-500 mt-1">
            Контролируйте финансовые показатели и активность подключенных POS-терминалов.
          </p>
        </div>

        {/* Quick Simulation Trigger */}
        <button
          onClick={simulateSale}
          disabled={isSimulating}
          className={`flex items-center gap-2 px-5 py-3 rounded-xl text-sm font-semibold transition-all duration-300 shadow-sm border ${
            isSimulating
              ? 'bg-slate-200 border-slate-300 text-slate-400 cursor-not-allowed'
              : 'bg-blue-600 hover:bg-blue-700 text-white border-blue-700 hover:shadow shadow-blue-200'
          }`}
        >
          <RefreshCw className={`w-4 h-4 ${isSimulating ? 'animate-spin' : ''}`} />
          {isSimulating ? 'Синхронизация...' : 'Имитировать чек с POS'}
        </button>
      </div>

      {/* KPI Stats Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
        {/* KPI 1: Revenue */}
        <div className="bg-white p-6 rounded-2xl border border-slate-200/80 shadow-sm hover:shadow-md transition-shadow">
          <div className="flex justify-between items-start">
            <div className="p-3 bg-blue-50 rounded-xl text-blue-600">
              <DollarSign className="w-5 h-5" />
            </div>
            <div className="flex items-center gap-1 font-mono text-xs font-semibold text-emerald-600 bg-emerald-50 px-2 py-1 rounded-lg">
              <TrendingUp className="w-3.5 h-3.5" />
              <span>+{trends.revenue}%</span>
            </div>
          </div>
          <div className="mt-4">
            <span className="text-xs uppercase font-semibold font-mono tracking-wider text-slate-400">
              Выручка сегодня
            </span>
            <h3 className="text-2xl font-bold text-slate-900 mt-1 font-sans">
              {stats.totalRevenue.toLocaleString('ru-RU')} <span className="text-lg font-medium text-slate-500 font-sans">₽</span>
            </h3>
          </div>
        </div>

        {/* KPI 2: Order count */}
        <div className="bg-white p-6 rounded-2xl border border-slate-200/80 shadow-sm hover:shadow-md transition-shadow">
          <div className="flex justify-between items-start">
            <div className="p-3 bg-emerald-50 rounded-xl text-emerald-600">
              <ShoppingBag className="w-5 h-5" />
            </div>
            <div className="flex items-center gap-1 font-mono text-xs font-semibold text-emerald-600 bg-emerald-50 px-2 py-1 rounded-lg">
              <TrendingUp className="w-3.5 h-3.5" />
              <span>+{trends.orders}%</span>
            </div>
          </div>
          <div className="mt-4">
            <span className="text-xs uppercase font-semibold font-mono tracking-wider text-slate-400">
              Количество чеков
            </span>
            <h3 className="text-2xl font-bold text-slate-900 mt-1 font-sans">
              {stats.orderCount} <span className="text-base font-normal text-slate-500">шт</span>
            </h3>
          </div>
        </div>

        {/* KPI 3: Average check */}
        <div className="bg-white p-6 rounded-2xl border border-slate-200/80 shadow-sm hover:shadow-md transition-shadow">
          <div className="flex justify-between items-start">
            <div className="p-3 bg-purple-50 rounded-xl text-purple-600">
              <CreditCard className="w-5 h-5" />
            </div>
            <div className="flex items-center gap-1 font-mono text-xs font-semibold text-rose-600 bg-rose-50 px-2 py-1 rounded-lg">
              <TrendingDown className="w-3.5 h-3.5" />
              <span>-1.2%</span>
            </div>
          </div>
          <div className="mt-4">
            <span className="text-xs uppercase font-semibold font-mono tracking-wider text-slate-400">
              Средний чек
            </span>
            <h3 className="text-2xl font-bold text-slate-900 mt-1 font-sans">
              {Math.round(stats.averageCheck).toLocaleString('ru-RU')} <span className="text-lg font-medium text-slate-500 font-sans">₽</span>
            </h3>
          </div>
        </div>

        {/* KPI 4: Margin */}
        <div className="bg-white p-6 rounded-2xl border border-slate-200/80 shadow-sm hover:shadow-md transition-shadow">
          <div className="flex justify-between items-start">
            <div className="p-3 bg-amber-50 rounded-xl text-amber-600">
              <Activity className="w-5 h-5" />
            </div>
            <div className="flex items-center gap-1 font-mono text-xs font-semibold text-emerald-600 bg-emerald-50 px-2 py-1 rounded-lg">
              <TrendingUp className="w-3.5 h-3.5" />
              <span>+{trends.margin}%</span>
            </div>
          </div>
          <div className="mt-4">
            <span className="text-xs uppercase font-semibold font-mono tracking-wider text-slate-400">
              Фуд кост маржа (ср)
            </span>
            <h3 className="text-2xl font-bold text-slate-900 mt-1 font-sans">
              {stats.margin.toFixed(1)} <span className="text-lg font-medium text-slate-500">%</span>
            </h3>
          </div>
        </div>
      </div>

      {/* Charts Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8 mb-8">
        {/* Main Line graph: Dynamic Sales */}
        <div className="lg:col-span-2 bg-white p-6 rounded-2xl border border-slate-200/80 shadow-sm flex flex-col">
          <div className="flex justify-between items-center mb-6">
            <div>
              <h3 className="font-sans font-bold text-base text-slate-900">
                Динамика выручки по часам
              </h3>
              <p className="text-xs text-slate-400 mt-0.5">Показатели продаж с шагом в 2 часа</p>
            </div>
            <div className="flex items-center gap-4 text-xs font-mono font-semibold">
              <div className="flex items-center gap-1.5 text-slate-500">
                <span className="w-2.5 h-2.5 rounded-full bg-blue-500"></span>
                <span>Выручка (₽)</span>
              </div>
            </div>
          </div>

          {/* Interactive SVG chart */}
          <div className="relative flex-1 min-h-[260px] flex items-center justify-center">
            <svg
              viewBox={`0 0 ${chartWidth} ${chartHeight}`}
              className="w-full h-full overflow-visible"
            >
              <defs>
                <linearGradient id="chartGradient" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor="#3b82f6" stopOpacity="0.18" />
                  <stop offset="100%" stopColor="#3b82f6" stopOpacity="0.0" />
                </linearGradient>
              </defs>

              {/* Horizontal Grid lines */}
              {[0, 0.25, 0.5, 0.75, 1].map((ratio, i) => {
                const y = chartPadding.top + ratio * (chartHeight - chartPadding.top - chartPadding.bottom);
                const maxVal = Math.max(...hourlyData.map(d => d.revenue), 2000) * 1.15;
                const value = Math.round(maxVal * (1 - ratio));
                return (
                  <g key={i}>
                    <line
                      x1={chartPadding.left}
                      y1={y}
                      x2={chartWidth - chartPadding.right}
                      y2={y}
                      stroke="#f1f5f9"
                      strokeWidth="1.5"
                    />
                    <text
                      x={chartPadding.left - 10}
                      y={y + 4}
                      className="font-mono text-[10px] text-slate-400 text-right font-medium"
                      textAnchor="end"
                    >
                      {value >= 1000 ? `${(value / 1000).toFixed(1)}k` : value} ₽
                    </text>
                  </g>
                );
              })}

              {/* Gradient Area under line */}
              {areaPath && (
                <path
                  d={areaPath}
                  fill="url(#chartGradient)"
                />
              )}

              {/* Line path */}
              {linePath && (
                <path
                  d={linePath.replace(/#4f46e5/g, '#3b82f6')}
                  fill="none"
                  stroke="#3b82f6"
                  strokeWidth="3.5"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                />
              )}

              {/* Interaction Circles & Tooltips */}
              {points.map((pt, i) => {
                const isHovered = hoveredIndex === i;
                return (
                  <g key={i}>
                    <circle
                      cx={pt.x}
                      cy={pt.y}
                      r={isHovered ? 7 : 4}
                      fill={isHovered ? '#2563eb' : '#ffffff'}
                      stroke="#2563eb"
                      strokeWidth={isHovered ? 3 : 2}
                      className="transition-all duration-200 cursor-pointer"
                      onMouseEnter={() => setHoveredIndex(i)}
                      onMouseLeave={() => setHoveredIndex(null)}
                    />
                    
                    {/* Tiny X axis Labels */}
                    <text
                      x={pt.x}
                      y={chartHeight - 12}
                      className="font-mono text-[10px] text-slate-400 text-center font-medium"
                      textAnchor="middle"
                    >
                      {pt.hour}
                    </text>

                    {/* Interactive Tooltip box */}
                    {isHovered && (
                      <g className="pointer-events-none transition-all duration-300">
                        {/* Shadow rectangle background */}
                        <rect
                          x={pt.x - 65}
                          y={pt.y - 65}
                          width="130"
                          height="48"
                          rx="8"
                          fill="#1e293b"
                          className="shadow-2xl"
                        />
                        <text
                          x={pt.x}
                          y={pt.y - 48}
                          fill="#ffffff"
                          className="font-sans text-[11px] font-bold"
                          textAnchor="middle"
                        >
                          Выручка: {pt.revenue.toLocaleString('ru-RU')} ₽
                        </text>
                        <text
                          x={pt.x}
                          y={pt.y - 34}
                          fill="#94a3b8"
                          className="font-mono text-[10px] font-semibold"
                          textAnchor="middle"
                        >
                          Чеков: {pt.orders} шт
                        </text>
                        <line
                          x1={pt.x}
                          y1={pt.y - 17}
                          x2={pt.x}
                          y2={pt.y}
                          stroke="#1e293b"
                          strokeWidth="2"
                        />
                      </g>
                    )}
                  </g>
                );
              })}
            </svg>
          </div>
        </div>

        {/* Categories & Payment methods breakdown side cards */}
        <div className="bg-white p-6 rounded-2xl border border-slate-200 md:col-span-1 shadow-sm flex flex-col justify-between gap-6">
          <div>
            <h3 className="font-sans font-bold text-base text-slate-900 mb-4">
              Продажи по категориям
            </h3>
            <div className="space-y-3.5">
              {categoryStats.map((cat, idx) => (
                <div key={cat.slug} className="group">
                  <div className="flex justify-between items-center text-xs font-semibold mb-1">
                    <span className="text-slate-600 hover:text-blue-600 transition-colors">
                      {cat.name}
                    </span>
                    <span className="font-mono text-slate-900">
                      {cat.revenue.toLocaleString('ru-RU')} ₽ ({Math.round(cat.percentage)}%)
                    </span>
                  </div>
                  {/* Custom progress bar */}
                  <div className="w-full bg-slate-100 h-2.5 rounded-full overflow-hidden">
                    <div
                      className={`h-full rounded-full transition-all duration-500 ${
                        idx === 0
                          ? 'bg-blue-600'
                          : idx === 1
                          ? 'bg-purple-500'
                          : idx === 2
                          ? 'bg-pink-500'
                          : idx === 3
                          ? 'bg-amber-500'
                          : 'bg-emerald-500'
                      }`}
                      style={{ width: `${cat.percentage}%` }}
                    />
                  </div>
                </div>
              ))}
            </div>
          </div>

          <div className="pt-4 border-t border-slate-100">
            <h3 className="font-sans font-bold text-sm text-slate-950 mb-3.5">
              Методы оплат (Доля)
            </h3>
            <div className="flex gap-2.5 h-6 rounded-lg overflow-hidden">
              {payments.map((p, idx) => (
                p.pct > 0 && (
                  <div
                    key={idx}
                    className={`${p.color.replace(/bg-indigo-500/g, 'bg-blue-500')} transition-all duration-500 relative group`}
                    style={{ width: `${p.pct}%` }}
                  >
                    <div className="absolute left-1/2 -top-12 -translate-x-1/2 bg-slate-800 text-white text-[10px] py-1 px-2 rounded secrecy hidden group-hover:block whitespace-nowrap z-30 font-semibold font-mono shadow">
                      {p.type}: {Math.round(p.pct)}%
                    </div>
                  </div>
                )
              ))}
            </div>
            <div className="flex flex-wrap gap-x-4 gap-y-2.5 mt-4">
              {payments.map((p, idx) => (
                <div key={idx} className="flex items-center gap-2">
                  <span className={`w-2.5 h-2.5 rounded ${p.color.replace(/bg-indigo-500/g, 'bg-blue-500')}`} />
                  <span className="text-xs text-slate-500">{p.type}</span>
                  <span className="font-mono text-xs font-bold text-slate-800">
                    ({Math.round(p.pct)}%)
                  </span>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>

      {/* Popular Dishes & Live Ledger Grid */}
      <div className="grid grid-cols-1 xl:grid-cols-2 gap-8">
        {/* Popular Dishes Card */}
        <div className="bg-white p-6 rounded-2xl border border-slate-200/80 shadow-sm">
          <div className="flex items-center gap-2.5 mb-6">
            <div className="p-2 bg-amber-50 rounded-lg text-amber-600">
              <Trophy className="w-5 h-5" />
            </div>
            <div>
              <h3 className="font-sans font-bold text-base text-slate-900">
                Самые популярные блюда
              </h3>
              <p className="text-xs text-slate-400 mt-0.5">Лидеры продаж по количеству заказанных позиций</p>
            </div>
          </div>
          <div className="divide-y divide-slate-100">
            {topDishes.map((dish, idx) => (
              <div key={dish.id} className="py-3.5 flex items-center justify-between first:pt-0 last:pb-0">
                <div className="flex items-center gap-3.5">
                  <span className="w-7 text-xs font-mono font-bold text-slate-300">
                    #{(idx + 1).toString().padStart(2, '0')}
                  </span>
                  <span className="text-2xl leading-none">{dish.emoji}</span>
                  <div>
                    <h4 className="text-sm font-semibold text-slate-900">{dish.name}</h4>
                    <span className="text-[11px] font-mono font-medium text-slate-400 uppercase bg-slate-50 px-1.5 py-0.5 rounded border border-slate-100">
                      {dish.category}
                    </span>
                  </div>
                </div>
                <div className="text-right">
                  <span className="text-sm font-bold text-slate-900 font-sans block">
                    {dish.sellCount} шт.
                  </span>
                  <span className="text-xs text-slate-500 font-mono">
                    {dish.totalRevenue.toLocaleString('ru-RU')} ₽
                  </span>
                </div>
              </div>
            ))}
            {topDishes.length === 0 && (
              <div className="py-12 text-center text-slate-400 text-sm font-medium">
                Нет данных для отображения лучших блюд.
              </div>
            )}
          </div>
        </div>

        {/* Live Transaction Ledger */}
        <div className="bg-white p-6 rounded-2xl border border-slate-200/80 shadow-sm flex flex-col justify-between">
          <div>
            <div className="flex items-center justify-between mb-6">
              <div>
                <h3 className="font-sans font-bold text-base text-slate-900">
                  История последних транзакций
                </h3>
                <p className="text-xs text-slate-400 mt-0.5">Служба реального времени POS-выгрузки</p>
              </div>
              <span className="bg-blue-50 border border-blue-100 text-blue-700 font-mono text-[10px] font-bold uppercase tracking-wider px-2.5 py-1 rounded-full animate-pulse">
                СВЕЖИЙ ЛОГ
              </span>
            </div>
            <div className="space-y-3.5 max-h-[290px] overflow-y-auto pr-1">
              {transactions.slice().reverse().slice(0, 5).map((tx) => (
                <div
                  key={tx.id}
                  className="p-3 rounded-xl border border-slate-100 bg-slate-50/60 flex items-center justify-between hover:bg-slate-50 transition-colors"
                >
                  <div className="flex items-center gap-3">
                    <span className="text-xl">🧾</span>
                    <div>
                      <div className="flex items-center gap-1.5">
                        <span className="text-xs font-mono font-bold text-slate-800">
                          {tx.id}
                        </span>
                        <span className="text-[10px] font-mono font-semibold text-slate-400">
                          {tx.timestamp}
                        </span>
                      </div>
                      <p className="text-xs text-slate-500 mt-0.5 max-w-[240px] truncate">
                        {tx.items.map((it) => `${it.itemName} (x${it.quantity})`).join(', ')}
                      </p>
                    </div>
                  </div>
                  <div className="text-right flex flex-col items-end gap-1">
                    <span className="text-xs font-bold text-blue-600 font-mono leading-none">
                      +{tx.totalAmount} ₽
                    </span>
                    <span className="text-[9px] font-mono tracking-wide uppercase font-semibold border px-1.5 py-0.5 rounded leading-none bg-white text-slate-400">
                      {tx.paymentMethod === 'card' ? 'карта' : tx.paymentMethod === 'cash' ? 'наличные' : 'сбп'}
                    </span>
                  </div>
                </div>
              ))}
              {transactions.length === 0 && (
                <div className="py-12 text-center text-slate-400 text-sm font-medium">
                  Нет чеков за сегодня.
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
