import type { LucideIcon } from 'lucide-react';
import type { ReactNode } from 'react';

type PanelHeaderProps = {
  icon: LucideIcon;
  title: string;
  description?: string;
  action?: ReactNode;
  meta?: ReactNode;
};

export default function PanelHeader({ icon: Icon, title, description, action, meta }: PanelHeaderProps) {
  return (
    <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
      <div className="flex min-w-0 items-start gap-3">
        <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border border-blue-100 bg-blue-50 text-blue-700 shadow-[0_12px_28px_-24px_rgba(37,99,235,0.9)]">
          <Icon className="h-4 w-4" />
        </div>
        <div className="min-w-0">
          {meta ? <div className="mb-2">{meta}</div> : null}
          <h3 className="text-lg font-semibold tracking-tight text-slate-950">{title}</h3>
          {description ? <p className="mt-1 max-w-3xl text-sm leading-6 text-slate-600">{description}</p> : null}
        </div>
      </div>
      {action ? <div className="shrink-0">{action}</div> : null}
    </div>
  );
}
