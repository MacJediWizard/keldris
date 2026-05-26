import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import type { DockerStack } from '../../lib/types';
import { DockerStackSelector } from './DockerStackSelector';

const stacks = [
	{
		id: 'stack-1',
		agent_id: 'agent-1',
		name: 'web',
		compose_path: '/srv/web/docker-compose.yml',
		is_running: true,
	} as unknown as DockerStack,
];

describe('DockerStackSelector', () => {
	it('renders stack in select for matching agent', () => {
		render(
			<DockerStackSelector
				stacks={stacks}
				selectedStackId={null}
				onChange={vi.fn()}
				agentId="agent-1"
			/>,
		);
		expect(screen.getByRole('option', { name: /web/ })).toBeDefined();
	});

	it('renders loading state', () => {
		const { container } = render(
			<DockerStackSelector
				stacks={[]}
				selectedStackId={null}
				onChange={vi.fn()}
				agentId="agent-1"
				isLoading
			/>,
		);
		expect(container.firstChild).not.toBeNull();
	});

	it('renders empty state when no stacks for agent', () => {
		const { container } = render(
			<DockerStackSelector
				stacks={[]}
				selectedStackId={null}
				onChange={vi.fn()}
				agentId="agent-1"
			/>,
		);
		expect(container.firstChild).not.toBeNull();
	});

	it('renders agent-first message when agentId empty', () => {
		render(
			<DockerStackSelector
				stacks={[]}
				selectedStackId={null}
				onChange={vi.fn()}
				agentId=""
			/>,
		);
		expect(screen.getByText(/Select an agent first/)).toBeDefined();
	});
});
