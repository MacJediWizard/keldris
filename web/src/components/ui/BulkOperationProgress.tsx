export interface BulkOperationResult {
	id: string;
	success: boolean;
	error?: string;
}

interface BulkOperationProgressProps {
	isOpen: boolean;
	onClose: () => void;
	title: string;
	total: number;
	completed: number;
	results: BulkOperationResult[];
	isComplete: boolean;
}

export function BulkOperationProgress({
	isOpen,
	onClose,
	title,
	total,
	completed,
	results,
	isComplete,
}: BulkOperationProgressProps) {
	if (!isOpen) return null;

	const progress = total > 0 ? (completed / total) * 100 : 0;
	const successCount = results.filter((r) => r.success).length;
	const failedCount = results.filter((r) => !r.success).length;

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-md w-full mx-4">
				<h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
					{title}
				</h3>

				<div className="mb-4">
					<div className="flex justify-between text-sm text-gray-600 dark:text-gray-400 mb-2">
						<span>Progress</span>
						<span>
							{completed} of {total}
						</span>
					</div>
					<div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2.5">
						<div
							className="bg-indigo-600 h-2.5 rounded-full transition-all duration-300"
							style={{ width: `${progress}%` }}
						/>
					</div>
				</div>

				{isComplete && (
					<div className="mb-4 space-y-2">
						<div className="flex items-center gap-2 text-sm">
							<svg
								aria-hidden="true"
								className="w-5 h-5 text-green-500"
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
							<span className="text-gray-700 dark:text-gray-300">
								{successCount} succeeded
							</span>
						</div>
						{failedCount > 0 && (
							<div className="flex items-center gap-2 text-sm">
								<svg
									aria-hidden="true"
									className="w-5 h-5 text-red-500"
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
								<span className="text-gray-700 dark:text-gray-300">
									{failedCount} failed
								</span>
							</div>
						)}
					</div>
				)}

				{isComplete && failedCount > 0 && (
					<div className="mb-4 max-h-40 overflow-y-auto">
						<p className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
							Failed items:
						</p>
						<ul className="space-y-1">
							{results
								.filter((r) => !r.success)
								.map((result) => (
									<li
										key={result.id}
										className="text-sm text-red-600 dark:text-red-400 bg-red-50 dark:bg-red-900/30 px-3 py-1.5 rounded"
									>
										{result.error || 'Unknown error'}
									</li>
								))}
						</ul>
					</div>
				)}

				{!isComplete && (
					<p className="text-sm text-gray-600 dark:text-gray-400 text-center">
						Please wait while the operation completes...
					</p>
				)}

				{isComplete && (
					<div className="flex justify-end">
						<button
							type="button"
							onClick={onClose}
							className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors text-sm font-medium"
						>
							Done
						</button>
					</div>
				)}
			</div>
		</div>
	);
}

interface UseBulkOperationResult {
	isRunning: boolean;
	isComplete: boolean;
	total: number;
	completed: number;
	results: BulkOperationResult[];
	start: <T>(
		ids: string[],
		operation: (id: string) => Promise<T>,
	) => Promise<void>;
	reset: () => void;
}

import { useCallback, useState } from 'react';

export function useBulkOperation(): UseBulkOperationResult {
	const [isRunning, setIsRunning] = useState(false);
	const [isComplete, setIsComplete] = useState(false);
	const [total, setTotal] = useState(0);
	const [completed, setCompleted] = useState(0);
	const [results, setResults] = useState<BulkOperationResult[]>([]);

	const start = useCallback(
		async <T,>(
			ids: string[],
			operation: (id: string) => Promise<T>,
		): Promise<void> => {
			setIsRunning(true);
			setIsComplete(false);
			setTotal(ids.length);
			setCompleted(0);
			setResults([]);

			const operationResults: BulkOperationResult[] = [];

			for (const id of ids) {
				try {
					await operation(id);
					operationResults.push({ id, success: true });
				} catch (error) {
					operationResults.push({
						id,
						success: false,
						error: error instanceof Error ? error.message : 'Unknown error',
					});
				}
				setCompleted((prev) => prev + 1);
				setResults([...operationResults]);
			}

			setIsRunning(false);
			setIsComplete(true);
		},
		[],
	);

	const reset = useCallback(() => {
		setIsRunning(false);
		setIsComplete(false);
		setTotal(0);
		setCompleted(0);
		setResults([]);
	}, []);

	return {
		isRunning,
		isComplete,
		total,
		completed,
		results,
		start,
		reset,
	};
}
