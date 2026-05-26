import { render } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

vi.mock('../../hooks/useAgents', () => ({
	useAgents: vi.fn(() => ({ data: [] })),
}));

vi.mock('../../hooks/useBackupCalendar', () => ({
	useBackupCalendar: vi.fn(() => ({ data: undefined, isLoading: false })),
}));

import { BackupCalendar, MiniBackupCalendar } from './BackupCalendar';

describe('BackupCalendar', () => {
	it('renders calendar without crashing', () => {
		const { container } = render(<BackupCalendar />);
		expect(container.firstChild).not.toBeNull();
	});
});

describe('MiniBackupCalendar', () => {
	it('renders mini calendar without crashing', () => {
		const { container } = render(<MiniBackupCalendar />);
		expect(container.firstChild).not.toBeNull();
	});
});
