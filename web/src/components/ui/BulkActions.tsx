import { useState } from 'react';

export interface BulkAction {
	id: string;
	label: string;
	icon?: React.ReactNode;
	variant?: 'default' | 'danger';
	disabled?: boolean;
	requiresConfirmation?: boolean;
	confirmationMessage?: string;
}

interface BulkActionsProps {
	actions: BulkAction[];
	onAction: (actionId: string) => void;
	disabled?: boolean;
	label?: string;
}

export function BulkActions({
	actions,
	onAction,
	disabled = false,
	label = 'Actions',
}: BulkActionsProps) {
	const [isOpen, setIsOpen] = useState(false);

	const handleAction = (action: BulkAction) => {
		setIsOpen(false);
		onAction(action.id);
	};

	return (
		<div className="relative">
			<button
				type="button"
				onClick={() => setIsOpen(!isOpen)}
				disabled={disabled}
				className="inline-flex items-center gap-2 px-4 py-2 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-lg text-sm font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-indigo-500 disabled:opacity-50 disabled:cursor-not-allowed"
			>
				{label}
				<svg
					aria-hidden="true"
					className={`w-4 h-4 transition-transform ${isOpen ? 'rotate-180' : ''}`}
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
			</button>

			{isOpen && (
				<>
					<div
						className="fixed inset-0 z-10"
						onClick={() => setIsOpen(false)}
						onKeyDown={(e) => e.key === 'Escape' && setIsOpen(false)}
					/>
					<div className="absolute right-0 z-20 mt-2 w-56 bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 py-1">
						{actions.map((action) => (
							<button
								key={action.id}
								type="button"
								onClick={() => handleAction(action)}
								disabled={action.disabled}
								className={`w-full text-left px-4 py-2 text-sm flex items-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed ${
									action.variant === 'danger'
										? 'text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/30'
										: 'text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700'
								}`}
							>
								{action.icon}
								{action.label}
							</button>
						))}
					</div>
				</>
			)}
		</div>
	);
}

interface BulkActionButtonProps {
	label: string;
	icon?: React.ReactNode;
	onClick: () => void;
	variant?: 'default' | 'primary' | 'danger';
	disabled?: boolean;
}

export function BulkActionButton({
	label,
	icon,
	onClick,
	variant = 'default',
	disabled = false,
}: BulkActionButtonProps) {
	const variantClasses = {
		default:
			'bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700',
		primary:
			'bg-indigo-600 border border-indigo-600 text-white hover:bg-indigo-700',
		danger: 'bg-red-600 border border-red-600 text-white hover:bg-red-700',
	};

	return (
		<button
			type="button"
			onClick={onClick}
			disabled={disabled}
			className={`inline-flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 disabled:opacity-50 disabled:cursor-not-allowed ${variantClasses[variant]}`}
		>
			{icon}
			{label}
		</button>
	);
}
