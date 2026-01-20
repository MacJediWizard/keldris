import { useState } from 'react';
import type { Repository, ScheduleRepositoryRequest } from '../../lib/types';

interface MultiRepoSelectorProps {
	repositories: Repository[];
	selectedRepos: ScheduleRepositoryRequest[];
	onChange: (repos: ScheduleRepositoryRequest[]) => void;
}

export function MultiRepoSelector({
	repositories,
	selectedRepos,
	onChange,
}: MultiRepoSelectorProps) {
	const [selectedId, setSelectedId] = useState('');

	const addRepository = () => {
		if (!selectedId) return;

		// Don't add duplicates
		if (selectedRepos.some((r) => r.repository_id === selectedId)) {
			setSelectedId('');
			return;
		}

		const newRepo: ScheduleRepositoryRequest = {
			repository_id: selectedId,
			priority: selectedRepos.length, // Next priority
			enabled: true,
		};

		onChange([...selectedRepos, newRepo]);
		setSelectedId('');
	};

	const removeRepository = (repoId: string) => {
		const updated = selectedRepos
			.filter((r) => r.repository_id !== repoId)
			.map((r, index) => ({ ...r, priority: index })); // Renumber priorities
		onChange(updated);
	};

	const moveUp = (index: number) => {
		if (index === 0) return;
		const updated = [...selectedRepos];
		[updated[index - 1], updated[index]] = [updated[index], updated[index - 1]];
		// Update priorities
		onChange(updated.map((r, i) => ({ ...r, priority: i })));
	};

	const moveDown = (index: number) => {
		if (index === selectedRepos.length - 1) return;
		const updated = [...selectedRepos];
		[updated[index], updated[index + 1]] = [updated[index + 1], updated[index]];
		// Update priorities
		onChange(updated.map((r, i) => ({ ...r, priority: i })));
	};

	const toggleEnabled = (repoId: string) => {
		const updated = selectedRepos.map((r) =>
			r.repository_id === repoId ? { ...r, enabled: !r.enabled } : r,
		);
		onChange(updated);
	};

	const getRepoName = (repoId: string) => {
		const repo = repositories.find((r) => r.id === repoId);
		return repo ? `${repo.name} (${repo.type})` : 'Unknown';
	};

	const availableRepos = repositories.filter(
		(r) => !selectedRepos.some((sr) => sr.repository_id === r.id),
	);

	return (
		<div className="space-y-3">
			<span className="block text-sm font-medium text-gray-700">
				Repositories (ordered by priority)
			</span>

			{/* Selected repositories list */}
			{selectedRepos.length > 0 && (
				<div className="space-y-2 border border-gray-200 rounded-lg p-3 bg-gray-50">
					{selectedRepos.map((repo, index) => (
						<div
							key={repo.repository_id}
							className="flex items-center gap-2 bg-white rounded-lg px-3 py-2 border border-gray-200"
						>
							<span
								className={`flex-shrink-0 w-6 h-6 flex items-center justify-center rounded-full text-xs font-medium ${
									index === 0
										? 'bg-indigo-100 text-indigo-700'
										: 'bg-gray-100 text-gray-600'
								}`}
							>
								{index === 0 ? 'P' : index}
							</span>

							<span
								className={`flex-1 text-sm ${
									repo.enabled ? 'text-gray-900' : 'text-gray-400 line-through'
								}`}
							>
								{getRepoName(repo.repository_id)}
							</span>

							<div className="flex items-center gap-1">
								<button
									type="button"
									onClick={() => toggleEnabled(repo.repository_id)}
									className={`p-1 rounded ${
										repo.enabled
											? 'text-green-600 hover:bg-green-50'
											: 'text-gray-400 hover:bg-gray-50'
									}`}
									title={repo.enabled ? 'Disable' : 'Enable'}
								>
									{repo.enabled ? (
										<svg
											aria-hidden="true"
											className="w-4 h-4"
											fill="none"
											stroke="currentColor"
											viewBox="0 0 24 24"
										>
											<path
												strokeLinecap="round"
												strokeLinejoin="round"
												strokeWidth={2}
												d="M5 13l4 4L19 7"
											/>
										</svg>
									) : (
										<svg
											aria-hidden="true"
											className="w-4 h-4"
											fill="none"
											stroke="currentColor"
											viewBox="0 0 24 24"
										>
											<path
												strokeLinecap="round"
												strokeLinejoin="round"
												strokeWidth={2}
												d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636"
											/>
										</svg>
									)}
								</button>

								<button
									type="button"
									onClick={() => moveUp(index)}
									disabled={index === 0}
									className="p-1 rounded text-gray-400 hover:text-gray-600 hover:bg-gray-50 disabled:opacity-30 disabled:cursor-not-allowed"
									title="Move up"
								>
									<svg
										aria-hidden="true"
										className="w-4 h-4"
										fill="none"
										stroke="currentColor"
										viewBox="0 0 24 24"
									>
										<path
											strokeLinecap="round"
											strokeLinejoin="round"
											strokeWidth={2}
											d="M5 15l7-7 7 7"
										/>
									</svg>
								</button>

								<button
									type="button"
									onClick={() => moveDown(index)}
									disabled={index === selectedRepos.length - 1}
									className="p-1 rounded text-gray-400 hover:text-gray-600 hover:bg-gray-50 disabled:opacity-30 disabled:cursor-not-allowed"
									title="Move down"
								>
									<svg
										aria-hidden="true"
										className="w-4 h-4"
										fill="none"
										stroke="currentColor"
										viewBox="0 0 24 24"
									>
										<path
											strokeLinecap="round"
											strokeLinejoin="round"
											strokeWidth={2}
											d="M19 9l-7 7-7-7"
										/>
									</svg>
								</button>

								<button
									type="button"
									onClick={() => removeRepository(repo.repository_id)}
									className="p-1 rounded text-red-400 hover:text-red-600 hover:bg-red-50"
									title="Remove"
								>
									<svg
										aria-hidden="true"
										className="w-4 h-4"
										fill="none"
										stroke="currentColor"
										viewBox="0 0 24 24"
									>
										<path
											strokeLinecap="round"
											strokeLinejoin="round"
											strokeWidth={2}
											d="M6 18L18 6M6 6l12 12"
										/>
									</svg>
								</button>
							</div>
						</div>
					))}

					<p className="text-xs text-gray-500 mt-2">
						P = Primary repository. Backups go to primary first, then replicate
						to secondaries.
					</p>
				</div>
			)}

			{/* Add repository dropdown */}
			{availableRepos.length > 0 && (
				<div className="flex gap-2">
					<select
						value={selectedId}
						onChange={(e) => setSelectedId(e.target.value)}
						className="flex-1 px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
					>
						<option value="">Select a repository to add...</option>
						{availableRepos.map((repo) => (
							<option key={repo.id} value={repo.id}>
								{repo.name} ({repo.type})
							</option>
						))}
					</select>
					<button
						type="button"
						onClick={addRepository}
						disabled={!selectedId}
						className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
					>
						Add
					</button>
				</div>
			)}

			{selectedRepos.length === 0 && (
				<p className="text-sm text-amber-600">
					Please add at least one repository for backups.
				</p>
			)}
		</div>
	);
}
