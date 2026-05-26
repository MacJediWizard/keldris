import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';

vi.mock('../../hooks/useSearch', () => ({
	useGroupedSearch: vi.fn(() => ({ data: undefined, isLoading: false })),
	useRecentSearches: vi.fn(() => ({ data: [] })),
	useSearchSuggestions: vi.fn(() => ({ data: [] })),
	useSaveRecentSearch: vi.fn(() => ({ mutateAsync: vi.fn() })),
	useDeleteRecentSearch: vi.fn(() => ({ mutateAsync: vi.fn() })),
	useClearRecentSearches: vi.fn(() => ({ mutateAsync: vi.fn() })),
}));

import { GlobalSearchBar } from './GlobalSearchBar';

function withRouter(ui: React.ReactNode) {
	return <MemoryRouter>{ui}</MemoryRouter>;
}

describe('GlobalSearchBar', () => {
	it('renders search input', () => {
		render(withRouter(<GlobalSearchBar />));
		expect(screen.getByRole('combobox')).toBeDefined();
	});

	it('uses custom placeholder', () => {
		render(withRouter(<GlobalSearchBar placeholder="Find anything" />));
		expect(screen.getByPlaceholderText('Find anything')).toBeDefined();
	});
});
