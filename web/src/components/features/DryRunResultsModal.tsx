import { useState } from 'react';
import type { DryRunResponse } from '../../lib/types';

interface DryRunResultsModalProps {
	isOpen: boolean;
	onClose: () => void;
	results: DryRunResponse | null;
	isLoading: boolean;
	error: Error | null;
}

function formatBytes(bytes: number): string {
	if (bytes === 0) return '0 B';
	const k = 1024;
	const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
	const i = Math.floor(Math.log(bytes) / Math.log(k));
	return `${Number.parseFloat((bytes / k ** i).toFixed(2))} ${sizes[i]}`;
}

export function DryRunResultsModal({
	isOpen,
	onClose,
	results,
	isLoading,
	error,
}: DryRunResultsModalProps) {
	const [activeTab, setActiveTab] = useState<'files' | 'excluded'>('files');

	if (!isOpen) return null;

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-3xl w-full mx-4 max-h-[90vh] overflow-hidden flex flex-col">
				<div className="flex items-center justify-between mb-4">
					<h3 className="text-lg font-semibold text-gray-900 dark:text-white">
						Dry Run Results
					</h3>
					<button
						type="button"
						onClick={onClose}
						className="text-gray-400 hover:text-gray-500 dark:hover:text-gray-300"
					>
						<svg
							className="w-6 h-6"
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
				</div>

				{isLoading && (
					<div className="flex-1 flex items-center justify-center py-12">
						<div className="text-center">
							<svg
								className="animate-spin h-8 w-8 text-indigo-600 mx-auto mb-4"
								fill="none"
								viewBox="0 0 24 24"
								aria-hidden="true"
							>
								<circle
									className="opacity-25"
									cx="12"
									cy="12"
									r="10"
									stroke="currentColor"
									strokeWidth="4"
								/>
								<path
									className="opacity-75"
									fill="currentColor"
									d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
								/>
							</svg>
							<p className="text-gray-500 dark:text-gray-400">
								Running dry run simulation...
							</p>
						</div>
					</div>
				)}

				{error && (
					<div className="flex-1 flex items-center justify-center py-12">
						<div className="text-center text-red-500">
							<svg
								className="w-12 h-12 mx-auto mb-4"
								fill="none"
								stroke="currentColor"
								viewBox="0 0 24 24"
								aria-hidden="true"
							>
								<path
									strokeLinecap="round"
									strokeLinejoin="round"
									strokeWidth={2}
									d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
								/>
							</svg>
							<p className="font-medium">Dry run failed</p>
							<p className="text-sm mt-1">{error.message}</p>
						</div>
					</div>
				)}

				{results && !isLoading && !error && (
					<>
						{/* Summary Stats */}
						<div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
							<div className="bg-gray-50 dark:bg-gray-700 rounded-lg p-3 text-center">
								<div className="text-2xl font-bold text-gray-900 dark:text-white">
									{results.total_files}
								</div>
								<div className="text-xs text-gray-500 dark:text-gray-400">
									Total Files
								</div>
							</div>
							<div className="bg-gray-50 dark:bg-gray-700 rounded-lg p-3 text-center">
								<div className="text-2xl font-bold text-gray-900 dark:text-white">
									{formatBytes(results.total_size)}
								</div>
								<div className="text-xs text-gray-500 dark:text-gray-400">
									Estimated Size
								</div>
							</div>
							<div className="bg-green-50 dark:bg-green-900/20 rounded-lg p-3 text-center">
								<div className="text-2xl font-bold text-green-600 dark:text-green-400">
									{results.new_files}
								</div>
								<div className="text-xs text-gray-500 dark:text-gray-400">
									New Files
								</div>
							</div>
							<div className="bg-amber-50 dark:bg-amber-900/20 rounded-lg p-3 text-center">
								<div className="text-2xl font-bold text-amber-600 dark:text-amber-400">
									{results.changed_files}
								</div>
								<div className="text-xs text-gray-500 dark:text-gray-400">
									Changed Files
								</div>
							</div>
						</div>

						{/* Message */}
						{results.message && (
							<div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-3 mb-4">
								<p className="text-sm text-blue-700 dark:text-blue-300">
									{results.message}
								</p>
							</div>
						)}

						{/* Tabs */}
						<div className="flex border-b border-gray-200 dark:border-gray-700 mb-4">
							<button
								type="button"
								onClick={() => setActiveTab('files')}
								className={`px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors ${
									activeTab === 'files'
										? 'border-indigo-500 text-indigo-600 dark:text-indigo-400'
										: 'border-transparent text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300'
								}`}
							>
								Files to Backup ({results.files_to_backup.length})
							</button>
							<button
								type="button"
								onClick={() => setActiveTab('excluded')}
								className={`px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors ${
									activeTab === 'excluded'
										? 'border-indigo-500 text-indigo-600 dark:text-indigo-400'
										: 'border-transparent text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300'
								}`}
							>
								Excluded ({results.excluded_files.length})
							</button>
						</div>

						{/* Tab Content */}
						<div className="flex-1 overflow-y-auto min-h-0">
							{activeTab === 'files' && (
								<div className="space-y-1">
									{results.files_to_backup.length === 0 ? (
										<div className="text-center py-8 text-gray-500 dark:text-gray-400">
											<p>No files would be backed up.</p>
											<p className="text-sm mt-1">
												This could mean all files are unchanged or the backup
												paths are empty.
											</p>
										</div>
									) : (
										results.files_to_backup.map((file) => (
											<div
												key={file.path}
												className="flex items-center justify-between py-2 px-3 hover:bg-gray-50 dark:hover:bg-gray-700 rounded"
											>
												<div className="flex items-center gap-2 min-w-0 flex-1">
													{file.type === 'dir' ? (
														<svg
															className="w-4 h-4 text-amber-500 flex-shrink-0"
															fill="currentColor"
															viewBox="0 0 20 20"
															aria-hidden="true"
														>
															<path d="M2 6a2 2 0 012-2h5l2 2h5a2 2 0 012 2v6a2 2 0 01-2 2H4a2 2 0 01-2-2V6z" />
														</svg>
													) : (
														<svg
															className="w-4 h-4 text-gray-400 flex-shrink-0"
															fill="currentColor"
															viewBox="0 0 20 20"
															aria-hidden="true"
														>
															<path
																fillRule="evenodd"
																d="M4 4a2 2 0 012-2h4.586A2 2 0 0112 2.586L15.414 6A2 2 0 0116 7.414V16a2 2 0 01-2 2H6a2 2 0 01-2-2V4z"
																clipRule="evenodd"
															/>
														</svg>
													)}
													<span
														className="text-sm text-gray-700 dark:text-gray-300 truncate"
														title={file.path}
													>
														{file.path}
													</span>
												</div>
												<div className="flex items-center gap-3 flex-shrink-0">
													<span
														className={`text-xs px-2 py-0.5 rounded ${
															file.action === 'new'
																? 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400'
																: file.action === 'changed'
																	? 'bg-amber-100 text-amber-800 dark:bg-amber-900/30 dark:text-amber-400'
																	: 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-400'
														}`}
													>
														{file.action}
													</span>
													<span className="text-xs text-gray-500 dark:text-gray-400 w-16 text-right">
														{formatBytes(file.size)}
													</span>
												</div>
											</div>
										))
									)}
								</div>
							)}

							{activeTab === 'excluded' && (
								<div className="space-y-1">
									{results.excluded_files.length === 0 ? (
										<div className="text-center py-8 text-gray-500 dark:text-gray-400">
											<p>No exclusion patterns configured.</p>
										</div>
									) : (
										results.excluded_files.map((excluded) => (
											<div
												key={excluded.path}
												className="flex items-center justify-between py-2 px-3 hover:bg-gray-50 dark:hover:bg-gray-700 rounded"
											>
												<div className="flex items-center gap-2 min-w-0 flex-1">
													<svg
														className="w-4 h-4 text-red-400 flex-shrink-0"
														fill="none"
														stroke="currentColor"
														viewBox="0 0 24 24"
														aria-hidden="true"
													>
														<path
															strokeLinecap="round"
															strokeLinejoin="round"
															strokeWidth={2}
															d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728L5.636 5.636"
														/>
													</svg>
													<code className="text-sm text-gray-700 dark:text-gray-300 truncate bg-gray-100 dark:bg-gray-700 px-1.5 py-0.5 rounded">
														{excluded.path}
													</code>
												</div>
												<span className="text-xs text-gray-500 dark:text-gray-400 flex-shrink-0">
													{excluded.reason}
												</span>
											</div>
										))
									)}
								</div>
							)}
						</div>
					</>
				)}

				{/* Footer */}
				<div className="flex justify-end gap-3 mt-6 pt-4 border-t border-gray-200 dark:border-gray-700">
					<button
						type="button"
						onClick={onClose}
						className="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
					>
						Close
					</button>
				</div>
			</div>
		</div>
	);
}
