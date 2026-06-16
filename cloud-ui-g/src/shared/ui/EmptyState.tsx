import { Inbox } from 'lucide-react';

export default function EmptyState({ title, description }: { title: string; description: string }) {
  return (
    <div className="rounded-2xl border border-dashed border-slate-300 bg-white/80 p-5 text-sm text-slate-600 shadow-[0_18px_42px_-34px_rgba(15,23,42,0.44)]">
      <div className="flex items-start gap-3">
        <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border border-slate-200 bg-slate-50 text-slate-500">
          <Inbox className="h-4 w-4" />
        </div>
        <div className="min-w-0">
          <p className="font-semibold tracking-tight text-slate-900">{title}</p>
          <p className="mt-1 leading-relaxed">{description}</p>
        </div>
      </div>
    </div>
  );
}
