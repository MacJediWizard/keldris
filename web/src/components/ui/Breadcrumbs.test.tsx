import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';
import type { BreadcrumbItem } from '../../hooks/useBreadcrumbs';

vi.mock('../../hooks/useBreadcrumbs', () => ({
	useBreadcrumbs: vi.fn(),
}));

import { useBreadcrumbs } from '../../hooks/useBreadcrumbs';
import { Breadcrumbs, BreadcrumbsWithItems } from './Breadcrumbs';

function setMock(breadcrumbs: BreadcrumbItem[], isLoading = false) {
	vi.mocked(useBreadcrumbs).mockReturnValue({ breadcrumbs, isLoading });
}

function withRouter(ui: React.ReactNode) {
	return <MemoryRouter>{ui}</MemoryRouter>;
}

describe('Breadcrumbs', () => {
	it('renders nothing when only Dashboard is present', () => {
		setMock([{ label: 'Dashboard', path: '/', isCurrentPage: true }]);
		const { container } = render(withRouter(<Breadcrumbs />));
		expect(container.firstChild).toBeNull();
	});

	it('renders breadcrumb labels with current page marker', () => {
		setMock([
			{ label: 'Dashboard', path: '/', isCurrentPage: false },
			{ label: 'Agents', path: '/agents', isCurrentPage: true },
		]);
		render(withRouter(<Breadcrumbs />));
		expect(screen.getByText('Agents')).toBeDefined();
		expect(screen.getByText('Agents').getAttribute('aria-current')).toBe(
			'page',
		);
	});

	it('renders explicit items prop when provided', () => {
		setMock([{ label: 'Dashboard', path: '/', isCurrentPage: true }]);
		render(
			withRouter(
				<Breadcrumbs
					items={[
						{ label: 'Dashboard', path: '/', isCurrentPage: false },
						{
							label: 'Repositories',
							path: '/repositories',
							isCurrentPage: true,
						},
					]}
				/>,
			),
		);
		expect(screen.getByText('Repositories')).toBeDefined();
	});

	it('renders loading skeleton for current page when isLoading', () => {
		setMock(
			[
				{ label: 'Dashboard', path: '/', isCurrentPage: false },
				{ label: 'loading', path: '/agents/abc', isCurrentPage: true },
			],
			true,
		);
		const { container } = render(withRouter(<Breadcrumbs />));
		expect(container.querySelector('.animate-pulse')).not.toBeNull();
	});
});

describe('BreadcrumbsWithItems', () => {
	it('renders nothing for single item', () => {
		const { container } = render(
			withRouter(
				<BreadcrumbsWithItems
					items={[{ label: 'Dashboard', path: '/', isCurrentPage: true }]}
				/>,
			),
		);
		expect(container.firstChild).toBeNull();
	});

	it('renders items with chevron separators', () => {
		render(
			withRouter(
				<BreadcrumbsWithItems
					items={[
						{ label: 'Dashboard', path: '/', isCurrentPage: false },
						{ label: 'Snapshots', path: '/snapshots', isCurrentPage: false },
						{
							label: 'Compare',
							path: '/snapshots/compare',
							isCurrentPage: true,
						},
					]}
				/>,
			),
		);
		expect(screen.getByText('Snapshots')).toBeDefined();
		expect(screen.getByText('Compare')).toBeDefined();
	});
});
