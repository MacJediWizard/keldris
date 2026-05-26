import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import type { Announcement } from '../../lib/types';

vi.mock('../../hooks/useAnnouncements', () => ({
	useActiveAnnouncements: vi.fn(),
	useDismissAnnouncement: vi.fn(),
}));

import {
	useActiveAnnouncements,
	useDismissAnnouncement,
} from '../../hooks/useAnnouncements';
import { AnnouncementBanner } from './AnnouncementBanner';

function makeAnnouncement(overrides: Partial<Announcement> = {}): Announcement {
	return {
		id: 'a1',
		title: 'System maintenance tonight',
		message: 'Brief downtime expected',
		type: 'info',
		dismissible: true,
		active: true,
		starts_at: null,
		ends_at: null,
		created_at: '',
		updated_at: '',
		...overrides,
	} as Announcement;
}

const mutate = vi.fn();

function setMocks(
	announcements: Announcement[] | undefined,
	isPending = false,
) {
	vi.mocked(useActiveAnnouncements).mockReturnValue({
		data: announcements,
		isLoading: false,
		isError: false,
		error: null,
	} as never);
	vi.mocked(useDismissAnnouncement).mockReturnValue({
		mutate,
		isPending,
	} as never);
}

describe('AnnouncementBanner', () => {
	it('renders nothing when no announcements', () => {
		setMocks([]);
		const { container } = render(<AnnouncementBanner />);
		expect(container.firstChild).toBeNull();
	});

	it('renders nothing when announcements undefined', () => {
		setMocks(undefined);
		const { container } = render(<AnnouncementBanner />);
		expect(container.firstChild).toBeNull();
	});

	it('renders title for each announcement', () => {
		setMocks([
			makeAnnouncement({ id: '1', title: 'First' }),
			makeAnnouncement({ id: '2', title: 'Second' }),
		]);
		render(<AnnouncementBanner />);
		expect(screen.getByText('First')).toBeDefined();
		expect(screen.getByText('Second')).toBeDefined();
	});

	it('fires mutate(id) when dismiss clicked', () => {
		mutate.mockReset();
		setMocks([makeAnnouncement({ id: 'abc', dismissible: true })]);
		render(<AnnouncementBanner />);
		screen.getByRole('button', { name: 'Dismiss announcement' }).click();
		expect(mutate).toHaveBeenCalledWith('abc');
	});

	it('hides dismiss button when dismissible=false', () => {
		setMocks([makeAnnouncement({ dismissible: false })]);
		render(<AnnouncementBanner />);
		expect(
			screen.queryByRole('button', { name: 'Dismiss announcement' }),
		).toBeNull();
	});

	it('disables dismiss button while pending', () => {
		setMocks([makeAnnouncement({ dismissible: true })], true);
		render(<AnnouncementBanner />);
		expect(
			screen.getByRole('button', { name: 'Dismiss announcement' }),
		).toBeDisabled();
	});
});
