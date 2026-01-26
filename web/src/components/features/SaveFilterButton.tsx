import { useState } from 'react';
import { useCreateSavedFilter } from '../../hooks/useSavedFilters';

interface SaveFilterButtonProps {
	entityType: string;
	filters: Record<string, unknown>;
	disabled?: boolean;
}

export function SaveFilterButton({
	entityType,
	filters,
	disabled,
}: SaveFilterButtonProps) {
	const [isOpen, setIsOpen] = useState(false);
	const [name, setName] = useState('');
	const [shared, setShared] = useState(false);
	const [isDefault, setIsDefault] = useState(false);
	const createFilter = useCreateSavedFilter();

	const handleSave = async () => {
		if (!name.trim()) return;

		try {
			await createFilter.mutateAsync({
				name: name.trim(),
				entity_type: entityType,
				filters,
				shared,
				is_default: isDefault,
			});
			setIsOpen(false);
			setName('');
			setShared(false);
			setIsDefault(false);
		} catch (error) {
			console.error('Failed to save filter:', error);
		}
	};

	const hasFilters = Object.keys(filters).some((key) => {
		const value = filters[key];
		return value !== undefined && value !== null && value !== '' && value !== 'all';
	});

	if (!hasFilters) return null;

	return (
		<div className="relative">
			<button
				type="button"
				onClick={() => setIsOpen(true)}
				disabled={disabled || createFilter.isPending}
				className="inline-flex items-center gap-2 px-3 py-1.5 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
			>
				<svg
					className="w-4 h-4"
					fill="none"
					stroke="currentColor"
					viewBox="0 0 24 24"
					aria-hidden="true"
				>
					<path
						strokeLinecap="round"
						strokeLinejoin="round"
						strokeWidth={2}
						d="M5 5a2 2 0 012-2h10a2 2 0 012 2v16l-7-3.5L5 21V5z"
					/>
				</svg>
				Save Filter
			</button>

			{isOpen && (
				<div className="fixed inset-0 z-50 flex items-center justify-center">
					<div
						className="absolute inset-0 bg-black/50"
						onClick={() => setIsOpen(false)}
						onKeyDown={(e) => e.key === 'Escape' && setIsOpen(false)}
						role="button"
						tabIndex={0}
						aria-label="Close modal"
					/>
					<div className="relative bg-white rounded-xl shadow-2xl w-full max-w-md mx-4 overflow-hidden">
						<div className="px-6 py-4 border-b border-gray-200">
							<h3 className="text-lg font-semibold text-gray-900">
								Save Current Filter
							</h3>
							<p className="text-sm text-gray-500 mt-1">
								Save this filter configuration for quick access later.
							</p>
						</div>

						<div className="px-6 py-4 space-y-4">
							<div>
								<label
									htmlFor="filter-name"
									className="block text-sm font-medium text-gray-700 mb-1"
								>
									Filter Name
								</label>
								<input
									id="filter-name"
									type="text"
									value={name}
									onChange={(e) => setName(e.target.value)}
									placeholder="e.g., Failed Backups Last Week"
									className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
									autoFocus
								/>
							</div>

							<div className="space-y-3">
								<label className="flex items-center gap-3 cursor-pointer">
									<input
										type="checkbox"
										checked={shared}
										onChange={(e) => setShared(e.target.checked)}
										className="w-4 h-4 text-indigo-600 border-gray-300 rounded focus:ring-indigo-500"
									/>
									<div>
										<span className="text-sm font-medium text-gray-700">
											Share with organization
										</span>
										<p className="text-xs text-gray-500">
											Other team members can use this filter
										</p>
									</div>
								</label>

								<label className="flex items-center gap-3 cursor-pointer">
									<input
										type="checkbox"
										checked={isDefault}
										onChange={(e) => setIsDefault(e.target.checked)}
										className="w-4 h-4 text-indigo-600 border-gray-300 rounded focus:ring-indigo-500"
									/>
									<div>
										<span className="text-sm font-medium text-gray-700">
											Set as default
										</span>
										<p className="text-xs text-gray-500">
											Apply this filter automatically when visiting this page
										</p>
									</div>
								</label>
							</div>
						</div>

						<div className="px-6 py-4 bg-gray-50 border-t border-gray-200 flex justify-end gap-3">
							<button
								type="button"
								onClick={() => setIsOpen(false)}
								className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50"
							>
								Cancel
							</button>
							<button
								type="button"
								onClick={handleSave}
								disabled={!name.trim() || createFilter.isPending}
								className="px-4 py-2 text-sm font-medium text-white bg-indigo-600 rounded-lg hover:bg-indigo-700 disabled:opacity-50 disabled:cursor-not-allowed"
							>
								{createFilter.isPending ? 'Saving...' : 'Save Filter'}
							</button>
						</div>
					</div>
				</div>
			)}
		</div>
	);
}
