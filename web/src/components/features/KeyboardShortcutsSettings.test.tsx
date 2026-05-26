import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import type { ShortcutDefinition } from '../../hooks/useKeyboardShortcuts';

vi.mock('../../hooks/useLocale', () => ({
	useLocale: () => ({
		t: (key: string, ..._args: unknown[]) => key,
	}),
}));

vi.mock('../../hooks/useKeyboardShortcuts', () => ({
	useKeyboardShortcuts: vi.fn(),
}));

import { useKeyboardShortcuts } from '../../hooks/useKeyboardShortcuts';
import { KeyboardShortcutsSettings } from './KeyboardShortcutsSettings';

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

function setMocks({
	customShortcuts = {} as Record<string, string[]>,
}: { customShortcuts?: Record<string, string[]> } = {}) {
	vi.mocked(useKeyboardShortcuts).mockReturnValue({
		shortcuts,
		customShortcuts,
		updateCustomShortcut: vi.fn(),
		resetShortcut: vi.fn(),
		resetAllShortcuts: vi.fn(),
	} as never);
}

describe('KeyboardShortcutsSettings', () => {
	it('renders nothing when closed', () => {
		setMocks();
		const { container } = render(
			<KeyboardShortcutsSettings isOpen={false} onClose={() => {}} />,
		);
		expect(container.firstChild).toBeNull();
	});

	it('renders title and section headers when open', () => {
		setMocks();
		render(<KeyboardShortcutsSettings isOpen={true} onClose={() => {}} />);
		expect(screen.getByText('shortcuts.customizeTitle')).toBeDefined();
		expect(screen.getByText('shortcuts.navigation')).toBeDefined();
		expect(screen.getByText('shortcuts.actionsSection')).toBeDefined();
		expect(screen.getByText('shortcuts.general')).toBeDefined();
	});

	it('shows Reset All button when custom shortcuts exist', () => {
		setMocks({ customShortcuts: { goToDashboard: ['g', 'x'] } });
		render(<KeyboardShortcutsSettings isOpen={true} onClose={() => {}} />);
		expect(
			screen.getByRole('button', { name: 'shortcuts.resetAll' }),
		).toBeDefined();
	});

	it('hides Reset All button when no custom shortcuts', () => {
		setMocks();
		render(<KeyboardShortcutsSettings isOpen={true} onClose={() => {}} />);
		expect(
			screen.queryByRole('button', { name: 'shortcuts.resetAll' }),
		).toBeNull();
	});

	it('fires onClose when Done clicked', () => {
		setMocks();
		const onClose = vi.fn();
		render(<KeyboardShortcutsSettings isOpen={true} onClose={onClose} />);
		screen.getByRole('button', { name: 'common.done' }).click();
		expect(onClose).toHaveBeenCalledOnce();
	});
});
