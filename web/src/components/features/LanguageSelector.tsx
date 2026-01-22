import { useState } from 'react';
import { useLocale } from '../../hooks/useLocale';
import type { SupportedLanguage } from '../../lib/i18n';

const LANGUAGE_FLAGS: Record<SupportedLanguage, string> = {
	en: '🇺🇸',
	es: '🇪🇸',
	pt: '🇧🇷',
};

export function LanguageSelector() {
	const [showDropdown, setShowDropdown] = useState(false);
	const { language, setLanguage, languages, languageNames, t } = useLocale();

	const handleSelect = (lang: SupportedLanguage) => {
		setLanguage(lang);
		setShowDropdown(false);
	};

	return (
		<div className="relative">
			<button
				type="button"
				onClick={() => setShowDropdown(!showDropdown)}
				className="flex items-center gap-2 px-3 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50"
				aria-label={t('settings.language')}
			>
				<span className="text-base">{LANGUAGE_FLAGS[language]}</span>
				<span className="hidden sm:inline">{languageNames[language]}</span>
				<svg
					aria-hidden="true"
					className="w-4 h-4 text-gray-400"
					fill="none"
					stroke="currentColor"
					viewBox="0 0 24 24"
				>
					<path
						strokeLinecap="round"
						strokeLinejoin="round"
						strokeWidth={2}
						d="M19 9l-7 7-7-7"
					/>
				</svg>
			</button>
			{showDropdown && (
				<>
					<div
						className="fixed inset-0 z-10"
						onClick={() => setShowDropdown(false)}
						onKeyDown={(e) => e.key === 'Escape' && setShowDropdown(false)}
					/>
					<div className="absolute right-0 mt-2 w-40 bg-white rounded-lg shadow-lg border border-gray-200 py-1 z-50">
						<div className="px-3 py-2 text-xs font-semibold text-gray-500 uppercase">
							{t('settings.language')}
						</div>
						{languages.map((lang) => (
							<button
								key={lang}
								type="button"
								onClick={() => handleSelect(lang)}
								className={`w-full text-left px-3 py-2 text-sm hover:bg-gray-100 flex items-center gap-2 ${
									lang === language
										? 'bg-indigo-50 text-indigo-700'
										: 'text-gray-700'
								}`}
							>
								<span className="text-base">{LANGUAGE_FLAGS[lang]}</span>
								<span>{languageNames[lang]}</span>
								{lang === language && (
									<svg
										aria-hidden="true"
										className="w-4 h-4 ml-auto text-indigo-600"
										fill="currentColor"
										viewBox="0 0 20 20"
									>
										<path
											fillRule="evenodd"
											d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
											clipRule="evenodd"
										/>
									</svg>
								)}
							</button>
						))}
					</div>
				</>
			)}
		</div>
	);
}
