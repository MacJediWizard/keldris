import { act, renderHook } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import { useLocale } from './useLocale';

const mockChangeLanguage = vi.fn();
const mockMutate = vi.fn();

vi.mock('react-i18next', () => ({
	useTranslation: () => ({
		t: (key: string) => key,
		i18n: {
			language: 'en',
			changeLanguage: mockChangeLanguage,
		},
	}),
}));

vi.mock('../lib/i18n', () => ({
	LANGUAGE_LOCALES: { en: 'en-US', es: 'es-ES', pt: 'pt-BR' },
	LANGUAGE_NAMES: { en: 'English', es: 'Español', pt: 'Português' },
	SUPPORTED_LANGUAGES: ['en', 'es', 'pt'],
}));

vi.mock('./useAuth', () => ({
	useMe: () => ({ data: null }),
	useUpdatePreferences: () => ({ mutate: mockMutate }),
}));

describe('useLocale', () => {
	beforeEach(() => {
		vi.clearAllMocks();
		localStorage.clear();
	});

	it('returns t function', () => {
		const { result } = renderHook(() => useLocale(), {
			wrapper: createWrapper(),
		});
		expect(result.current.t).toBeDefined();
		expect(result.current.t('test.key')).toBe('test.key');
	});

	it('returns language and language lists', () => {
		const { result } = renderHook(() => useLocale(), {
			wrapper: createWrapper(),
		});
		expect(result.current.language).toBe('en');
		expect(result.current.languages).toEqual(['en', 'es', 'pt']);
		expect(result.current.languageNames).toEqual({
			en: 'English',
			es: 'Español',
			pt: 'Português',
		});
	});

	it('setLanguage changes language and saves to localStorage', () => {
		const { result } = renderHook(() => useLocale(), {
			wrapper: createWrapper(),
		});
		act(() => {
			result.current.setLanguage('es' as 'en');
		});
		expect(mockChangeLanguage).toHaveBeenCalledWith('es');
		expect(localStorage.getItem('keldris-language')).toBe('es');
		expect(mockMutate).toHaveBeenCalledWith({ language: 'es' });
	});

	it('formatDate returns common.never for undefined', () => {
		const { result } = renderHook(() => useLocale(), {
			wrapper: createWrapper(),
		});
		expect(result.current.formatDate(undefined)).toBe('common.never');
	});

	it('formatDate formats a valid date', () => {
		const { result } = renderHook(() => useLocale(), {
			wrapper: createWrapper(),
		});
		const formatted = result.current.formatDate('2024-06-15T12:00:00Z');
		expect(formatted).toBeTruthy();
		expect(formatted).not.toBe('common.never');
	});

	it('formatDateTime returns N/A for undefined', () => {
		const { result } = renderHook(() => useLocale(), {
			wrapper: createWrapper(),
		});
		expect(result.current.formatDateTime(undefined)).toBe('N/A');
	});

	it('formatDateTime formats a valid date', () => {
		const { result } = renderHook(() => useLocale(), {
			wrapper: createWrapper(),
		});
		const formatted = result.current.formatDateTime('2024-06-15T12:00:00Z');
		expect(formatted).toBeTruthy();
		expect(formatted).not.toBe('N/A');
	});

	it('formatRelativeTime returns common.never for undefined', () => {
		const { result } = renderHook(() => useLocale(), {
			wrapper: createWrapper(),
		});
		expect(result.current.formatRelativeTime(undefined)).toBe('common.never');
	});

	it('formatNumber returns N/A for undefined', () => {
		const { result } = renderHook(() => useLocale(), {
			wrapper: createWrapper(),
		});
		expect(result.current.formatNumber(undefined)).toBe('N/A');
	});

	it('formatNumber formats a number', () => {
		const { result } = renderHook(() => useLocale(), {
			wrapper: createWrapper(),
		});
		expect(result.current.formatNumber(1234)).toBeTruthy();
	});

	it('formatBytes returns N/A for undefined', () => {
		const { result } = renderHook(() => useLocale(), {
			wrapper: createWrapper(),
		});
		expect(result.current.formatBytes(undefined)).toBe('N/A');
	});

	it('formatBytes returns 0 B for zero', () => {
		const { result } = renderHook(() => useLocale(), {
			wrapper: createWrapper(),
		});
		expect(result.current.formatBytes(0)).toBe('0 B');
	});

	it('formatBytes formats bytes', () => {
		const { result } = renderHook(() => useLocale(), {
			wrapper: createWrapper(),
		});
		const formatted = result.current.formatBytes(1048576);
		expect(formatted).toContain('MB');
	});

	it('formatPercent returns N/A for undefined', () => {
		const { result } = renderHook(() => useLocale(), {
			wrapper: createWrapper(),
		});
		expect(result.current.formatPercent(undefined)).toBe('N/A');
	});

	it('formatPercent formats a percentage', () => {
		const { result } = renderHook(() => useLocale(), {
			wrapper: createWrapper(),
		});
		const formatted = result.current.formatPercent(75);
		expect(formatted).toBeTruthy();
	});

	it('formatDuration returns time.inProgress when endDate is undefined', () => {
		const { result } = renderHook(() => useLocale(), {
			wrapper: createWrapper(),
		});
		expect(
			result.current.formatDuration('2024-01-01T00:00:00Z', undefined),
		).toBe('time.inProgress');
	});

	it('formatDuration formats a duration in seconds', () => {
		const { result } = renderHook(() => useLocale(), {
			wrapper: createWrapper(),
		});
		const formatted = result.current.formatDuration(
			'2024-01-01T00:00:00Z',
			'2024-01-01T00:00:30Z',
		);
		expect(formatted).toContain('time.seconds');
	});

	it('formatDuration formats a duration in minutes', () => {
		const { result } = renderHook(() => useLocale(), {
			wrapper: createWrapper(),
		});
		const formatted = result.current.formatDuration(
			'2024-01-01T00:00:00Z',
			'2024-01-01T00:05:30Z',
		);
		expect(formatted).toContain('time.minutes');
	});

	it('formatDuration formats a duration in hours', () => {
		const { result } = renderHook(() => useLocale(), {
			wrapper: createWrapper(),
		});
		const formatted = result.current.formatDuration(
			'2024-01-01T00:00:00Z',
			'2024-01-01T02:30:00Z',
		);
		expect(formatted).toContain('time.hours');
	});
});
