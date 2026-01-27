import { useEffect, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
	useClearRecentItems,
	useDeleteRecentItem,
	useRecentItems,
} from '../../hooks/useRecentItems';
import type { RecentItem, RecentItemType } from '../../lib/types';

// Icon mapping for different item types
const typeIcons: Record<RecentItemType, React.ReactNode> = {
	agent: (
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
				d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"
			/>
		</svg>
	),
	repository: (
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
				d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4"
			/>
		</svg>
	),
	schedule: (
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
				d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
			/>
		</svg>
	),
	backup: (
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
				d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12"
			/>
		</svg>
	),
	policy: (
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
				d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"
			/>
		</svg>
	),
	snapshot: (
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
				d="M3 9a2 2 0 012-2h.93a2 2 0 001.664-.89l.812-1.22A2 2 0 0110.07 4h3.86a2 2 0 011.664.89l.812 1.22A2 2 0 0018.07 7H19a2 2 0 012 2v9a2 2 0 01-2 2H5a2 2 0 01-2-2V9z"
			/>
			<path
				strokeLinecap="round"
				strokeLinejoin="round"
				strokeWidth={2}
				d="M15 13a3 3 0 11-6 0 3 3 0 016 0z"
			/>
		</svg>
	),
};

const typeLabels: Record<RecentItemType, string> = {
	agent: 'Agents',
	repository: 'Repositories',
	schedule: 'Schedules',
	backup: 'Backups',
	policy: 'Policies',
	snapshot: 'Snapshots',
};

function formatRelativeTime(dateString: string): string {
	const date = new Date(dateString);
	const now = new Date();
	const diffMs = now.getTime() - date.getTime();
	const diffMins = Math.floor(diffMs / 60000);
	const diffHours = Math.floor(diffMs / 3600000);
	const diffDays = Math.floor(diffMs / 86400000);

	if (diffMins < 1) return 'Just now';
	if (diffMins < 60) return `${diffMins}m ago`;
	if (diffHours < 24) return `${diffHours}h ago`;
	if (diffDays < 7) return `${diffDays}d ago`;
	return date.toLocaleDateString();
}

interface RecentItemsDropdownProps {
	className?: string;
}

export function RecentItemsDropdown({ className }: RecentItemsDropdownProps) {
	const [isOpen, setIsOpen] = useState(false);
	const dropdownRef = useRef<HTMLDivElement>(null);
	const navigate = useNavigate();
	const { data: items, isLoading } = useRecentItems();
	const deleteItem = useDeleteRecentItem();
	const clearAll = useClearRecentItems();

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

	const handleNavigate = (item: RecentItem) => {
		navigate(item.page_path);
		setIsOpen(false);
	};

	const handleDelete = async (e: React.MouseEvent, id: string) => {
		e.stopPropagation();
		await deleteItem.mutateAsync(id);
	};

	const handleClearAll = async () => {
		if (confirm('Are you sure you want to clear all recent items?')) {
			await clearAll.mutateAsync();
		}
	};

	// Group items by type
	const groupedItems = (items ?? []).reduce(
		(acc, item) => {
			if (!acc[item.item_type]) {
				acc[item.item_type] = [];
			}
			acc[item.item_type].push(item);
			return acc;
		},
		{} as Record<RecentItemType, RecentItem[]>,
	);

	const itemTypes = Object.keys(groupedItems) as RecentItemType[];
	const hasItems = itemTypes.length > 0;

	return (
		<div className={`relative ${className ?? ''}`} ref={dropdownRef}>
			<button
				type="button"
				onClick={() => setIsOpen(!isOpen)}
				disabled={isLoading}
				className="inline-flex items-center justify-center w-9 h-9 text-gray-500 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 hover:text-gray-700 disabled:opacity-50 transition-colors"
				title="Recent Items"
			>
				<svg
					className="w-5 h-5"
					fill="none"
					stroke="currentColor"
					viewBox="0 0 24 24"
					aria-hidden="true"
				>
					<path
						strokeLinecap="round"
						strokeLinejoin="round"
						strokeWidth={2}
						d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
					/>
				</svg>
			</button>

			{isOpen && (
				<div className="absolute right-0 z-50 mt-2 w-80 bg-white border border-gray-200 rounded-lg shadow-lg">
					<div className="flex items-center justify-between px-4 py-3 border-b border-gray-100">
						<h3 className="text-sm font-semibold text-gray-900">
							Recently Viewed
						</h3>
						{hasItems && (
							<button
								type="button"
								onClick={handleClearAll}
								className="text-xs text-gray-500 hover:text-red-600 transition-colors"
							>
								Clear all
							</button>
						)}
					</div>

					<div className="max-h-96 overflow-y-auto">
						{!hasItems ? (
							<div className="px-4 py-8 text-center text-gray-500">
								<svg
									className="w-8 h-8 mx-auto mb-2 text-gray-400"
									fill="none"
									stroke="currentColor"
									viewBox="0 0 24 24"
									aria-hidden="true"
								>
									<path
										strokeLinecap="round"
										strokeLinejoin="round"
										strokeWidth={2}
										d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
									/>
								</svg>
								<p className="text-sm">No recent items</p>
								<p className="text-xs text-gray-400 mt-1">
									Items you view will appear here
								</p>
							</div>
						) : (
							<div className="p-2">
								{itemTypes.map((type, typeIndex) => (
									<div key={type}>
										{typeIndex > 0 && (
											<div className="border-t border-gray-100 my-2" />
										)}
										<p className="px-2 py-1 text-xs font-semibold text-gray-500 uppercase tracking-wider">
											{typeLabels[type]}
										</p>
										{groupedItems[type].map((item) => (
											<RecentItemRow
												key={item.id}
												item={item}
												icon={typeIcons[type]}
												onNavigate={() => handleNavigate(item)}
												onDelete={(e) => handleDelete(e, item.id)}
											/>
										))}
									</div>
								))}
							</div>
						)}
					</div>
				</div>
			)}
		</div>
	);
}

interface RecentItemRowProps {
	item: RecentItem;
	icon: React.ReactNode;
	onNavigate: () => void;
	onDelete: (e: React.MouseEvent) => void;
}

function RecentItemRow({ item, icon, onNavigate, onDelete }: RecentItemRowProps) {
	return (
		<button
			type="button"
			className="w-full flex items-center gap-3 px-2 py-2 rounded-lg hover:bg-gray-50 cursor-pointer group text-left"
			onClick={onNavigate}
		>
			<span className="flex-shrink-0 text-gray-400">{icon}</span>
			<div className="flex-1 min-w-0">
				<p className="text-sm font-medium text-gray-900 truncate">
					{item.item_name}
				</p>
				<p className="text-xs text-gray-500">
					{formatRelativeTime(item.viewed_at)}
				</p>
			</div>
			<button
				type="button"
				onClick={onDelete}
				className="flex-shrink-0 p-1 text-gray-400 opacity-0 group-hover:opacity-100 hover:text-red-600 transition-all"
				title="Remove from recent"
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
						d="M6 18L18 6M6 6l12 12"
					/>
				</svg>
			</button>
		</button>
	);
}
