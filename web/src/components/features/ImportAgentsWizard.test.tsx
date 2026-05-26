import { render } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

vi.mock('../../hooks/useAgentGroups', () => ({
	useAgentGroups: vi.fn(() => ({ data: [] })),
}));

vi.mock('../../hooks/useAgentImport', () => ({
	useAgentImportPreview: vi.fn(() => ({ mutateAsync: vi.fn() })),
	useAgentImport: vi.fn(() => ({ mutateAsync: vi.fn(), isPending: false })),
	useAgentImportTemplate: vi.fn(() => ({ data: undefined })),
	useAgentImportTemplateDownload: vi.fn(() => ({ refetch: vi.fn() })),
	useAgentRegistrationScript: vi.fn(() => ({ refetch: vi.fn() })),
	useAgentImportTokensExport: vi.fn(() => ({ refetch: vi.fn() })),
}));

import { ImportAgentsWizard } from './ImportAgentsWizard';

describe('ImportAgentsWizard', () => {
	it('renders nothing when closed', () => {
		const { container } = render(
			<ImportAgentsWizard isOpen={false} onClose={vi.fn()} />,
		);
		expect(container.firstChild).toBeNull();
	});

	it('renders wizard when open', () => {
		const { container } = render(
			<ImportAgentsWizard isOpen onClose={vi.fn()} />,
		);
		expect(container.firstChild).not.toBeNull();
	});
});
