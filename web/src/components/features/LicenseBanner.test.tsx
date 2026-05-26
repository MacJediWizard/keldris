import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';

vi.mock('../../hooks/useLicenses', () => ({
	useLicensePurchaseUrl: vi.fn(),
	useLicenseWarnings: vi.fn(),
}));

import {
	useLicensePurchaseUrl,
	useLicenseWarnings,
} from '../../hooks/useLicenses';
import { LicenseBanner } from './LicenseBanner';

function setMocks({
	warnings,
	purchaseUrl,
	isLoading = false,
}: {
	warnings?: unknown;
	purchaseUrl?: string;
	isLoading?: boolean;
}) {
	vi.mocked(useLicenseWarnings).mockReturnValue({
		data: warnings ? { warnings } : undefined,
		isLoading,
	} as never);
	vi.mocked(useLicensePurchaseUrl).mockReturnValue({
		data: purchaseUrl ? { url: purchaseUrl } : undefined,
	} as never);
}

function renderInRouter(ui: React.ReactElement) {
	return render(<MemoryRouter>{ui}</MemoryRouter>);
}

describe('LicenseBanner', () => {
	it('renders nothing while loading', () => {
		setMocks({ isLoading: true });
		const { container } = renderInRouter(<LicenseBanner />);
		expect(container.firstChild).toBeNull();
	});

	it('renders nothing when warnings undefined', () => {
		setMocks({});
		const { container } = renderInRouter(<LicenseBanner />);
		expect(container.firstChild).toBeNull();
	});

	it('renders expired blocking modal when expired + no grace', () => {
		setMocks({
			warnings: {
				expiration: { is_expired: true, is_in_grace_period: false },
				limits: [],
			},
			purchaseUrl: 'https://buy.example.com',
		});
		renderInRouter(<LicenseBanner />);
		expect(screen.getByText('License Expired')).toBeDefined();
		expect(screen.getByText('Renew License')).toBeDefined();
	});

	it('renders grace period banner', () => {
		const grace = new Date(Date.now() + 5 * 86_400_000).toISOString();
		setMocks({
			warnings: {
				expiration: {
					is_expired: true,
					is_in_grace_period: true,
					grace_period_ends_at: grace,
				},
				limits: [],
			},
		});
		renderInRouter(<LicenseBanner />);
		expect(
			screen.getByText(/License Expired - Grace Period Active/),
		).toBeDefined();
	});

	it('renders expiring soon banner', () => {
		setMocks({
			warnings: {
				expiration: {
					is_expired: false,
					is_in_grace_period: false,
					days_until_expiry: 5,
				},
				limits: [],
			},
		});
		renderInRouter(<LicenseBanner />);
		expect(
			screen.getByText(/Your license will expire in 5 days/),
		).toBeDefined();
	});

	it('renders critical limits banner', () => {
		setMocks({
			warnings: {
				expiration: null,
				limits: [{ type: 'agents', percentage: 95, current: 95, limit: 100 }],
			},
		});
		renderInRouter(<LicenseBanner />);
		expect(screen.getByText('Limit Reached!')).toBeDefined();
		expect(screen.getByText(/95% of your agents limit/)).toBeDefined();
	});

	it('renders nothing when no actionable conditions', () => {
		setMocks({
			warnings: {
				expiration: {
					is_expired: false,
					is_in_grace_period: false,
					days_until_expiry: 120,
				},
				limits: [{ type: 'agents', percentage: 10, current: 1, limit: 10 }],
			},
		});
		const { container } = renderInRouter(<LicenseBanner />);
		expect(container.firstChild).toBeNull();
	});
});
