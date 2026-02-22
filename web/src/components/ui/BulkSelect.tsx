interface BulkSelectCheckboxProps {
	checked: boolean;
	onChange: () => void;
	disabled?: boolean;
}

export function BulkSelectCheckbox({
	checked,
	onChange,
	disabled = false,
}: BulkSelectCheckboxProps) {
	return (
		<input
			type="checkbox"
			checked={checked}
			onChange={onChange}
			disabled={disabled}
			className="h-4 w-4 rounded border-gray-300 text-indigo-600 focus:ring-indigo-500 disabled:opacity-50 disabled:cursor-not-allowed"
		/>
	);
}

interface BulkSelectHeaderProps {
	isAllSelected: boolean;
	isPartiallySelected: boolean;
	onToggleAll: () => void;
	selectedCount: number;
	totalCount: number;
	disabled?: boolean;
}

export function BulkSelectHeader({
	isAllSelected,
	isPartiallySelected,
	onToggleAll,
	totalCount,
	disabled = false,
}: BulkSelectHeaderProps) {
	return (
		<div className="flex items-center gap-2">
			<input
				type="checkbox"
				checked={isAllSelected}
				ref={(el) => {
					if (el) {
						el.indeterminate = isPartiallySelected;
					}
				}}
				onChange={onToggleAll}
				disabled={disabled || totalCount === 0}
				className="h-4 w-4 rounded border-gray-300 text-indigo-600 focus:ring-indigo-500 disabled:opacity-50 disabled:cursor-not-allowed"
				aria-label={isAllSelected ? 'Deselect all' : 'Select all'}
			/>
		</div>
	);
}

interface SelectionIndicatorProps {
	selectedCount: number;
	totalCount: number;
	itemLabel?: string;
}

export function SelectionIndicator({
	selectedCount,
	totalCount,
	itemLabel = 'item',
}: SelectionIndicatorProps) {
	if (selectedCount === 0) return null;

	const pluralLabel = selectedCount === 1 ? itemLabel : `${itemLabel}s`;

	return (
		<div className="flex items-center gap-2 px-3 py-1.5 bg-indigo-50 dark:bg-indigo-900/30 text-indigo-700 dark:text-indigo-300 rounded-lg text-sm font-medium">
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
			<span>
				{selectedCount} of {totalCount} {pluralLabel} selected
			</span>
		</div>
	);
}

interface BulkSelectToolbarProps {
	selectedCount: number;
	totalCount: number;
	onSelectAll: () => void;
	onDeselectAll: () => void;
	itemLabel?: string;
	children?: React.ReactNode;
}

export function BulkSelectToolbar({
	selectedCount,
	totalCount,
	onSelectAll,
	onDeselectAll,
	itemLabel = 'item',
	children,
}: BulkSelectToolbarProps) {
	if (selectedCount === 0) return null;

	const pluralLabel = selectedCount === 1 ? itemLabel : `${itemLabel}s`;

	return (
		<div className="flex items-center justify-between gap-4 px-4 py-3 bg-indigo-50 dark:bg-indigo-900/30 border border-indigo-200 dark:border-indigo-800 rounded-lg">
			<div className="flex items-center gap-3">
				<span className="text-sm font-medium text-indigo-700 dark:text-indigo-300">
					{selectedCount} {pluralLabel} selected
				</span>
				{selectedCount < totalCount && (
					<button
						type="button"
						onClick={onSelectAll}
						className="text-sm text-indigo-600 dark:text-indigo-400 hover:text-indigo-800 dark:hover:text-indigo-300 underline"
					>
						Select all {totalCount}
					</button>
				)}
				<button
					type="button"
					onClick={onDeselectAll}
					className="text-sm text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-300"
				>
					Clear selection
				</button>
			</div>
			{children && <div className="flex items-center gap-2">{children}</div>}
		</div>
	);
}
