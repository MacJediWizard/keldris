import { createContext, useCallback, useContext, useState } from 'react';

export type ToastVariant = 'success' | 'error' | 'warning' | 'info';

export interface Toast {
	id: string;
	message: string;
	variant: ToastVariant;
	duration?: number;
}

interface ToastContextValue {
	toasts: Toast[];
	addToast: (
		message: string,
		variant?: ToastVariant,
		duration?: number,
	) => string;
	removeToast: (id: string) => void;
	success: (message: string, duration?: number) => string;
	error: (message: string, duration?: number) => string;
	warning: (message: string, duration?: number) => string;
	info: (message: string, duration?: number) => string;
}

const ToastContext = createContext<ToastContextValue | null>(null);

export function useToast(): ToastContextValue {
	const context = useContext(ToastContext);
	if (!context) {
		throw new Error('useToast must be used within a ToastProvider');
	}
	return context;
}

export function useToastValue(): ToastContextValue {
	const [toasts, setToasts] = useState<Toast[]>([]);

	const removeToast = useCallback((id: string) => {
		setToasts((prev) => prev.filter((toast) => toast.id !== id));
	}, []);

	const addToast = useCallback(
		(
			message: string,
			variant: ToastVariant = 'info',
			duration: number = 5000,
		): string => {
			const id = `toast-${Date.now()}-${Math.random().toString(36).slice(2, 9)}`;
			const toast: Toast = { id, message, variant, duration };

			setToasts((prev) => [...prev, toast]);

			if (duration > 0) {
				setTimeout(() => {
					removeToast(id);
				}, duration);
			}

			return id;
		},
		[removeToast],
	);

	const success = useCallback(
		(message: string, duration?: number) => addToast(message, 'success', duration),
		[addToast],
	);

	const error = useCallback(
		(message: string, duration?: number) => addToast(message, 'error', duration),
		[addToast],
	);

	const warning = useCallback(
		(message: string, duration?: number) => addToast(message, 'warning', duration),
		[addToast],
	);

	const info = useCallback(
		(message: string, duration?: number) => addToast(message, 'info', duration),
		[addToast],
	);

	return {
		toasts,
		addToast,
		removeToast,
		success,
		error,
		warning,
		info,
	};
}

export { ToastContext };
