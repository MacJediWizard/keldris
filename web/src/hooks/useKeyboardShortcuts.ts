import { useCallback, useEffect, useRef, useState } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';

export type ShortcutAction =
	| 'goToDashboard'
	| 'goToAgents'
	| 'goToBackups'
	| 'goToSchedules'
	| 'goToRepositories'
	| 'goToAlerts'
	| 'goToRestore'
	| 'focusSearch'
	| 'showHelp'
	| 'newItem'
	| 'closeModal';

export interface ShortcutDefinition {
	keys: string[];
	action: ShortcutAction;
	description: string;
	category: 'navigation' | 'actions' | 'general';
}

export interface ShortcutConfig {
	[key: string]: string[];
}

const SHORTCUTS_STORAGE_KEY = 'keldris-keyboard-shortcuts';

const DEFAULT_SHORTCUTS: ShortcutDefinition[] = [
	// Navigation shortcuts (g + key)
	{
		keys: ['g', 'd'],
		action: 'goToDashboard',
		description: 'Go to Dashboard',
		category: 'navigation',
	},
	{
		keys: ['g', 'a'],
		action: 'goToAgents',
		description: 'Go to Agents',
		category: 'navigation',
	},
	{
		keys: ['g', 'b'],
		action: 'goToBackups',
		description: 'Go to Backups',
		category: 'navigation',
	},
	{
		keys: ['g', 's'],
		action: 'goToSchedules',
		description: 'Go to Schedules',
		category: 'navigation',
	},
	{
		keys: ['g', 'r'],
		action: 'goToRepositories',
		description: 'Go to Repositories',
		category: 'navigation',
	},
	{
		keys: ['g', 'l'],
		action: 'goToAlerts',
		description: 'Go to Alerts',
		category: 'navigation',
	},
	{
		keys: ['g', 'e'],
		action: 'goToRestore',
		description: 'Go to Restore',
		category: 'navigation',
	},
	// Action shortcuts
	{
		keys: ['/'],
		action: 'focusSearch',
		description: 'Focus search',
		category: 'actions',
	},
	{
		keys: ['?'],
		action: 'showHelp',
		description: 'Show keyboard shortcuts',
		category: 'general',
	},
	{
		keys: ['n'],
		action: 'newItem',
		description: 'New item (context-aware)',
		category: 'actions',
	},
	{
		keys: ['Escape'],
		action: 'closeModal',
		description: 'Close modal',
		category: 'general',
	},
];

// Routes for navigation shortcuts
const ROUTE_MAP: Record<string, string> = {
	goToDashboard: '/',
	goToAgents: '/agents',
	goToBackups: '/backups',
	goToSchedules: '/schedules',
	goToRepositories: '/repositories',
	goToAlerts: '/alerts',
	goToRestore: '/restore',
};

// Context-aware new item actions per route
const NEW_ITEM_CONTEXT: Record<
	string,
	{ selector: string; fallbackAction?: () => void }
> = {
	'/agents': { selector: '[data-action="register-agent"]' },
	'/repositories': { selector: '[data-action="create-repository"]' },
	'/schedules': { selector: '[data-action="create-schedule"]' },
	'/policies': { selector: '[data-action="create-policy"]' },
	'/alerts': { selector: '[data-action="create-alert-rule"]' },
	'/notifications': { selector: '[data-action="create-channel"]' },
	'/dr-runbooks': { selector: '[data-action="create-runbook"]' },
	'/tags': { selector: '[data-action="create-tag"]' },
};

function getStoredCustomShortcuts(): ShortcutConfig | null {
	if (typeof window === 'undefined') return null;
	try {
		const stored = localStorage.getItem(SHORTCUTS_STORAGE_KEY);
		return stored ? JSON.parse(stored) : null;
	} catch {
		return null;
	}
}

function saveCustomShortcuts(config: ShortcutConfig): void {
	if (typeof window === 'undefined') return;
	localStorage.setItem(SHORTCUTS_STORAGE_KEY, JSON.stringify(config));
}

interface UseKeyboardShortcutsOptions {
	onShowHelp?: () => void;
	onCloseModal?: () => void;
	enabled?: boolean;
}

export function useKeyboardShortcuts(
	options: UseKeyboardShortcutsOptions = {},
) {
	const { onShowHelp, onCloseModal, enabled = true } = options;
	const navigate = useNavigate();
	const location = useLocation();

	// Track pressed keys for multi-key shortcuts
	const keySequence = useRef<string[]>([]);
	const keyTimeout = useRef<ReturnType<typeof setTimeout> | null>(null);
	const [customShortcuts, setCustomShortcuts] = useState<ShortcutConfig>(
		() => getStoredCustomShortcuts() ?? {},
	);

	// Get effective shortcuts (default merged with custom)
	const getEffectiveShortcuts = useCallback((): ShortcutDefinition[] => {
		return DEFAULT_SHORTCUTS.map((shortcut) => {
			const customKeys = customShortcuts[shortcut.action];
			return customKeys ? { ...shortcut, keys: customKeys } : shortcut;
		});
	}, [customShortcuts]);

	// Check if current element should prevent shortcuts
	const shouldPreventShortcut = useCallback((e: KeyboardEvent): boolean => {
		const target = e.target as HTMLElement;
		const tagName = target.tagName.toLowerCase();

		// Allow shortcuts when:
		// 1. Escape key (always allow for closing modals)
		// 2. '?' key with Shift (always allow for help)
		if (e.key === 'Escape') return false;
		if (e.key === '?' && e.shiftKey) return false;

		// Prevent shortcuts in input elements
		if (
			tagName === 'input' ||
			tagName === 'textarea' ||
			tagName === 'select' ||
			target.isContentEditable
		) {
			return true;
		}

		return false;
	}, []);

	// Execute shortcut action
	const executeAction = useCallback(
		(action: ShortcutAction): boolean => {
			switch (action) {
				case 'goToDashboard':
				case 'goToAgents':
				case 'goToBackups':
				case 'goToSchedules':
				case 'goToRepositories':
				case 'goToAlerts':
				case 'goToRestore': {
					const route = ROUTE_MAP[action];
					if (route && location.pathname !== route) {
						navigate(route);
						return true;
					}
					return false;
				}

				case 'focusSearch': {
					// Find and focus the search input
					const searchInput = document.querySelector<HTMLInputElement>(
						'input[type="text"][placeholder*="Search"], input[type="search"]',
					);
					if (searchInput) {
						searchInput.focus();
						searchInput.select();
						return true;
					}
					return false;
				}

				case 'showHelp': {
					onShowHelp?.();
					return true;
				}

				case 'newItem': {
					// Context-aware new action based on current route
					const context = NEW_ITEM_CONTEXT[location.pathname];
					if (context) {
						const button = document.querySelector<HTMLButtonElement>(
							context.selector,
						);
						if (button) {
							button.click();
							return true;
						}
						// Try fallback generic button
						const genericNewButton = document.querySelector<HTMLButtonElement>(
							'button[type="button"]:has(svg path[d*="M12 4v16m8-8H4"])',
						);
						if (genericNewButton) {
							genericNewButton.click();
							return true;
						}
					}
					return false;
				}

				case 'closeModal': {
					onCloseModal?.();
					// Also try clicking any visible close button
					const closeButton = document.querySelector<HTMLButtonElement>(
						'.fixed.inset-0 button[type="button"]:has(svg)',
					);
					if (closeButton) {
						closeButton.click();
						return true;
					}
					return onCloseModal !== undefined;
				}

				default:
					return false;
			}
		},
		[navigate, location.pathname, onShowHelp, onCloseModal],
	);

	// Match key sequence against shortcuts
	const matchShortcut = useCallback(
		(sequence: string[]): ShortcutDefinition | null => {
			const shortcuts = getEffectiveShortcuts();

			for (const shortcut of shortcuts) {
				if (shortcut.keys.length !== sequence.length) continue;

				const matches = shortcut.keys.every(
					(key, index) => key.toLowerCase() === sequence[index]?.toLowerCase(),
				);

				if (matches) return shortcut;
			}

			return null;
		},
		[getEffectiveShortcuts],
	);

	// Handle keydown
	useEffect(() => {
		if (!enabled) return;

		const handleKeyDown = (e: KeyboardEvent) => {
			// Check if we should prevent this shortcut
			if (shouldPreventShortcut(e)) {
				// But allow Escape in modals
				if (e.key === 'Escape') {
					const modal = document.querySelector('.fixed.inset-0');
					if (modal) {
						executeAction('closeModal');
						return;
					}
				}
				return;
			}

			// Handle ? for help (special case since it requires Shift)
			if (e.key === '?') {
				e.preventDefault();
				executeAction('showHelp');
				return;
			}

			// Handle Escape
			if (e.key === 'Escape') {
				executeAction('closeModal');
				return;
			}

			// Handle single-key shortcuts
			if (e.key === '/') {
				e.preventDefault();
				executeAction('focusSearch');
				return;
			}

			if (e.key === 'n' && !e.ctrlKey && !e.metaKey && !e.altKey) {
				e.preventDefault();
				executeAction('newItem');
				return;
			}

			// Handle multi-key shortcuts (g + key)
			if (
				e.key.toLowerCase() === 'g' &&
				!e.ctrlKey &&
				!e.metaKey &&
				!e.altKey
			) {
				// Start key sequence
				keySequence.current = ['g'];

				// Clear previous timeout
				if (keyTimeout.current) {
					clearTimeout(keyTimeout.current);
				}

				// Set timeout to clear sequence
				keyTimeout.current = setTimeout(() => {
					keySequence.current = [];
				}, 500);

				return;
			}

			// If we're in a key sequence starting with 'g'
			if (keySequence.current.length > 0 && keySequence.current[0] === 'g') {
				keySequence.current.push(e.key.toLowerCase());

				// Try to match shortcut
				const shortcut = matchShortcut(keySequence.current);
				if (shortcut) {
					e.preventDefault();
					executeAction(shortcut.action);
				}

				// Clear sequence
				if (keyTimeout.current) {
					clearTimeout(keyTimeout.current);
				}
				keySequence.current = [];
			}
		};

		window.addEventListener('keydown', handleKeyDown);
		return () => window.removeEventListener('keydown', handleKeyDown);
	}, [enabled, shouldPreventShortcut, executeAction, matchShortcut]);

	// Update custom shortcuts
	const updateCustomShortcut = useCallback(
		(action: ShortcutAction, keys: string[]) => {
			setCustomShortcuts((prev) => {
				const newConfig = { ...prev, [action]: keys };
				saveCustomShortcuts(newConfig);
				return newConfig;
			});
		},
		[],
	);

	// Reset shortcut to default
	const resetShortcut = useCallback((action: ShortcutAction) => {
		setCustomShortcuts((prev) => {
			const newConfig = { ...prev };
			delete newConfig[action];
			saveCustomShortcuts(newConfig);
			return newConfig;
		});
	}, []);

	// Reset all shortcuts to defaults
	const resetAllShortcuts = useCallback(() => {
		setCustomShortcuts({});
		localStorage.removeItem(SHORTCUTS_STORAGE_KEY);
	}, []);

	return {
		shortcuts: getEffectiveShortcuts(),
		defaultShortcuts: DEFAULT_SHORTCUTS,
		customShortcuts,
		updateCustomShortcut,
		resetShortcut,
		resetAllShortcuts,
		executeAction,
	};
}

// Format shortcut keys for display
export function formatShortcutKeys(keys: string[]): string {
	return keys
		.map((key) => {
			if (key === 'Escape') return 'Esc';
			if (key === '/') return '/';
			if (key === '?') return '?';
			return key.toUpperCase();
		})
		.join(' then ');
}

// Get shortcut display for tooltips
export function getShortcutHint(keys: string[]): string {
	return keys
		.map((key) => {
			if (key === 'Escape') return 'Esc';
			return key.toUpperCase();
		})
		.join('+');
}
