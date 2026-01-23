import { useState } from 'react';
import {
	type ShortcutAction,
	type ShortcutDefinition,
	useKeyboardShortcuts,
} from '../../hooks/useKeyboardShortcuts';
import { useLocale } from '../../hooks/useLocale';

interface KeyboardShortcutsSettingsProps {
	isOpen: boolean;
	onClose: () => void;
}

function ShortcutEditor({
	shortcut,
	onUpdate,
	onReset,
	isCustom,
}: {
	shortcut: ShortcutDefinition;
	onUpdate: (action: ShortcutAction, keys: string[]) => void;
	onReset: (action: ShortcutAction) => void;
	isCustom: boolean;
}) {
	const [isEditing, setIsEditing] = useState(false);
	const [capturedKeys, setCapturedKeys] = useState<string[]>([]);
	const { t } = useLocale();

	const handleKeyCapture = (e: React.KeyboardEvent) => {
		e.preventDefault();
		e.stopPropagation();

		const key = e.key;

		// Skip modifier keys alone
		if (['Shift', 'Control', 'Alt', 'Meta'].includes(key)) {
			return;
		}

		// For multi-key shortcuts like g+d
		if (capturedKeys.length === 0 && key.toLowerCase() === 'g') {
			setCapturedKeys(['g']);
			return;
		}

		// Complete the shortcut
		if (capturedKeys.length > 0) {
			const newKeys = [...capturedKeys, key.toLowerCase()];
			onUpdate(shortcut.action, newKeys);
			setCapturedKeys([]);
			setIsEditing(false);
		} else {
			// Single key shortcut
			onUpdate(shortcut.action, [key]);
			setIsEditing(false);
		}
	};

	const handleStartEditing = () => {
		setIsEditing(true);
		setCapturedKeys([]);
	};

	const handleCancelEditing = () => {
		setIsEditing(false);
		setCapturedKeys([]);
	};

	return (
		<div className="flex items-center justify-between py-3 border-b border-gray-100 last:border-0">
			<div className="flex-1">
				<span className="text-sm text-gray-700">
					{t(`shortcuts.actions.${shortcut.action}`) || shortcut.description}
				</span>
			</div>
			<div className="flex items-center gap-2">
				{isEditing ? (
					<div className="flex items-center gap-2">
						<input
							type="text"
							className="w-32 px-3 py-1.5 text-sm border border-indigo-500 rounded-lg bg-indigo-50 text-center focus:outline-none"
							placeholder={t('shortcuts.pressKey')}
							value={
								capturedKeys.length > 0 ? `${capturedKeys.join(' + ')}...` : ''
							}
							onKeyDown={handleKeyCapture}
							readOnly
						/>
						<button
							type="button"
							onClick={handleCancelEditing}
							className="p-1 text-gray-400 hover:text-gray-600"
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
				) : (
					<>
						<button
							type="button"
							onClick={handleStartEditing}
							className="flex items-center gap-1"
						>
							{shortcut.keys.map((key, index) => (
								<span key={key}>
									<kbd className="inline-flex items-center justify-center min-w-[24px] h-6 px-1.5 text-xs font-mono font-medium text-gray-700 bg-gray-100 border border-gray-300 rounded shadow-sm hover:bg-gray-200 transition-colors">
										{key === 'Escape'
											? 'Esc'
											: key === '?'
												? '?'
												: key.toUpperCase()}
									</kbd>
									{index < shortcut.keys.length - 1 && (
										<span className="mx-0.5 text-gray-400 text-xs">+</span>
									)}
								</span>
							))}
						</button>
						{isCustom && (
							<button
								type="button"
								onClick={() => onReset(shortcut.action)}
								className="p-1 text-gray-400 hover:text-gray-600"
								title={t('shortcuts.resetToDefault')}
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
										d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
									/>
								</svg>
							</button>
						)}
					</>
				)}
			</div>
		</div>
	);
}

export function KeyboardShortcutsSettings({
	isOpen,
	onClose,
}: KeyboardShortcutsSettingsProps) {
	const { t } = useLocale();
	const {
		shortcuts,
		customShortcuts,
		updateCustomShortcut,
		resetShortcut,
		resetAllShortcuts,
	} = useKeyboardShortcuts({
		enabled: false, // Don't enable shortcuts while editing
	});

	if (!isOpen) return null;

	const navigationShortcuts = shortcuts.filter(
		(s) => s.category === 'navigation',
	);
	const actionShortcuts = shortcuts.filter((s) => s.category === 'actions');
	const generalShortcuts = shortcuts.filter((s) => s.category === 'general');

	const hasCustomShortcuts = Object.keys(customShortcuts).length > 0;

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
				className="bg-white rounded-lg shadow-xl max-w-2xl w-full mx-4 max-h-[80vh] overflow-hidden"
				aria-labelledby="shortcut-settings-title"
			>
				<div className="flex items-center justify-between px-6 py-4 border-b border-gray-200">
					<div>
						<h2
							id="shortcut-settings-title"
							className="text-lg font-semibold text-gray-900"
						>
							{t('shortcuts.customizeTitle')}
						</h2>
						<p className="text-sm text-gray-500 mt-1">
							{t('shortcuts.customizeDescription')}
						</p>
					</div>
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
						<div>
							{navigationShortcuts.map((shortcut) => (
								<ShortcutEditor
									key={shortcut.action}
									shortcut={shortcut}
									onUpdate={updateCustomShortcut}
									onReset={resetShortcut}
									isCustom={!!customShortcuts[shortcut.action]}
								/>
							))}
						</div>
					</div>

					{/* Action Shortcuts */}
					<div className="mb-6">
						<h3 className="text-sm font-semibold text-gray-500 uppercase tracking-wider mb-3">
							{t('shortcuts.actions')}
						</h3>
						<div>
							{actionShortcuts.map((shortcut) => (
								<ShortcutEditor
									key={shortcut.action}
									shortcut={shortcut}
									onUpdate={updateCustomShortcut}
									onReset={resetShortcut}
									isCustom={!!customShortcuts[shortcut.action]}
								/>
							))}
						</div>
					</div>

					{/* General Shortcuts */}
					<div>
						<h3 className="text-sm font-semibold text-gray-500 uppercase tracking-wider mb-3">
							{t('shortcuts.general')}
						</h3>
						<div>
							{generalShortcuts.map((shortcut) => (
								<ShortcutEditor
									key={shortcut.action}
									shortcut={shortcut}
									onUpdate={updateCustomShortcut}
									onReset={resetShortcut}
									isCustom={!!customShortcuts[shortcut.action]}
								/>
							))}
						</div>
					</div>
				</div>

				<div className="px-6 py-4 bg-gray-50 border-t border-gray-200 flex justify-between items-center">
					{hasCustomShortcuts && (
						<button
							type="button"
							onClick={resetAllShortcuts}
							className="text-sm text-red-600 hover:text-red-800"
						>
							{t('shortcuts.resetAll')}
						</button>
					)}
					<div className="flex-1" />
					<button
						type="button"
						onClick={onClose}
						className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors"
					>
						{t('common.done')}
					</button>
				</div>
			</div>
		</div>
	);
}
