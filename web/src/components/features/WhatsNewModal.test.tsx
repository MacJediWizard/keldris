import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { ChangelogEntry } from '../../lib/types';
import { WhatsNewModal } from './WhatsNewModal';

function renderInRouter(ui: React.ReactElement) {
	return render(<MemoryRouter>{ui}</MemoryRouter>);
}

const sampleEntry: ChangelogEntry = {
	version: '1.2.3',
	date: '2026-05-26',
	added: ['Cool feature'],
	changed: ['Improved foo'],
	fixed: ['Fixed bar'],
} as ChangelogEntry;

describe('WhatsNewModal', () => {
	beforeEach(() => {
		localStorage.clear();
	});

	it('renders nothing when entry is null', () => {
		const { container } = renderInRouter(
			<WhatsNewModal
				entry={null}
				currentVersion="1.2.3"
				onDismiss={() => {}}
			/>,
		);
		expect(container.firstChild).toBeNull();
	});

	it('renders nothing when currentVersion undefined', () => {
		const { container } = renderInRouter(
			<WhatsNewModal
				entry={sampleEntry}
				currentVersion={undefined}
				onDismiss={() => {}}
			/>,
		);
		expect(container.firstChild).toBeNull();
	});

	it('renders modal with version header when version is unseen', () => {
		renderInRouter(
			<WhatsNewModal
				entry={sampleEntry}
				currentVersion="1.2.3"
				onDismiss={() => {}}
			/>,
		);
		expect(screen.getByText('Keldris v1.2.3')).toBeDefined();
		expect(screen.getByText('New Features')).toBeDefined();
		expect(screen.getByText('Cool feature')).toBeDefined();
	});

	it('renders nothing when version already seen', () => {
		localStorage.setItem('keldris_seen_version', '1.2.3');
		const { container } = renderInRouter(
			<WhatsNewModal
				entry={sampleEntry}
				currentVersion="1.2.3"
				onDismiss={() => {}}
			/>,
		);
		expect(container.firstChild).toBeNull();
	});

	it('renders fallback copy when no changes are listed', () => {
		renderInRouter(
			<WhatsNewModal
				entry={{ version: '2.0.0', date: '' } as ChangelogEntry}
				currentVersion="2.0.0"
				onDismiss={() => {}}
			/>,
		);
		expect(
			screen.getByText(
				'This release includes various improvements and bug fixes.',
			),
		).toBeDefined();
	});

	it('fires onDismiss when Got it clicked', () => {
		const onDismiss = vi.fn();
		renderInRouter(
			<WhatsNewModal
				entry={sampleEntry}
				currentVersion="1.2.3"
				onDismiss={onDismiss}
			/>,
		);
		screen.getByRole('button', { name: 'Got it' }).click();
		expect(onDismiss).toHaveBeenCalledOnce();
		expect(localStorage.getItem('keldris_seen_version')).toBe('1.2.3');
	});
});
