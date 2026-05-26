import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import type { ShortcutDefinition } from '../../hooks/useKeyboardShortcuts';

vi.mock('../../hooks/useLocale', () => ({
	useLocale: () => ({
		t: (key: string, ..._args: unknown[]) => key,
	}),
}));

import { ShortcutHelpModal } from './ShortcutHelpModal';

const shortcuts: ShortcutDefinition[] = [
	{
		keys: ['g', 'd'],
		action: 'goToDashboard',
		description: 'Go to Dashboard',
		category: 'navigation',
	},
	{
		keys: ['/'],
		action: 'focusSearch',
		description: 'Focus search',
		category: 'actions',
	},
	{
		keys: ['?'],
		action: 'showHelp',
		description: 'Show help',
		category: 'general',
	},
];

describe('ShortcutHelpModal', () => {
	it('renders nothing when closed', () => {
		const { container } = render(
			<ShortcutHelpModal
				isOpen={false}
				onClose={() => {}}
				shortcuts={shortcuts}
			/>,
		);
		expect(container.firstChild).toBeNull();
	});

	it('renders modal title and sections when open', () => {
		render(
			<ShortcutHelpModal
				isOpen={true}
				onClose={() => {}}
				shortcuts={shortcuts}
			/>,
		);
		expect(screen.getByText('shortcuts.title')).toBeDefined();
		expect(screen.getByText('shortcuts.navigation')).toBeDefined();
		expect(screen.getByText('shortcuts.actionsSection')).toBeDefined();
		expect(screen.getByText('shortcuts.general')).toBeDefined();
	});

	it('renders kbd elements for each shortcut key', () => {
		const { container } = render(
			<ShortcutHelpModal
				isOpen={true}
				onClose={() => {}}
				shortcuts={shortcuts}
			/>,
		);
		const kbds = container.querySelectorAll('kbd');
		expect(kbds.length).toBeGreaterThan(0);
	});

	it('fires onClose when close button clicked', () => {
		const onClose = vi.fn();
		render(
			<ShortcutHelpModal
				isOpen={true}
				onClose={onClose}
				shortcuts={shortcuts}
			/>,
		);
		screen.getByRole('button', { name: 'common.close' }).click();
		expect(onClose).toHaveBeenCalledOnce();
	});
});
