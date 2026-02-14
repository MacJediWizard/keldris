import { screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '../test/helpers';

vi.mock('../hooks/useAirGap', () => ({
	useAirGapStatus: vi.fn().mockReturnValue({
		data: {
			enabled: true,
			disabled_features: [
				{ name: 'auto_update', reason: 'Requires internet access' },
				{
					name: 'external_webhooks',
					reason: 'External webhooks require internet access',
				},
			],
			license: {
				customer_id: 'cust-123',
				tier: 'enterprise',
				expires_at: '2025-12-31T00:00:00Z',
				issued_at: '2024-01-01T00:00:00Z',
				valid: true,
			},
		},
		isLoading: false,
		error: null,
	}),
	useUploadLicense: vi.fn().mockReturnValue({
		mutateAsync: vi.fn(),
		isPending: false,
		isSuccess: false,
	}),
}));

vi.mock('../hooks/useLocale', () => ({
	useLocale: vi.fn().mockReturnValue({
		t: (key: string) => {
			const translations: Record<string, string> = {
				'airGap.title': 'Air-Gap Mode',
				'airGap.subtitle': 'Manage air-gap deployment settings',
				'airGap.modeStatus': 'Mode Status',
				'airGap.airGapEnabled': 'Enabled',
				'airGap.airGapDisabled': 'Disabled',
				'airGap.disabledFeatures': 'Disabled Features',
				'airGap.notInAirGapMode': 'Not in air-gap mode',
				'airGap.offlineLicense': 'Offline License',
				'airGap.noLicense': 'No license uploaded',
				'airGap.uploadLicense': 'Upload License',
				'airGap.uploading': 'Uploading...',
				'airGap.licenseUploaded': 'License uploaded successfully',
				'airGap.licenseValid': 'Valid',
				'airGap.licenseExpired': 'Expired',
				'airGap.customerIdLabel': 'Customer ID',
				'airGap.tierLabel': 'Tier',
				'airGap.expiresLabel': 'Expires',
				'airGap.issuedLabel': 'Issued',
				'airGap.failedToLoadStatus': 'Failed to load status',
				'common.status': 'Status',
				'errors.generic': 'An error occurred',
			};
			return translations[key] || key;
		},
		formatDateTime: (d: string) => d || 'N/A',
	}),
}));

import AirGapLicensePage from './AirGapLicense';

describe('AirGapLicense page', () => {
	it('renders the page title', () => {
		renderWithProviders(<AirGapLicensePage />);
		expect(screen.getByText('Air-Gap Mode')).toBeInTheDocument();
	});

	it('renders the subtitle', () => {
		renderWithProviders(<AirGapLicensePage />);
		expect(
			screen.getByText('Manage air-gap deployment settings'),
		).toBeInTheDocument();
	});

	it('shows air-gap enabled badge', () => {
		renderWithProviders(<AirGapLicensePage />);
		expect(screen.getByText('Enabled')).toBeInTheDocument();
	});

	it('renders disabled features list', () => {
		renderWithProviders(<AirGapLicensePage />);
		expect(screen.getByText('auto update')).toBeInTheDocument();
		expect(screen.getByText('external webhooks')).toBeInTheDocument();
	});

	it('shows license information', () => {
		renderWithProviders(<AirGapLicensePage />);
		expect(screen.getByText('cust-123')).toBeInTheDocument();
		expect(screen.getByText('enterprise')).toBeInTheDocument();
	});

	it('shows valid license badge', () => {
		renderWithProviders(<AirGapLicensePage />);
		expect(screen.getByText('Valid')).toBeInTheDocument();
	});

	it('renders the upload license button', () => {
		renderWithProviders(<AirGapLicensePage />);
		expect(screen.getByText('Upload License')).toBeInTheDocument();
	});
});
