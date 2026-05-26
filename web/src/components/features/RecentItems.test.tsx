import { fireEvent, render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';

vi.mock('../../hooks/useRecentItems', () => ({
	useRecentItems: vi.fn(),
	useDeleteRecentItem: vi.fn(),
	useClearRecentItems: vi.fn(),
}));

import {
	useClearRecentItems,
	useDeleteRecentItem,
	useRecentItems,
} from '../../hooks/useRecentItems';
import { RecentItemsDropdown } from './RecentItems';

function setItems(items: unknown[] | undefined, isLoading = false) {
	vi.mocked(useRecentItems).mockReturnValue({
		data: items,
		isLoading,
	} as never);
	vi.mocked(useDeleteRecentItem).mockReturnValue({
		mutateAsync: vi.fn(),
	} as never);
	vi.mocked(useClearRecentItems).mockReturnValue({
		mutateAsync: vi.fn(),
	} as never);
}

function withRouter(ui: React.ReactNode) {
	return <MemoryRouter>{ui}</MemoryRouter>;
}

describe('RecentItemsDropdown', () => {
	it('renders the trigger button', () => {
		setItems([]);
		render(withRouter(<RecentItemsDropdown />));
		expect(screen.getByRole('button', { name: 'Recent Items' })).toBeDefined();
	});

	it('opens dropdown on click and shows empty state', () => {
		setItems([]);
		render(withRouter(<RecentItemsDropdown />));
		fireEvent.click(screen.getByRole('button', { name: 'Recent Items' }));
		expect(screen.getByText('No recent items')).toBeDefined();
	});

	it('renders grouped items by type', () => {
		setItems([
			{
				id: '1',
				item_type: 'agent',
				item_name: 'web-prod-01',
				page_path: '/agents/1',
				viewed_at: new Date(Date.now() - 60_000).toISOString(),
			},
			{
				id: '2',
				item_type: 'repository',
				item_name: 's3-backup',
				page_path: '/repositories/2',
				viewed_at: new Date().toISOString(),
			},
		]);
		render(withRouter(<RecentItemsDropdown />));
		fireEvent.click(screen.getByRole('button', { name: 'Recent Items' }));
		expect(screen.getByText('Agents')).toBeDefined();
		expect(screen.getByText('web-prod-01')).toBeDefined();
		expect(screen.getByText('s3-backup')).toBeDefined();
	});

	it('disables trigger while loading', () => {
		setItems(undefined, true);
		render(withRouter(<RecentItemsDropdown />));
		expect(screen.getByRole('button', { name: 'Recent Items' })).toBeDisabled();
	});
});
