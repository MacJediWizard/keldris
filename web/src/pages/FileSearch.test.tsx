import { screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '../test/helpers';

vi.mock('../hooks/useAgents', () => ({
	useAgents: vi.fn().mockReturnValue({
		data: [{ id: 'a1', hostname: 'host-1' }],
	}),
}));

vi.mock('../hooks/useRepositories', () => ({
	useRepositories: vi.fn().mockReturnValue({
		data: [{ id: 'r1', name: 'repo-1' }],
	}),
}));

vi.mock('../hooks/useFileSearch', () => ({
	useFileSearch: vi.fn().mockReturnValue({
		data: undefined,
		isLoading: false,
		isError: false,
		refetch: vi.fn(),
	}),
}));

vi.mock('../hooks/useRestore', () => ({
	useCreateRestore: vi.fn().mockReturnValue({
		mutate: vi.fn(),
		isPending: false,
	}),
}));

import { FileSearch } from './FileSearch';

describe('FileSearch page', () => {
	it('renders the title', () => {
		renderWithProviders(<FileSearch />);
		expect(screen.getByText('File Search')).toBeInTheDocument();
	});

	it('shows empty state before any search', () => {
		renderWithProviders(<FileSearch />);
		expect(
			screen.getByText('Search for files across snapshots'),
		).toBeInTheDocument();
	});
});
