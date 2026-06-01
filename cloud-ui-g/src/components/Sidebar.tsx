interface SidebarProps {
  appTitle: string;
  appSubtitle: string;
}

export default function Sidebar({ appTitle, appSubtitle }: SidebarProps) {
  return (
    <aside className="w-full rounded-2xl border border-slate-200 bg-white p-6 lg:w-80">
      <h1 className="text-lg font-semibold text-slate-900">{appTitle}</h1>
      <p className="mt-2 text-sm text-slate-500">{appSubtitle}</p>
    </aside>
  );
}
