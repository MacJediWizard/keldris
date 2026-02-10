import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi, beforeEach } from 'vitest';

const mockSetLanguage = vi.fn();

vi.mock('../../hooks/useLocale', () => ({
	useLocale: vi.fn(() => ({
		language: 'en',
		setLanguage: mockSetLanguage,
		languages: ['en', 'es', 'pt'],
		languageNames: { en: 'English', es: 'Español', pt: 'Português' },
		t: (key: string) => {
			const translations: Record<string, string> = {
				'settings.language': 'Language',
			};
			return translations[key] || key;
		},
	})),
}));

import { LanguageSelector } from './LanguageSelector';

describe('LanguageSelector', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('renders language button with current flag', () => {
		render(<LanguageSelector />);
		expect(screen.getByRole('button', { name: 'Language' })).toBeInTheDocument();
	});

	it('does not show dropdown by default', () => {
		render(<LanguageSelector />);
		expect(screen.queryByText('Español')).not.toBeInTheDocument();
	});

	it('opens dropdown on click', async () => {
		const user = userEvent.setup();
		render(<LanguageSelector />);
		await user.click(screen.getByRole('button', { name: 'Language' }));
		// English appears in both button and dropdown
		expect(screen.getAllByText('English').length).toBeGreaterThanOrEqual(2);
		expect(screen.getByText('Español')).toBeInTheDocument();
		expect(screen.getByText('Português')).toBeInTheDocument();
	});

	it('shows Language header in dropdown', async () => {
		const user = userEvent.setup();
		render(<LanguageSelector />);
		await user.click(screen.getByRole('button', { name: 'Language' }));
		const headers = screen.getAllByText('Language');
		expect(headers.length).toBeGreaterThan(0);
	});

	it('calls setLanguage when selecting a language', async () => {
		const user = userEvent.setup();
		render(<LanguageSelector />);
		await user.click(screen.getByRole('button', { name: 'Language' }));
		await user.click(screen.getByText('Español'));
		expect(mockSetLanguage).toHaveBeenCalledWith('es');
	});

	it('closes dropdown after selecting a language', async () => {
		const user = userEvent.setup();
		render(<LanguageSelector />);
		await user.click(screen.getByRole('button', { name: 'Language' }));
		expect(screen.getByText('Español')).toBeInTheDocument();
		await user.click(screen.getByText('Español'));
		expect(screen.queryByText('Português')).not.toBeInTheDocument();
	});

	it('toggles dropdown on repeated clicks', async () => {
		const user = userEvent.setup();
		render(<LanguageSelector />);
		const button = screen.getByRole('button', { name: 'Language' });
		await user.click(button);
		expect(screen.getByText('Español')).toBeInTheDocument();
		await user.click(button);
		expect(screen.queryByText('Español')).not.toBeInTheDocument();
	});
});
