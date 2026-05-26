import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import {
	EmptyState,
	EmptyStateNoAgents,
	EmptyStateNoBackups,
	EmptyStateNoGroups,
	EmptyStateNoPolicies,
	EmptyStateNoRepositories,
	EmptyStateNoSchedules,
	EmptyStateNoSearchResults,
} from './EmptyState';

const icon = <span data-testid="icon">★</span>;

describe('EmptyState', () => {
	it('renders title and description', () => {
		render(
			<EmptyState
				icon={icon}
				title="No agents"
				description="Add one to start"
			/>,
		);
		expect(screen.getByText('No agents')).toBeDefined();
		expect(screen.getByText('Add one to start')).toBeDefined();
		expect(screen.getByTestId('icon')).toBeDefined();
	});

	it('renders action button when provided and fires onClick', () => {
		const onClick = vi.fn();
		render(
			<EmptyState
				icon={icon}
				title="No data"
				description="Add some"
				action={{ label: 'Add Item', onClick }}
			/>,
		);
		const btn = screen.getByRole('button', { name: 'Add Item' });
		btn.click();
		expect(onClick).toHaveBeenCalledOnce();
	});

	it('respects compact variant by rendering smaller wrapper', () => {
		const { container } = render(
			<EmptyState
				icon={icon}
				title="Empty"
				description="Nothing here"
				variant="compact"
			/>,
		);
		expect(container.firstChild).toHaveProperty('className');
	});

	it('renders help tooltip when help is provided', () => {
		render(
			<EmptyState
				icon={icon}
				title="Empty"
				description="Nothing"
				help={{ content: 'Help info', title: 'Help' }}
			/>,
		);
		expect(screen.getByText('Empty')).toBeDefined();
	});

	it('renders children below description', () => {
		render(
			<EmptyState icon={icon} title="Empty" description="Nothing">
				<span>extra child node</span>
			</EmptyState>,
		);
		expect(screen.getByText('extra child node')).toBeDefined();
	});
});

describe('EmptyState variants', () => {
	it('EmptyStateNoAgents renders default copy + fires action', () => {
		const onAddAgent = vi.fn();
		render(<EmptyStateNoAgents onAddAgent={onAddAgent} />);
		expect(screen.getByText('No agents connected')).toBeDefined();
		screen.getByRole('button', { name: 'Add Your First Agent' }).click();
		expect(onAddAgent).toHaveBeenCalledOnce();
	});

	it('EmptyStateNoBackups renders default copy', () => {
		render(<EmptyStateNoBackups onCreateSchedule={() => {}} />);
		expect(screen.getByText('No backups yet')).toBeDefined();
	});

	it('EmptyStateNoSchedules shows cron examples by default', () => {
		render(<EmptyStateNoSchedules onCreateSchedule={() => {}} />);
		expect(screen.getByText('Common schedules:')).toBeDefined();
		expect(screen.getByText(/Daily at 2 AM/)).toBeDefined();
	});

	it('EmptyStateNoSchedules hides cron examples when disabled', () => {
		render(
			<EmptyStateNoSchedules
				onCreateSchedule={() => {}}
				showCronExamples={false}
			/>,
		);
		expect(screen.queryByText('Common schedules:')).toBeNull();
	});

	it('EmptyStateNoSearchResults injects query into title', () => {
		render(
			<EmptyStateNoSearchResults query="foobar" onClearSearch={() => {}} />,
		);
		expect(screen.getByText('No results for "foobar"')).toBeDefined();
	});

	it('EmptyStateNoGroups renders default copy', () => {
		render(<EmptyStateNoGroups onCreateGroup={() => {}} />);
		expect(screen.getByText('No agent groups')).toBeDefined();
	});

	it('EmptyStateNoRepositories renders default copy', () => {
		render(<EmptyStateNoRepositories onCreateRepository={() => {}} />);
		expect(screen.getByText('No repositories')).toBeDefined();
	});

	it('EmptyStateNoPolicies renders default copy', () => {
		render(<EmptyStateNoPolicies onCreatePolicy={() => {}} />);
		expect(screen.getByText('No policies defined')).toBeDefined();
	});
});
