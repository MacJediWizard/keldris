import { useEffect, useRef } from 'react';
import type { ShortcutDefinition } from '../../hooks/useKeyboardShortcuts';
import { useLocale } from '../../hooks/useLocale';

interface ShortcutHelpModalProps {
	isOpen: boolean;
	onClose: () => void;
	shortcuts: ShortcutDefinition[];
}

function ShortcutKey({ keys }: { keys: string[] }) {
	return (
		<span className="flex items-center gap-1">
			{keys.map((key, index) => (
				<span key={key}>
					<kbd className="inline-flex items-center justify-center min-w-[24px] h-6 px-1.5 text-xs font-mono font-medium text-gray-700 bg-gray-100 border border-gray-300 rounded shadow-sm">
						{key === 'Escape' ? 'Esc' : key === '?' ? '?' : key.toUpperCase()}
					</kbd>
					{index < keys.length - 1 && (
						<span className="mx-1 text-gray-400 text-xs">then</span>
					)}
				</span>
			))}
		</span>
	);
}

export function ShortcutHelpModal({
	isOpen,
	onClose,
	shortcuts,
}: ShortcutHelpModalProps) {
	const { t } = useLocale();
	const modalRef = useRef<HTMLDivElement>(null);

	// Handle escape key to close
	useEffect(() => {
		if (!isOpen) return;

		const handleKeyDown = (e: KeyboardEvent) => {
			if (e.key === 'Escape') {
				e.preventDefault();
				e.stopPropagation();
				onClose();
			}
		};

		window.addEventListener('keydown', handleKeyDown);
		return () => window.removeEventListener('keydown', handleKeyDown);
	}, [isOpen, onClose]);

	// Focus trap
	useEffect(() => {
		if (!isOpen) return;
		modalRef.current?.focus();
	}, [isOpen]);

	if (!isOpen) return null;

	const navigationShortcuts = shortcuts.filter(
		(s) => s.category === 'navigation',
	);
	const actionShortcuts = shortcuts.filter((s) => s.category === 'actions');
	const generalShortcuts = shortcuts.filter((s) => s.category === 'general');

	return (
		<div
			className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50"
			onClick={(e) => {
				if (e.target === e.currentTarget) onClose();
			}}
			onKeyDown={(e) => {
				if (e.key === 'Escape') onClose();
			}}
		>
			<div
				ref={modalRef}
				className="bg-white rounded-lg shadow-xl max-w-2xl w-full mx-4 max-h-[80vh] overflow-hidden"
				role="dialog"
				aria-labelledby="shortcut-help-title"
				aria-modal="true"
				tabIndex={-1}
			>
				<div className="flex items-center justify-between px-6 py-4 border-b border-gray-200">
					<h2
						id="shortcut-help-title"
						className="text-lg font-semibold text-gray-900"
					>
						{t('shortcuts.title')}
					</h2>
					<button
						type="button"
						onClick={onClose}
						className="p-2 text-gray-400 hover:text-gray-600 rounded-lg hover:bg-gray-100 transition-colors"
						aria-label={t('common.close')}
					>
						<svg
							aria-hidden="true"
							className="w-5 h-5"
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

				<div className="px-6 py-4 overflow-y-auto max-h-[60vh]">
					{/* Navigation Shortcuts */}
					<div className="mb-6">
						<h3 className="text-sm font-semibold text-gray-500 uppercase tracking-wider mb-3">
							{t('shortcuts.navigation')}
						</h3>
						<div className="space-y-2">
							{navigationShortcuts.map((shortcut) => (
								<div
									key={shortcut.action}
									className="flex items-center justify-between py-2"
								>
									<span className="text-sm text-gray-700">
										{t(`shortcuts.actions.${shortcut.action}`) ||
											shortcut.description}
									</span>
									<ShortcutKey keys={shortcut.keys} />
								</div>
							))}
						</div>
					</div>

					{/* Action Shortcuts */}
					<div className="mb-6">
						<h3 className="text-sm font-semibold text-gray-500 uppercase tracking-wider mb-3">
							{t('shortcuts.actions')}
						</h3>
						<div className="space-y-2">
							{actionShortcuts.map((shortcut) => (
								<div
									key={shortcut.action}
									className="flex items-center justify-between py-2"
								>
									<span className="text-sm text-gray-700">
										{t(`shortcuts.actions.${shortcut.action}`) ||
											shortcut.description}
									</span>
									<ShortcutKey keys={shortcut.keys} />
								</div>
							))}
						</div>
					</div>

					{/* General Shortcuts */}
					<div>
						<h3 className="text-sm font-semibold text-gray-500 uppercase tracking-wider mb-3">
							{t('shortcuts.general')}
						</h3>
						<div className="space-y-2">
							{generalShortcuts.map((shortcut) => (
								<div
									key={shortcut.action}
									className="flex items-center justify-between py-2"
								>
									<span className="text-sm text-gray-700">
										{t(`shortcuts.actions.${shortcut.action}`) ||
											shortcut.description}
									</span>
									<ShortcutKey keys={shortcut.keys} />
								</div>
							))}
						</div>
					</div>
				</div>

				<div className="px-6 py-4 bg-gray-50 border-t border-gray-200">
					<p className="text-xs text-gray-500 text-center">
						{t('shortcuts.hint')}
					</p>
				</div>
			</div>
		</div>
	);
}
