import { screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '../test/helpers';

vi.mock('../hooks/useChangelog', () => ({
	useChangelog: vi.fn().mockReturnValue({
		data: {
			current_version: '1.2.3',
			entries: [
				{
					version: '1.2.3',
					date: '2024-05-01',
					is_unreleased: false,
					added: ['New backup engine'],
					changed: [],
					deprecated: [],
					removed: [],
					fixed: [],
					security: [],
				},
			],
		},
		isLoading: false,
		error: null,
	}),
}));

import { Changelog } from './Changelog';

describe('Changelog page', () => {
	it('renders the title', () => {
		renderWithProviders(<Changelog />);
		expect(screen.getByText('Changelog')).toBeInTheDocument();
	});

	it('renders version entries', () => {
		renderWithProviders(<Changelog />);
		expect(screen.getByText('v1.2.3')).toBeInTheDocument();
	});
});
