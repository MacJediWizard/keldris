import { screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '../test/helpers';

vi.mock('../hooks/useBackupHookTemplates', () => ({
	useBackupHookTemplates: vi.fn().mockReturnValue({
		data: [
			{
				id: 't1',
				name: 'Postgres dump',
				description: 'Dumps a postgres database before backup',
				service_type: 'postgresql',
				icon: 'database',
				visibility: 'built_in',
				tags: ['db'],
				variables: [],
				scripts: {},
				usage_count: 0,
			},
		],
		isLoading: false,
	}),
	useCreateBackupHookTemplate: vi.fn().mockReturnValue({
		mutateAsync: vi.fn(),
		isPending: false,
	}),
	useUpdateBackupHookTemplate: vi.fn().mockReturnValue({
		mutateAsync: vi.fn(),
		isPending: false,
	}),
	useDeleteBackupHookTemplate: vi.fn().mockReturnValue({
		mutateAsync: vi.fn(),
		isPending: false,
	}),
	useApplyBackupHookTemplate: vi.fn().mockReturnValue({
		mutateAsync: vi.fn(),
		isPending: false,
	}),
}));

vi.mock('../hooks/useSchedules', () => ({
	useSchedules: vi.fn().mockReturnValue({
		data: [],
		isLoading: false,
	}),
}));

import { BackupHookTemplates } from './BackupHookTemplates';

describe('BackupHookTemplates page', () => {
	it('renders the title', () => {
		renderWithProviders(<BackupHookTemplates />);
		expect(screen.getByText('Backup Hook Templates')).toBeInTheDocument();
	});

	it('renders templates from data', () => {
		renderWithProviders(<BackupHookTemplates />);
		expect(screen.getByText('Postgres dump')).toBeInTheDocument();
	});
});
