import { type ReactNode, useEffect } from 'react';

interface ModalProps {
	open: boolean;
	onClose: () => void;
	children: ReactNode;
}

export function Modal({ open, onClose, children }: ModalProps) {
	useEffect(() => {
		function handleKeyDown(e: KeyboardEvent) {
			if (e.key === 'Escape') onClose();
		}
		if (open) {
			document.addEventListener('keydown', handleKeyDown);
			document.body.style.overflow = 'hidden';
		}
		return () => {
			document.removeEventListener('keydown', handleKeyDown);
			document.body.style.overflow = '';
		};
	}, [open, onClose]);

	if (!open) return null;

	return (
		<div className="fixed inset-0 z-50 flex items-center justify-center">
			<div
				className="fixed inset-0 bg-black/50"
				onClick={onClose}
				onKeyDown={(e) => {
					if (e.key === 'Enter' || e.key === ' ') onClose();
				}}
				role="button"
				tabIndex={-1}
				data-testid="modal-overlay"
			/>
			<dialog
				className="relative z-10 w-full max-w-lg rounded-lg bg-white shadow-xl"
				open
				aria-modal="true"
			>
				<button
					type="button"
					onClick={onClose}
					className="absolute right-4 top-4 text-gray-400 hover:text-gray-600"
					aria-label="Close"
				>
					<svg
						className="h-5 w-5"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
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
				{children}
			</dialog>
		</div>
	);
}

interface ModalHeaderProps {
	children: ReactNode;
}

export function ModalHeader({ children }: ModalHeaderProps) {
	return (
		<div className="border-b border-gray-200 px-6 py-4">
			<h3 className="text-lg font-semibold text-gray-900">{children}</h3>
		</div>
	);
}

interface ModalBodyProps {
	children: ReactNode;
}

export function ModalBody({ children }: ModalBodyProps) {
	return <div className="px-6 py-4">{children}</div>;
}

interface ModalFooterProps {
	children: ReactNode;
}

export function ModalFooter({ children }: ModalFooterProps) {
	return (
		<div className="flex justify-end gap-3 border-t border-gray-200 px-6 py-4">
			{children}
		</div>
	);
}
