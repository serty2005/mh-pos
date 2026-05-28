interface MenuPanelProps {
  title: string;
  description: string;
}

export default function MenuPanel({ title, description }: MenuPanelProps) {
  return (
    <section className="rounded-2xl border border-slate-200 bg-white p-5">
      <h2 className="text-sm font-semibold text-slate-900">{title}</h2>
      <p className="mt-2 text-sm text-slate-500">{description}</p>
    </section>
  );
}
