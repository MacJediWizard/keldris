import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import {
	BulkSelectCheckbox,
	BulkSelectHeader,
	BulkSelectToolbar,
	SelectionIndicator,
} from './BulkSelect';

describe('BulkSelectCheckbox', () => {
	it('renders checked when checked=true', () => {
		const onChange = vi.fn();
		render(<BulkSelectCheckbox checked onChange={onChange} />);
		const cb = screen.getByRole('checkbox') as HTMLInputElement;
		expect(cb.checked).toBe(true);
	});

	it('fires onChange when clicked', () => {
		const onChange = vi.fn();
		render(<BulkSelectCheckbox checked={false} onChange={onChange} />);
		screen.getByRole('checkbox').click();
		expect(onChange).toHaveBeenCalledOnce();
	});

	it('disables when disabled=true', () => {
		render(<BulkSelectCheckbox checked={false} onChange={() => {}} disabled />);
		expect(screen.getByRole('checkbox')).toBeDisabled();
	});
});

describe('BulkSelectHeader', () => {
	it('renders Select all label when not all selected', () => {
		render(
			<BulkSelectHeader
				isAllSelected={false}
				isPartiallySelected={false}
				onToggleAll={() => {}}
				selectedCount={0}
				totalCount={5}
			/>,
		);
		expect(screen.getByRole('checkbox', { name: 'Select all' })).toBeDefined();
	});

	it('renders Deselect all label when all selected', () => {
		render(
			<BulkSelectHeader
				isAllSelected
				isPartiallySelected={false}
				onToggleAll={() => {}}
				selectedCount={5}
				totalCount={5}
			/>,
		);
		expect(
			screen.getByRole('checkbox', { name: 'Deselect all' }),
		).toBeDefined();
	});

	it('fires onToggleAll when clicked', () => {
		const onToggle = vi.fn();
		render(
			<BulkSelectHeader
				isAllSelected={false}
				isPartiallySelected={false}
				onToggleAll={onToggle}
				selectedCount={0}
				totalCount={5}
			/>,
		);
		screen.getByRole('checkbox').click();
		expect(onToggle).toHaveBeenCalledOnce();
	});

	it('disables when totalCount=0', () => {
		render(
			<BulkSelectHeader
				isAllSelected={false}
				isPartiallySelected={false}
				onToggleAll={() => {}}
				selectedCount={0}
				totalCount={0}
			/>,
		);
		expect(screen.getByRole('checkbox')).toBeDisabled();
	});
});

describe('SelectionIndicator', () => {
	it('renders nothing when selectedCount=0', () => {
		const { container } = render(
			<SelectionIndicator selectedCount={0} totalCount={5} />,
		);
		expect(container.firstChild).toBeNull();
	});

	it('renders count with default label', () => {
		render(<SelectionIndicator selectedCount={3} totalCount={5} />);
		expect(screen.getByText('3 of 5 items selected')).toBeDefined();
	});

	it('renders singular label when selectedCount=1', () => {
		render(<SelectionIndicator selectedCount={1} totalCount={5} />);
		expect(screen.getByText('1 of 5 item selected')).toBeDefined();
	});

	it('uses custom itemLabel', () => {
		render(
			<SelectionIndicator selectedCount={2} totalCount={5} itemLabel="agent" />,
		);
		expect(screen.getByText('2 of 5 agents selected')).toBeDefined();
	});
});

describe('BulkSelectToolbar', () => {
	it('renders nothing when selectedCount=0', () => {
		const { container } = render(
			<BulkSelectToolbar
				selectedCount={0}
				totalCount={5}
				onSelectAll={() => {}}
				onDeselectAll={() => {}}
			/>,
		);
		expect(container.firstChild).toBeNull();
	});

	it('renders Select all link when not all selected', () => {
		render(
			<BulkSelectToolbar
				selectedCount={2}
				totalCount={5}
				onSelectAll={() => {}}
				onDeselectAll={() => {}}
			/>,
		);
		expect(screen.getByRole('button', { name: 'Select all 5' })).toBeDefined();
	});

	it('hides Select all link when all selected', () => {
		render(
			<BulkSelectToolbar
				selectedCount={5}
				totalCount={5}
				onSelectAll={() => {}}
				onDeselectAll={() => {}}
			/>,
		);
		expect(screen.queryByRole('button', { name: 'Select all 5' })).toBeNull();
	});

	it('fires onSelectAll when Select all clicked', () => {
		const onSelectAll = vi.fn();
		render(
			<BulkSelectToolbar
				selectedCount={2}
				totalCount={5}
				onSelectAll={onSelectAll}
				onDeselectAll={() => {}}
			/>,
		);
		screen.getByRole('button', { name: 'Select all 5' }).click();
		expect(onSelectAll).toHaveBeenCalledOnce();
	});

	it('fires onDeselectAll when Clear clicked', () => {
		const onDeselectAll = vi.fn();
		render(
			<BulkSelectToolbar
				selectedCount={2}
				totalCount={5}
				onSelectAll={() => {}}
				onDeselectAll={onDeselectAll}
			/>,
		);
		screen.getByRole('button', { name: 'Clear selection' }).click();
		expect(onDeselectAll).toHaveBeenCalledOnce();
	});

	it('renders children', () => {
		render(
			<BulkSelectToolbar
				selectedCount={2}
				totalCount={5}
				onSelectAll={() => {}}
				onDeselectAll={() => {}}
			>
				<span>child-action</span>
			</BulkSelectToolbar>,
		);
		expect(screen.getByText('child-action')).toBeDefined();
	});
});
