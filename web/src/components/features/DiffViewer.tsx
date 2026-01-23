import { useState } from 'react';
import type { FileDiffResponse } from '../../lib/types';
import { formatBytes } from '../../lib/utils';

interface DiffViewerProps {
	diff: FileDiffResponse;
}

type ViewMode = 'unified' | 'split';

interface DiffLine {
	type: 'added' | 'removed' | 'context' | 'header';
	content: string;
	oldLineNum?: number;
	newLineNum?: number;
}

function parseUnifiedDiff(unifiedDiff: string): DiffLine[] {
	if (!unifiedDiff) return [];

	const lines: DiffLine[] = [];
	const diffLines = unifiedDiff.split('\n');
	let oldLineNum = 0;
	let newLineNum = 0;

	for (const line of diffLines) {
		if (line.startsWith('@@')) {
			// Parse hunk header: @@ -start,count +start,count @@
			const match = line.match(/@@ -(\d+),?\d* \+(\d+),?\d* @@/);
			if (match) {
				oldLineNum = parseInt(match[1], 10) - 1;
				newLineNum = parseInt(match[2], 10) - 1;
			}
			lines.push({ type: 'header', content: line });
		} else if (line.startsWith('---') || line.startsWith('+++')) {
			lines.push({ type: 'header', content: line });
		} else if (line.startsWith('+')) {
			newLineNum++;
			lines.push({
				type: 'added',
				content: line.slice(1),
				newLineNum,
			});
		} else if (line.startsWith('-')) {
			oldLineNum++;
			lines.push({
				type: 'removed',
				content: line.slice(1),
				oldLineNum,
			});
		} else if (line.startsWith(' ') || line === '') {
			oldLineNum++;
			newLineNum++;
			lines.push({
				type: 'context',
				content: line.slice(1) || '',
				oldLineNum,
				newLineNum,
			});
		}
	}

	return lines;
}

interface SplitLine {
	left?: { lineNum?: number; content: string; type: 'removed' | 'context' };
	right?: { lineNum?: number; content: string; type: 'added' | 'context' };
}

function convertToSplitView(diffLines: DiffLine[]): SplitLine[] {
	const result: SplitLine[] = [];
	let i = 0;

	while (i < diffLines.length) {
		const line = diffLines[i];

		if (line.type === 'header') {
			result.push({
				left: { content: line.content, type: 'context' },
				right: { content: line.content, type: 'context' },
			});
			i++;
		} else if (line.type === 'context') {
			result.push({
				left: {
					lineNum: line.oldLineNum,
					content: line.content,
					type: 'context',
				},
				right: {
					lineNum: line.newLineNum,
					content: line.content,
					type: 'context',
				},
			});
			i++;
		} else if (line.type === 'removed') {
			// Check if next line is an addition (paired change)
			const nextLine = diffLines[i + 1];
			if (nextLine && nextLine.type === 'added') {
				result.push({
					left: {
						lineNum: line.oldLineNum,
						content: line.content,
						type: 'removed',
					},
					right: {
						lineNum: nextLine.newLineNum,
						content: nextLine.content,
						type: 'added',
					},
				});
				i += 2;
			} else {
				result.push({
					left: {
						lineNum: line.oldLineNum,
						content: line.content,
						type: 'removed',
					},
				});
				i++;
			}
		} else if (line.type === 'added') {
			result.push({
				right: {
					lineNum: line.newLineNum,
					content: line.content,
					type: 'added',
				},
			});
			i++;
		} else {
			i++;
		}
	}

	return result;
}

function getLineClass(type: 'added' | 'removed' | 'context' | 'header'): string {
	switch (type) {
		case 'added':
			return 'bg-green-50 text-green-900';
		case 'removed':
			return 'bg-red-50 text-red-900';
		case 'header':
			return 'bg-blue-50 text-blue-700 font-semibold';
		default:
			return '';
	}
}

function getLineNumClass(
	type: 'added' | 'removed' | 'context' | 'header',
): string {
	switch (type) {
		case 'added':
			return 'bg-green-100 text-green-600';
		case 'removed':
			return 'bg-red-100 text-red-600';
		default:
			return 'bg-gray-50 text-gray-400';
	}
}

export function DiffViewer({ diff }: DiffViewerProps) {
	const [viewMode, setViewMode] = useState<ViewMode>('unified');

	if (diff.is_binary) {
		return (
			<div className="border border-gray-200 rounded-lg overflow-hidden">
				<div className="bg-gray-50 px-4 py-3 border-b border-gray-200">
					<div className="flex items-center justify-between">
						<h3 className="font-mono text-sm text-gray-700 truncate">
							{diff.path}
						</h3>
						<span className="text-xs bg-yellow-100 text-yellow-800 px-2 py-1 rounded">
							Binary file
						</span>
					</div>
				</div>
				<div className="p-6 text-center text-gray-500">
					<svg
						aria-hidden="true"
						className="w-12 h-12 mx-auto mb-3 text-gray-300"
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
					<p className="font-medium text-gray-900 mb-2">Binary file</p>
					<p className="text-sm">
						Content comparison is not available for binary files
					</p>
					<div className="mt-4 flex justify-center gap-8 text-sm">
						{diff.old_size !== undefined && (
							<div>
								<span className="text-gray-500">Old size: </span>
								<span className="font-medium">{formatBytes(diff.old_size)}</span>
							</div>
						)}
						{diff.new_size !== undefined && (
							<div>
								<span className="text-gray-500">New size: </span>
								<span className="font-medium">{formatBytes(diff.new_size)}</span>
							</div>
						)}
					</div>
					{diff.old_hash && diff.new_hash && (
						<div className="mt-4 text-xs text-gray-400">
							<p>
								{diff.old_hash === diff.new_hash
									? 'Files are identical'
									: 'Files have different content'}
							</p>
						</div>
					)}
				</div>
			</div>
		);
	}

	const diffLines = parseUnifiedDiff(diff.unified_diff || '');
	const splitLines = convertToSplitView(diffLines);
	const hasContent = diffLines.length > 0;

	return (
		<div className="border border-gray-200 rounded-lg overflow-hidden">
			<div className="bg-gray-50 px-4 py-3 border-b border-gray-200">
				<div className="flex items-center justify-between">
					<div className="flex items-center gap-3">
						<h3 className="font-mono text-sm text-gray-700 truncate">
							{diff.path}
						</h3>
						<span
							className={`text-xs px-2 py-0.5 rounded ${
								diff.change_type === 'added'
									? 'bg-green-100 text-green-800'
									: diff.change_type === 'removed'
										? 'bg-red-100 text-red-800'
										: 'bg-blue-100 text-blue-800'
							}`}
						>
							{diff.change_type}
						</span>
					</div>
					{hasContent && (
						<div className="flex items-center gap-2">
							<span className="text-xs text-gray-500">View:</span>
							<div className="flex border border-gray-300 rounded overflow-hidden">
								<button
									type="button"
									onClick={() => setViewMode('unified')}
									className={`px-3 py-1 text-xs ${
										viewMode === 'unified'
											? 'bg-indigo-600 text-white'
											: 'bg-white text-gray-700 hover:bg-gray-50'
									}`}
								>
									Unified
								</button>
								<button
									type="button"
									onClick={() => setViewMode('split')}
									className={`px-3 py-1 text-xs border-l border-gray-300 ${
										viewMode === 'split'
											? 'bg-indigo-600 text-white'
											: 'bg-white text-gray-700 hover:bg-gray-50'
									}`}
								>
									Split
								</button>
							</div>
						</div>
					)}
				</div>
			</div>

			{!hasContent ? (
				<div className="p-6 text-center text-gray-500">
					<svg
						aria-hidden="true"
						className="w-12 h-12 mx-auto mb-3 text-gray-300"
						fill="none"
						stroke="currentColor"
						viewBox="0 0 24 24"
					>
						<path
							strokeLinecap="round"
							strokeLinejoin="round"
							strokeWidth={2}
							d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
						/>
					</svg>
					<p className="font-medium text-gray-900 mb-2">
						{diff.change_type === 'added'
							? 'New file'
							: diff.change_type === 'removed'
								? 'Deleted file'
								: 'No changes detected'}
					</p>
					<p className="text-sm">
						{diff.change_type === 'added'
							? 'This file was added in the newer snapshot'
							: diff.change_type === 'removed'
								? 'This file was removed in the newer snapshot'
								: 'The file content is identical between snapshots'}
					</p>
					<div className="mt-4 flex justify-center gap-8 text-sm">
						{diff.old_size !== undefined && diff.old_size > 0 && (
							<div>
								<span className="text-gray-500">Old size: </span>
								<span className="font-medium">{formatBytes(diff.old_size)}</span>
							</div>
						)}
						{diff.new_size !== undefined && diff.new_size > 0 && (
							<div>
								<span className="text-gray-500">New size: </span>
								<span className="font-medium">{formatBytes(diff.new_size)}</span>
							</div>
						)}
					</div>
				</div>
			) : viewMode === 'unified' ? (
				<div className="overflow-x-auto">
					<table className="w-full text-sm font-mono">
						<tbody>
							{diffLines.map((line, index) => (
								<tr key={index} className={getLineClass(line.type)}>
									<td
										className={`px-2 py-0 text-right select-none w-12 ${getLineNumClass(line.type)}`}
									>
										{line.oldLineNum ?? ''}
									</td>
									<td
										className={`px-2 py-0 text-right select-none w-12 border-r border-gray-200 ${getLineNumClass(line.type)}`}
									>
										{line.newLineNum ?? ''}
									</td>
									<td className="px-2 py-0 whitespace-pre">
										{line.type === 'added' && (
											<span className="text-green-600 mr-1">+</span>
										)}
										{line.type === 'removed' && (
											<span className="text-red-600 mr-1">-</span>
										)}
										{line.type === 'context' && (
											<span className="text-gray-400 mr-1"> </span>
										)}
										{line.content}
									</td>
								</tr>
							))}
						</tbody>
					</table>
				</div>
			) : (
				<div className="overflow-x-auto">
					<table className="w-full text-sm font-mono">
						<thead className="bg-gray-100 border-b border-gray-200">
							<tr>
								<th className="px-2 py-1 text-left text-xs font-medium text-gray-500 w-1/2 border-r border-gray-200">
									{diff.snapshot_id_1?.slice(0, 8) || 'Old'}
								</th>
								<th className="px-2 py-1 text-left text-xs font-medium text-gray-500 w-1/2">
									{diff.snapshot_id_2?.slice(0, 8) || 'New'}
								</th>
							</tr>
						</thead>
						<tbody>
							{splitLines.map((row, index) => (
								<tr key={index}>
									<td
										className={`px-0 py-0 w-1/2 border-r border-gray-200 ${
											row.left?.type === 'removed'
												? 'bg-red-50'
												: row.left?.type === 'context'
													? ''
													: 'bg-gray-50'
										}`}
									>
										{row.left && (
											<div className="flex">
												<span
													className={`px-2 py-0 text-right select-none w-12 ${getLineNumClass(row.left.type)}`}
												>
													{row.left.lineNum ?? ''}
												</span>
												<span
													className={`flex-1 px-2 py-0 whitespace-pre ${row.left.type === 'removed' ? 'text-red-900' : ''}`}
												>
													{row.left.content}
												</span>
											</div>
										)}
									</td>
									<td
										className={`px-0 py-0 w-1/2 ${
											row.right?.type === 'added'
												? 'bg-green-50'
												: row.right?.type === 'context'
													? ''
													: 'bg-gray-50'
										}`}
									>
										{row.right && (
											<div className="flex">
												<span
													className={`px-2 py-0 text-right select-none w-12 ${getLineNumClass(row.right.type)}`}
												>
													{row.right.lineNum ?? ''}
												</span>
												<span
													className={`flex-1 px-2 py-0 whitespace-pre ${row.right.type === 'added' ? 'text-green-900' : ''}`}
												>
													{row.right.content}
												</span>
											</div>
										)}
									</td>
								</tr>
							))}
						</tbody>
					</table>
				</div>
			)}
		</div>
	);
}
