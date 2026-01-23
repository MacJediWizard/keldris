import type { ReactNode } from 'react';
import { useReadOnlyMode } from '../../hooks/useReadOnlyMode';

interface ReadOnlyBlockerProps {
	children: ReactNode;
	fallback?: ReactNode;
	showMessage?: boolean;
}

export function ReadOnlyBlocker({
	children,
	fallback,
	showMessage = true,
}: ReadOnlyBlockerProps) {
	const { isReadOnly, maintenanceTitle } = useReadOnlyMode();

	if (!isReadOnly) {
		return <>{children}</>;
	}

	if (fallback) {
		return <>{fallback}</>;
	}

	if (showMessage) {
		return (
			<div className="inline-flex items-center gap-2 px-3 py-2 bg-amber-50 border border-amber-200 rounded-md text-amber-800 text-sm">
				<svg
					className="w-4 h-4"
					fill="none"
					stroke="currentColor"
					viewBox="0 0 24 24"
					aria-hidden="true"
				>
					<path
						strokeLinecap="round"
						strokeLinejoin="round"
						strokeWidth={2}
						d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"
					/>
				</svg>
				<span>
					Read-only mode
					{maintenanceTitle ? `: ${maintenanceTitle}` : ''}
				</span>
			</div>
		);
	}

	return null;
}

interface ReadOnlyDisabledButtonProps {
	children: ReactNode;
	onClick?: () => void;
	className?: string;
	type?: 'button' | 'submit' | 'reset';
	disabled?: boolean;
}

export function ReadOnlyDisabledButton({
	children,
	onClick,
	className = '',
	type = 'button',
	disabled = false,
}: ReadOnlyDisabledButtonProps) {
	const { isReadOnly, maintenanceTitle } = useReadOnlyMode();

	const isDisabled = disabled || isReadOnly;

	return (
		<div className="relative inline-block">
			<button
				type={type}
				onClick={isDisabled ? undefined : onClick}
				disabled={isDisabled}
				className={`${className} ${isDisabled ? 'opacity-50 cursor-not-allowed' : ''}`}
				title={
					isReadOnly
						? `Read-only mode: ${maintenanceTitle ?? 'Maintenance in progress'}`
						: undefined
				}
			>
				{children}
			</button>
			{isReadOnly && (
				<div className="absolute -top-1 -right-1">
					<svg
						className="w-4 h-4 text-amber-500"
						fill="currentColor"
						viewBox="0 0 20 20"
						aria-hidden="true"
					>
						<path
							fillRule="evenodd"
							d="M5 9V7a5 5 0 0110 0v2a2 2 0 012 2v5a2 2 0 01-2 2H5a2 2 0 01-2-2v-5a2 2 0 012-2zm8-2v2H7V7a3 3 0 016 0z"
							clipRule="evenodd"
						/>
					</svg>
				</div>
			)}
		</div>
	);
}

export function useIsReadOnly(): boolean {
	const { isReadOnly } = useReadOnlyMode();
	return isReadOnly;
}
