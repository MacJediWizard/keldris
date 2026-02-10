import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { BrowserRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { Tags } from './Tags';

vi.mock('../hooks/useTags', () => ({
	useTags: vi.fn(),
	useCreateTag: () => ({
		mutateAsync: vi.fn(),
		isPending: false,
		isError: false,
	}),
	useUpdateTag: () => ({
		mutateAsync: vi.fn(),
		isPending: false,
		isError: false,
	}),
	useDeleteTag: () => ({
		mutateAsync: vi.fn(),
		isPending: false,
		isError: false,
	}),
}));

import { useTags } from '../hooks/useTags';

function renderPage() {
	return render(
		<BrowserRouter>
			<Tags />
		</BrowserRouter>,
	);
}

describe('Tags', () => {
	beforeEach(() => vi.clearAllMocks());

	it('renders title', () => {
		vi.mocked(useTags).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useTags>);
		renderPage();
		expect(screen.getByText('Tags')).toBeInTheDocument();
	});

	it('shows subtitle', () => {
		vi.mocked(useTags).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useTags>);
		renderPage();
		expect(
			screen.getByText('Organize and categorize your backups with tags'),
		).toBeInTheDocument();
	});

	it('shows loading state', () => {
		vi.mocked(useTags).mockReturnValue({
			data: undefined,
			isLoading: true,
			isError: false,
		} as ReturnType<typeof useTags>);
		renderPage();
		const skeletons = document.querySelectorAll('.animate-pulse');
		expect(skeletons.length).toBeGreaterThan(0);
	});

	it('shows error state', () => {
		vi.mocked(useTags).mockReturnValue({
			data: undefined,
			isLoading: false,
			isError: true,
		} as ReturnType<typeof useTags>);
		renderPage();
		expect(screen.getByText('Failed to load tags')).toBeInTheDocument();
	});

	it('shows empty state', () => {
		vi.mocked(useTags).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useTags>);
		renderPage();
		expect(screen.getByText('No tags yet')).toBeInTheDocument();
	});

	it('shows empty state help text', () => {
		vi.mocked(useTags).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useTags>);
		renderPage();
		expect(
			screen.getByText('Create your first tag to organize backups'),
		).toBeInTheDocument();
	});

	it('renders tag rows', () => {
		vi.mocked(useTags).mockReturnValue({
			data: [
				{
					id: '1',
					name: 'production',
					color: '#ef4444',
					created_at: '2024-01-01T00:00:00Z',
				},
				{
					id: '2',
					name: 'staging',
					color: '#3b82f6',
					created_at: '2024-01-02T00:00:00Z',
				},
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useTags>);
		renderPage();
		expect(screen.getByText('production')).toBeInTheDocument();
		expect(screen.getByText('staging')).toBeInTheDocument();
		expect(screen.getByText('#ef4444')).toBeInTheDocument();
	});

	it('shows table headers when tags exist', () => {
		vi.mocked(useTags).mockReturnValue({
			data: [
				{
					id: '1',
					name: 'test',
					color: '#6366f1',
					created_at: '2024-01-01T00:00:00Z',
				},
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useTags>);
		renderPage();
		expect(screen.getByText('Actions')).toBeInTheDocument();
	});

	it('shows Create Tag button', () => {
		vi.mocked(useTags).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useTags>);
		renderPage();
		expect(screen.getByText('Create Tag')).toBeInTheDocument();
	});

	it('opens create modal on button click', async () => {
		const user = userEvent.setup();
		vi.mocked(useTags).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useTags>);
		renderPage();
		await user.click(screen.getByText('Create Tag'));
		expect(screen.getByText('Create New Tag')).toBeInTheDocument();
	});

	it('shows create modal form fields', async () => {
		const user = userEvent.setup();
		vi.mocked(useTags).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useTags>);
		renderPage();
		await user.click(screen.getByText('Create Tag'));
		expect(screen.getByLabelText('Name')).toBeInTheDocument();
	});

	it('shows cancel button in create modal', async () => {
		const user = userEvent.setup();
		vi.mocked(useTags).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useTags>);
		renderPage();
		await user.click(screen.getByText('Create Tag'));
		expect(screen.getByText('Cancel')).toBeInTheDocument();
	});

	it('shows edit and delete buttons for tags', () => {
		vi.mocked(useTags).mockReturnValue({
			data: [
				{
					id: '1',
					name: 'prod',
					color: '#ef4444',
					created_at: '2024-01-01T00:00:00Z',
				},
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useTags>);
		renderPage();
		expect(screen.getByText('Edit')).toBeInTheDocument();
		expect(screen.getByText('Delete')).toBeInTheDocument();
	});

	it('opens edit modal on edit click', async () => {
		const user = userEvent.setup();
		vi.mocked(useTags).mockReturnValue({
			data: [
				{
					id: '1',
					name: 'prod',
					color: '#ef4444',
					created_at: '2024-01-01T00:00:00Z',
				},
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useTags>);
		renderPage();
		await user.click(screen.getByText('Edit'));
		expect(screen.getByText('Edit Tag')).toBeInTheDocument();
	});

	it('opens delete modal on delete click', async () => {
		const user = userEvent.setup();
		vi.mocked(useTags).mockReturnValue({
			data: [
				{
					id: '1',
					name: 'prod',
					color: '#ef4444',
					created_at: '2024-01-01T00:00:00Z',
				},
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useTags>);
		renderPage();
		await user.click(screen.getByText('Delete'));
		expect(screen.getByText('Delete Tag')).toBeInTheDocument();
	});

	it('shows delete confirmation message', async () => {
		const user = userEvent.setup();
		vi.mocked(useTags).mockReturnValue({
			data: [
				{
					id: '1',
					name: 'prod',
					color: '#ef4444',
					created_at: '2024-01-01T00:00:00Z',
				},
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useTags>);
		renderPage();
		await user.click(screen.getByText('Delete'));
		expect(
			screen.getByText(/This will remove it from all associated backups/),
		).toBeInTheDocument();
	});

	it('shows error page try again text', () => {
		vi.mocked(useTags).mockReturnValue({
			data: undefined,
			isLoading: false,
			isError: true,
		} as ReturnType<typeof useTags>);
		renderPage();
		expect(
			screen.getByText('Please try refreshing the page'),
		).toBeInTheDocument();
	});

	it('shows multiple tags', () => {
		vi.mocked(useTags).mockReturnValue({
			data: [
				{
					id: '1',
					name: 'production',
					color: '#ef4444',
					created_at: '2024-01-01T00:00:00Z',
				},
				{
					id: '2',
					name: 'staging',
					color: '#3b82f6',
					created_at: '2024-01-02T00:00:00Z',
				},
				{
					id: '3',
					name: 'daily',
					color: '#22c55e',
					created_at: '2024-01-03T00:00:00Z',
				},
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useTags>);
		renderPage();
		expect(screen.getByText('production')).toBeInTheDocument();
		expect(screen.getByText('staging')).toBeInTheDocument();
		expect(screen.getByText('daily')).toBeInTheDocument();
	});
});
