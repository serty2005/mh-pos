import { renderToString } from '@vue/server-renderer';
import { createSSRApp } from 'vue';
import { describe, expect, it } from 'vitest';

import PosActionRail from './PosActionRail.vue';
import PosDataRow from './PosDataRow.vue';
import PosPanel from './PosPanel.vue';
import PosReadinessCard from './PosReadinessCard.vue';

describe('POS reusable UI primitives', () => {
  it('renders action rail header, summary and actions slots', async () => {
    const html = await renderToString(createSSRApp({
      components: { PosActionRail },
      template: `
        <PosActionRail eyebrow="Data" title="Summary" aria-label="Summary">
          <template #summary><div><span>Pending</span><strong>2</strong></div></template>
          <template #actions><button>Close</button></template>
        </PosActionRail>
      `,
    }));

    expect(html).toContain('Data');
    expect(html).toContain('Summary');
    expect(html).toContain('Pending');
    expect(html).toContain('Close');
  });

  it('renders panel title and body content', async () => {
    const html = await renderToString(createSSRApp({
      components: { PosPanel },
      template: '<PosPanel title="Cash"><p>Operations</p></PosPanel>',
    }));

    expect(html).toContain('Cash');
    expect(html).toContain('Operations');
  });

  it('renders data row main and side content with tone class', async () => {
    const html = await renderToString(createSSRApp({
      components: { PosDataRow },
      template: '<PosDataRow label="Payment" meta="captured" value="100" tone="warning" />',
    }));

    expect(html).toContain('Payment');
    expect(html).toContain('captured');
    expect(html).toContain('100');
    expect(html).toContain('warning');
  });

  it('renders passive readiness content without owning business state', async () => {
    const html = await renderToString(createSSRApp({
      components: { PosReadinessCard },
      template: '<PosReadinessCard title="Future action" description="No endpoint" badge="planned" tone="warning" passive />',
    }));

    expect(html).toContain('Future action');
    expect(html).toContain('No endpoint');
    expect(html).toContain('planned');
    expect(html).toContain('aria-disabled="true"');
    expect(html).toContain('warning');
  });
});
