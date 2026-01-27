import { useCallback, useEffect, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
	useClearRecentSearches,
	useDeleteRecentSearch,
	useGroupedSearch,
	useRecentSearches,
	useSaveRecentSearch,
	useSearchSuggestions,
} from '../../hooks/useSearch';
import type {
	GroupedSearchResult,
	RecentSearch,
	SearchFilter,
	SearchResultType,
	SearchSuggestion,
} from '../../lib/types';

interface GlobalSearchBarProps {
	placeholder?: string;
	className?: string;
}

type ResultItem =
	| { type: 'suggestion'; data: SearchSuggestion }
	| { type: 'recent'; data: RecentSearch }
	| { type: 'result'; data: GroupedSearchResult; category: SearchResultType };

const RESULT_TYPE_LABELS: Record<SearchResultType, string> = {
	agent: 'Agents',
	backup: 'Backups',
	snapshot: 'Snapshots',
	schedule: 'Schedules',
	repository: 'Repositories',
};

const RESULT_TYPE_ROUTES: Record<SearchResultType, string> = {
	agent: '/agents',
	backup: '/backups',
	snapshot: '/backups',
	schedule: '/schedules',
	repository: '/repositories',
};

const RESULT_TYPE_ICONS: Record<SearchResultType, React.ReactNode> = {
	agent: (
		<svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
			<path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z" />
		</svg>
	),
	backup: (
		<svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
			<path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" />
		</svg>
	),
	snapshot: (
		<svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
			<path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 9a2 2 0 012-2h.93a2 2 0 001.664-.89l.812-1.22A2 2 0 0110.07 4h3.86a2 2 0 011.664.89l.812 1.22A2 2 0 0018.07 7H19a2 2 0 012 2v9a2 2 0 01-2 2H5a2 2 0 01-2-2V9z" />
			<path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 13a3 3 0 11-6 0 3 3 0 016 0z" />
		</svg>
	),
	schedule: (
		<svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
			<path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
		</svg>
	),
	repository: (
		<svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
			<path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4" />
		</svg>
	),
};

export function GlobalSearchBar({ placeholder = 'Search...', className = '' }: GlobalSearchBarProps) {
	const navigate = useNavigate();
	const inputRef = useRef<HTMLInputElement>(null);
	const dropdownRef = useRef<HTMLDivElement>(null);
	const [isOpen, setIsOpen] = useState(false);
	const [query, setQuery] = useState('');
	const [selectedIndex, setSelectedIndex] = useState(-1);
	const [activeFilter, setActiveFilter] = useState<SearchResultType | null>(null);
	const [dateFrom, setDateFrom] = useState<string>('');
	const [dateTo, setDateTo] = useState<string>('');
	const [showFilters, setShowFilters] = useState(false);

	const searchFilter: SearchFilter | null = query.length >= 2
		? {
				q: query,
				types: activeFilter ? [activeFilter] : undefined,
				date_from: dateFrom || undefined,
				date_to: dateTo || undefined,
				limit: 5,
		  }
		: null;

	const { data: suggestionsData } = useSearchSuggestions(query, query.length >= 2 && !searchFilter);
	const { data: searchResults, isLoading: isSearching } = useGroupedSearch(searchFilter);
	const { data: recentData } = useRecentSearches(5);
	const saveSearch = useSaveRecentSearch();
	const deleteSearch = useDeleteRecentSearch();
	const clearSearches = useClearRecentSearches();

	const suggestions = suggestionsData?.suggestions ?? [];
	const recentSearches = recentData?.recent_searches ?? [];

	// Build flattened list of results for keyboard navigation
	const buildResultItems = useCallback((): ResultItem[] => {
		const items: ResultItem[] = [];

		if (query.length < 2) {
			// Show recent searches when no query
			for (const recent of recentSearches) {
				items.push({ type: 'recent', data: recent });
			}
			return items;
		}

		// Show suggestions first
		for (const suggestion of suggestions.slice(0, 5)) {
			items.push({ type: 'suggestion', data: suggestion });
		}

		// Then show grouped search results
		if (searchResults) {
			const categories: SearchResultType[] = ['agent', 'backup', 'snapshot', 'schedule', 'repository'];
			for (const category of categories) {
				if (activeFilter && activeFilter !== category) continue;
				const results = searchResults[`${category}s` as keyof typeof searchResults];
				if (Array.isArray(results)) {
					for (const result of results.slice(0, 3)) {
						items.push({ type: 'result', data: result as GroupedSearchResult, category });
					}
				}
			}
		}

		return items;
	}, [query, suggestions, searchResults, recentSearches, activeFilter]);

	const resultItems = buildResultItems();

	// Handle click outside to close dropdown
	useEffect(() => {
		const handleClickOutside = (event: MouseEvent) => {
			if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
				setIsOpen(false);
			}
		};

		document.addEventListener('mousedown', handleClickOutside);
		return () => document.removeEventListener('mousedown', handleClickOutside);
	}, []);

	// Global keyboard shortcut to focus search
	useEffect(() => {
		const handleGlobalKeyDown = (event: KeyboardEvent) => {
			// Cmd/Ctrl + K to focus search
			if ((event.metaKey || event.ctrlKey) && event.key === 'k') {
				event.preventDefault();
				inputRef.current?.focus();
				setIsOpen(true);
			}
		};

		document.addEventListener('keydown', handleGlobalKeyDown);
		return () => document.removeEventListener('keydown', handleGlobalKeyDown);
	}, []);

	const handleKeyDown = (event: React.KeyboardEvent) => {
		switch (event.key) {
			case 'ArrowDown':
				event.preventDefault();
				setSelectedIndex((prev) => Math.min(prev + 1, resultItems.length - 1));
				break;
			case 'ArrowUp':
				event.preventDefault();
				setSelectedIndex((prev) => Math.max(prev - 1, -1));
				break;
			case 'Enter':
				event.preventDefault();
				if (selectedIndex >= 0 && selectedIndex < resultItems.length) {
					handleSelectItem(resultItems[selectedIndex]);
				} else if (query.length >= 2) {
					handleSearch();
				}
				break;
			case 'Escape':
				setIsOpen(false);
				inputRef.current?.blur();
				break;
			case 'Tab':
				if (event.shiftKey) {
					// Cycle through filters backwards
					const types: (SearchResultType | null)[] = [null, 'agent', 'backup', 'snapshot', 'schedule', 'repository'];
					const currentIdx = types.indexOf(activeFilter);
					const prevIdx = currentIdx <= 0 ? types.length - 1 : currentIdx - 1;
					setActiveFilter(types[prevIdx]);
					event.preventDefault();
				} else {
					// Cycle through filters
					const types: (SearchResultType | null)[] = [null, 'agent', 'backup', 'snapshot', 'schedule', 'repository'];
					const currentIdx = types.indexOf(activeFilter);
					const nextIdx = (currentIdx + 1) % types.length;
					setActiveFilter(types[nextIdx]);
					event.preventDefault();
				}
				break;
		}
	};

	const handleSelectItem = (item: ResultItem) => {
		if (item.type === 'suggestion') {
			setQuery(item.data.text);
			navigate(`${RESULT_TYPE_ROUTES[item.data.type]}/${item.data.id}`);
		} else if (item.type === 'recent') {
			setQuery(item.data.query);
			handleSearch(item.data.query);
		} else if (item.type === 'result') {
			navigate(`${RESULT_TYPE_ROUTES[item.category]}/${item.data.id}`);
		}
		setIsOpen(false);
	};

	const handleSearch = (searchQuery?: string) => {
		const q = searchQuery ?? query;
		if (q.length < 2) return;

		// Save to recent searches
		saveSearch.mutate({ query: q, types: activeFilter ? [activeFilter] : undefined });

		// Navigate to search results page (or you could show results inline)
		const params = new URLSearchParams({ q });
		if (activeFilter) params.set('type', activeFilter);
		if (dateFrom) params.set('from', dateFrom);
		if (dateTo) params.set('to', dateTo);

		// For now, close dropdown and let user see results in dropdown
		// In a full implementation, you might navigate to a search results page
		setIsOpen(false);
	};

	const handleDeleteRecent = (event: React.MouseEvent, id: string) => {
		event.stopPropagation();
		deleteSearch.mutate(id);
	};

	const handleClearRecent = () => {
		clearSearches.mutate();
	};

	const formatDate = (dateStr: string) => {
		const date = new Date(dateStr);
		return date.toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
	};

	return (
		<div ref={dropdownRef} className={`relative ${className}`}>
			<div className="relative">
				<div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
					<svg className="h-5 w-5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
						<path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
					</svg>
				</div>
				<input
					ref={inputRef}
					type="text"
					value={query}
					onChange={(e) => {
						setQuery(e.target.value);
						setSelectedIndex(-1);
					}}
					onFocus={() => setIsOpen(true)}
					onKeyDown={handleKeyDown}
					placeholder={placeholder}
					className="block w-full pl-10 pr-20 py-2 border border-gray-300 rounded-lg bg-white text-sm placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
					aria-label="Search"
					aria-expanded={isOpen}
					aria-haspopup="listbox"
					role="combobox"
				/>
				<div className="absolute inset-y-0 right-0 flex items-center pr-2 gap-1">
					{query && (
						<button
							type="button"
							onClick={() => {
								setQuery('');
								setSelectedIndex(-1);
								inputRef.current?.focus();
							}}
							className="p-1 text-gray-400 hover:text-gray-600"
							aria-label="Clear search"
						>
							<svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
								<path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
							</svg>
						</button>
					)}
					<button
						type="button"
						onClick={() => setShowFilters(!showFilters)}
						className={`p-1 rounded ${showFilters ? 'text-indigo-600 bg-indigo-50' : 'text-gray-400 hover:text-gray-600'}`}
						aria-label="Toggle filters"
					>
						<svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
							<path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 4a1 1 0 011-1h16a1 1 0 011 1v2.586a1 1 0 01-.293.707l-6.414 6.414a1 1 0 00-.293.707V17l-4 4v-6.586a1 1 0 00-.293-.707L3.293 7.293A1 1 0 013 6.586V4z" />
						</svg>
					</button>
					<kbd className="hidden sm:inline-flex items-center px-1.5 text-xs font-medium text-gray-400 bg-gray-100 border border-gray-200 rounded">
						{navigator.platform.includes('Mac') ? '⌘' : 'Ctrl'}K
					</kbd>
				</div>
			</div>

			{/* Filters Panel */}
			{showFilters && (
				<div className="absolute top-full left-0 right-0 mt-1 p-3 bg-white border border-gray-200 rounded-lg shadow-lg z-50">
					<div className="space-y-3">
						<div>
							<label className="block text-xs font-medium text-gray-500 mb-1.5">Filter by type</label>
							<div className="flex flex-wrap gap-1.5">
								<button
									type="button"
									onClick={() => setActiveFilter(null)}
									className={`px-2 py-1 text-xs rounded-full transition-colors ${
										activeFilter === null
											? 'bg-indigo-100 text-indigo-700'
											: 'bg-gray-100 text-gray-600 hover:bg-gray-200'
									}`}
								>
									All
								</button>
								{(Object.keys(RESULT_TYPE_LABELS) as SearchResultType[]).map((type) => (
									<button
										key={type}
										type="button"
										onClick={() => setActiveFilter(activeFilter === type ? null : type)}
										className={`px-2 py-1 text-xs rounded-full transition-colors ${
											activeFilter === type
												? 'bg-indigo-100 text-indigo-700'
												: 'bg-gray-100 text-gray-600 hover:bg-gray-200'
										}`}
									>
										{RESULT_TYPE_LABELS[type]}
									</button>
								))}
							</div>
						</div>
						<div className="flex gap-3">
							<div className="flex-1">
								<label htmlFor="search-date-from" className="block text-xs font-medium text-gray-500 mb-1">From</label>
								<input
									id="search-date-from"
									type="date"
									value={dateFrom}
									onChange={(e) => setDateFrom(e.target.value)}
									className="w-full px-2 py-1 text-sm border border-gray-300 rounded focus:ring-1 focus:ring-indigo-500 focus:border-indigo-500"
								/>
							</div>
							<div className="flex-1">
								<label htmlFor="search-date-to" className="block text-xs font-medium text-gray-500 mb-1">To</label>
								<input
									id="search-date-to"
									type="date"
									value={dateTo}
									onChange={(e) => setDateTo(e.target.value)}
									className="w-full px-2 py-1 text-sm border border-gray-300 rounded focus:ring-1 focus:ring-indigo-500 focus:border-indigo-500"
								/>
							</div>
						</div>
					</div>
				</div>
			)}

			{/* Dropdown Results */}
			{isOpen && (
				<div
					className="absolute top-full left-0 right-0 mt-1 bg-white border border-gray-200 rounded-lg shadow-lg max-h-96 overflow-y-auto z-40"
					role="listbox"
				>
					{query.length < 2 && recentSearches.length > 0 && (
						<div className="p-2">
							<div className="flex items-center justify-between px-2 py-1">
								<span className="text-xs font-medium text-gray-500">Recent Searches</span>
								<button
									type="button"
									onClick={handleClearRecent}
									className="text-xs text-gray-400 hover:text-gray-600"
								>
									Clear all
								</button>
							</div>
							{recentSearches.map((recent, idx) => (
								<button
									key={recent.id}
									type="button"
									onClick={() => handleSelectItem({ type: 'recent', data: recent })}
									className={`w-full flex items-center gap-3 px-3 py-2 rounded-lg text-left ${
										selectedIndex === idx ? 'bg-indigo-50' : 'hover:bg-gray-50'
									}`}
									role="option"
									aria-selected={selectedIndex === idx}
								>
									<svg className="w-4 h-4 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
										<path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
									</svg>
									<span className="flex-1 text-sm text-gray-700 truncate">{recent.query}</span>
									{recent.types && recent.types.length > 0 && (
										<span className="text-xs text-gray-400">{recent.types.join(', ')}</span>
									)}
									<span className="text-xs text-gray-400">{formatDate(recent.created_at)}</span>
									<button
										type="button"
										onClick={(e) => handleDeleteRecent(e, recent.id)}
										className="p-1 text-gray-400 hover:text-gray-600"
										aria-label="Remove from recent"
									>
										<svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
											<path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
										</svg>
									</button>
								</button>
							))}
						</div>
					)}

					{query.length >= 2 && (
						<>
							{/* Loading State */}
							{isSearching && (
								<div className="p-4 text-center">
									<div className="w-5 h-5 border-2 border-indigo-200 border-t-indigo-600 rounded-full animate-spin mx-auto" />
									<span className="text-sm text-gray-500 mt-2 block">Searching...</span>
								</div>
							)}

							{/* Suggestions */}
							{suggestions.length > 0 && (
								<div className="p-2 border-b border-gray-100">
									<span className="px-2 text-xs font-medium text-gray-500">Suggestions</span>
									{suggestions.slice(0, 5).map((suggestion, idx) => (
										<button
											key={`${suggestion.type}-${suggestion.id}`}
											type="button"
											onClick={() => handleSelectItem({ type: 'suggestion', data: suggestion })}
											className={`w-full flex items-center gap-3 px-3 py-2 rounded-lg text-left ${
												selectedIndex === idx ? 'bg-indigo-50' : 'hover:bg-gray-50'
											}`}
											role="option"
											aria-selected={selectedIndex === idx}
										>
											<span className="text-gray-400">{RESULT_TYPE_ICONS[suggestion.type]}</span>
											<span className="flex-1 text-sm text-gray-700">{suggestion.text}</span>
											<span className="text-xs text-gray-400">{suggestion.detail}</span>
										</button>
									))}
								</div>
							)}

							{/* Grouped Results */}
							{searchResults && !isSearching && (
								<div className="p-2">
									{(Object.keys(RESULT_TYPE_LABELS) as SearchResultType[]).map((type) => {
										if (activeFilter && activeFilter !== type) return null;
										const results = searchResults[`${type}s` as keyof typeof searchResults];
										if (!Array.isArray(results) || results.length === 0) return null;

										const categoryStartIdx = suggestions.length +
											(Object.keys(RESULT_TYPE_LABELS) as SearchResultType[])
												.filter(t => t !== type && (!activeFilter || activeFilter === t))
												.reduce((acc, t) => {
													const r = searchResults[`${t}s` as keyof typeof searchResults];
													return acc + (Array.isArray(r) ? Math.min(r.length, 3) : 0);
												}, 0);

										return (
											<div key={type} className="mb-2">
												<span className="px-2 text-xs font-medium text-gray-500">{RESULT_TYPE_LABELS[type]}</span>
												{(results as GroupedSearchResult[]).slice(0, 3).map((result, idx) => {
													const globalIdx = categoryStartIdx + idx;
													return (
														<button
															key={result.id}
															type="button"
															onClick={() => handleSelectItem({ type: 'result', data: result, category: type })}
															className={`w-full flex items-center gap-3 px-3 py-2 rounded-lg text-left ${
																selectedIndex === globalIdx ? 'bg-indigo-50' : 'hover:bg-gray-50'
															}`}
															role="option"
															aria-selected={selectedIndex === globalIdx}
														>
															<span className="text-gray-400">{RESULT_TYPE_ICONS[type]}</span>
															<div className="flex-1 min-w-0">
																<div className="text-sm text-gray-700 truncate">{result.name}</div>
																{result.description && (
																	<div className="text-xs text-gray-500 truncate">{result.description}</div>
																)}
															</div>
															{result.status && (
																<span className={`px-1.5 py-0.5 text-xs rounded-full ${
																	result.status === 'active' || result.status === 'completed' || result.status === 'enabled'
																		? 'bg-green-100 text-green-700'
																		: result.status === 'failed' || result.status === 'offline' || result.status === 'disabled'
																		? 'bg-red-100 text-red-700'
																		: 'bg-gray-100 text-gray-600'
																}`}>
																	{result.status}
																</span>
															)}
														</button>
													);
												})}
											</div>
										);
									})}
								</div>
							)}

							{/* No Results */}
							{searchResults && !isSearching && searchResults.total === 0 && (
								<div className="p-4 text-center text-sm text-gray-500">
									No results found for "{query}"
								</div>
							)}

							{/* Search tip */}
							<div className="p-2 border-t border-gray-100 bg-gray-50">
								<div className="flex items-center justify-between px-2 text-xs text-gray-500">
									<span>
										<kbd className="px-1 py-0.5 bg-white border border-gray-200 rounded text-[10px]">Tab</kbd>
										{' '}to filter by type
									</span>
									<span>
										<kbd className="px-1 py-0.5 bg-white border border-gray-200 rounded text-[10px]">Enter</kbd>
										{' '}to search
									</span>
								</div>
							</div>
						</>
					)}

					{query.length < 2 && recentSearches.length === 0 && (
						<div className="p-4 text-center text-sm text-gray-500">
							Start typing to search (min 2 characters)
						</div>
					)}
				</div>
			)}
		</div>
	);
}
