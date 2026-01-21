import { useCallback, useEffect, useState } from 'react';

type Theme = 'light' | 'dark' | 'system';

const THEME_KEY = 'keldris-theme';

function getSystemTheme(): 'light' | 'dark' {
	if (typeof window !== 'undefined') {
		return window.matchMedia('(prefers-color-scheme: dark)').matches
			? 'dark'
			: 'light';
	}
	return 'light';
}

function getStoredTheme(): Theme {
	if (typeof window !== 'undefined') {
		const stored = localStorage.getItem(THEME_KEY);
		if (stored === 'light' || stored === 'dark' || stored === 'system') {
			return stored;
		}
	}
	return 'system';
}

function applyTheme(theme: Theme) {
	const root = document.documentElement;
	const effectiveTheme = theme === 'system' ? getSystemTheme() : theme;

	if (effectiveTheme === 'dark') {
		root.classList.add('dark');
	} else {
		root.classList.remove('dark');
	}
}

export function useTheme() {
	const [theme, setThemeState] = useState<Theme>(getStoredTheme);
	const [resolvedTheme, setResolvedTheme] = useState<'light' | 'dark'>(() => {
		const stored = getStoredTheme();
		return stored === 'system' ? getSystemTheme() : stored;
	});

	const setTheme = useCallback((newTheme: Theme) => {
		setThemeState(newTheme);
		localStorage.setItem(THEME_KEY, newTheme);
		applyTheme(newTheme);
		setResolvedTheme(newTheme === 'system' ? getSystemTheme() : newTheme);
	}, []);

	const toggleTheme = useCallback(() => {
		const nextTheme = theme === 'light' ? 'dark' : theme === 'dark' ? 'system' : 'light';
		setTheme(nextTheme);
	}, [theme, setTheme]);

	// Apply theme on mount and listen for system theme changes
	useEffect(() => {
		applyTheme(theme);

		const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
		const handleChange = () => {
			if (theme === 'system') {
				applyTheme('system');
				setResolvedTheme(getSystemTheme());
			}
		};

		mediaQuery.addEventListener('change', handleChange);
		return () => mediaQuery.removeEventListener('change', handleChange);
	}, [theme]);

	return {
		theme,
		resolvedTheme,
		setTheme,
		toggleTheme,
		isDark: resolvedTheme === 'dark',
	};
}
