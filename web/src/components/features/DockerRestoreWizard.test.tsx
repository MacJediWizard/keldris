import { render } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

vi.mock('../../hooks/useAgents', () => ({
	useAgents: vi.fn(() => ({ data: [] })),
}));

vi.mock('../../hooks/useDockerRestore', () => ({
	useDockerRestores: vi.fn(() => ({ data: [] })),
	useDockerRestore: vi.fn(() => ({ data: undefined })),
	useCreateDockerRestore: vi.fn(() => ({
		mutateAsync: vi.fn(),
		isPending: false,
	})),
	useDockerRestorePreview: vi.fn(() => ({ mutateAsync: vi.fn() })),
	useDockerRestoreProgress: vi.fn(() => ({ data: undefined })),
	useContainersInSnapshot: vi.fn(() => ({ data: [] })),
	useVolumesInSnapshot: vi.fn(() => ({ data: [] })),
	useCancelDockerRestore: vi.fn(() => ({ mutateAsync: vi.fn() })),
}));

vi.mock('../../hooks/useSnapshots', () => ({
	useSnapshots: vi.fn(() => ({ data: [], isLoading: false })),
	useSnapshot: vi.fn(() => ({ data: undefined })),
	useSnapshotFiles: vi.fn(() => ({ data: [] })),
	useSnapshotCompare: vi.fn(() => ({ data: undefined })),
	useFileDiff: vi.fn(() => ({ data: undefined })),
}));

import { DockerRestoreWizard } from './DockerRestoreWizard';

describe('DockerRestoreWizard', () => {
	it('renders nothing when closed', () => {
		const { container } = render(
			<DockerRestoreWizard isOpen={false} onClose={vi.fn()} />,
		);
		expect(container.firstChild).toBeNull();
	});

	it('renders wizard when open', () => {
		const { container } = render(
			<DockerRestoreWizard isOpen onClose={vi.fn()} />,
		);
		expect(container.firstChild).not.toBeNull();
	});
});
