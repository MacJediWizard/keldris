import { useState } from 'react';
import { useChangelog } from '../hooks/useChangelog';
import type { ChangelogEntry } from '../lib/types';

function VersionBadge({ isCurrent }: { version: string; isCurrent: boolean }) {
	if (isCurrent) {
		return (
			<span className="ml-2 px-2 py-0.5 text-xs font-medium bg-indigo-100 text-indigo-800 rounded-full">
				Current
			</span>
		);
	}
	return null;
}

function ChangeSection({
	title,
	items,
	color,
}: {
	title: string;
	items: string[] | undefined;
	color: string;
}) {
	if (!items || items.length === 0) return null;

	const colorClasses: Record<string, string> = {
		green: 'bg-green-50 border-green-200 text-green-800',
		blue: 'bg-blue-50 border-blue-200 text-blue-800',
		yellow: 'bg-yellow-50 border-yellow-200 text-yellow-800',
		red: 'bg-red-50 border-red-200 text-red-800',
		orange: 'bg-orange-50 border-orange-200 text-orange-800',
		purple: 'bg-purple-50 border-purple-200 text-purple-800',
	};

	const badgeClasses: Record<string, string> = {
		green: 'bg-green-100 text-green-800',
		blue: 'bg-blue-100 text-blue-800',
		yellow: 'bg-yellow-100 text-yellow-800',
		red: 'bg-red-100 text-red-800',
		orange: 'bg-orange-100 text-orange-800',
		purple: 'bg-purple-100 text-purple-800',
	};

	return (
		<div className={`rounded-lg border p-4 ${colorClasses[color]}`}>
			<h4 className="font-medium mb-2 flex items-center gap-2">
				<span className={`px-2 py-0.5 text-xs font-medium rounded ${badgeClasses[color]}`}>
					{title}
				</span>
				<span className="text-xs text-gray-500">({items.length})</span>
			</h4>
			<ul className="space-y-1">
				{items.map((item, idx) => (
					<li key={idx} className="text-sm flex items-start gap-2">
						<span className="mt-1.5 w-1.5 h-1.5 rounded-full bg-current flex-shrink-0" />
						<span>{item}</span>
					</li>
				))}
			</ul>
		</div>
	);
}

function VersionCard({
	entry,
	isExpanded,
	onToggle,
	isCurrent,
}: {
	entry: ChangelogEntry;
	isExpanded: boolean;
	onToggle: () => void;
	isCurrent: boolean;
}) {
	const hasChanges =
		(entry.added?.length ?? 0) +
			(entry.changed?.length ?? 0) +
			(entry.deprecated?.length ?? 0) +
			(entry.removed?.length ?? 0) +
			(entry.fixed?.length ?? 0) +
			(entry.security?.length ?? 0) >
		0;

	return (
		<div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
			<button
				type="button"
				onClick={onToggle}
				className="w-full px-6 py-4 flex items-center justify-between hover:bg-gray-50 transition-colors"
			>
				<div className="flex items-center gap-4">
					<div className="flex items-center gap-2">
						<span className="text-lg font-semibold text-gray-900">
							{entry.is_unreleased ? 'Unreleased' : `v${entry.version}`}
						</span>
						<VersionBadge version={entry.version} isCurrent={isCurrent} />
					</div>
					{entry.date && (
						<span className="text-sm text-gray-500">{entry.date}</span>
					)}
				</div>
				<div className="flex items-center gap-4">
					{!isExpanded && hasChanges && (
						<div className="flex gap-2">
							{entry.added && entry.added.length > 0 && (
								<span className="px-2 py-0.5 text-xs bg-green-100 text-green-800 rounded">
									+{entry.added.length} added
								</span>
							)}
							{entry.fixed && entry.fixed.length > 0 && (
								<span className="px-2 py-0.5 text-xs bg-blue-100 text-blue-800 rounded">
									{entry.fixed.length} fixed
								</span>
							)}
							{entry.changed && entry.changed.length > 0 && (
								<span className="px-2 py-0.5 text-xs bg-yellow-100 text-yellow-800 rounded">
									{entry.changed.length} changed
								</span>
							)}
						</div>
					)}
					<svg
						className={`w-5 h-5 text-gray-400 transition-transform ${isExpanded ? 'rotate-180' : ''}`}
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
				</div>
			</button>

			{isExpanded && (
				<div className="px-6 pb-6 border-t border-gray-100 pt-4">
					{!hasChanges ? (
						<p className="text-gray-500 text-sm italic">No changes documented yet.</p>
					) : (
						<div className="grid gap-4 md:grid-cols-2">
							<ChangeSection title="Added" items={entry.added} color="green" />
							<ChangeSection title="Changed" items={entry.changed} color="yellow" />
							<ChangeSection title="Fixed" items={entry.fixed} color="blue" />
							<ChangeSection title="Deprecated" items={entry.deprecated} color="orange" />
							<ChangeSection title="Removed" items={entry.removed} color="red" />
							<ChangeSection title="Security" items={entry.security} color="purple" />
						</div>
					)}
				</div>
			)}
		</div>
	);
}

export function Changelog() {
	const { data, isLoading, error } = useChangelog();
	const [expandedVersions, setExpandedVersions] = useState<Set<string>>(new Set());

	const toggleVersion = (version: string) => {
		setExpandedVersions((prev) => {
			const next = new Set(prev);
			if (next.has(version)) {
				next.delete(version);
			} else {
				next.add(version);
			}
			return next;
		});
	};

	const expandAll = () => {
		if (data?.entries) {
			setExpandedVersions(new Set(data.entries.map((e) => e.version)));
		}
	};

	const collapseAll = () => {
		setExpandedVersions(new Set());
	};

	if (isLoading) {
		return (
			<div className="space-y-6">
				<div className="flex items-center justify-between">
					<div>
						<div className="h-8 w-48 bg-gray-200 rounded animate-pulse" />
						<div className="h-5 w-64 bg-gray-200 rounded animate-pulse mt-2" />
					</div>
				</div>
				<div className="space-y-4">
					{[1, 2, 3].map((i) => (
						<div key={i} className="h-20 bg-gray-100 rounded-lg animate-pulse" />
					))}
				</div>
			</div>
		);
	}

	if (error) {
		return (
			<div className="space-y-6">
				<div>
					<h1 className="text-2xl font-bold text-gray-900">Changelog</h1>
					<p className="text-gray-600 mt-1">Version history and release notes</p>
				</div>
				<div className="bg-red-50 border border-red-200 rounded-lg p-4 text-red-800">
					<p>Failed to load changelog. Please try again later.</p>
				</div>
			</div>
		);
	}

	return (
		<div className="space-y-6">
			<div className="flex items-center justify-between">
				<div>
					<h1 className="text-2xl font-bold text-gray-900">Changelog</h1>
					<p className="text-gray-600 mt-1">
						Version history and release notes
						{data?.current_version && (
							<span className="ml-2 text-indigo-600 font-medium">
								(Current: v{data.current_version})
							</span>
						)}
					</p>
				</div>
				<div className="flex gap-2">
					<button
						type="button"
						onClick={expandAll}
						className="px-3 py-1.5 text-sm text-gray-600 hover:text-gray-900 border border-gray-300 rounded-md hover:bg-gray-50"
					>
						Expand All
					</button>
					<button
						type="button"
						onClick={collapseAll}
						className="px-3 py-1.5 text-sm text-gray-600 hover:text-gray-900 border border-gray-300 rounded-md hover:bg-gray-50"
					>
						Collapse All
					</button>
				</div>
			</div>

			<div className="space-y-4">
				{data?.entries.map((entry) => (
					<VersionCard
						key={entry.version}
						entry={entry}
						isExpanded={expandedVersions.has(entry.version)}
						onToggle={() => toggleVersion(entry.version)}
						isCurrent={entry.version === data.current_version}
					/>
				))}
			</div>

			{(!data?.entries || data.entries.length === 0) && (
				<div className="bg-white rounded-lg border border-gray-200 p-8 text-center">
					<svg
						className="w-12 h-12 mx-auto text-gray-300 mb-4"
						fill="none"
						stroke="currentColor"
						viewBox="0 0 24 24"
						aria-hidden="true"
					>
						<path
							strokeLinecap="round"
							strokeLinejoin="round"
							strokeWidth={2}
							d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
						/>
					</svg>
					<p className="text-gray-500">No changelog entries found.</p>
				</div>
			)}
		</div>
	);
}
