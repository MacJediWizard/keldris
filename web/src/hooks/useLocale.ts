import { useCallback, useEffect, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import {
	LANGUAGE_LOCALES,
	LANGUAGE_NAMES,
	SUPPORTED_LANGUAGES,
	type SupportedLanguage,
} from '../lib/i18n';
import { useMe, useUpdatePreferences } from './useAuth';

interface LocaleFormatters {
	formatDate: (dateString: string | undefined) => string;
	formatDateTime: (dateString: string | undefined) => string;
	formatRelativeTime: (dateString: string | undefined) => string;
	formatNumber: (value: number | undefined) => string;
	formatBytes: (bytes: number | undefined) => string;
	formatPercent: (percent: number | undefined) => string;
	formatDuration: (startDate: string, endDate: string | undefined) => string;
}

interface UseLocaleReturn extends LocaleFormatters {
	t: ReturnType<typeof useTranslation>['t'];
	language: SupportedLanguage;
	setLanguage: (lang: SupportedLanguage) => void;
	languages: typeof SUPPORTED_LANGUAGES;
	languageNames: typeof LANGUAGE_NAMES;
}

export function useLocale(): UseLocaleReturn {
	const { t, i18n } = useTranslation();
	const { data: user } = useMe();
	const updatePreferences = useUpdatePreferences();

	const language = (i18n.language || 'en') as SupportedLanguage;
	const locale = LANGUAGE_LOCALES[language] || 'en-US';

	// Sync language with user preference from server on mount
	useEffect(() => {
		if (user?.language && user.language !== i18n.language) {
			i18n.changeLanguage(user.language);
			localStorage.setItem('keldris-language', user.language);
		}
	}, [user?.language, i18n]);

	const setLanguage = useCallback(
		(lang: SupportedLanguage) => {
			i18n.changeLanguage(lang);
			localStorage.setItem('keldris-language', lang);
			// Save to server (fire and forget, local storage is the source of truth for offline)
			updatePreferences.mutate({ language: lang });
		},
		[i18n, updatePreferences],
	);

	const formatDate = useCallback(
		(dateString: string | undefined): string => {
			if (!dateString) return t('common.never');

			const date = new Date(dateString);
			const now = new Date();

			return date.toLocaleDateString(locale, {
				month: 'short',
				day: 'numeric',
				year: date.getFullYear() !== now.getFullYear() ? 'numeric' : undefined,
			});
		},
		[locale, t],
	);

	const formatDateTime = useCallback(
		(dateString: string | undefined): string => {
			if (!dateString) return 'N/A';

			const date = new Date(dateString);
			return date.toLocaleString(locale, {
				month: 'short',
				day: 'numeric',
				year: 'numeric',
				hour: 'numeric',
				minute: '2-digit',
			});
		},
		[locale],
	);

	const formatRelativeTime = useCallback(
		(dateString: string | undefined): string => {
			if (!dateString) return t('common.never');

			const date = new Date(dateString);
			const now = new Date();
			const diffMs = now.getTime() - date.getTime();
			const diffSeconds = Math.floor(diffMs / 1000);
			const diffMinutes = Math.floor(diffSeconds / 60);
			const diffHours = Math.floor(diffMinutes / 60);
			const diffDays = Math.floor(diffHours / 24);

			if (diffSeconds < 60) return t('common.justNow');
			if (diffMinutes < 60) return t('time.minutesAgo', { count: diffMinutes });
			if (diffHours < 24) return t('time.hoursAgo', { count: diffHours });
			if (diffDays < 7) return t('time.daysAgo', { count: diffDays });

			return formatDate(dateString);
		},
		[t, formatDate],
	);

	const formatNumber = useCallback(
		(value: number | undefined): string => {
			if (value === undefined || value === null) return 'N/A';
			return new Intl.NumberFormat(locale).format(value);
		},
		[locale],
	);

	const formatBytes = useCallback(
		(bytes: number | undefined): string => {
			if (bytes === undefined || bytes === null) return 'N/A';
			if (bytes === 0) return '0 B';

			const units = ['B', 'KB', 'MB', 'GB', 'TB'];
			const k = 1024;
			const i = Math.floor(Math.log(bytes) / Math.log(k));
			const value = bytes / k ** i;

			return `${new Intl.NumberFormat(locale, { maximumFractionDigits: i > 0 ? 1 : 0 }).format(value)} ${units[i]}`;
		},
		[locale],
	);

	const formatPercent = useCallback(
		(percent: number | undefined): string => {
			if (percent === undefined || percent === null) return 'N/A';
			return new Intl.NumberFormat(locale, {
				style: 'percent',
				minimumFractionDigits: 1,
				maximumFractionDigits: 1,
			}).format(percent / 100);
		},
		[locale],
	);

	const formatDuration = useCallback(
		(startDate: string, endDate: string | undefined): string => {
			if (!endDate) return t('time.inProgress');

			const start = new Date(startDate);
			const end = new Date(endDate);
			const diffMs = end.getTime() - start.getTime();
			const diffSeconds = Math.floor(diffMs / 1000);
			const diffMinutes = Math.floor(diffSeconds / 60);
			const diffHours = Math.floor(diffMinutes / 60);

			if (diffSeconds < 60) return t('time.seconds', { count: diffSeconds });
			if (diffMinutes < 60)
				return `${t('time.minutes', { count: diffMinutes })} ${t('time.seconds', { count: diffSeconds % 60 })}`;
			return `${t('time.hours', { count: diffHours })} ${t('time.minutes', { count: diffMinutes % 60 })}`;
		},
		[t],
	);

	return useMemo(
		() => ({
			t,
			language,
			setLanguage,
			languages: SUPPORTED_LANGUAGES,
			languageNames: LANGUAGE_NAMES,
			formatDate,
			formatDateTime,
			formatRelativeTime,
			formatNumber,
			formatBytes,
			formatPercent,
			formatDuration,
		}),
		[
			t,
			language,
			setLanguage,
			formatDate,
			formatDateTime,
			formatRelativeTime,
			formatNumber,
			formatBytes,
			formatPercent,
			formatDuration,
		],
	);
}
