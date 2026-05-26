import { screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '../test/helpers';

vi.mock('../hooks/useConfigExport', () => ({
	useTemplates: vi.fn(),
	useCreateTemplate: () => ({ mutateAsync: vi.fn(), isPending: false }),
	useDeleteTemplate: () => ({ mutate: vi.fn(), isPending: false }),
	useUseTemplate: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

import { useTemplates } from '../hooks/useConfigExport';
import { Templates } from './Templates';

describe('Templates page', () => {
	beforeEach(() => vi.clearAllMocks());

	it('renders title', () => {
		vi.mocked(useTemplates).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useTemplates>);
		renderWithProviders(<Templates />);
		expect(screen.getByText('Templates')).toBeInTheDocument();
	});

	it('shows empty state when no templates', () => {
		vi.mocked(useTemplates).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useTemplates>);
		renderWithProviders(<Templates />);
		expect(screen.getByText('No templates yet')).toBeInTheDocument();
	});

	it('renders template rows from data', () => {
		vi.mocked(useTemplates).mockReturnValue({
			data: [
				{
					id: 'tpl-1',
					name: 'Web Server Backup',
					description: 'Nginx + content',
					type: 'schedule',
					visibility: 'organization',
					tags: ['web', 'production'],
					config: {},
					usage_count: 3,
					created_at: '2024-01-01T00:00:00Z',
				},
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useTemplates>);
		renderWithProviders(<Templates />);
		expect(screen.getByText('Web Server Backup')).toBeInTheDocument();
		expect(screen.getByText('Nginx + content')).toBeInTheDocument();
	});
});
