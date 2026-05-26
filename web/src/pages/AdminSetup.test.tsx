import { screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '../test/helpers';

vi.mock('../hooks/useSetup', () => ({
	useRerunStatus: vi.fn().mockReturnValue({
		data: {
			license: {
				license_type: 'pro',
				status: 'active',
				expires_at: '2030-01-01T00:00:00Z',
				company_name: 'Acme Inc',
			},
		},
		isLoading: false,
	}),
	useRerunConfigureSMTP: vi.fn().mockReturnValue({
		mutate: vi.fn(),
		isPending: false,
		isError: false,
		isSuccess: false,
		error: null,
	}),
	useRerunConfigureOIDC: vi.fn().mockReturnValue({
		mutate: vi.fn(),
		isPending: false,
		isError: false,
		isSuccess: false,
		error: null,
	}),
	useRerunUpdateLicense: vi.fn().mockReturnValue({
		mutate: vi.fn(),
		isPending: false,
		isError: false,
		isSuccess: false,
		error: null,
	}),
}));

import { AdminSetup } from './AdminSetup';

describe('AdminSetup page', () => {
	it('renders the title', () => {
		renderWithProviders(<AdminSetup />);
		expect(screen.getByText('Server Setup')).toBeInTheDocument();
	});

	it('renders license details when available', () => {
		renderWithProviders(<AdminSetup />);
		expect(screen.getByText('Current License')).toBeInTheDocument();
		expect(screen.getByText('Acme Inc')).toBeInTheDocument();
	});
});
