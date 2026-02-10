import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const mockMutate = vi.fn();
const mockDeleteMutate = vi.fn();

vi.mock('../../hooks/useAuth', () => ({
	useMe: vi.fn(() => ({
		data: { id: 'user-1', name: 'Test User', email: 'test@example.com' },
	})),
}));

vi.mock('../../hooks/useSnapshotComments', () => ({
	useSnapshotComments: vi.fn(),
	useCreateSnapshotComment: vi.fn(() => ({
		mutate: mockMutate,
		isPending: false,
	})),
	useDeleteSnapshotComment: vi.fn(() => ({
		mutate: mockDeleteMutate,
		isPending: false,
	})),
}));

vi.mock('../../lib/utils', () => ({
	formatDateTime: (d: string) => d || 'N/A',
}));

import {
	useCreateSnapshotComment,
	useDeleteSnapshotComment,
	useSnapshotComments,
} from '../../hooks/useSnapshotComments';
import { SnapshotComments } from './SnapshotComments';

describe('SnapshotComments', () => {
	beforeEach(() => {
		vi.clearAllMocks();
		vi.mocked(useCreateSnapshotComment).mockReturnValue({
			mutate: mockMutate,
			isPending: false,
		} as ReturnType<typeof useCreateSnapshotComment>);
		vi.mocked(useDeleteSnapshotComment).mockReturnValue({
			mutate: mockDeleteMutate,
			isPending: false,
		} as ReturnType<typeof useDeleteSnapshotComment>);
	});

	it('shows loading state', () => {
		vi.mocked(useSnapshotComments).mockReturnValue({
			data: undefined,
			isLoading: true,
			isError: false,
		} as ReturnType<typeof useSnapshotComments>);
		render(<SnapshotComments snapshotId="snap-1" />);
		const pulses = document.querySelectorAll('.animate-pulse');
		expect(pulses.length).toBeGreaterThan(0);
	});

	it('shows error state', () => {
		vi.mocked(useSnapshotComments).mockReturnValue({
			data: undefined,
			isLoading: false,
			isError: true,
		} as ReturnType<typeof useSnapshotComments>);
		render(<SnapshotComments snapshotId="snap-1" />);
		expect(screen.getByText('Failed to load comments')).toBeInTheDocument();
	});

	it('shows empty state when no comments', () => {
		vi.mocked(useSnapshotComments).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useSnapshotComments>);
		render(<SnapshotComments snapshotId="snap-1" />);
		expect(screen.getByText(/No notes yet/)).toBeInTheDocument();
	});

	it('renders comments with user info', () => {
		vi.mocked(useSnapshotComments).mockReturnValue({
			data: [
				{
					id: 'c1',
					snapshot_id: 'snap-1',
					user_id: 'user-1',
					user_name: 'Test User',
					user_email: 'test@example.com',
					content: 'First comment',
					created_at: '2024-01-01T00:00:00Z',
				},
				{
					id: 'c2',
					snapshot_id: 'snap-1',
					user_id: 'user-2',
					user_name: 'Other User',
					user_email: 'other@example.com',
					content: 'Second comment',
					created_at: '2024-01-02T00:00:00Z',
				},
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useSnapshotComments>);
		render(<SnapshotComments snapshotId="snap-1" />);
		expect(screen.getByText('First comment')).toBeInTheDocument();
		expect(screen.getByText('Second comment')).toBeInTheDocument();
		expect(screen.getByText('Test User')).toBeInTheDocument();
		expect(screen.getByText('Other User')).toBeInTheDocument();
	});

	it('shows user initial from name', () => {
		vi.mocked(useSnapshotComments).mockReturnValue({
			data: [
				{
					id: 'c1',
					snapshot_id: 'snap-1',
					user_id: 'user-1',
					user_name: 'Alice',
					user_email: 'alice@example.com',
					content: 'Hello',
					created_at: '2024-01-01T00:00:00Z',
				},
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useSnapshotComments>);
		render(<SnapshotComments snapshotId="snap-1" />);
		expect(screen.getByText('A')).toBeInTheDocument();
	});

	it('shows comment count badge', () => {
		vi.mocked(useSnapshotComments).mockReturnValue({
			data: [
				{
					id: 'c1',
					snapshot_id: 'snap-1',
					user_id: 'user-1',
					user_name: 'User',
					content: 'One',
					created_at: '2024-01-01T00:00:00Z',
				},
				{
					id: 'c2',
					snapshot_id: 'snap-1',
					user_id: 'user-1',
					user_name: 'User',
					content: 'Two',
					created_at: '2024-01-02T00:00:00Z',
				},
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useSnapshotComments>);
		render(<SnapshotComments snapshotId="snap-1" />);
		expect(screen.getByText('2')).toBeInTheDocument();
	});

	it('shows delete button only for own comments', () => {
		vi.mocked(useSnapshotComments).mockReturnValue({
			data: [
				{
					id: 'c1',
					snapshot_id: 'snap-1',
					user_id: 'user-1',
					user_name: 'Test User',
					content: 'My comment',
					created_at: '2024-01-01T00:00:00Z',
				},
				{
					id: 'c2',
					snapshot_id: 'snap-1',
					user_id: 'user-2',
					user_name: 'Other User',
					content: 'Their comment',
					created_at: '2024-01-02T00:00:00Z',
				},
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useSnapshotComments>);
		render(<SnapshotComments snapshotId="snap-1" />);
		const deleteButtons = screen.getAllByText('Delete');
		expect(deleteButtons).toHaveLength(1);
	});

	it('renders textarea and Add Note button', () => {
		vi.mocked(useSnapshotComments).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useSnapshotComments>);
		render(<SnapshotComments snapshotId="snap-1" />);
		expect(screen.getByPlaceholderText(/Add a note/)).toBeInTheDocument();
		expect(screen.getByText('Add Note')).toBeInTheDocument();
	});

	it('disables submit button when textarea is empty', () => {
		vi.mocked(useSnapshotComments).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useSnapshotComments>);
		render(<SnapshotComments snapshotId="snap-1" />);
		const submitButton = screen.getByText('Add Note');
		expect(submitButton).toBeDisabled();
	});

	it('submits new comment', async () => {
		const user = userEvent.setup();
		vi.mocked(useSnapshotComments).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useSnapshotComments>);
		render(<SnapshotComments snapshotId="snap-1" />);

		await user.type(screen.getByPlaceholderText(/Add a note/), 'New comment');
		await user.click(screen.getByText('Add Note'));

		expect(mockMutate).toHaveBeenCalledWith(
			{ content: 'New comment' },
			expect.objectContaining({ onSuccess: expect.any(Function) }),
		);
	});

	it('does not submit whitespace-only comment', async () => {
		const user = userEvent.setup();
		vi.mocked(useSnapshotComments).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useSnapshotComments>);
		render(<SnapshotComments snapshotId="snap-1" />);

		await user.type(screen.getByPlaceholderText(/Add a note/), '   ');
		const submitButton = screen.getByText('Add Note');
		expect(submitButton).toBeDisabled();
	});

	it('shows Saving... when creating comment', () => {
		vi.mocked(useSnapshotComments).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useSnapshotComments>);
		vi.mocked(useCreateSnapshotComment).mockReturnValue({
			mutate: mockMutate,
			isPending: true,
		} as unknown as ReturnType<typeof useCreateSnapshotComment>);
		render(<SnapshotComments snapshotId="snap-1" />);
		expect(screen.getByText('Saving...')).toBeInTheDocument();
	});

	it('calls delete with confirmation', async () => {
		const user = userEvent.setup();
		vi.spyOn(window, 'confirm').mockReturnValue(true);
		vi.mocked(useSnapshotComments).mockReturnValue({
			data: [
				{
					id: 'c1',
					snapshot_id: 'snap-1',
					user_id: 'user-1',
					user_name: 'Test User',
					content: 'My comment',
					created_at: '2024-01-01T00:00:00Z',
				},
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useSnapshotComments>);
		render(<SnapshotComments snapshotId="snap-1" />);

		await user.click(screen.getByText('Delete'));
		expect(window.confirm).toHaveBeenCalled();
		expect(mockDeleteMutate).toHaveBeenCalledWith('c1');
	});

	it('does not delete when confirmation is cancelled', async () => {
		const user = userEvent.setup();
		vi.spyOn(window, 'confirm').mockReturnValue(false);
		vi.mocked(useSnapshotComments).mockReturnValue({
			data: [
				{
					id: 'c1',
					snapshot_id: 'snap-1',
					user_id: 'user-1',
					user_name: 'Test User',
					content: 'My comment',
					created_at: '2024-01-01T00:00:00Z',
				},
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useSnapshotComments>);
		render(<SnapshotComments snapshotId="snap-1" />);

		await user.click(screen.getByText('Delete'));
		expect(mockDeleteMutate).not.toHaveBeenCalled();
	});

	it('shows email initial when user has no name', () => {
		vi.mocked(useSnapshotComments).mockReturnValue({
			data: [
				{
					id: 'c1',
					snapshot_id: 'snap-1',
					user_id: 'user-3',
					user_email: 'bob@example.com',
					content: 'Hello',
					created_at: '2024-01-01T00:00:00Z',
				},
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useSnapshotComments>);
		render(<SnapshotComments snapshotId="snap-1" />);
		expect(screen.getByText('B')).toBeInTheDocument();
	});

	it('renders Notes heading', () => {
		vi.mocked(useSnapshotComments).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useSnapshotComments>);
		render(<SnapshotComments snapshotId="snap-1" />);
		expect(screen.getByText('Notes')).toBeInTheDocument();
	});
});
