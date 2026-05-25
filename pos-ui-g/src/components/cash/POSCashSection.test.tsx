import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const usePOSMock = vi.fn();

vi.mock('../../context/POSContext', () => ({
  usePOS: () => usePOSMock(),
}));

describe('POSCashSection', () => {
  beforeEach(() => {
    usePOSMock.mockReset();
  });

  it('offers opening the employee shift before the cash session', async () => {
    usePOSMock.mockReturnValue({
      currentOperator: null,
      openEmployeeShift: vi.fn(),
      closeEmployeeShift: vi.fn(),
      cashSession: null,
      openCashSession: vi.fn(),
      closeCashSession: vi.fn(),
      cashDrawerEvents: [],
      addCashDrawerEvent: vi.fn(),
      outboxCount: 0,
      syncOutbox: vi.fn(),
      syncStatus: 'online',
      logEvents: [],
    });

    const { POSCashSection } = await import('./POSCashSection');
    const html = renderToStaticMarkup(<POSCashSection />);

    expect(html).toContain('cash-open-shift-btn');
    expect(html).toContain('Открыть личную смену');
    expect(html).toContain('cash-open-session-btn');
  });
});
