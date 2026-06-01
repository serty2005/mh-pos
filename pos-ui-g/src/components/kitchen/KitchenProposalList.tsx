import { AlertTriangle } from 'lucide-react';

import { t } from '../../shared/i18n';
import { PosEmptyState, PosInlineStatusBadge } from '../../shared/ui';
import type { BackendKitchenProposal } from '../../shared/schemas';
import { formatDateTime, proposalKindLabel, proposalStatusLabel } from './kitchenHelpers';

export function KitchenProposalList({ proposals }: { proposals: BackendKitchenProposal[] }) {
  if (proposals.length === 0) {
    return (
      <PosEmptyState
        title={t.kitchen.emptyProposals}
        description={t.kitchen.tabSuggestions}
        icon={<AlertTriangle className="w-10 h-10" />}
      />
    );
  }
  return (
    <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
      {proposals.map((proposal) => (
        <article key={proposal.id} className="border border-[var(--pos-border)] bg-[var(--pos-surface)] p-4 space-y-3">
          <div className="flex items-start justify-between gap-3">
            <div>
              <div className="font-mono text-[10px] uppercase tracking-widest text-[var(--pos-text-muted)]">{proposalKindLabel(proposal.kind)}</div>
              <h3 className="font-sans text-sm font-bold text-[var(--pos-text-primary)]">{proposal.action || proposal.outbox_event_type}</h3>
            </div>
            <PosInlineStatusBadge variant="neutral">
              {proposalStatusLabel(proposal.status)}
            </PosInlineStatusBadge>
          </div>
          <div className="font-mono text-[10px] uppercase tracking-wider text-[var(--pos-text-muted)]">
            {formatDateTime(proposal.created_at)}
          </div>
        </article>
      ))}
    </div>
  );
}
