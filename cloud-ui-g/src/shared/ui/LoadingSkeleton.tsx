type LoadingSkeletonProps = {
  cards?: number;
  className?: string;
};

export default function LoadingSkeleton({ cards = 3, className = 'grid gap-3 sm:grid-cols-3' }: LoadingSkeletonProps) {
  return (
    <div className={className} aria-busy="true" aria-live="polite">
      {Array.from({ length: cards }, (_, item) => (
        <div key={item} className="rounded-2xl border border-slate-200 bg-white p-4 shadow-[0_18px_44px_-32px_rgba(15,23,42,0.32)]">
          <div className="flex items-center justify-between gap-3">
            <div className="h-3 w-20 animate-pulse rounded-full bg-slate-200" />
            <div className="h-8 w-8 animate-pulse rounded-xl bg-blue-100" />
          </div>
          <div className="mt-5 h-6 w-28 animate-pulse rounded-lg bg-slate-200" />
          <div className="mt-3 space-y-2">
            <div className="h-3 w-full animate-pulse rounded-full bg-slate-100" />
            <div className="h-3 w-2/3 animate-pulse rounded-full bg-slate-100" />
          </div>
        </div>
      ))}
    </div>
  );
}
