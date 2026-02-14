import { screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '../test/helpers';

vi.mock('../hooks/useBranding', () => ({
	useBranding: vi.fn().mockReturnValue({
		data: {
			id: 'branding-1',
			org_id: 'org-1',
			product_name: 'MyBackup',
			logo_url: 'https://example.com/logo.png',
			favicon_url: 'https://example.com/favicon.ico',
			primary_color: '#FF0000',
			secondary_color: '#00FF00',
			support_url: 'https://support.example.com',
			custom_css: '',
		},
		isLoading: false,
		error: null,
	}),
	useUpdateBranding: vi.fn().mockReturnValue({
		mutateAsync: vi.fn(),
		isPending: false,
		isSuccess: false,
		isError: false,
	}),
	useResetBranding: vi.fn().mockReturnValue({
		mutateAsync: vi.fn(),
		isPending: false,
	}),
}));

vi.mock('../hooks/useAuth', () => ({
	useMe: vi.fn().mockReturnValue({
		data: {
			id: 'user-1',
			email: 'admin@example.com',
			current_org_role: 'admin',
		},
	}),
}));

import { Branding } from './Branding';

describe('Branding page', () => {
	it('renders the page title', () => {
		renderWithProviders(<Branding />);
		expect(screen.getByText('White Label Branding')).toBeInTheDocument();
	});

	it('renders the page description', () => {
		renderWithProviders(<Branding />);
		expect(
			screen.getByText('Customize the appearance of your Keldris instance.'),
		).toBeInTheDocument();
	});

	it('renders product identity section', () => {
		renderWithProviders(<Branding />);
		expect(screen.getByText('Product Identity')).toBeInTheDocument();
	});

	it('renders logo and favicon section', () => {
		renderWithProviders(<Branding />);
		expect(screen.getByText('Logo & Favicon')).toBeInTheDocument();
	});

	it('renders brand colors section', () => {
		renderWithProviders(<Branding />);
		expect(screen.getByText('Brand Colors')).toBeInTheDocument();
	});

	it('renders custom CSS section', () => {
		renderWithProviders(<Branding />);
		expect(screen.getByText('Custom CSS')).toBeInTheDocument();
	});

	it('populates form fields from branding data', () => {
		renderWithProviders(<Branding />);
		const productNameInput = screen.getByLabelText(
			'Product Name',
		) as HTMLInputElement;
		expect(productNameInput.value).toBe('MyBackup');
	});

	it('shows save button for admin users', () => {
		renderWithProviders(<Branding />);
		expect(screen.getByText('Save Changes')).toBeInTheDocument();
	});

	it('shows reset button for admin users', () => {
		renderWithProviders(<Branding />);
		expect(screen.getByText('Reset to Defaults')).toBeInTheDocument();
	});
});
