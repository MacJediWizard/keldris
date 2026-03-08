import { type ReactNode, useState } from 'react';

export interface Column<T> {
	key: string;
	header: string;
	sortable?: boolean;
	render?: (row: T) => ReactNode;
}

interface DataTableProps<T> {
	columns: Column<T>[];
	data: T[];
	keyField: keyof T;
	loading?: boolean;
	emptyMessage?: string;
	pageSize?: number;
}

type SortDirection = 'asc' | 'desc';

export function DataTable<T extends Record<string, unknown>>({
	columns,
	data,
	keyField,
	loading = false,
	emptyMessage = 'No data available',
	pageSize = 10,
}: DataTableProps<T>) {
	const [sortKey, setSortKey] = useState<string | null>(null);
	const [sortDir, setSortDir] = useState<SortDirection>('asc');
	const [currentPage, setCurrentPage] = useState(1);

	function handleSort(key: string) {
		if (sortKey === key) {
			setSortDir(sortDir === 'asc' ? 'desc' : 'asc');
		} else {
			setSortKey(key);
			setSortDir('asc');
		}
		setCurrentPage(1);
	}

	const sortedData = [...data].sort((a, b) => {
		if (!sortKey) return 0;
		const aVal = a[sortKey];
		const bVal = b[sortKey];
		if (aVal == null || bVal == null) return 0;
		const cmp = String(aVal).localeCompare(String(bVal));
		return sortDir === 'asc' ? cmp : -cmp;
	});

	const totalPages = Math.max(1, Math.ceil(sortedData.length / pageSize));
	const start = (currentPage - 1) * pageSize;
	const pageData = sortedData.slice(start, start + pageSize);

	if (loading) {
		return (
			<div className="flex h-40 items-center justify-center">
				<div className="h-8 w-8 animate-spin rounded-full border-4 border-gray-200 border-t-indigo-600 dark:border-gray-700 dark:border-t-indigo-400" />
			</div>
		);
	}

	return (
		<div>
			<div className="overflow-x-auto">
				<table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
					<thead className="bg-gray-50 dark:bg-gray-900">
						<tr>
							{columns.map((col) => (
								<th
									key={col.key}
									scope="col"
									className={`px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-gray-400 ${
										col.sortable
											? 'cursor-pointer select-none hover:text-gray-700 dark:hover:text-gray-200'
											: ''
									}`}
									onClick={col.sortable ? () => handleSort(col.key) : undefined}
									onKeyDown={
										col.sortable
											? (e) => {
													if (e.key === 'Enter' || e.key === ' ') {
														e.preventDefault();
														handleSort(col.key);
													}
												}
											: undefined
									}
								>
									<span className="inline-flex items-center gap-1">
										{col.header}
										{col.sortable && sortKey === col.key && (
											<span
												aria-label={
													sortDir === 'asc'
														? 'sorted ascending'
														: 'sorted descending'
												}
											>
												{sortDir === 'asc' ? '\u2191' : '\u2193'}
											</span>
										)}
									</span>
								</th>
							))}
						</tr>
					</thead>
					<tbody className="divide-y divide-gray-200 bg-white dark:divide-gray-700 dark:bg-gray-800">
						{pageData.length === 0 ? (
							<tr>
								<td
									colSpan={columns.length}
									className="px-6 py-8 text-center text-sm text-gray-500 dark:text-gray-400"
								>
									{emptyMessage}
								</td>
							</tr>
						) : (
							pageData.map((row) => (
								<tr
									key={String(row[keyField])}
									className="hover:bg-gray-50 dark:hover:bg-gray-700"
								>
									{columns.map((col) => (
										<td
											key={col.key}
											className="whitespace-nowrap px-6 py-4 text-sm text-gray-900 dark:text-white"
										>
											{col.render
												? col.render(row)
												: String(row[col.key] ?? '')}
										</td>
									))}
								</tr>
							))
						)}
					</tbody>
				</table>
			</div>
			{totalPages > 1 && (
				<div className="flex items-center justify-between border-t border-gray-200 px-4 py-3 dark:border-gray-700">
					<p className="text-sm text-gray-700 dark:text-gray-300">
						Showing {start + 1} to{' '}
						{Math.min(start + pageSize, sortedData.length)} of{' '}
						{sortedData.length} results
					</p>
					<div className="flex gap-1">
						<button
							type="button"
							onClick={() => setCurrentPage((p) => Math.max(1, p - 1))}
							disabled={currentPage === 1}
							className="rounded-md px-3 py-1 text-sm text-gray-700 hover:bg-gray-100 disabled:cursor-not-allowed disabled:opacity-50 dark:text-gray-300 dark:hover:bg-gray-700"
						>
							Previous
						</button>
						<button
							type="button"
							onClick={() => setCurrentPage((p) => Math.min(totalPages, p + 1))}
							disabled={currentPage === totalPages}
							className="rounded-md px-3 py-1 text-sm text-gray-700 hover:bg-gray-100 disabled:cursor-not-allowed disabled:opacity-50 dark:text-gray-300 dark:hover:bg-gray-700"
						>
							Next
						</button>
					</div>
				</div>
			)}
		</div>
	);
}
