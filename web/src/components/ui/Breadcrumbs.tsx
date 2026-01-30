import { Link } from 'react-router-dom';
import type { BreadcrumbItem } from '../../hooks/useBreadcrumbs';
import { useBreadcrumbs } from '../../hooks/useBreadcrumbs';

interface BreadcrumbsProps {
	items?: BreadcrumbItem[];
	className?: string;
}

export function Breadcrumbs({ items, className = '' }: BreadcrumbsProps) {
	const { breadcrumbs: autoBreadcrumbs, isLoading } = useBreadcrumbs();
	const breadcrumbs = items ?? autoBreadcrumbs;

	// Don't render if only Dashboard (home page)
	if (breadcrumbs.length <= 1) {
		return null;
	}

	return (
		<nav aria-label="Breadcrumb" className={`mb-4 ${className}`}>
			<ol className="flex items-center space-x-2 text-sm">
				{breadcrumbs.map((item, index) => (
					<li key={item.path} className="flex items-center">
						{index > 0 && (
							<svg
								aria-hidden="true"
								className="w-4 h-4 text-gray-400 dark:text-gray-500 mx-2"
								fill="none"
								stroke="currentColor"
								viewBox="0 0 24 24"
							>
								<path
									strokeLinecap="round"
									strokeLinejoin="round"
									strokeWidth={2}
									d="M9 5l7 7-7 7"
								/>
							</svg>
						)}
						{item.isCurrentPage ? (
							<span
								className="text-gray-900 dark:text-white font-medium"
								aria-current="page"
							>
								{isLoading && index === breadcrumbs.length - 1 ? (
									<span className="inline-block w-20 h-4 bg-gray-200 dark:bg-gray-700 rounded animate-pulse" />
								) : (
									item.label
								)}
							</span>
						) : (
							<Link
								to={item.path}
								className="text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300 transition-colors"
							>
								{index === 0 ? (
									<span className="flex items-center gap-1">
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
												d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6"
											/>
										</svg>
										<span className="sr-only">{item.label}</span>
									</span>
								) : (
									item.label
								)}
							</Link>
						)}
					</li>
				))}
			</ol>
		</nav>
	);
}

interface BreadcrumbsWithItemsProps {
	items: BreadcrumbItem[];
	className?: string;
}

export function BreadcrumbsWithItems({
	items,
	className = '',
}: BreadcrumbsWithItemsProps) {
	// Don't render if only one item
	if (items.length <= 1) {
		return null;
	}

	return (
		<nav aria-label="Breadcrumb" className={`mb-4 ${className}`}>
			<ol className="flex items-center space-x-2 text-sm">
				{items.map((item, index) => (
					<li key={item.path} className="flex items-center">
						{index > 0 && (
							<svg
								aria-hidden="true"
								className="w-4 h-4 text-gray-400 dark:text-gray-500 mx-2"
								fill="none"
								stroke="currentColor"
								viewBox="0 0 24 24"
							>
								<path
									strokeLinecap="round"
									strokeLinejoin="round"
									strokeWidth={2}
									d="M9 5l7 7-7 7"
								/>
							</svg>
						)}
						{item.isCurrentPage ? (
							<span
								className="text-gray-900 dark:text-white font-medium"
								aria-current="page"
							>
								{item.label}
							</span>
						) : (
							<Link
								to={item.path}
								className="text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300 transition-colors"
							>
								{index === 0 ? (
									<span className="flex items-center gap-1">
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
												d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6"
											/>
										</svg>
										<span className="sr-only">{item.label}</span>
									</span>
								) : (
									item.label
								)}
							</Link>
						)}
					</li>
				))}
			</ol>
		</nav>
	);
}
