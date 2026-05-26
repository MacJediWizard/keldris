import { screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '../test/helpers';

vi.mock('../hooks/useAnnouncements', () => ({
	useAnnouncements: vi.fn().mockReturnValue({
		data: [
			{
				id: 'a1',
				title: 'System maintenance',
				message: 'Scheduled downtime tonight',
				type: 'info',
				dismissible: true,
				active: true,
				starts_at: null,
				ends_at: null,
				created_at: '2024-01-01T00:00:00Z',
			},
		],
		isLoading: false,
		isError: false,
	}),
	useCreateAnnouncement: vi.fn().mockReturnValue({
		mutateAsync: vi.fn(),
		isPending: false,
	}),
	useUpdateAnnouncement: vi.fn().mockReturnValue({
		mutateAsync: vi.fn(),
		isPending: false,
	}),
	useDeleteAnnouncement: vi.fn().mockReturnValue({
		mutateAsync: vi.fn(),
		isPending: false,
	}),
}));

vi.mock('../hooks/useAuth', () => ({
	useMe: vi.fn().mockReturnValue({
		data: { id: 'u1', current_org_id: 'org1', current_org_role: 'admin' },
	}),
}));

import { Announcements } from './Announcements';

describe('Announcements page', () => {
	it('renders the title', () => {
		renderWithProviders(<Announcements />);
		expect(screen.getByText('Announcements')).toBeInTheDocument();
	});

	it('renders existing announcements', () => {
		renderWithProviders(<Announcements />);
		expect(screen.getByText('System maintenance')).toBeInTheDocument();
	});
});
