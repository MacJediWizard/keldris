import { useEffect, useRef, useState } from 'react';
import { useMe } from '../../hooks/useAuth';
import {
	useDeleteSavedFilter,
	useSavedFilters,
	useUpdateSavedFilter,
} from '../../hooks/useSavedFilters';
import type { SavedFilter } from '../../lib/types';

interface SavedFiltersDropdownProps {
	entityType: string;
	onApplyFilter: (filters: Record<string, unknown>) => void;
	currentFilters?: Record<string, unknown>;
}

export function SavedFiltersDropdown({
	entityType,
	onApplyFilter,
}: SavedFiltersDropdownProps) {
	const [isOpen, setIsOpen] = useState(false);
	const dropdownRef = useRef<HTMLDivElement>(null);
	const { data: user } = useMe();
	const { data: filters, isLoading } = useSavedFilters(entityType);
	const deleteFilter = useDeleteSavedFilter();
	const updateFilter = useUpdateSavedFilter();

	// Close dropdown when clicking outside
	useEffect(() => {
		function handleClickOutside(event: MouseEvent) {
			if (
				dropdownRef.current &&
				!dropdownRef.current.contains(event.target as Node)
			) {
				setIsOpen(false);
			}
		}

		document.addEventListener('mousedown', handleClickOutside);
		return () => document.removeEventListener('mousedown', handleClickOutside);
	}, []);

	const handleApplyFilter = (filter: SavedFilter) => {
		onApplyFilter(filter.filters);
		setIsOpen(false);
	};

	const handleDeleteFilter = async (e: React.MouseEvent, filterId: string) => {
		e.stopPropagation();
		if (confirm('Are you sure you want to delete this saved filter?')) {
			await deleteFilter.mutateAsync(filterId);
		}
	};

	const handleToggleDefault = async (
		e: React.MouseEvent,
		filter: SavedFilter,
	) => {
		e.stopPropagation();
		await updateFilter.mutateAsync({
			id: filter.id,
			data: { is_default: !filter.is_default },
		});
	};

	if (!filters || filters.length === 0) return null;

	const myFilters = filters.filter((f) => f.user_id === user?.id);
	const sharedFilters = filters.filter(
		(f) => f.user_id !== user?.id && f.shared,
	);

	return (
		<div className="relative" ref={dropdownRef}>
			<button
				type="button"
				onClick={() => setIsOpen(!isOpen)}
				disabled={isLoading}
				className="inline-flex items-center gap-2 px-3 py-1.5 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 disabled:opacity-50"
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
						d="M3 4a1 1 0 011-1h16a1 1 0 011 1v2.586a1 1 0 01-.293.707l-6.414 6.414a1 1 0 00-.293.707V17l-4 4v-6.586a1 1 0 00-.293-.707L3.293 7.293A1 1 0 013 6.586V4z"
					/>
				</svg>
				Saved Filters
				<svg
					className={`w-4 h-4 transition-transform ${isOpen ? 'rotate-180' : ''}`}
					fill="none"
					stroke="currentColor"
					viewBox="0 0 24 24"
					aria-hidden="true"
				>
					<path
						strokeLinecap="round"
						strokeLinejoin="round"
						strokeWidth={2}
						d="M19 9l-7 7-7-7"
					/>
				</svg>
			</button>

			{isOpen && (
				<div className="absolute z-50 mt-2 w-72 bg-white border border-gray-200 rounded-lg shadow-lg">
					<div className="p-2 max-h-80 overflow-y-auto">
						{myFilters.length > 0 && (
							<div className="mb-2">
								<p className="px-2 py-1 text-xs font-semibold text-gray-500 uppercase tracking-wider">
									My Filters
								</p>
								{myFilters.map((filter) => (
									<FilterItem
										key={filter.id}
										filter={filter}
										isOwner={true}
										onApply={() => handleApplyFilter(filter)}
										onDelete={(e) => handleDeleteFilter(e, filter.id)}
										onToggleDefault={(e) => handleToggleDefault(e, filter)}
									/>
								))}
							</div>
						)}

						{sharedFilters.length > 0 && (
							<div>
								{myFilters.length > 0 && (
									<div className="border-t border-gray-100 my-2" />
								)}
								<p className="px-2 py-1 text-xs font-semibold text-gray-500 uppercase tracking-wider">
									Shared Filters
								</p>
								{sharedFilters.map((filter) => (
									<FilterItem
										key={filter.id}
										filter={filter}
										isOwner={false}
										onApply={() => handleApplyFilter(filter)}
									/>
								))}
							</div>
						)}
					</div>
				</div>
			)}
		</div>
	);
}

interface FilterItemProps {
	filter: SavedFilter;
	isOwner: boolean;
	onApply: () => void;
	onDelete?: (e: React.MouseEvent) => void;
	onToggleDefault?: (e: React.MouseEvent) => void;
}

function FilterItem({
	filter,
	isOwner,
	onApply,
	onDelete,
	onToggleDefault,
}: FilterItemProps) {
	return (
		<button
			type="button"
			className="w-full flex items-center justify-between px-2 py-2 rounded-lg hover:bg-gray-50 cursor-pointer group text-left"
			onClick={onApply}
		>
			<div className="flex items-center gap-2 min-w-0">
				<div className="min-w-0 flex-1">
					<div className="flex items-center gap-2">
						<span className="text-sm font-medium text-gray-900 truncate">
							{filter.name}
						</span>
						{filter.is_default && (
							<span className="flex-shrink-0 px-1.5 py-0.5 text-xs font-medium text-indigo-600 bg-indigo-50 rounded">
								Default
							</span>
						)}
						{filter.shared && isOwner && (
							<span className="flex-shrink-0 px-1.5 py-0.5 text-xs font-medium text-green-600 bg-green-50 rounded">
								Shared
							</span>
						)}
					</div>
				</div>
			</div>

			{isOwner && (
				<div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
					{onToggleDefault && (
						<button
							type="button"
							onClick={onToggleDefault}
							className="p-1 text-gray-400 hover:text-indigo-600"
							title={filter.is_default ? 'Remove as default' : 'Set as default'}
						>
							<svg
								className={`w-4 h-4 ${filter.is_default ? 'fill-current text-indigo-600' : ''}`}
								fill={filter.is_default ? 'currentColor' : 'none'}
								stroke="currentColor"
								viewBox="0 0 24 24"
								aria-hidden="true"
							>
								<path
									strokeLinecap="round"
									strokeLinejoin="round"
									strokeWidth={2}
									d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z"
								/>
							</svg>
						</button>
					)}
					{onDelete && (
						<button
							type="button"
							onClick={onDelete}
							className="p-1 text-gray-400 hover:text-red-600"
							title="Delete filter"
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
									d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
								/>
							</svg>
						</button>
					)}
				</div>
			)}
		</button>
	);
}
