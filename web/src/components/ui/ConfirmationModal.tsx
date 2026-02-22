interface ConfirmationModalProps {
	isOpen: boolean;
	onClose: () => void;
	onConfirm: () => void;
	title: string;
	message: string;
	confirmLabel?: string;
	cancelLabel?: string;
	variant?: 'default' | 'danger';
	isLoading?: boolean;
	itemCount?: number;
}

export function ConfirmationModal({
	isOpen,
	onClose,
	onConfirm,
	title,
	message,
	confirmLabel = 'Confirm',
	cancelLabel = 'Cancel',
	variant = 'default',
	isLoading = false,
	itemCount,
}: ConfirmationModalProps) {
	if (!isOpen) return null;

	const iconBgColor =
		variant === 'danger' ? 'bg-red-100 dark:bg-red-900/30' : 'bg-indigo-100';
	const iconColor =
		variant === 'danger' ? 'text-red-600 dark:text-red-400' : 'text-indigo-600';
	const confirmButtonClasses =
		variant === 'danger'
			? 'bg-red-600 hover:bg-red-700 focus:ring-red-500'
			: 'bg-indigo-600 hover:bg-indigo-700 focus:ring-indigo-500';

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-md w-full mx-4">
				<div className="flex items-start gap-4">
					<div className={`p-3 rounded-full ${iconBgColor}`}>
						{variant === 'danger' ? (
							<svg
								aria-hidden="true"
								className={`w-6 h-6 ${iconColor}`}
								fill="none"
								stroke="currentColor"
								viewBox="0 0 24 24"
							>
								<path
									strokeLinecap="round"
									strokeLinejoin="round"
									strokeWidth={2}
									d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
								/>
							</svg>
						) : (
							<svg
								aria-hidden="true"
								className={`w-6 h-6 ${iconColor}`}
								fill="none"
								stroke="currentColor"
								viewBox="0 0 24 24"
							>
								<path
									strokeLinecap="round"
									strokeLinejoin="round"
									strokeWidth={2}
									d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
								/>
							</svg>
						)}
					</div>
					<div className="flex-1">
						<h3 className="text-lg font-semibold text-gray-900 dark:text-white">
							{title}
						</h3>
						<p className="mt-2 text-sm text-gray-600 dark:text-gray-400">
							{message}
						</p>
						{itemCount !== undefined && itemCount > 0 && (
							<p className="mt-2 text-sm font-medium text-gray-900 dark:text-white">
								This action will affect {itemCount}{' '}
								{itemCount === 1 ? 'item' : 'items'}.
							</p>
						)}
					</div>
				</div>

				<div className="mt-6 flex justify-end gap-3">
					<button
						type="button"
						onClick={onClose}
						disabled={isLoading}
						className="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors disabled:opacity-50"
					>
						{cancelLabel}
					</button>
					<button
						type="button"
						onClick={onConfirm}
						disabled={isLoading}
						className={`px-4 py-2 text-sm font-medium text-white rounded-lg transition-colors disabled:opacity-50 focus:outline-none focus:ring-2 focus:ring-offset-2 ${confirmButtonClasses}`}
					>
						{isLoading ? (
							<span className="flex items-center gap-2">
								<svg
									aria-hidden="true"
									className="w-4 h-4 animate-spin"
									fill="none"
									viewBox="0 0 24 24"
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
								Processing...
							</span>
						) : (
							confirmLabel
						)}
					</button>
				</div>
			</div>
		</div>
	);
}
