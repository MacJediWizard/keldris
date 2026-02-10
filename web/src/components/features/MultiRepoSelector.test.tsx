import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import type { Repository, ScheduleRepositoryRequest } from '../../lib/types';
import { MultiRepoSelector } from './MultiRepoSelector';

const repositories: Repository[] = [
	{
		id: 'repo-1',
		name: 'Local Backup',
		type: 'local',
		org_id: 'org-1',
		created_at: '2024-01-01T00:00:00Z',
		updated_at: '2024-01-01T00:00:00Z',
	} as Repository,
	{
		id: 'repo-2',
		name: 'S3 Backup',
		type: 's3',
		org_id: 'org-1',
		created_at: '2024-01-01T00:00:00Z',
		updated_at: '2024-01-01T00:00:00Z',
	} as Repository,
	{
		id: 'repo-3',
		name: 'B2 Backup',
		type: 'b2',
		org_id: 'org-1',
		created_at: '2024-01-01T00:00:00Z',
		updated_at: '2024-01-01T00:00:00Z',
	} as Repository,
];

describe('MultiRepoSelector', () => {
	it('renders available repos in dropdown', () => {
		render(
			<MultiRepoSelector
				repositories={repositories}
				selectedRepos={[]}
				onChange={vi.fn()}
			/>,
		);
		expect(
			screen.getByText('Select a repository to add...'),
		).toBeInTheDocument();
	});

	it('shows empty state message when no repos selected', () => {
		render(
			<MultiRepoSelector
				repositories={repositories}
				selectedRepos={[]}
				onChange={vi.fn()}
			/>,
		);
		expect(
			screen.getByText('Please add at least one repository for backups.'),
		).toBeInTheDocument();
	});

	it('renders selected repositories', () => {
		const selected: ScheduleRepositoryRequest[] = [
			{ repository_id: 'repo-1', priority: 0, enabled: true },
		];
		render(
			<MultiRepoSelector
				repositories={repositories}
				selectedRepos={selected}
				onChange={vi.fn()}
			/>,
		);
		expect(screen.getByText('Local Backup (local)')).toBeInTheDocument();
	});

	it('adds a repository when clicking Add', () => {
		const onChange = vi.fn();
		render(
			<MultiRepoSelector
				repositories={repositories}
				selectedRepos={[]}
				onChange={onChange}
			/>,
		);

		const select = screen.getByRole('combobox');
		fireEvent.change(select, { target: { value: 'repo-1' } });

		const addButton = screen.getByText('Add');
		fireEvent.click(addButton);

		expect(onChange).toHaveBeenCalledWith([
			{ repository_id: 'repo-1', priority: 0, enabled: true },
		]);
	});

	it('removes a repository', () => {
		const onChange = vi.fn();
		const selected: ScheduleRepositoryRequest[] = [
			{ repository_id: 'repo-1', priority: 0, enabled: true },
			{ repository_id: 'repo-2', priority: 1, enabled: true },
		];
		render(
			<MultiRepoSelector
				repositories={repositories}
				selectedRepos={selected}
				onChange={onChange}
			/>,
		);

		const removeButtons = screen.getAllByTitle('Remove');
		fireEvent.click(removeButtons[0]);

		expect(onChange).toHaveBeenCalledWith([
			{ repository_id: 'repo-2', priority: 0, enabled: true },
		]);
	});

	it('toggles enabled state', () => {
		const onChange = vi.fn();
		const selected: ScheduleRepositoryRequest[] = [
			{ repository_id: 'repo-1', priority: 0, enabled: true },
		];
		render(
			<MultiRepoSelector
				repositories={repositories}
				selectedRepos={selected}
				onChange={onChange}
			/>,
		);

		const disableButton = screen.getByTitle('Disable');
		fireEvent.click(disableButton);

		expect(onChange).toHaveBeenCalledWith([
			{ repository_id: 'repo-1', priority: 0, enabled: false },
		]);
	});

	it('moves a repository up in priority', () => {
		const onChange = vi.fn();
		const selected: ScheduleRepositoryRequest[] = [
			{ repository_id: 'repo-1', priority: 0, enabled: true },
			{ repository_id: 'repo-2', priority: 1, enabled: true },
		];
		render(
			<MultiRepoSelector
				repositories={repositories}
				selectedRepos={selected}
				onChange={onChange}
			/>,
		);

		const moveUpButtons = screen.getAllByTitle('Move up');
		fireEvent.click(moveUpButtons[1]); // Move second repo up

		expect(onChange).toHaveBeenCalledWith([
			{ repository_id: 'repo-2', priority: 0, enabled: true },
			{ repository_id: 'repo-1', priority: 1, enabled: true },
		]);
	});

	it('does not show already-selected repos in dropdown', () => {
		const selected: ScheduleRepositoryRequest[] = [
			{ repository_id: 'repo-1', priority: 0, enabled: true },
		];
		render(
			<MultiRepoSelector
				repositories={repositories}
				selectedRepos={selected}
				onChange={vi.fn()}
			/>,
		);

		const options = screen.getAllByRole('option');
		const optionTexts = options.map((o) => o.textContent);
		expect(optionTexts).not.toContain('Local Backup (local)');
		expect(optionTexts).toContain('S3 Backup (s3)');
	});

	it('disables Add button when no repo selected', () => {
		render(
			<MultiRepoSelector
				repositories={repositories}
				selectedRepos={[]}
				onChange={vi.fn()}
			/>,
		);
		const addButton = screen.getByText('Add');
		expect(addButton).toBeDisabled();
	});
});
