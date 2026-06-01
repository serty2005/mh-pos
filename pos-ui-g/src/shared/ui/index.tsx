import React from 'react';
import { ChevronLeft, ChevronRight, Minus, Plus, Search, X } from 'lucide-react';

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

interface PosIconButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  icon: React.ReactNode;
  label: string;
  variant?: 'neutral' | 'danger' | 'ghost';
  size?: 'sm' | 'md';
}

export const PosIconButton: React.FC<PosIconButtonProps> = ({
  icon,
  label,
  variant = 'neutral',
  size = 'md',
  className = '',
  ...props
}) => {
  const variantStyles = {
    neutral: 'border-[var(--pos-border)] bg-[var(--pos-surface-raised)] text-[var(--pos-text-primary)] hover:bg-[var(--pos-border)]',
    danger: 'border-[var(--pos-status-danger)] bg-red-500/10 text-[var(--pos-status-danger)] hover:bg-red-500 hover:text-white',
    ghost: 'border-transparent bg-transparent text-[var(--pos-text-muted)] hover:text-[var(--pos-text-primary)] hover:bg-[var(--pos-surface-raised)]',
  };
  const sizeStyles = {
    sm: 'w-8 h-8',
    md: 'w-10 h-10',
  };

  return (
    <button
      type="button"
      aria-label={label}
      title={label}
      className={`inline-flex items-center justify-center border rounded-none cursor-pointer select-none transition-colors outline-none focus:ring-2 focus:ring-[var(--pos-focus-ring)] disabled:opacity-40 disabled:cursor-not-allowed ${variantStyles[variant]} ${sizeStyles[size]} ${className}`}
      {...props}
    >
      {icon}
    </button>
  );
};

interface PosSelectableChipProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  active?: boolean;
  fullWidth?: boolean;
  tone?: 'default' | 'danger';
}

export const PosSelectableChip: React.FC<PosSelectableChipProps> = ({
  active = false,
  fullWidth = false,
  tone = 'default',
  className = '',
  children,
  ...props
}) => {
  const activeStyle = tone === 'danger'
    ? 'bg-[var(--pos-status-danger)] border-[var(--pos-status-danger)] text-[var(--pos-surface)]'
    : 'bg-[var(--pos-action-primary)] border-[var(--pos-action-primary)] text-[var(--pos-surface)]';
  const widthStyle = fullWidth ? 'w-full' : '';

  return (
    <button
      type="button"
      aria-pressed={active}
      className={`h-11 px-4 border font-mono text-xs uppercase font-bold rounded-none cursor-pointer select-none transition-all outline-none focus:ring-2 focus:ring-[var(--pos-focus-ring)] disabled:opacity-40 disabled:cursor-not-allowed ${widthStyle} ${
        active
          ? activeStyle
          : 'bg-[var(--pos-surface)] text-[var(--pos-text-secondary)] border-[var(--pos-border)] hover:bg-[var(--pos-surface-raised)] hover:text-[var(--pos-text-primary)]'
      } ${className}`}
      {...props}
    >
      {children}
    </button>
  );
};

interface PosSearchInputProps {
  id: string;
  value: string;
  onChange: (value: string) => void;
  placeholder: string;
  clearLabel: string;
  className?: string;
  onClear?: () => void;
}

export const PosSearchInput: React.FC<PosSearchInputProps> = ({
  id,
  value,
  onChange,
  placeholder,
  clearLabel,
  className = '',
  onClear,
}) => {
  const handleClear = () => {
    onChange('');
    onClear?.();
  };

  return (
    <div className={`relative w-full ${className}`}>
      <span className="absolute inset-y-0 left-3 flex items-center text-[var(--pos-text-muted)] pointer-events-none">
        <Search className="w-4 h-4" />
      </span>
      <input
        id={id}
        type="text"
        placeholder={placeholder}
        className="w-full h-11 pl-10 pr-10 border border-[var(--pos-border)] bg-[var(--pos-surface)] text-[var(--pos-text-primary)] font-sans text-xs focus:outline-none focus:border-[var(--pos-border-strong)] rounded-none"
        value={value}
        onChange={(event) => onChange(event.target.value)}
      />
      {value && (
        <PosIconButton
          variant="ghost"
          size="sm"
          label={clearLabel}
          onClick={handleClear}
          className="absolute inset-y-1 right-1 h-9 w-9"
          icon={<X className="w-3.5 h-3.5" />}
        />
      )}
    </div>
  );
};

interface PosBottomNavItem {
  id: string;
  label: string;
  icon: React.ComponentType<{ className?: string }>;
  badge?: string | number;
  disabled?: boolean;
}

interface PosBottomNavProps {
  items: PosBottomNavItem[];
  activeId: string;
  onChange: (id: string) => void;
  idPrefix?: string;
  horizontal?: boolean;
  className?: string;
}

export const PosBottomNav: React.FC<PosBottomNavProps> = ({
  items,
  activeId,
  onChange,
  idPrefix = 'nav',
  horizontal = false,
  className = '',
}) => {
  return (
    <div className={`flex w-full divide-x divide-[var(--pos-border)] h-full overflow-hidden ${className}`}>
      {items.map((item) => {
        const active = item.id === activeId;
        const Icon = item.icon;
        return (
          <button
            key={item.id}
            id={`${idPrefix}-${item.id}`}
            type="button"
            disabled={item.disabled}
            onClick={() => onChange(item.id)}
            className={`flex-1 h-full font-mono text-center flex ${horizontal ? 'flex-row' : 'flex-col md:flex-row'} items-center justify-center gap-1.5 border-t-2 select-none whitespace-nowrap cursor-pointer transition-colors disabled:opacity-40 disabled:cursor-not-allowed ${
              active
                ? 'bg-[var(--pos-surface-raised)] border-t-[var(--pos-action-primary)] text-[var(--pos-text-primary)] font-black'
                : 'bg-transparent border-t-transparent text-[var(--pos-text-muted)] hover:text-[var(--pos-text-primary)]'
            }`}
          >
            <Icon className={`w-4 h-4 md:w-5 md:h-5 shrink-0 ${active ? 'text-[var(--pos-text-primary)]' : 'text-[var(--pos-text-muted)]'}`} />
            <span className="text-[10px] md:text-xs font-bold uppercase tracking-widest">{item.label}</span>
            {item.badge !== undefined && (
              <span className="ml-1 text-[8px] border border-amber-500 text-amber-600 px-1 uppercase">
                {item.badge}
              </span>
            )}
          </button>
        );
      })}
    </div>
  );
};

interface PosSegmentedControlItem {
  id: string;
  label: React.ReactNode;
  disabled?: boolean;
}

interface PosSegmentedControlProps {
  items: PosSegmentedControlItem[];
  activeId: string;
  onChange: (id: string) => void;
  idPrefix?: string;
  className?: string;
  itemClassName?: string;
}

export const PosSegmentedControl: React.FC<PosSegmentedControlProps> = ({
  items,
  activeId,
  onChange,
  idPrefix = 'segment',
  className = '',
  itemClassName = '',
}) => {
  return (
    <div className={`flex gap-1.5 overflow-x-auto pos-scrollbar-thin ${className}`}>
      {items.map((item) => (
        <PosSelectableChip
          key={item.id}
          id={`${idPrefix}-${item.id}`}
          active={item.id === activeId}
          disabled={item.disabled}
          onClick={() => onChange(item.id)}
          className={itemClassName}
        >
          {item.label}
        </PosSelectableChip>
      ))}
    </div>
  );
};

interface PosStatusBadgeProps {
  children: React.ReactNode;
  variant?: 'success' | 'warning' | 'danger' | 'neutral' | 'info';
  className?: string;
}

export const PosStatusBadge: React.FC<PosStatusBadgeProps> = ({
  children,
  variant = 'neutral',
  className = '',
}) => {
  const styles = {
    success: 'bg-emerald-500 text-white',
    warning: 'bg-amber-500 text-zinc-950',
    danger: 'bg-red-500 text-white',
    neutral: 'bg-neutral-600 text-white',
    info: 'bg-[var(--pos-status-info)] text-white',
  };

  return (
    <span className={`px-1.5 py-0.5 font-mono text-[9px] font-bold uppercase rounded-none ${styles[variant]} ${className}`}>
      {children}
    </span>
  );
};

interface PosInlineStatusBadgeProps {
  children: React.ReactNode;
  variant?: 'success' | 'warning' | 'danger' | 'neutral' | 'info';
  className?: string;
}

export const PosInlineStatusBadge: React.FC<PosInlineStatusBadgeProps> = ({
  children,
  variant = 'neutral',
  className = '',
}) => {
  const styles = {
    success: 'border-[var(--pos-status-success)] text-[var(--pos-status-success)] bg-emerald-50/10',
    warning: 'border-[var(--pos-status-warning)] text-amber-700 dark:text-amber-300 bg-amber-50/10',
    danger: 'border-[var(--pos-status-danger)] text-[var(--pos-status-danger)] bg-red-50/10',
    neutral: 'border-[var(--pos-border)] text-[var(--pos-text-secondary)] bg-[var(--pos-action-secondary)]',
    info: 'border-[var(--pos-status-info)] text-[var(--pos-status-info)] bg-blue-50/10',
  };

  return (
    <span className={`inline-flex items-center border px-2 py-1 font-mono text-[10px] font-bold uppercase tracking-wider rounded-none ${styles[variant]} ${className}`}>
      {children}
    </span>
  );
};

interface PosSelectableTileProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  active?: boolean;
  title?: React.ReactNode;
  subtitle?: React.ReactNode;
  value?: React.ReactNode;
  status?: React.ReactNode;
  statusVariant?: 'success' | 'warning' | 'danger' | 'neutral' | 'info';
  leading?: React.ReactNode;
  trailing?: React.ReactNode;
  children?: React.ReactNode;
}

export const PosSelectableTile: React.FC<PosSelectableTileProps> = ({
  active = false,
  disabled = false,
  title,
  subtitle,
  value,
  status,
  statusVariant = 'neutral',
  leading,
  trailing,
  children,
  className = '',
  ...props
}) => {
  return (
    <button
      type="button"
      aria-pressed={active}
      disabled={disabled}
      className={`w-full text-left border rounded-none cursor-pointer select-none transition-all focus:ring-2 focus:ring-[var(--pos-focus-ring)] outline-none disabled:opacity-40 disabled:cursor-not-allowed ${
        active
          ? 'bg-[var(--pos-selected-order)] border-l-4 border-l-[var(--pos-action-primary)] border-[var(--pos-border)] text-[var(--pos-text-primary)]'
          : 'bg-[var(--pos-surface)] border-[var(--pos-border)] text-[var(--pos-text-primary)] hover:bg-[var(--pos-surface-raised)] hover:border-[var(--pos-border-strong)]'
      } ${className}`}
      {...props}
    >
      {children ?? (
        <div className="flex items-center justify-between gap-3">
          <div className="flex min-w-0 items-start gap-3">
            {leading && <span className="shrink-0">{leading}</span>}
            <div className="min-w-0">
              {title && <span className="font-sans text-sm font-semibold text-[var(--pos-text-primary)] truncate block">{title}</span>}
              {subtitle && <span className="font-mono text-[10px] text-[var(--pos-text-muted)] uppercase tracking-wider truncate block mt-0.5">{subtitle}</span>}
            </div>
          </div>
          <div className="shrink-0 flex items-center gap-2">
            {value && <span className="font-mono text-sm font-bold text-[var(--pos-text-primary)]">{value}</span>}
            {status && <PosInlineStatusBadge variant={statusVariant}>{status}</PosInlineStatusBadge>}
            {trailing}
          </div>
        </div>
      )}
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
          <PosIconButton
            label={title}
            variant="neutral"
            onClick={onClose}
            icon={<X className="w-4 h-4" />}
          />
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

interface PosRailHeaderProps {
  title: React.ReactNode;
  badge?: React.ReactNode;
  actions?: React.ReactNode;
  className?: string;
}

export const PosRailHeader: React.FC<PosRailHeaderProps> = ({
  title,
  badge,
  actions,
  className = '',
}) => {
  return (
    <div className={`h-14 border-b border-[var(--pos-border)] px-4 flex items-center justify-between bg-[var(--pos-surface-raised)] select-none shrink-0 ${className}`}>
      <div className="min-w-0 flex items-center gap-2">
        <span className="font-mono text-xs font-bold uppercase tracking-wider text-[var(--pos-text-secondary)] truncate">
          {title}
        </span>
        {badge && <span className="shrink-0">{badge}</span>}
      </div>
      {actions && <div className="shrink-0 flex items-center gap-2">{actions}</div>}
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
  idPrefix?: string;
}

export const PosTabs: React.FC<PosTabsProps> = ({
  id,
  items,
  activeId,
  onChange,
  idPrefix = 'tab'
}) => {
  return (
    <div id={id} className="flex border-b border-[var(--pos-border)] bg-[var(--pos-surface)] pos-scrollarea-x pos-scrollbar-thin max-w-full">
      {items.map((item) => {
        const isActive = item.id === activeId;
        return (
          <button
            key={item.id}
            id={`${idPrefix}-${item.id}`}
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
  prevLabel?: string;
  nextLabel?: string;
}

export const PosPagination: React.FC<PosPaginationProps> = ({
  currentPage,
  isFirstPage,
  isLastPage,
  onPrev,
  onNext,
  label = String(currentPage),
  prevLabel = 'previous',
  nextLabel = 'next',
}) => {
  return (
    <div className="flex items-center justify-between py-2 px-4 border border-[var(--pos-border)] bg-[var(--pos-surface)] select-none">
      <span className="font-mono text-xs font-bold uppercase text-[var(--pos-text-secondary)]">{label}</span>
      <div className="flex gap-1">
        <PosIconButton
          onClick={onPrev}
          disabled={isFirstPage}
          label={prevLabel}
          variant="neutral"
          icon={<ChevronLeft className="w-4 h-4" />}
        />
        <PosIconButton
          onClick={onNext}
          disabled={isLastPage}
          label={nextLabel}
          variant="neutral"
          icon={<ChevronRight className="w-4 h-4" />}
        />
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
