import { useEffect, useState } from 'react';
import type { Toast as ToastType, ToastVariant } from '../../hooks/useToast';
import { ToastContext, useToast, useToastValue } from '../../hooks/useToast';

const variantStyles: Record<
	ToastVariant,
	{ bg: string; icon: string; border: string }
> = {
	success: {
		bg: 'bg-green-50 dark:bg-green-900/30',
		icon: 'text-green-500 dark:text-green-400',
		border: 'border-green-200 dark:border-green-800',
	},
	error: {
		bg: 'bg-red-50 dark:bg-red-900/30',
		icon: 'text-red-500 dark:text-red-400',
		border: 'border-red-200 dark:border-red-800',
	},
	warning: {
		bg: 'bg-amber-50 dark:bg-amber-900/30',
		icon: 'text-amber-500 dark:text-amber-400',
		border: 'border-amber-200 dark:border-amber-800',
	},
	info: {
		bg: 'bg-blue-50 dark:bg-blue-900/30',
		icon: 'text-blue-500 dark:text-blue-400',
		border: 'border-blue-200 dark:border-blue-800',
	},
};

function ToastIcon({ variant }: { variant: ToastVariant }) {
	const styles = variantStyles[variant];

	switch (variant) {
		case 'success':
			return (
				<svg
					aria-hidden="true"
					className={`w-5 h-5 ${styles.icon}`}
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
			);
		case 'error':
			return (
				<svg
					aria-hidden="true"
					className={`w-5 h-5 ${styles.icon}`}
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
			);
		case 'warning':
			return (
				<svg
					aria-hidden="true"
					className={`w-5 h-5 ${styles.icon}`}
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
			);
		default:
			return (
				<svg
					aria-hidden="true"
					className={`w-5 h-5 ${styles.icon}`}
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
			);
	}
}

interface ToastItemProps {
	toast: ToastType;
	onDismiss: (id: string) => void;
}

function ToastItem({ toast, onDismiss }: ToastItemProps) {
	const [isExiting, setIsExiting] = useState(false);
	const styles = variantStyles[toast.variant];

	const handleDismiss = () => {
		setIsExiting(true);
		setTimeout(() => {
			onDismiss(toast.id);
		}, 150);
	};

	useEffect(() => {
		if (toast.duration && toast.duration > 0) {
			const exitTime = toast.duration - 150;
			const timer = setTimeout(() => {
				setIsExiting(true);
			}, exitTime);
			return () => clearTimeout(timer);
		}
	}, [toast.duration]);

	return (
		<div
			role="alert"
			className={`flex items-start gap-3 p-4 rounded-lg border shadow-lg transition-all duration-150 ${styles.bg} ${styles.border} ${
				isExiting ? 'opacity-0 translate-x-4' : 'opacity-100 translate-x-0'
			}`}
		>
			<ToastIcon variant={toast.variant} />
			<p className="flex-1 text-sm text-gray-900 dark:text-gray-100">
				{toast.message}
			</p>
			<button
				type="button"
				onClick={handleDismiss}
				className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 transition-colors"
				aria-label="Dismiss notification"
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
		</div>
	);
}

export function ToastContainer() {
	const { toasts, removeToast } = useToast();

	if (toasts.length === 0) {
		return null;
	}

	return (
		<div
			aria-live="polite"
			className="fixed bottom-4 right-4 z-50 flex flex-col gap-2 max-w-sm w-full"
		>
			{toasts.map((toast) => (
				<ToastItem key={toast.id} toast={toast} onDismiss={removeToast} />
			))}
		</div>
	);
}

interface ToastProviderProps {
	children: React.ReactNode;
}

export function ToastProvider({ children }: ToastProviderProps) {
	const toastValue = useToastValue();

	return (
		<ToastContext.Provider value={toastValue}>
			{children}
			<ToastContainer />
		</ToastContext.Provider>
	);
}
