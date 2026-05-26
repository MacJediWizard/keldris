import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

vi.mock('../../hooks/useAgents', () => ({
	useAgentLogs: vi.fn(),
}));

import { useAgentLogs } from '../../hooks/useAgents';
import { AgentLogViewer } from './AgentLogViewer';

function setLogs(logs: unknown[], isLoading = false) {
	vi.mocked(useAgentLogs).mockReturnValue({
		data: { logs, total: logs.length },
		isLoading,
		refetch: vi.fn(),
	} as never);
}

describe('AgentLogViewer', () => {
	it('renders loading state', () => {
		setLogs([], true);
		const { container } = render(<AgentLogViewer agentId="abc" />);
		expect(container.firstChild).not.toBeNull();
	});

	it('renders log entries', () => {
		setLogs([
			{
				id: '1',
				agent_id: 'abc',
				timestamp: new Date().toISOString(),
				level: 'info',
				message: 'Agent connected',
				component: 'main',
			},
		]);
		render(<AgentLogViewer agentId="abc" />);
		expect(screen.getByText('Agent connected')).toBeDefined();
	});

	it('renders empty state when no logs', () => {
		setLogs([]);
		const { container } = render(<AgentLogViewer agentId="abc" />);
		expect(container.firstChild).not.toBeNull();
	});
});
