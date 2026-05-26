import { screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '../test/helpers';

vi.mock('../hooks/useAuth', () => ({
	useMe: vi.fn(),
}));

vi.mock('../hooks/useMigration', () => ({
	useGenerateExportKey: () => ({ mutateAsync: vi.fn(), isPending: false }),
	useMigrationExport: () => ({ mutateAsync: vi.fn(), isPending: false }),
	useValidateMigrationImport: () => ({
		mutateAsync: vi.fn(),
		isPending: false,
	}),
	useMigrationImport: () => ({ mutateAsync: vi.fn(), isPending: false }),
	downloadMigrationExport: vi.fn(),
	readFileAsText: vi.fn(),
}));

import { useMe } from '../hooks/useAuth';
import { MigrationSettings } from './MigrationSettings';

describe('MigrationSettings page', () => {
	beforeEach(() => vi.clearAllMocks());

	it('shows loading state', () => {
		vi.mocked(useMe).mockReturnValue({
			data: undefined,
			isLoading: true,
		} as ReturnType<typeof useMe>);
		renderWithProviders(<MigrationSettings />);
		const skeletons = document.querySelectorAll('.animate-pulse');
		expect(skeletons.length).toBeGreaterThan(0);
	});

	it('shows access denied for non-superuser', () => {
		vi.mocked(useMe).mockReturnValue({
			data: { is_superuser: false },
			isLoading: false,
		} as ReturnType<typeof useMe>);
		renderWithProviders(<MigrationSettings />);
		expect(screen.getByText('Access Denied')).toBeInTheDocument();
	});

	it('renders title and export tab for superuser', () => {
		vi.mocked(useMe).mockReturnValue({
			data: { is_superuser: true },
			isLoading: false,
		} as ReturnType<typeof useMe>);
		renderWithProviders(<MigrationSettings />);
		expect(screen.getByText('Migration')).toBeInTheDocument();
		expect(screen.getByText('Export Configuration')).toBeInTheDocument();
	});
});
