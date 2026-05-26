import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import type { SavedFilter } from '../../lib/types';

vi.mock('../../hooks/useAuth', () => ({
	useMe: vi.fn(),
}));

vi.mock('../../hooks/useSavedFilters', () => ({
	useSavedFilters: vi.fn(),
	useDeleteSavedFilter: vi.fn(),
	useUpdateSavedFilter: vi.fn(),
}));

import { useMe } from '../../hooks/useAuth';
import {
	useDeleteSavedFilter,
	useSavedFilters,
	useUpdateSavedFilter,
} from '../../hooks/useSavedFilters';
import { SavedFiltersDropdown } from './SavedFiltersDropdown';

function makeFilter(overrides: Partial<SavedFilter> = {}): SavedFilter {
	return {
		id: 'f1',
		user_id: 'u1',
		name: 'My filter',
		entity_type: 'agents',
		filters: {},
		shared: false,
		is_default: false,
		created_at: '',
		updated_at: '',
		...overrides,
	} as SavedFilter;
}

function setMocks(filters: SavedFilter[] | undefined, userId = 'u1') {
	vi.mocked(useMe).mockReturnValue({ data: { id: userId } } as never);
	vi.mocked(useSavedFilters).mockReturnValue({
		data: filters,
		isLoading: false,
	} as never);
	vi.mocked(useDeleteSavedFilter).mockReturnValue({
		mutateAsync: vi.fn(),
	} as never);
	vi.mocked(useUpdateSavedFilter).mockReturnValue({
		mutateAsync: vi.fn(),
	} as never);
}

describe('SavedFiltersDropdown', () => {
	it('renders nothing when no filters', () => {
		setMocks([]);
		const { container } = render(
			<SavedFiltersDropdown entityType="agents" onApplyFilter={() => {}} />,
		);
		expect(container.firstChild).toBeNull();
	});

	it('renders nothing when filters undefined', () => {
		setMocks(undefined);
		const { container } = render(
			<SavedFiltersDropdown entityType="agents" onApplyFilter={() => {}} />,
		);
		expect(container.firstChild).toBeNull();
	});

	it('renders dropdown trigger when filters present', () => {
		setMocks([makeFilter()]);
		render(
			<SavedFiltersDropdown entityType="agents" onApplyFilter={() => {}} />,
		);
		expect(screen.getByRole('button', { name: /Saved Filters/ })).toBeDefined();
	});

	it('opens menu and shows My Filters section', async () => {
		setMocks([makeFilter({ name: 'Mine' })]);
		const user = userEvent.setup();
		render(
			<SavedFiltersDropdown entityType="agents" onApplyFilter={() => {}} />,
		);
		await user.click(screen.getByRole('button', { name: /Saved Filters/ }));
		expect(screen.getByText('My Filters')).toBeDefined();
		expect(screen.getByText('Mine')).toBeDefined();
	});

	it('shows Shared Filters section when other-owned filters are shared', async () => {
		setMocks([
			makeFilter({ id: 'a', name: 'Mine' }),
			makeFilter({
				id: 'b',
				user_id: 'someone-else',
				name: 'Theirs',
				shared: true,
			}),
		]);
		const user = userEvent.setup();
		render(
			<SavedFiltersDropdown entityType="agents" onApplyFilter={() => {}} />,
		);
		await user.click(screen.getByRole('button', { name: /Saved Filters/ }));
		expect(screen.getByText('Shared Filters')).toBeDefined();
		expect(screen.getByText('Theirs')).toBeDefined();
	});

	it('applies a filter when clicked', async () => {
		const onApplyFilter = vi.fn();
		const filter = makeFilter({ filters: { status: 'active' } });
		setMocks([filter]);
		const user = userEvent.setup();
		render(
			<SavedFiltersDropdown
				entityType="agents"
				onApplyFilter={onApplyFilter}
			/>,
		);
		await user.click(screen.getByRole('button', { name: /Saved Filters/ }));
		await user.click(screen.getByText('My filter'));
		expect(onApplyFilter).toHaveBeenCalledWith({ status: 'active' });
	});
});
