export default function EmptyState({ title, description }: { title: string; description: string }) {
  return (
    <div className="rounded-xl border border-dashed border-slate-300 p-4 text-sm text-slate-600">
      <p className="font-medium text-slate-900">{title}</p>
      <p className="mt-1">{description}</p>
    </div>
  );
}
