import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it, vi } from 'vitest';
import { Circle } from 'lucide-react';

import {
  PosButton,
  PosInlineStatusBadge,
  PosQuantityStepper,
  PosSearchInput,
  PosTabs,
} from './index';

function asElement(node: React.ReactNode) {
  return node as React.ReactElement<Record<string, any>>;
}

function renderPrimitive(node: React.ReactNode | null) {
  return asElement(node);
}

describe('shared POS UI primitives', () => {
  it('renders PosButton variant classes and exposes click handler when enabled', () => {
    const onClick = vi.fn();
    const button = renderPrimitive(PosButton({ variant: 'danger', onClick, children: 'Run' }));

    expect(button.props.className).toContain('var(--pos-status-danger)');
    button.props.onClick?.();
    expect(onClick).toHaveBeenCalledTimes(1);

    const disabledHtml = renderToStaticMarkup(<PosButton disabled>Locked</PosButton>);
    expect(disabledHtml).toContain('disabled=""');
    expect(disabledHtml).toContain('cursor-not-allowed');
  });

  it('renders PosTabs active tab and calls onChange', () => {
    const onChange = vi.fn();
    const tabs = renderPrimitive(PosTabs({
      id: 'test-tabs',
      activeId: 'second',
      onChange,
      items: [
        { id: 'first', label: 'First' },
        { id: 'second', label: 'Second', count: 2 },
      ],
    }));
    const buttons = React.Children.toArray(tabs.props.children).map(asElement);

    expect(renderToStaticMarkup(tabs)).toContain('border-b-[var(--pos-action-primary)]');
    buttons[0].props.onClick?.();
    expect(onChange).toHaveBeenCalledWith('first');
  });

  it('keeps PosQuantityStepper min and max controls bounded', () => {
    const onChange = vi.fn();
    const atMin = renderPrimitive(PosQuantityStepper({ id: 'qty', value: 1, min: 1, max: 3, onChange }));
    const minChildren = React.Children.toArray(atMin.props.children).map(asElement);
    expect(minChildren[0].props.disabled).toBe(true);

    const atMax = renderPrimitive(PosQuantityStepper({ id: 'qty', value: 3, min: 1, max: 3, onChange }));
    const maxChildren = React.Children.toArray(atMax.props.children).map(asElement);
    expect(maxChildren[2].props.disabled).toBe(true);

    const middle = renderPrimitive(PosQuantityStepper({ id: 'qty', value: 2, min: 1, max: 3, onChange }));
    const middleChildren = React.Children.toArray(middle.props.children).map(asElement);
    middleChildren[0].props.onClick?.();
    middleChildren[2].props.onClick?.();
    expect(onChange).toHaveBeenNthCalledWith(1, 1);
    expect(onChange).toHaveBeenNthCalledWith(2, 3);
  });

  it('renders PosSearchInput as controlled input and clears value', () => {
    const onChange = vi.fn();
    const onClear = vi.fn();
    const input = renderPrimitive(PosSearchInput({
      id: 'search',
      value: 'latte',
      onChange,
      onClear,
      placeholder: 'Search',
      clearLabel: 'Clear',
    }));
    const children = React.Children.toArray(input.props.children).map(asElement);
    const textInput = children.find((child) => child.props.id === 'search');
    const clearButton = children.find((child) => child.props.label === 'Clear') ?? children[2];

    expect(textInput?.props.value).toBe('latte');
    asElement(clearButton).props.onClick?.();
    expect(onChange).toHaveBeenCalledWith('');
    expect(onClear).toHaveBeenCalledTimes(1);
  });

  it('renders PosInlineStatusBadge variant styling', () => {
    const html = renderToStaticMarkup(
      <PosInlineStatusBadge variant="success">
        <Circle />
        Online
      </PosInlineStatusBadge>,
    );

    expect(html).toContain('var(--pos-status-success)');
    expect(html).toContain('Online');
  });
});
