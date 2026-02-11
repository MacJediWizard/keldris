import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { DropdownMenu } from './DropdownMenu';

const items = [
	{ label: 'Edit', onClick: vi.fn() },
	{ label: 'Delete', onClick: vi.fn(), variant: 'danger' as const },
	{ label: 'Disabled', onClick: vi.fn(), disabled: true },
];

describe('DropdownMenu', () => {
	it('renders trigger element', () => {
		render(<DropdownMenu trigger={<span>Actions</span>} items={items} />);
		expect(screen.getByText('Actions')).toBeInTheDocument();
	});

	it('does not show menu items by default', () => {
		render(<DropdownMenu trigger={<span>Actions</span>} items={items} />);
		expect(screen.queryByRole('menu')).not.toBeInTheDocument();
	});

	it('shows menu items when trigger is clicked', () => {
		render(<DropdownMenu trigger={<span>Actions</span>} items={items} />);
		fireEvent.click(screen.getByText('Actions'));
		expect(screen.getByRole('menu')).toBeInTheDocument();
		expect(screen.getByText('Edit')).toBeInTheDocument();
		expect(screen.getByText('Delete')).toBeInTheDocument();
	});

	it('calls item onClick and closes menu', () => {
		render(<DropdownMenu trigger={<span>Actions</span>} items={items} />);
		fireEvent.click(screen.getByText('Actions'));
		fireEvent.click(screen.getByText('Edit'));
		expect(items[0].onClick).toHaveBeenCalled();
		expect(screen.queryByRole('menu')).not.toBeInTheDocument();
	});

	it('applies danger styling to danger variant items', () => {
		render(<DropdownMenu trigger={<span>Actions</span>} items={items} />);
		fireEvent.click(screen.getByText('Actions'));
		expect(screen.getByText('Delete')).toHaveClass('text-red-600');
	});

	it('disables disabled items', () => {
		render(<DropdownMenu trigger={<span>Actions</span>} items={items} />);
		fireEvent.click(screen.getByText('Actions'));
		expect(screen.getByText('Disabled')).toBeDisabled();
	});

	it('closes when clicking outside', () => {
		render(
			<div>
				<span>Outside</span>
				<DropdownMenu trigger={<span>Actions</span>} items={items} />
			</div>,
		);
		fireEvent.click(screen.getByText('Actions'));
		expect(screen.getByRole('menu')).toBeInTheDocument();
		fireEvent.mouseDown(screen.getByText('Outside'));
		expect(screen.queryByRole('menu')).not.toBeInTheDocument();
	});
});
