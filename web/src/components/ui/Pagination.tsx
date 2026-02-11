interface PaginationProps {
	currentPage: number;
	totalPages: number;
	onPageChange: (page: number) => void;
}

export function Pagination({
	currentPage,
	totalPages,
	onPageChange,
}: PaginationProps) {
	if (totalPages <= 1) return null;

	function getPageNumbers(): (number | string)[] {
		const pages: (number | string)[] = [];

		if (totalPages <= 7) {
			for (let i = 1; i <= totalPages; i++) pages.push(i);
			return pages;
		}

		pages.push(1);
		if (currentPage > 3) pages.push('ellipsis-start');

		const start = Math.max(2, currentPage - 1);
		const end = Math.min(totalPages - 1, currentPage + 1);
		for (let i = start; i <= end; i++) pages.push(i);

		if (currentPage < totalPages - 2) pages.push('ellipsis-end');
		pages.push(totalPages);

		return pages;
	}

	return (
		<nav
			aria-label="Pagination"
			className="flex items-center justify-center gap-1"
		>
			<button
				type="button"
				onClick={() => onPageChange(currentPage - 1)}
				disabled={currentPage === 1}
				className="rounded-md px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-100 disabled:cursor-not-allowed disabled:opacity-50"
			>
				Previous
			</button>
			{getPageNumbers().map((page) =>
				typeof page === 'string' ? (
					<span key={page} className="px-2 text-gray-400">
						...
					</span>
				) : (
					<button
						key={page}
						type="button"
						onClick={() => onPageChange(page)}
						className={`rounded-md px-3 py-2 text-sm font-medium ${
							page === currentPage
								? 'bg-indigo-600 text-white'
								: 'text-gray-700 hover:bg-gray-100'
						}`}
						aria-current={page === currentPage ? 'page' : undefined}
					>
						{page}
					</button>
				),
			)}
			<button
				type="button"
				onClick={() => onPageChange(currentPage + 1)}
				disabled={currentPage === totalPages}
				className="rounded-md px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-100 disabled:cursor-not-allowed disabled:opacity-50"
			>
				Next
			</button>
		</nav>
	);
}
