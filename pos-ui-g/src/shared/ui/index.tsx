import React from 'react';
import { ChevronLeft, ChevronRight, Minus, Plus } from 'lucide-react';

// PosButton component for direct touch-first targets
interface PosButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'primary' | 'secondary' | 'danger' | 'success' | 'warning' | 'financial' | 'ghost';
  size?: 'sm' | 'md' | 'lg' | 'critical';
  icon?: React.ReactNode;
  fullWidth?: boolean;
}

export const PosButton: React.FC<PosButtonProps> = ({
  children,
  variant = 'secondary',
  size = 'md',
  icon,
  fullWidth = false,
  className = '',
  ...props
}) => {
  const baseStyle = 'inline-flex items-center justify-center font-semibold rounded-none uppercase tracking-wider transition-all cursor-pointer border border-transparent select-none shrink-0 outline-none focus:ring-2 focus:ring-[var(--pos-focus-ring)]';
  
  const variantStyles = {
    primary: 'bg-[var(--pos-action-primary)] hover:bg-[var(--pos-action-primary-hover)] text-[var(--pos-surface)]',
    secondary: 'bg-[var(--pos-action-secondary)] hover:bg-[var(--pos-action-secondary-hover)] border-[var(--pos-border)] text-[var(--pos-text-primary)]',
    danger: 'bg-[var(--pos-status-danger)] hover:opacity-90 text-[var(--pos-surface)]',
    success: 'bg-[var(--pos-status-success)] hover:opacity-90 text-[var(--pos-surface)]',
    warning: 'bg-[var(--pos-status-warning)] hover:opacity-90 text-zinc-950',
    financial: 'bg-[var(--pos-status-success)] hover:opacity-95 text-[var(--pos-surface)] shadow-sm font-bold',
    ghost: 'bg-transparent text-[var(--pos-text-secondary)] hover:bg-zinc-50 dark:hover:bg-zinc-800 border-none'
  };

  const sizeStyles = {
    sm: 'h-10 px-3 text-xs md:text-sm',
    md: 'h-[52px] px-5 text-sm md:text-base',
    lg: 'h-[64px] px-8 text-base md:text-lg',
    critical: 'h-[80px] px-10 text-lg md:text-xl font-bold'
  };

  const widthStyle = fullWidth ? 'w-full' : '';
  const disabledStyle = props.disabled ? 'opacity-40 cursor-not-allowed filter saturate-50' : '';

  return (
    <button
      className={`${baseStyle} ${variantStyles[variant]} ${sizeStyles[size]} ${widthStyle} ${disabledStyle} ${className}`}
      {...props}
    >
      {icon && <span className="mr-2 inline-flex items-center">{icon}</span>}
      {children}
    </button>
  );
};

// Compact Context Button
interface PosContextButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  active?: boolean;
}

export const PosContextButton: React.FC<PosContextButtonProps> = ({
  children,
  active = false,
  className = '',
  ...props
}) => {
  return (
    <button
      className={`h-[44px] px-4 font-mono font-medium text-xs rounded-none border text-left flex items-center justify-between select-none cursor-pointer tracking-tight
        ${active 
          ? 'bg-[var(--pos-action-primary)] text-[var(--pos-surface)] border-[var(--pos-action-primary)]' 
          : 'bg-[var(--pos-surface)] text-[var(--pos-text-secondary)] border-[var(--pos-border)] hover:bg-[var(--pos-bg)]'
        } ${props.disabled ? 'opacity-40 cursor-not-allowed' : ''} ${className}`}
      {...props}
    >
      {children}
    </button>
  );
};

// Generic accessible Modal dialog
interface PosDialogProps {
  isOpen: boolean;
  onClose: () => void;
  title: string;
  children: React.ReactNode;
  footer?: React.ReactNode;
}

export const PosDialog: React.FC<PosDialogProps> = ({
  isOpen,
  onClose,
  title,
  children,
  footer
}) => {
  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      {/* Backdrop */}
      <div 
        className="fixed inset-0 bg-neutral-950/70"
        onClick={onClose}
      />
      
      {/* Dialog content Container */}
      <div className="relative w-full max-w-lg bg-[var(--pos-surface)] border border-[var(--pos-border)] text-[var(--pos-text-primary)] rounded-none flex flex-col shadow-xl z-10 max-h-[90vh]">
        {/* Header */}
        <div className="px-6 py-4 border-b border-[var(--pos-border)] flex items-center justify-between shrink-0 bg-[var(--pos-surface-raised)]">
          <h3 className="font-mono text-base font-bold tracking-wider uppercase">{title}</h3>
          <button 
            className="w-10 h-10 border border-[var(--pos-border)] hover:bg-[var(--pos-border)] flex items-center justify-center font-mono text-lg font-bold cursor-pointer rounded-none"
            onClick={onClose}
          >
            ×
          </button>
        </div>

        {/* Scrollable Body */}
        <div className="p-6 pos-scrollarea-y pos-scrollbar-thin flex-1 overflow-y-auto">
          {children}
        </div>

        {/* Footer */}
        {footer && (
          <div className="px-6 py-4 border-t border-[var(--pos-border)] bg-[var(--pos-surface-raised)] flex items-center justify-end gap-3 shrink-0">
            {footer}
          </div>
        )}
      </div>
    </div>
  );
};

// Section header
interface PosSectionHeaderProps {
  title: string;
  badge?: string;
  actions?: React.ReactNode;
}

export const PosSectionHeader: React.FC<PosSectionHeaderProps> = ({
  title,
  badge,
  actions
}) => {
  return (
    <div className="h-[60px] border-b border-[var(--pos-border)] bg-[var(--pos-surface)] px-6 flex items-center justify-between select-none">
      <div className="flex items-center gap-3">
        <h2 className="font-mono uppercase tracking-widest text-sm font-bold md:text-base text-[var(--pos-text-primary)]">
          {title}
        </h2>
        {badge && (
          <span className="bg-[var(--pos-action-secondary)] border border-[var(--pos-border)] font-mono text-xs px-2 py-0.5 font-bold uppercase tracking-tight text-[var(--pos-text-secondary)]">
            {badge}
          </span>
        )}
      </div>
      {actions && <div className="flex gap-2 shrink-0">{actions}</div>}
    </div>
  );
};

// Tabs component
interface PosTabItem {
  id: string;
  label: string;
  count?: number;
}

interface PosTabsProps {
  id: string;
  items: PosTabItem[];
  activeId: string;
  onChange: (id: string) => void;
}

export const PosTabs: React.FC<PosTabsProps> = ({
  items,
  activeId,
  onChange
}) => {
  return (
    <div className="flex border-b border-[var(--pos-border)] bg-[var(--pos-surface)] pos-scrollarea-x pos-scrollbar-thin max-w-full">
      {items.map((item) => {
        const isActive = item.id === activeId;
        return (
          <button
            key={item.id}
            id={`tab-${item.id}`}
            onClick={() => onChange(item.id)}
            className={`h-14 px-6 font-mono text-xs uppercase tracking-widest font-bold border-r border-[var(--pos-border)] select-none cursor-pointer transition-all shrink-0 relative
              ${isActive 
                ? 'bg-[var(--pos-surface-raised)] text-[var(--pos-text-primary)] border-b-2 border-b-[var(--pos-action-primary)]' 
                : 'bg-transparent text-[var(--pos-text-muted)] hover:text-[var(--pos-text-primary)]'
              }`}
          >
            {item.label}
            {item.count !== undefined && item.count > 0 && (
              <span className="ml-2 font-mono bg-[var(--pos-action-primary)] text-[var(--pos-surface)] text-[10px] px-1.5 py-0.5 rounded-none font-bold">
                {item.count}
              </span>
            )}
          </button>
        );
      })}
    </div>
  );
};

// Stepper for Quantities
interface PosQuantityStepperProps {
  value: number;
  id: string;
  onChange: (value: number) => void;
  min?: number;
  max?: number;
  disabled?: boolean;
}

export const PosQuantityStepper: React.FC<PosQuantityStepperProps> = ({
  value,
  id,
  onChange,
  min = 1,
  max = 99,
  disabled = false
}) => {
  return (
    <div className="flex items-center border border-[var(--pos-border)] bg-[var(--pos-surface-raised)] h-[44px]">
      <button
        id={`${id}-decrease`}
        type="button"
        disabled={disabled || value <= min}
        className="w-12 h-full flex items-center justify-center border-r border-[var(--pos-border)] hover:bg-[var(--pos-border)] cursor-pointer select-none text-[var(--pos-text-primary)] disabled:opacity-30 disabled:cursor-not-allowed"
        onClick={() => onChange(value - 1)}
      >
        <Minus className="w-4 h-4" />
      </button>
      <span className="w-12 text-center font-mono font-bold text-base text-[var(--pos-text-primary)]">
        {value}
      </span>
      <button
        id={`${id}-increase`}
        type="button"
        disabled={disabled || value >= max}
        className="w-12 h-full flex items-center justify-center border-l border-[var(--pos-border)] hover:bg-[var(--pos-border)] cursor-pointer select-none text-[var(--pos-text-primary)] disabled:opacity-30 disabled:cursor-not-allowed"
        onClick={() => onChange(value + 1)}
      >
        <Plus className="w-4 h-4" />
      </button>
    </div>
  );
};

// Information banner / Notification warning
interface PosBannerProps {
  type?: 'info' | 'warning' | 'danger' | 'success';
  message: string;
  action?: React.ReactNode;
}

export const PosBanner: React.FC<PosBannerProps> = ({
  type = 'info',
  message,
  action
}) => {
  const styles = {
    info: 'bg-blue-50 dark:bg-blue-950 border-blue-200 dark:border-blue-900 text-blue-800 dark:text-blue-100',
    warning: 'bg-amber-50 dark:bg-amber-950 border-amber-200 dark:border-amber-900 text-amber-800 dark:text-amber-100',
    danger: 'bg-red-50 dark:bg-red-950 border-red-200 dark:border-red-900 text-red-800 dark:text-red-100',
    success: 'bg-emerald-50 dark:bg-emerald-950 border-emerald-200 dark:border-emerald-900 text-emerald-800 dark:text-emerald-100'
  };

  return (
    <div className={`p-4 border border-l-4 rounded-none flex items-center justify-between gap-4 font-sans text-sm tracking-tight ${styles[type]}`}>
      <div className="flex items-center gap-2">
        <span className="font-mono uppercase font-bold text-xs tracking-wider">[{type}]</span>
        <span>{message}</span>
      </div>
      {action && <div className="shrink-0">{action}</div>}
    </div>
  );
};

// Status Strip
interface PosStatusStripProps {
  title: string;
  message: string;
  variant?: 'neutral' | 'accent' | 'success' | 'warning' | 'danger';
}

export const PosStatusStrip: React.FC<PosStatusStripProps> = ({
  title,
  message,
  variant = 'neutral'
}) => {
  const borderColors = {
    neutral: 'border-l-[var(--pos-border-strong)] bg-[var(--pos-surface-raised)]',
    accent: 'border-l-[var(--pos-sync-pending)] bg-violet-50/10',
    success: 'border-l-[var(--pos-status-success)] bg-emerald-50/10',
    warning: 'border-l-[var(--pos-status-warning)] bg-amber-50/10',
    danger: 'border-l-[var(--pos-status-danger)] bg-red-50/10',
  };

  return (
    <div className={`border-l-4 px-4 py-2 border border-y-[var(--pos-border)] border-r-[var(--pos-border)] rounded-none flex flex-col gap-0.5 select-none ${borderColors[variant]}`}>
      <span className="font-mono text-[10px] font-bold uppercase tracking-widest text-[var(--pos-text-muted)]">
        {title}
      </span>
      <span className="font-sans text-xs font-semibold text-[var(--pos-text-secondary)]">
        {message}
      </span>
    </div>
  );
};

// Pagination controls
interface PosPaginationProps {
  currentPage: number;
  isFirstPage: boolean;
  isLastPage: boolean;
  onPrev: () => void;
  onNext: () => void;
  label?: string;
}

export const PosPagination: React.FC<PosPaginationProps> = ({
  currentPage,
  isFirstPage,
  isLastPage,
  onPrev,
  onNext,
  label = `Страница ${currentPage}`
}) => {
  return (
    <div className="flex items-center justify-between py-2 px-4 border border-[var(--pos-border)] bg-[var(--pos-surface)] select-none">
      <span className="font-mono text-xs font-bold uppercase text-[var(--pos-text-secondary)]">{label}</span>
      <div className="flex gap-1">
        <button
          onClick={onPrev}
          disabled={isFirstPage}
          className="w-10 h-10 border border-[var(--pos-border)] flex items-center justify-center bg-[var(--pos-surface)] hover:bg-[var(--pos-bg)] cursor-pointer disabled:opacity-30 disabled:cursor-not-allowed rounded-none"
        >
          <ChevronLeft className="w-4 h-4" />
        </button>
        <button
          onClick={onNext}
          disabled={isLastPage}
          className="w-10 h-10 border border-[var(--pos-border)] flex items-center justify-center bg-[var(--pos-surface)] hover:bg-[var(--pos-bg)] cursor-pointer disabled:opacity-30 disabled:cursor-not-allowed rounded-none"
        >
          <ChevronRight className="w-4 h-4" />
        </button>
      </div>
    </div>
  );
};

// Empty State visualizer
interface PosEmptyStateProps {
  title: string;
  description: string;
  buttonLabel?: string;
  onAction?: () => void;
  icon?: React.ReactNode;
}

export const PosEmptyState: React.FC<PosEmptyStateProps> = ({
  title,
  description,
  buttonLabel,
  onAction,
  icon
}) => {
  return (
    <div className="flex flex-col items-center justify-center text-center p-8 border border-dashed border-[var(--pos-border)] bg-[var(--pos-surface)] py-16 select-none max-w-lg mx-auto my-6">
      {icon && <div className="text-[var(--pos-text-muted)] mb-4 shrink-0">{icon}</div>}
      <h3 className="font-mono text-sm font-bold uppercase tracking-widest text-[var(--pos-text-primary)] mb-2">
        {title}
      </h3>
      <p className="font-sans text-xs md:text-sm text-[var(--pos-text-muted)] mb-6 max-w-xs leading-relaxed">
        {description}
      </p>
      {buttonLabel && onAction && (
        <PosButton variant="primary" size="sm" onClick={onAction}>
          {buttonLabel}
        </PosButton>
      )}
    </div>
  );
};

// Metric and session indicators
interface PosMetricCardProps {
  title: string;
  value: string | number;
  highlight?: boolean;
  statusText?: string;
  statusVariant?: 'success' | 'warning' | 'danger' | 'info';
}

export const PosMetricCard: React.FC<PosMetricCardProps> = ({
  title,
  value,
  highlight = false,
  statusText,
  statusVariant = 'info'
}) => {
  const statusColors = {
    info: 'text-[var(--pos-status-info)] border-[var(--pos-status-info)]',
    success: 'text-[var(--pos-status-success)] border-[var(--pos-status-success)]',
    warning: 'text-[var(--pos-status-warning)] border-[var(--pos-status-warning)]',
    danger: 'text-[var(--pos-status-danger)] border-[var(--pos-status-danger)]'
  };

  return (
    <div className={`p-5 border border-[var(--pos-border)] rounded-none flex flex-col justify-between shrink-0 select-none bg-[var(--pos-surface)]`}>
      <span className="font-mono text-[10px] font-bold uppercase tracking-widest text-[var(--pos-text-muted)] mb-3">
        {title}
      </span>
      <div className="flex items-baseline justify-between gap-2 mt-auto">
        <span className={`font-mono text-2xl font-bold tracking-tight text-[var(--pos-text-primary)]`}>
          {value}
        </span>
        {statusText && (
          <span className={`font-mono text-[10px] px-1.5 py-0.5 border font-semibold uppercase ${statusColors[statusVariant]}`}>
            {statusText}
          </span>
        )}
      </div>
    </div>
  );
};

// Skeletons for loaders
export const PosSkeleton: React.FC<{ type?: 'grid' | 'lines' | 'card' }> = ({ type = 'grid' }) => {
  if (type === 'lines') {
    return (
      <div className="animate-pulse space-y-4">
        {[1, 2, 3, 4].map((i) => (
          <div key={i} className="h-10 bg-[var(--pos-action-secondary)] border border-[var(--pos-border)] w-full" />
        ))}
      </div>
    );
  }

  if (type === 'card') {
    return (
      <div className="animate-pulse border border-[var(--pos-border)] p-6 bg-[var(--pos-surface)] space-y-3">
        <div className="h-4 bg-[var(--pos-action-secondary)] w-1/3" />
        <div className="h-8 bg-[var(--pos-action-secondary)] w-1/2" />
        <div className="h-4 bg-[var(--pos-action-secondary)] w-2/3" />
      </div>
    );
  }

  return (
    <div className="animate-pulse grid grid-cols-2 shadow-sm md:grid-cols-3 lg:grid-cols-4 gap-4 p-2">
      {[1, 2, 3, 4, 5, 6].map((i) => (
        <div key={i} className="h-[112px] bg-[var(--pos-surface-raised)] border border-[var(--pos-border)] rounded-none" />
      ))}
    </div>
  );
};

// Standard Action Rail Panel
export const PosActionRail: React.FC<{ children: React.ReactNode; className?: string }> = ({
  children,
  className = ''
}) => {
  return (
    <div className={`bg-[var(--pos-surface)] border-l border-[var(--pos-border)] w-full flex flex-col min-h-0 select-none ${className}`}>
      {children}
    </div>
  );
};

// PosDataRow for list items
interface PosDataRowProps {
  title: string;
  subtitle?: string;
  value: string | number;
  highlightValue?: boolean;
}

export const PosDataRow: React.FC<PosDataRowProps> = ({
  title,
  subtitle,
  value,
  highlightValue = false
}) => {
  return (
    <div className="py-3 px-4 border-b border-[var(--pos-border)] flex items-center justify-between hover:bg-[var(--pos-surface-raised)] transition-all">
      <div className="flex flex-col shrink-0">
        <span className="font-sans text-xs font-semibold text-[var(--pos-text-secondary)]">{title}</span>
        {subtitle && <span className="font-mono text-[10px] text-[var(--pos-text-muted)] tracking-tight mt-0.5">{subtitle}</span>}
      </div>
      <span className={`font-mono text-sm font-bold ${highlightValue ? 'text-[var(--pos-status-success)]' : 'text-[var(--pos-text-primary)]'}`}>
        {value}
      </span>
    </div>
  );
};

// PosFormRow for standard dialog layouts
interface PosFormRowProps {
  label: string;
  id: string;
  error?: string;
  children: React.ReactNode;
}

export const PosFormRow: React.FC<PosFormRowProps> = ({
  label,
  id,
  error,
  children
}) => {
  return (
    <div className="flex flex-col gap-1.5 mb-4">
      <label htmlFor={id} className="font-mono text-xs font-bold uppercase tracking-wider text-[var(--pos-text-secondary)] select-none">
        {label}
      </label>
      <div className="relative">
        {children}
      </div>
      {error && (
        <span className="font-sans text-xs font-semibold text-[var(--pos-status-danger)] animate-fade-in">
          {error}
        </span>
      )}
    </div>
  );
};
