import { screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '../test/helpers';

vi.mock('../hooks/useAuth', () => ({
	useMe: vi.fn().mockReturnValue({
		data: { id: 'u1', current_org_id: 'org1', current_org_role: 'admin' },
	}),
}));

vi.mock('../hooks/useBranding', () => ({
	useBranding: vi.fn().mockReturnValue({
		data: {
			id: 'b1',
			org_id: 'org1',
			enabled: true,
			product_name: 'CustomKeldris',
			company_name: 'Acme Inc',
			logo_url: '',
			logo_dark_url: '',
			favicon_url: '',
			primary_color: '#4f46e5',
			secondary_color: '#64748b',
			accent_color: '#06b6d4',
			support_url: '',
			support_email: '',
			privacy_url: '',
			terms_url: '',
			footer_text: '',
			login_title: '',
			login_subtitle: '',
			login_bg_url: '',
			hide_powered_by: false,
			custom_css: '',
			created_at: '2024-01-01T00:00:00Z',
			updated_at: '2024-01-01T00:00:00Z',
		},
		isLoading: false,
		isError: false,
		error: null,
	}),
	useUpdateBranding: vi.fn().mockReturnValue({
		mutateAsync: vi.fn(),
		isPending: false,
	}),
}));

import { BrandingSettings } from './BrandingSettings';

describe('BrandingSettings page', () => {
	it('renders the title', () => {
		renderWithProviders(<BrandingSettings />);
		expect(screen.getByText('Branding')).toBeInTheDocument();
	});

	it('renders product identity section with current values', () => {
		renderWithProviders(<BrandingSettings />);
		expect(screen.getByText('Product Identity')).toBeInTheDocument();
		expect(screen.getByDisplayValue('CustomKeldris')).toBeInTheDocument();
	});
});
