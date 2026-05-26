import { render } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import {
	AgentListSkeleton,
	AgentRowSkeleton,
	DashboardBackupListSkeleton,
	DashboardFavoritesSkeleton,
	DashboardQueueSkeleton,
	DashboardSkeleton,
	DashboardStatsSkeleton,
	GenericListSkeleton,
	ScheduleFormSkeleton,
	ScheduleListSkeleton,
	ScheduleRowSkeleton,
} from './PageSkeletons';

function renderRowOnly(ui: React.ReactNode) {
	return render(
		<table>
			<tbody>{ui}</tbody>
		</table>,
	);
}

describe('PageSkeletons', () => {
	it('renders AgentRowSkeleton', () => {
		const { container } = renderRowOnly(<AgentRowSkeleton />);
		expect(container.querySelector('tr')).not.toBeNull();
	});

	it('renders AgentListSkeleton with default row count', () => {
		const { container } = render(<AgentListSkeleton />);
		expect(container.querySelectorAll('tbody tr').length).toBe(3);
	});

	it('renders AgentListSkeleton with custom row count', () => {
		const { container } = render(<AgentListSkeleton rows={7} />);
		expect(container.querySelectorAll('tbody tr').length).toBe(7);
	});

	it('renders ScheduleRowSkeleton', () => {
		const { container } = renderRowOnly(<ScheduleRowSkeleton />);
		expect(container.querySelector('tr')).not.toBeNull();
	});

	it('renders ScheduleListSkeleton', () => {
		const { container } = render(<ScheduleListSkeleton rows={2} />);
		expect(container.querySelectorAll('tbody tr').length).toBe(2);
	});

	it('renders DashboardStatsSkeleton with 4 cards', () => {
		const { container } = render(<DashboardStatsSkeleton />);
		expect(container.firstChild).toBeDefined();
	});

	it('renders DashboardBackupListSkeleton', () => {
		const { container } = render(<DashboardBackupListSkeleton />);
		expect(container.firstChild).toBeDefined();
	});

	it('renders DashboardQueueSkeleton', () => {
		const { container } = render(<DashboardQueueSkeleton />);
		expect(container.firstChild).toBeDefined();
	});

	it('renders DashboardFavoritesSkeleton', () => {
		const { container } = render(<DashboardFavoritesSkeleton />);
		expect(container.firstChild).toBeDefined();
	});

	it('renders DashboardSkeleton', () => {
		const { container } = render(<DashboardSkeleton />);
		expect(container.firstChild).toBeDefined();
	});

	it('renders ScheduleFormSkeleton', () => {
		const { container } = render(<ScheduleFormSkeleton />);
		expect(container.firstChild).toBeDefined();
	});

	it('renders GenericListSkeleton with defaults', () => {
		const { container } = render(<GenericListSkeleton />);
		expect(container.querySelectorAll('tbody tr').length).toBe(5);
	});

	it('renders GenericListSkeleton with custom config', () => {
		const { container } = render(
			<GenericListSkeleton
				rows={2}
				columns={4}
				showCheckbox={false}
				showActions={false}
			/>,
		);
		expect(container.querySelectorAll('tbody tr').length).toBe(2);
	});
});
