import { render } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

vi.mock('../../hooks/useAgents', () => ({
	useAgents: vi.fn(() => ({ data: [] })),
}));

vi.mock('../../hooks/useRepositoryImport', () => ({
	useVerifyImportAccess: vi.fn(() => ({ mutateAsync: vi.fn() })),
	useImportPreview: vi.fn(() => ({ mutateAsync: vi.fn() })),
	useImportRepository: vi.fn(() => ({
		mutateAsync: vi.fn(),
		isPending: false,
	})),
}));

import { ImportRepositoryWizard } from './ImportRepositoryWizard';

describe('ImportRepositoryWizard', () => {
	it('renders nothing when closed', () => {
		const { container } = render(
			<ImportRepositoryWizard isOpen={false} onClose={vi.fn()} />,
		);
		expect(container.firstChild).toBeNull();
	});

	it('renders wizard when open', () => {
		const { container } = render(
			<ImportRepositoryWizard isOpen onClose={vi.fn()} />,
		);
		expect(container.firstChild).not.toBeNull();
	});
});
