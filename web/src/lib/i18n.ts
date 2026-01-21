import i18n from 'i18next';
import LanguageDetector from 'i18next-browser-languagedetector';
import { initReactI18next } from 'react-i18next';

import en from '../locales/en.json';
import es from '../locales/es.json';
import pt from '../locales/pt.json';

export const SUPPORTED_LANGUAGES = ['en', 'es', 'pt'] as const;
export type SupportedLanguage = (typeof SUPPORTED_LANGUAGES)[number];

export const LANGUAGE_NAMES: Record<SupportedLanguage, string> = {
	en: 'English',
	es: 'Español',
	pt: 'Português',
};

export const LANGUAGE_LOCALES: Record<SupportedLanguage, string> = {
	en: 'en-US',
	es: 'es-ES',
	pt: 'pt-BR',
};

const resources = {
	en: { translation: en },
	es: { translation: es },
	pt: { translation: pt },
};

i18n
	.use(LanguageDetector)
	.use(initReactI18next)
	.init({
		resources,
		fallbackLng: 'en',
		supportedLngs: SUPPORTED_LANGUAGES,
		interpolation: {
			escapeValue: false,
		},
		detection: {
			order: ['localStorage', 'navigator'],
			caches: ['localStorage'],
			lookupLocalStorage: 'keldris-language',
		},
	});

export default i18n;
