import { createContext, useContext, useEffect, useMemo, type ReactNode } from 'react';
import { useBrandingSettings } from '../hooks/useBranding';
import type { BrandingSettings } from '../lib/types';

interface BrandingContextValue {
	branding: BrandingSettings | null;
	isLoading: boolean;
	productName: string;
	logoUrl: string;
	logoDarkUrl: string;
}

const defaultBranding: BrandingSettings = {
	enabled: false,
	product_name: 'Keldris',
	company_name: '',
	logo_url: '',
	logo_dark_url: '',
	favicon_url: '',
	primary_color: '#4f46e5',
	secondary_color: '#64748b',
	accent_color: '#06b6d4',
	support_url: '',
	support_email: '',
	privacy_url: '',
	terms_url: '',
	footer_text: '',
	login_title: '',
	login_subtitle: '',
	login_bg_url: '',
	hide_powered_by: false,
	custom_css: '',
};

const BrandingContext = createContext<BrandingContextValue>({
	branding: null,
	isLoading: false,
	productName: 'Keldris',
	logoUrl: '',
	logoDarkUrl: '',
});

export function useBranding() {
	return useContext(BrandingContext);
}

// Convert hex color to RGB values
function hexToRgb(hex: string): { r: number; g: number; b: number } | null {
	const result = /^#?([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i.exec(hex);
	return result
		? {
				r: parseInt(result[1], 16),
				g: parseInt(result[2], 16),
				b: parseInt(result[3], 16),
			}
		: null;
}

export function BrandingProvider({ children }: { children: ReactNode }) {
	const { data: branding, isLoading } = useBrandingSettings();

	// Memoize the effective branding settings
	const effectiveBranding = useMemo(() => {
		if (!branding?.enabled) {
			return defaultBranding;
		}
		return branding;
	}, [branding]);

	// Apply CSS custom properties for colors
	useEffect(() => {
		if (!effectiveBranding.enabled) {
			// Reset to default styles
			document.documentElement.style.removeProperty('--brand-primary');
			document.documentElement.style.removeProperty('--brand-primary-rgb');
			document.documentElement.style.removeProperty('--brand-secondary');
			document.documentElement.style.removeProperty('--brand-secondary-rgb');
			document.documentElement.style.removeProperty('--brand-accent');
			document.documentElement.style.removeProperty('--brand-accent-rgb');
			return;
		}

		// Apply primary color
		if (effectiveBranding.primary_color) {
			document.documentElement.style.setProperty(
				'--brand-primary',
				effectiveBranding.primary_color
			);
			const rgb = hexToRgb(effectiveBranding.primary_color);
			if (rgb) {
				document.documentElement.style.setProperty(
					'--brand-primary-rgb',
					`${rgb.r}, ${rgb.g}, ${rgb.b}`
				);
			}
		}

		// Apply secondary color
		if (effectiveBranding.secondary_color) {
			document.documentElement.style.setProperty(
				'--brand-secondary',
				effectiveBranding.secondary_color
			);
			const rgb = hexToRgb(effectiveBranding.secondary_color);
			if (rgb) {
				document.documentElement.style.setProperty(
					'--brand-secondary-rgb',
					`${rgb.r}, ${rgb.g}, ${rgb.b}`
				);
			}
		}

		// Apply accent color
		if (effectiveBranding.accent_color) {
			document.documentElement.style.setProperty(
				'--brand-accent',
				effectiveBranding.accent_color
			);
			const rgb = hexToRgb(effectiveBranding.accent_color);
			if (rgb) {
				document.documentElement.style.setProperty(
					'--brand-accent-rgb',
					`${rgb.r}, ${rgb.g}, ${rgb.b}`
				);
			}
		}
	}, [effectiveBranding]);

	// Update document title
	useEffect(() => {
		if (effectiveBranding.enabled && effectiveBranding.product_name) {
			document.title = effectiveBranding.product_name;
		} else {
			document.title = 'Keldris';
		}
	}, [effectiveBranding.enabled, effectiveBranding.product_name]);

	// Update favicon
	useEffect(() => {
		if (effectiveBranding.enabled && effectiveBranding.favicon_url) {
			const link: HTMLLinkElement =
				document.querySelector("link[rel*='icon']") ||
				document.createElement('link');
			link.type = 'image/x-icon';
			link.rel = 'shortcut icon';
			link.href = effectiveBranding.favicon_url;
			document.getElementsByTagName('head')[0].appendChild(link);
		}
	}, [effectiveBranding.enabled, effectiveBranding.favicon_url]);

	// Apply custom CSS
	useEffect(() => {
		const styleId = 'custom-branding-css';
		let styleElement = document.getElementById(styleId) as HTMLStyleElement;

		if (effectiveBranding.enabled && effectiveBranding.custom_css) {
			if (!styleElement) {
				styleElement = document.createElement('style');
				styleElement.id = styleId;
				document.head.appendChild(styleElement);
			}
			styleElement.textContent = effectiveBranding.custom_css;
		} else if (styleElement) {
			styleElement.remove();
		}

		return () => {
			if (styleElement) {
				styleElement.remove();
			}
		};
	}, [effectiveBranding.enabled, effectiveBranding.custom_css]);

	const value = useMemo(
		() => ({
			branding: effectiveBranding,
			isLoading,
			productName: effectiveBranding.product_name || 'Keldris',
			logoUrl: effectiveBranding.logo_url || '',
			logoDarkUrl: effectiveBranding.logo_dark_url || '',
		}),
		[effectiveBranding, isLoading]
	);

	return (
		<BrandingContext.Provider value={value}>
			{children}
		</BrandingContext.Provider>
	);
}
