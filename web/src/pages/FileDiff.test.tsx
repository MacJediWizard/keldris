import { screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '../test/helpers';

vi.mock('../hooks/useAgents', () => ({
	useAgents: vi.fn().mockReturnValue({
		data: [],
	}),
}));

vi.mock('../hooks/useSnapshots', () => ({
	useSnapshot: vi.fn().mockReturnValue({
		data: undefined,
	}),
	useFileDiff: vi.fn().mockReturnValue({
		data: undefined,
		isLoading: false,
		isError: false,
		error: null,
	}),
}));

vi.mock('../components/features/DiffViewer', () => ({
	DiffViewer: () => null,
}));

import { FileDiff } from './FileDiff';

describe('FileDiff page', () => {
	it('renders the title', () => {
		renderWithProviders(<FileDiff />);
		expect(
			screen.getByRole('heading', { name: 'File Diff' }),
		).toBeInTheDocument();
	});

	it('shows missing parameters warning when query params are absent', () => {
		renderWithProviders(<FileDiff />);
		expect(screen.getByText('Missing parameters')).toBeInTheDocument();
	});
});
