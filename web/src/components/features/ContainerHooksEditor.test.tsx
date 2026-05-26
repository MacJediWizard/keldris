import { render } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

vi.mock('../../hooks/useContainerHooks', () => ({
	useContainerHooks: vi.fn(() => ({ data: [], isLoading: false })),
	useContainerHook: vi.fn(() => ({ data: undefined })),
	useCreateContainerHook: vi.fn(() => ({ mutateAsync: vi.fn() })),
	useUpdateContainerHook: vi.fn(() => ({ mutateAsync: vi.fn() })),
	useDeleteContainerHook: vi.fn(() => ({ mutateAsync: vi.fn() })),
	useContainerHookTemplates: vi.fn(() => ({ data: [] })),
	useContainerHookExecutions: vi.fn(() => ({ data: [] })),
}));

import { ContainerHooksEditor } from './ContainerHooksEditor';

describe('ContainerHooksEditor', () => {
	it('renders without crashing', () => {
		const { container } = render(<ContainerHooksEditor scheduleId="sched-1" />);
		expect(container.firstChild).not.toBeNull();
	});
});
