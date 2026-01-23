import { useCallback, useEffect, useRef, useState } from 'react';
import { useAgentLogs } from '../../hooks/useAgents';
import type { AgentLog, AgentLogFilter, LogLevel } from '../../lib/types';
import { formatDateTime } from '../../lib/utils';

interface AgentLogViewerProps {
	agentId: string;
}

const LOG_LEVEL_COLORS: Record<LogLevel, { bg: string; text: string; dot: string }> = {
	debug: { bg: 'bg-gray-100', text: 'text-gray-700', dot: 'bg-gray-400' },
	info: { bg: 'bg-blue-100', text: 'text-blue-700', dot: 'bg-blue-400' },
	warn: { bg: 'bg-yellow-100', text: 'text-yellow-700', dot: 'bg-yellow-400' },
	error: { bg: 'bg-red-100', text: 'text-red-700', dot: 'bg-red-400' },
};

function LogRow({ log }: { log: AgentLog }) {
	const colors = LOG_LEVEL_COLORS[log.level] || LOG_LEVEL_COLORS.info;
	const [expanded, setExpanded] = useState(false);

	return (
		<div className="border-b border-gray-100 last:border-0 hover:bg-gray-50">
			<div
				className="px-4 py-2 flex items-start gap-3 cursor-pointer"
				onClick={() => setExpanded(!expanded)}
				onKeyDown={(e) => {
					if (e.key === 'Enter' || e.key === ' ') {
						e.preventDefault();
						setExpanded(!expanded);
					}
				}}
				role="button"
				tabIndex={0}
			>
				<span className="text-xs text-gray-400 whitespace-nowrap font-mono pt-0.5">
					{formatDateTime(log.timestamp)}
				</span>
				<span
					className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium ${colors.bg} ${colors.text}`}
				>
					<span className={`w-1.5 h-1.5 ${colors.dot} rounded-full`} />
					{log.level.toUpperCase()}
				</span>
				{log.component && (
					<span className="text-xs text-gray-500 font-mono">
						[{log.component}]
					</span>
				)}
				<span className="text-sm text-gray-900 flex-1 font-mono break-all">
					{log.message}
				</span>
				<svg
					aria-hidden="true"
					className={`w-4 h-4 text-gray-400 transition-transform ${expanded ? 'rotate-180' : ''}`}
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
			</div>
			{expanded && log.metadata && Object.keys(log.metadata).length > 0 && (
				<div className="px-4 pb-3 ml-[180px]">
					<pre className="text-xs bg-gray-50 p-3 rounded-lg overflow-x-auto font-mono text-gray-600">
						{JSON.stringify(log.metadata, null, 2)}
					</pre>
				</div>
			)}
		</div>
	);
}

export function AgentLogViewer({ agentId }: AgentLogViewerProps) {
	const [filter, setFilter] = useState<AgentLogFilter>({
		limit: 100,
		offset: 0,
	});
	const [autoScroll, setAutoScroll] = useState(true);
	const [searchText, setSearchText] = useState('');
	const logsContainerRef = useRef<HTMLDivElement>(null);

	const { data, isLoading, isError, refetch } = useAgentLogs(agentId, filter);
	const logs = data?.logs ?? [];
	const totalCount = data?.total_count ?? 0;
	const hasMore = data?.has_more ?? false;

	// Auto-scroll to bottom when new logs arrive
	useEffect(() => {
		if (autoScroll && logsContainerRef.current) {
			logsContainerRef.current.scrollTop = logsContainerRef.current.scrollHeight;
		}
	}, [logs, autoScroll]);

	const handleLevelChange = useCallback((level: LogLevel | '') => {
		setFilter((prev) => ({
			...prev,
			level: level || undefined,
			offset: 0,
		}));
	}, []);

	const handleSearch = useCallback(() => {
		setFilter((prev) => ({
			...prev,
			search: searchText || undefined,
			offset: 0,
		}));
	}, [searchText]);

	const handleKeyDown = useCallback(
		(e: React.KeyboardEvent) => {
			if (e.key === 'Enter') {
				handleSearch();
			}
		},
		[handleSearch],
	);

	const handleClearSearch = useCallback(() => {
		setSearchText('');
		setFilter((prev) => ({
			...prev,
			search: undefined,
			offset: 0,
		}));
	}, []);

	const handleLoadMore = useCallback(() => {
		setFilter((prev) => ({
			...prev,
			offset: (prev.offset ?? 0) + (prev.limit ?? 100),
		}));
	}, []);

	const handleDownload = useCallback(() => {
		const content = logs
			.map(
				(log) =>
					`${log.timestamp} [${log.level.toUpperCase()}]${log.component ? ` [${log.component}]` : ''} ${log.message}`,
			)
			.join('\n');
		const blob = new Blob([content], { type: 'text/plain' });
		const url = URL.createObjectURL(blob);
		const a = document.createElement('a');
		a.href = url;
		a.download = `agent-logs-${agentId}-${new Date().toISOString()}.txt`;
		document.body.appendChild(a);
		a.click();
		document.body.removeChild(a);
		URL.revokeObjectURL(url);
	}, [logs, agentId]);

	return (
		<div className="bg-white rounded-lg border border-gray-200 overflow-hidden flex flex-col h-[600px]">
			{/* Toolbar */}
			<div className="px-4 py-3 border-b border-gray-200 flex items-center gap-4 flex-wrap">
				{/* Level Filter */}
				<div className="flex items-center gap-2">
					<label htmlFor="log-level" className="text-sm text-gray-600">
						Level:
					</label>
					<select
						id="log-level"
						value={filter.level ?? ''}
						onChange={(e) => handleLevelChange(e.target.value as LogLevel | '')}
						className="text-sm border border-gray-300 rounded-lg px-2 py-1 focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
					>
						<option value="">All Levels</option>
						<option value="debug">Debug</option>
						<option value="info">Info</option>
						<option value="warn">Warning</option>
						<option value="error">Error</option>
					</select>
				</div>

				{/* Search */}
				<div className="flex items-center gap-2 flex-1 min-w-[200px]">
					<div className="relative flex-1">
						<input
							type="text"
							placeholder="Search logs..."
							value={searchText}
							onChange={(e) => setSearchText(e.target.value)}
							onKeyDown={handleKeyDown}
							className="w-full text-sm border border-gray-300 rounded-lg pl-8 pr-8 py-1 focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						/>
						<svg
							aria-hidden="true"
							className="w-4 h-4 text-gray-400 absolute left-2.5 top-1/2 -translate-y-1/2"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
						>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
							/>
						</svg>
						{searchText && (
							<button
								type="button"
								onClick={handleClearSearch}
								className="absolute right-2.5 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600"
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
						)}
					</div>
					<button
						type="button"
						onClick={handleSearch}
						className="px-3 py-1 text-sm bg-indigo-600 text-white rounded-lg hover:bg-indigo-700"
					>
						Search
					</button>
				</div>

				{/* Actions */}
				<div className="flex items-center gap-2">
					<button
						type="button"
						onClick={() => refetch()}
						className="p-1.5 text-gray-500 hover:text-gray-700 hover:bg-gray-100 rounded-lg"
						title="Refresh logs"
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
								d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
							/>
						</svg>
					</button>
					<button
						type="button"
						onClick={handleDownload}
						disabled={logs.length === 0}
						className="p-1.5 text-gray-500 hover:text-gray-700 hover:bg-gray-100 rounded-lg disabled:opacity-50"
						title="Download logs"
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
								d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"
							/>
						</svg>
					</button>
					<label className="flex items-center gap-1 text-sm text-gray-600 cursor-pointer">
						<input
							type="checkbox"
							checked={autoScroll}
							onChange={(e) => setAutoScroll(e.target.checked)}
							className="rounded border-gray-300 text-indigo-600 focus:ring-indigo-500"
						/>
						Auto-scroll
					</label>
				</div>
			</div>

			{/* Stats */}
			<div className="px-4 py-2 bg-gray-50 border-b border-gray-200 text-xs text-gray-500">
				{totalCount} log entries {filter.search && `matching "${filter.search}"`}
				{filter.level && ` at ${filter.level} level`}
			</div>

			{/* Log Content */}
			<div
				ref={logsContainerRef}
				className="flex-1 overflow-auto font-mono text-sm"
			>
				{isLoading && logs.length === 0 ? (
					<div className="flex items-center justify-center h-full text-gray-500">
						<div className="animate-spin rounded-full h-8 w-8 border-b-2 border-indigo-600" />
					</div>
				) : isError ? (
					<div className="flex items-center justify-center h-full text-red-500">
						Failed to load logs
					</div>
				) : logs.length === 0 ? (
					<div className="flex flex-col items-center justify-center h-full text-gray-500">
						<svg
							aria-hidden="true"
							className="w-12 h-12 mb-4 text-gray-300"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
						>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
							/>
						</svg>
						<p>No logs available</p>
						<p className="text-xs mt-1">
							Logs will appear here when the agent sends them
						</p>
					</div>
				) : (
					<div>
						{logs.map((log) => (
							<LogRow key={log.id} log={log} />
						))}
						{hasMore && (
							<div className="p-4 text-center">
								<button
									type="button"
									onClick={handleLoadMore}
									className="text-sm text-indigo-600 hover:text-indigo-700"
								>
									Load more logs...
								</button>
							</div>
						)}
					</div>
				)}
			</div>
		</div>
	);
}
