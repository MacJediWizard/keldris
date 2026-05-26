import { screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '../test/helpers';

vi.mock('../hooks/useServerLogs', () => ({
	useServerLogs: vi.fn().mockReturnValue({
		data: {
			logs: [
				{
					timestamp: '2024-01-01T00:00:00Z',
					level: 'info',
					component: 'api',
					message: 'Server started',
					fields: null,
				},
			],
			total_count: 1,
		},
		isLoading: false,
		isError: false,
		error: null,
		refetch: vi.fn(),
	}),
	useServerLogComponents: vi.fn().mockReturnValue({
		data: ['api', 'scheduler'],
	}),
	useExportServerLogsCsv: vi.fn().mockReturnValue({
		mutate: vi.fn(),
		isPending: false,
	}),
	useExportServerLogsJson: vi.fn().mockReturnValue({
		mutate: vi.fn(),
		isPending: false,
	}),
	useClearServerLogs: vi.fn().mockReturnValue({
		mutate: vi.fn(),
		isPending: false,
	}),
}));

import { AdminLogs } from './AdminLogs';

describe('AdminLogs page', () => {
	it('renders the title', () => {
		renderWithProviders(<AdminLogs />);
		expect(screen.getByText('Server Logs')).toBeInTheDocument();
	});

	it('renders log entries from data', () => {
		renderWithProviders(<AdminLogs />);
		expect(screen.getByText('Server started')).toBeInTheDocument();
	});
});
