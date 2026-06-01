import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const usePOSMock = vi.fn();

vi.mock('../../context/POSContext', () => ({
  usePOS: () => usePOSMock(),
}));

describe('POSFloorSection', () => {
  beforeEach(() => {
    usePOSMock.mockReset();
  });

  it('blocks the floor empty state until the employee shift is open', async () => {
    usePOSMock.mockReturnValue({
      tables: [],
      activeHallId: '',
      setActiveHallId: vi.fn(),
      setSelectedTableId: vi.fn(),
      createOrderForTable: vi.fn(),
      activeOrders: [],
      setCurrentSection: vi.fn(),
      currentOperator: null,
      halls: [],
    });

    const { POSFloorSection } = await import('./POSFloorSection');
    const html = renderToStaticMarkup(<POSFloorSection />);

    expect(html).toContain('Смена не открыта');
    expect(html).toContain('Открыть личную смену');
    expect(html).not.toContain('Нет столов');
  });
});
