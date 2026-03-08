import { screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '../test/helpers';

vi.mock('../hooks/useAirGap', () => ({
	useAirGapStatus: vi.fn().mockReturnValue({
		data: {
			airgap_mode: true,
			disable_update_checker: true,
			disable_telemetry: true,
			disable_external_links: true,
			license_valid: true,
		},
		isLoading: false,
		error: null,
	}),
	useUploadLicense: vi.fn().mockReturnValue({
		mutateAsync: vi.fn(),
		isPending: false,
		isSuccess: false,
	}),
	useLicenseStatus: vi.fn().mockReturnValue({
		data: {
			valid: true,
			type: 'enterprise',
			organization: 'Test Org',
			expires_at: '2025-12-31T00:00:00Z',
			airgap_mode: true,
		},
		isLoading: false,
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
		expect(screen.getByText('Update checker disabled')).toBeInTheDocument();
		expect(screen.getByText('Telemetry disabled')).toBeInTheDocument();
		expect(screen.getByText('External links disabled')).toBeInTheDocument();
	});

	it('shows license information', () => {
		renderWithProviders(<AirGapLicensePage />);
		expect(screen.getByText('Test Org')).toBeInTheDocument();
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
