import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi, beforeEach } from 'vitest';

const mockCreateMutateAsync = vi.fn();
const mockUpdateMutateAsync = vi.fn();
const mockDeleteMutateAsync = vi.fn();

vi.mock('../../hooks/useBackupScripts', () => ({
	useBackupScripts: vi.fn(),
	useCreateBackupScript: vi.fn(() => ({ mutateAsync: mockCreateMutateAsync, isPending: false })),
	useUpdateBackupScript: vi.fn(() => ({ mutateAsync: mockUpdateMutateAsync, isPending: false })),
	useDeleteBackupScript: vi.fn(() => ({ mutateAsync: mockDeleteMutateAsync, isPending: false })),
}));

import { useBackupScripts, useCreateBackupScript, useUpdateBackupScript, useDeleteBackupScript } from '../../hooks/useBackupScripts';
import { BackupScriptsEditor } from './BackupScriptsEditor';

const mockOnClose = vi.fn();

describe('BackupScriptsEditor', () => {
	beforeEach(() => {
		vi.clearAllMocks();
		vi.mocked(useCreateBackupScript).mockReturnValue({ mutateAsync: mockCreateMutateAsync, isPending: false } as unknown as ReturnType<typeof useCreateBackupScript>);
		vi.mocked(useUpdateBackupScript).mockReturnValue({ mutateAsync: mockUpdateMutateAsync, isPending: false } as unknown as ReturnType<typeof useUpdateBackupScript>);
		vi.mocked(useDeleteBackupScript).mockReturnValue({ mutateAsync: mockDeleteMutateAsync, isPending: false } as unknown as ReturnType<typeof useDeleteBackupScript>);
	});

	it('shows loading state', () => {
		vi.mocked(useBackupScripts).mockReturnValue({ data: undefined, isLoading: true, isError: false } as ReturnType<typeof useBackupScripts>);
		render(<BackupScriptsEditor scheduleId="sched-1" onClose={mockOnClose} />);
		const pulses = document.querySelectorAll('.animate-pulse');
		expect(pulses.length).toBeGreaterThan(0);
	});

	it('shows error state with close button', async () => {
		const user = userEvent.setup();
		vi.mocked(useBackupScripts).mockReturnValue({ data: undefined, isLoading: false, isError: true } as ReturnType<typeof useBackupScripts>);
		render(<BackupScriptsEditor scheduleId="sched-1" onClose={mockOnClose} />);
		expect(screen.getByText('Failed to load scripts')).toBeInTheDocument();
		await user.click(screen.getByText('Close'));
		expect(mockOnClose).toHaveBeenCalled();
	});

	it('renders empty state with all script type buttons', () => {
		vi.mocked(useBackupScripts).mockReturnValue({ data: [], isLoading: false, isError: false } as ReturnType<typeof useBackupScripts>);
		render(<BackupScriptsEditor scheduleId="sched-1" onClose={mockOnClose} />);
		expect(screen.getByText('Backup Scripts')).toBeInTheDocument();
		expect(screen.getByText('Pre-Backup')).toBeInTheDocument();
		expect(screen.getByText('Post-Backup (Success)')).toBeInTheDocument();
		expect(screen.getByText('Post-Backup (Failure)')).toBeInTheDocument();
		expect(screen.getByText('Post-Backup (Always)')).toBeInTheDocument();
	});

	it('renders existing scripts as cards', () => {
		vi.mocked(useBackupScripts).mockReturnValue({
			data: [
				{ id: 's1', schedule_id: 'sched-1', type: 'pre_backup', script: '#!/bin/bash\necho "pre"', timeout_seconds: 300, fail_on_error: true, enabled: true, created_at: '2024-01-01T00:00:00Z', updated_at: '2024-01-01T00:00:00Z' },
				{ id: 's2', schedule_id: 'sched-1', type: 'post_success', script: '#!/bin/bash\necho "post"', timeout_seconds: 60, fail_on_error: false, enabled: false, created_at: '2024-01-01T00:00:00Z', updated_at: '2024-01-01T00:00:00Z' },
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useBackupScripts>);
		render(<BackupScriptsEditor scheduleId="sched-1" onClose={mockOnClose} />);
		// Shows existing script types
		expect(screen.getAllByText('Pre-Backup').length).toBeGreaterThan(0);
		expect(screen.getAllByText('Post-Backup (Success)').length).toBeGreaterThan(0);
		// Shows disabled badge
		expect(screen.getByText('Disabled')).toBeInTheDocument();
		// Shows fail_on_error badge for pre_backup
		expect(screen.getByText('Fails backup on error')).toBeInTheDocument();
		// Shows timeout
		expect(screen.getByText('Timeout: 300s')).toBeInTheDocument();
		expect(screen.getByText('Timeout: 60s')).toBeInTheDocument();
	});

	it('hides already-used script types from add buttons', () => {
		vi.mocked(useBackupScripts).mockReturnValue({
			data: [
				{ id: 's1', schedule_id: 'sched-1', type: 'pre_backup', script: 'echo hi', timeout_seconds: 300, fail_on_error: false, enabled: true, created_at: '2024-01-01T00:00:00Z', updated_at: '2024-01-01T00:00:00Z' },
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useBackupScripts>);
		render(<BackupScriptsEditor scheduleId="sched-1" onClose={mockOnClose} />);
		// Add section should have remaining 3 types
		expect(screen.getByText('Add a script:')).toBeInTheDocument();
		const addButtons = screen.getAllByText(/Post-Backup/);
		expect(addButtons.length).toBe(3); // Success, Failure, Always
	});

	it('opens script form when clicking add button', async () => {
		const user = userEvent.setup();
		vi.mocked(useBackupScripts).mockReturnValue({ data: [], isLoading: false, isError: false } as ReturnType<typeof useBackupScripts>);
		render(<BackupScriptsEditor scheduleId="sched-1" onClose={mockOnClose} />);
		await user.click(screen.getByText('Pre-Backup'));
		expect(screen.getByLabelText('Script')).toBeInTheDocument();
		expect(screen.getByLabelText('Timeout (seconds)')).toBeInTheDocument();
		expect(screen.getByText('Create Script')).toBeInTheDocument();
	});

	it('shows fail_on_error checkbox only for pre_backup type', async () => {
		const user = userEvent.setup();
		vi.mocked(useBackupScripts).mockReturnValue({ data: [], isLoading: false, isError: false } as ReturnType<typeof useBackupScripts>);
		render(<BackupScriptsEditor scheduleId="sched-1" onClose={mockOnClose} />);
		await user.click(screen.getByText('Pre-Backup'));
		expect(screen.getByText('Fail backup if script fails')).toBeInTheDocument();
	});

	it('hides fail_on_error checkbox for non pre_backup types', async () => {
		const user = userEvent.setup();
		vi.mocked(useBackupScripts).mockReturnValue({ data: [], isLoading: false, isError: false } as ReturnType<typeof useBackupScripts>);
		render(<BackupScriptsEditor scheduleId="sched-1" onClose={mockOnClose} />);
		await user.click(screen.getByText('Post-Backup (Success)'));
		expect(screen.queryByText('Fail backup if script fails')).not.toBeInTheDocument();
	});

	it('creates a new script on form submit', async () => {
		const user = userEvent.setup();
		mockCreateMutateAsync.mockResolvedValue({});
		vi.mocked(useBackupScripts).mockReturnValue({ data: [], isLoading: false, isError: false } as ReturnType<typeof useBackupScripts>);
		render(<BackupScriptsEditor scheduleId="sched-1" onClose={mockOnClose} />);

		await user.click(screen.getByText('Pre-Backup'));
		await user.type(screen.getByLabelText('Script'), '#!/bin/bash\necho hello');
		await user.click(screen.getByText('Create Script'));

		expect(mockCreateMutateAsync).toHaveBeenCalledWith({
			scheduleId: 'sched-1',
			data: expect.objectContaining({
				type: 'pre_backup',
				script: '#!/bin/bash\necho hello',
			}),
		});
	});

	it('opens edit form when clicking Edit on a card', async () => {
		const user = userEvent.setup();
		vi.mocked(useBackupScripts).mockReturnValue({
			data: [
				{ id: 's1', schedule_id: 'sched-1', type: 'pre_backup', script: 'echo existing', timeout_seconds: 300, fail_on_error: false, enabled: true, created_at: '2024-01-01T00:00:00Z', updated_at: '2024-01-01T00:00:00Z' },
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useBackupScripts>);
		render(<BackupScriptsEditor scheduleId="sched-1" onClose={mockOnClose} />);

		await user.click(screen.getByText('Edit'));
		expect(screen.getByText('Update Script')).toBeInTheDocument();
		expect(screen.getByDisplayValue('echo existing')).toBeInTheDocument();
	});

	it('calls delete with confirmation', async () => {
		const user = userEvent.setup();
		vi.spyOn(window, 'confirm').mockReturnValue(true);
		mockDeleteMutateAsync.mockResolvedValue({});
		vi.mocked(useBackupScripts).mockReturnValue({
			data: [
				{ id: 's1', schedule_id: 'sched-1', type: 'pre_backup', script: 'echo hi', timeout_seconds: 300, fail_on_error: false, enabled: true, created_at: '2024-01-01T00:00:00Z', updated_at: '2024-01-01T00:00:00Z' },
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useBackupScripts>);
		render(<BackupScriptsEditor scheduleId="sched-1" onClose={mockOnClose} />);

		await user.click(screen.getAllByText('Delete')[0]);
		expect(window.confirm).toHaveBeenCalled();
		expect(mockDeleteMutateAsync).toHaveBeenCalledWith({ scheduleId: 'sched-1', id: 's1' });
	});

	it('cancels script form', async () => {
		const user = userEvent.setup();
		vi.mocked(useBackupScripts).mockReturnValue({ data: [], isLoading: false, isError: false } as ReturnType<typeof useBackupScripts>);
		render(<BackupScriptsEditor scheduleId="sched-1" onClose={mockOnClose} />);

		await user.click(screen.getByText('Pre-Backup'));
		expect(screen.getByText('Create Script')).toBeInTheDocument();

		await user.click(screen.getByText('Cancel'));
		expect(screen.queryByText('Create Script')).not.toBeInTheDocument();
	});

	it('calls onClose when clicking close button in header', async () => {
		const user = userEvent.setup();
		vi.mocked(useBackupScripts).mockReturnValue({ data: [], isLoading: false, isError: false } as ReturnType<typeof useBackupScripts>);
		render(<BackupScriptsEditor scheduleId="sched-1" onClose={mockOnClose} />);

		await user.click(screen.getByLabelText('Close'));
		expect(mockOnClose).toHaveBeenCalled();
	});

	it('truncates long scripts in card view', () => {
		const longScript = 'x'.repeat(600);
		vi.mocked(useBackupScripts).mockReturnValue({
			data: [
				{ id: 's1', schedule_id: 'sched-1', type: 'pre_backup', script: longScript, timeout_seconds: 300, fail_on_error: false, enabled: true, created_at: '2024-01-01T00:00:00Z', updated_at: '2024-01-01T00:00:00Z' },
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useBackupScripts>);
		render(<BackupScriptsEditor scheduleId="sched-1" onClose={mockOnClose} />);
		expect(screen.getByText(/\.\.\.$/)).toBeInTheDocument();
	});
});
