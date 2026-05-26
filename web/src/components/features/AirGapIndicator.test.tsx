import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';

vi.mock('../../hooks/useAirGap', () => ({
	useAirGap: vi.fn(),
	useLicenseStatus: vi.fn(),
}));

import { useAirGap, useLicenseStatus } from '../../hooks/useAirGap';
import {
	AirGapIndicator,
	AirGapStatusCard,
	ExternalLink,
} from './AirGapIndicator';

function setAirGap(value: Record<string, unknown>) {
	vi.mocked(useAirGap).mockReturnValue(value as never);
}

function setLicense(value: unknown, isLoading = false) {
	vi.mocked(useLicenseStatus).mockReturnValue({
		data: value,
		isLoading,
	} as never);
}

function withRouter(ui: React.ReactNode) {
	return <MemoryRouter>{ui}</MemoryRouter>;
}

describe('AirGapIndicator', () => {
	it('renders nothing while loading', () => {
		setAirGap({ isAirGapMode: true, licenseValid: true, isLoading: true });
		const { container } = render(withRouter(<AirGapIndicator />));
		expect(container.firstChild).toBeNull();
	});

	it('renders nothing when not air-gapped', () => {
		setAirGap({ isAirGapMode: false, licenseValid: true, isLoading: false });
		const { container } = render(withRouter(<AirGapIndicator />));
		expect(container.firstChild).toBeNull();
	});

	it('renders Air-Gapped badge when in air-gap mode', () => {
		setAirGap({ isAirGapMode: true, licenseValid: true, isLoading: false });
		render(withRouter(<AirGapIndicator />));
		expect(screen.getByText('Air-Gapped')).toBeDefined();
	});

	it('shows Check License chip when invalid + showDetails', () => {
		setAirGap({ isAirGapMode: true, licenseValid: false, isLoading: false });
		render(withRouter(<AirGapIndicator showDetails />));
		expect(screen.getByText('Check License')).toBeDefined();
	});
});

describe('AirGapStatusCard', () => {
	it('shows loading skeleton', () => {
		setAirGap({ isAirGapMode: true, isLoading: true });
		setLicense(undefined, true);
		const { container } = render(withRouter(<AirGapStatusCard />));
		expect(container.querySelector('.animate-pulse')).not.toBeNull();
	});

	it('renders Connected Mode when not air-gapped', () => {
		setAirGap({ isAirGapMode: false, isLoading: false });
		setLicense({}, false);
		render(withRouter(<AirGapStatusCard />));
		expect(screen.getByText('Connected Mode')).toBeDefined();
	});

	it('renders Air-Gapped Mode card with license valid badge', () => {
		setAirGap({
			isAirGapMode: true,
			disableExternalLinks: true,
			offlineDocsVersion: '1.0',
			isLoading: false,
		});
		setLicense({ valid: true, type: 'enterprise' }, false);
		render(withRouter(<AirGapStatusCard />));
		expect(screen.getByText('Air-Gapped Mode')).toBeDefined();
		expect(screen.getByText('Valid')).toBeDefined();
		expect(screen.getByText('Blocked')).toBeDefined();
	});
});

describe('ExternalLink', () => {
	it('renders anchor with target=_blank when not blocked', () => {
		setAirGap({ shouldBlockExternalLink: () => false });
		render(<ExternalLink href="https://example.com">External</ExternalLink>);
		const a = screen.getByText('External').closest('a');
		expect(a?.getAttribute('target')).toBe('_blank');
		expect(a?.getAttribute('rel')).toBe('noopener noreferrer');
	});

	it('renders inert span when blocked', () => {
		setAirGap({ shouldBlockExternalLink: () => true });
		render(<ExternalLink href="https://example.com">External</ExternalLink>);
		expect(screen.getByText('External').tagName).toBe('SPAN');
	});
});
