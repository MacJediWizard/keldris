import { render } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

vi.mock('../../hooks/useBackupQueue', () => ({
	useBackupQueue: vi.fn(() => ({ data: undefined, isLoading: false })),
	useBackupQueueSummary: vi.fn(() => ({ data: undefined })),
	useCancelQueuedBackup: vi.fn(() => ({ mutate: vi.fn() })),
	useOrgConcurrency: vi.fn(() => ({ data: undefined })),
	useUpdateOrgConcurrency: vi.fn(() => ({ mutate: vi.fn() })),
	useAgentConcurrency: vi.fn(() => ({ data: undefined })),
	useUpdateAgentConcurrency: vi.fn(() => ({ mutate: vi.fn() })),
}));

import { BackupQueuePanel } from './BackupQueuePanel';

describe('BackupQueuePanel', () => {
	it('renders without crashing', () => {
		const { container } = render(<BackupQueuePanel />);
		expect(container.firstChild).not.toBeNull();
	});
});
