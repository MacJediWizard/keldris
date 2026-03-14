import {
	type ReactNode,
	createContext,
	useCallback,
	useContext,
	useEffect,
	useId,
	useRef,
} from 'react';

const FOCUSABLE_SELECTOR =
	'a[href], button:not([disabled]), textarea:not([disabled]), input:not([disabled]), select:not([disabled]), [tabindex]:not([tabindex="-1"])';

const ModalTitleIdContext = createContext<string | undefined>(undefined);

interface ModalProps {
	open: boolean;
	onClose: () => void;
	children: ReactNode;
}

export function Modal({ open, onClose, children }: ModalProps) {
	const titleId = useId();
	const dialogRef = useRef<HTMLDialogElement>(null);
	const previousFocusRef = useRef<Element | null>(null);

	// Save previous focus and restore on unmount/close
	useEffect(() => {
		if (open) {
			previousFocusRef.current = document.activeElement;
		}
		return () => {
			if (previousFocusRef.current instanceof HTMLElement) {
				previousFocusRef.current.focus();
				previousFocusRef.current = null;
			}
		};
	}, [open]);

	// Focus the dialog when it opens
	useEffect(() => {
		if (open && dialogRef.current) {
			const focusable =
				dialogRef.current.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR);
			if (focusable.length > 0) {
				focusable[0].focus();
			} else {
				dialogRef.current.focus();
			}
		}
	}, [open]);

	// Handle keyboard: Escape to close, Tab to trap focus
	const handleKeyDown = useCallback(
		(e: KeyboardEvent) => {
			if (e.key === 'Escape') {
				onClose();
				return;
			}

			if (e.key === 'Tab' && dialogRef.current) {
				const focusable = Array.from(
					dialogRef.current.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR),
				);
				if (focusable.length === 0) return;

				const first = focusable[0];
				const last = focusable[focusable.length - 1];

				if (e.shiftKey) {
					if (document.activeElement === first) {
						e.preventDefault();
						last.focus();
					}
				} else {
					if (document.activeElement === last) {
						e.preventDefault();
						first.focus();
					}
				}
			}
		},
		[onClose],
	);

	useEffect(() => {
		if (open) {
			document.addEventListener('keydown', handleKeyDown);
			document.body.style.overflow = 'hidden';
		}
		return () => {
			document.removeEventListener('keydown', handleKeyDown);
			document.body.style.overflow = '';
		};
	}, [open, handleKeyDown]);

	if (!open) return null;

	return (
		<ModalTitleIdContext.Provider value={titleId}>
			<div className="fixed inset-0 z-50 flex items-center justify-center">
				{/* biome-ignore lint/a11y/useKeyWithClickEvents: backdrop overlay, not an interactive element */}
				<div
					className="fixed inset-0 bg-black/50"
					onClick={onClose}
					data-testid="modal-overlay"
				/>
				<dialog
					ref={dialogRef}
					className="relative z-10 w-full max-w-lg rounded-lg bg-white dark:bg-gray-800 shadow-xl"
					open
					aria-modal="true"
					aria-labelledby={titleId}
					tabIndex={-1}
				>
					<button
						type="button"
						onClick={onClose}
						className="absolute right-4 top-4 text-gray-400 hover:text-gray-600 dark:text-gray-500 dark:hover:text-gray-300"
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
		</ModalTitleIdContext.Provider>
	);
}

interface ModalHeaderProps {
	children: ReactNode;
}

export function ModalHeader({ children }: ModalHeaderProps) {
	const titleId = useContext(ModalTitleIdContext);
	return (
		<div className="border-b border-gray-200 dark:border-gray-700 px-6 py-4">
			<h3
				id={titleId}
				className="text-lg font-semibold text-gray-900 dark:text-white"
			>
				{children}
			</h3>
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
		<div className="flex justify-end gap-3 border-t border-gray-200 dark:border-gray-700 px-6 py-4">
			{children}
		</div>
	);
}
