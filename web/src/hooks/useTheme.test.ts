import { renderHook, act } from '@testing-library/react';
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { useTheme } from './useTheme';

describe('useTheme', () => {
	beforeEach(() => {
		localStorage.clear();
		document.documentElement.classList.remove('dark');
		vi.restoreAllMocks();
	});

	it('defaults to system theme when no stored preference', () => {
		const { result } = renderHook(() => useTheme());
		expect(result.current.theme).toBe('system');
	});

	it('reads stored theme from localStorage', () => {
		localStorage.setItem('keldris-theme', 'dark');
		const { result } = renderHook(() => useTheme());
		expect(result.current.theme).toBe('dark');
	});

	it('setTheme updates theme and localStorage', () => {
		const { result } = renderHook(() => useTheme());
		act(() => {
			result.current.setTheme('dark');
		});
		expect(result.current.theme).toBe('dark');
		expect(localStorage.getItem('keldris-theme')).toBe('dark');
	});

	it('toggleTheme cycles through light -> dark -> system -> light', () => {
		localStorage.setItem('keldris-theme', 'light');
		const { result } = renderHook(() => useTheme());
		expect(result.current.theme).toBe('light');

		act(() => result.current.toggleTheme());
		expect(result.current.theme).toBe('dark');

		act(() => result.current.toggleTheme());
		expect(result.current.theme).toBe('system');

		act(() => result.current.toggleTheme());
		expect(result.current.theme).toBe('light');
	});

	it('applies dark class to document root for dark theme', () => {
		const { result } = renderHook(() => useTheme());
		act(() => {
			result.current.setTheme('dark');
		});
		expect(document.documentElement.classList.contains('dark')).toBe(true);
	});

	it('removes dark class for light theme', () => {
		document.documentElement.classList.add('dark');
		const { result } = renderHook(() => useTheme());
		act(() => {
			result.current.setTheme('light');
		});
		expect(document.documentElement.classList.contains('dark')).toBe(false);
	});

	it('isDark reflects resolved theme', () => {
		const { result } = renderHook(() => useTheme());
		act(() => {
			result.current.setTheme('dark');
		});
		expect(result.current.isDark).toBe(true);

		act(() => {
			result.current.setTheme('light');
		});
		expect(result.current.isDark).toBe(false);
	});

	it('resolvedTheme returns effective theme', () => {
		const { result } = renderHook(() => useTheme());
		act(() => {
			result.current.setTheme('dark');
		});
		expect(result.current.resolvedTheme).toBe('dark');
	});
});
