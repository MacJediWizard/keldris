import { useEffect, useState } from 'react';
import { useMe } from '../hooks/useAuth';
import {
	useBrandingSettings,
	useUpdateBrandingSettings,
} from '../hooks/useBranding';
import type {
	BrandingSettings as BrandingSettingsType,
	OrgRole,
} from '../lib/types';

export function BrandingSettings() {
	const { data: user } = useMe();
	const currentUserRole = (user?.current_org_role ?? 'member') as OrgRole;
	const canEdit = currentUserRole === 'owner' || currentUserRole === 'admin';

	const { data: branding, isLoading, isError, error } = useBrandingSettings();
	const updateBranding = useUpdateBrandingSettings();

	const [isEditing, setIsEditing] = useState(false);
	const [form, setForm] = useState<BrandingSettingsType>({
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
	});

	useEffect(() => {
		if (branding) {
			setForm(branding);
		}
	}, [branding]);

	const handleSave = async () => {
		try {
			await updateBranding.mutateAsync(form);
			setIsEditing(false);
		} catch {
			// Error handled by mutation
		}
	};

	const handleReset = () => {
		if (branding) {
			setForm(branding);
		}
		setIsEditing(false);
	};

	if (isLoading) {
		return (
			<div className="space-y-6">
				<div>
					<div className="h-8 w-48 bg-gray-200 dark:bg-gray-700 rounded animate-pulse" />
					<div className="h-4 w-64 bg-gray-200 dark:bg-gray-700 rounded animate-pulse mt-2" />
				</div>
				<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
					<div className="space-y-4">
						{[1, 2, 3, 4, 5].map((i) => (
							<div
								key={i}
								className="h-12 w-full bg-gray-200 dark:bg-gray-700 rounded animate-pulse"
							/>
						))}
					</div>
				</div>
			</div>
		);
	}

	if (isError) {
		const errorMessage =
			(error as Error)?.message || 'Failed to load branding settings';
		const isEnterpriseError = errorMessage.toLowerCase().includes('enterprise');

		return (
			<div className="space-y-6">
				<div>
					<h1 className="text-2xl font-bold text-gray-900 dark:text-white">
						Branding
					</h1>
					<p className="text-gray-600 dark:text-gray-400 mt-1">
						Customize your organization's branding
					</p>
				</div>
				<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-8 text-center">
					<div className="p-3 bg-purple-100 dark:bg-purple-900/30 rounded-full inline-block mb-4">
						<svg
							aria-hidden="true"
							className="w-8 h-8 text-purple-600 dark:text-purple-400"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
						>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M5 3v4M3 5h4M6 17v4m-2-2h4m5-16l2.286 6.857L21 12l-5.714 2.143L13 21l-2.286-6.857L5 12l5.714-2.143L13 3z"
							/>
						</svg>
					</div>
					<h2 className="text-xl font-semibold text-gray-900 dark:text-white mb-2">
						{isEnterpriseError
							? 'Enterprise Feature'
							: 'Unable to Load Branding'}
					</h2>
					<p className="text-gray-600 dark:text-gray-400 max-w-md mx-auto">
						{isEnterpriseError
							? 'White-label branding is available for Enterprise organizations. Contact your administrator to enable this feature.'
							: errorMessage}
					</p>
				</div>
			</div>
		);
	}

	if (!canEdit) {
		return (
			<div className="text-center py-12">
				<div className="p-3 bg-red-100 dark:bg-red-900/30 rounded-full inline-block mb-4">
					<svg
						aria-hidden="true"
						className="w-8 h-8 text-red-600 dark:text-red-400"
						fill="none"
						stroke="currentColor"
						viewBox="0 0 24 24"
					>
						<path
							strokeLinecap="round"
							strokeLinejoin="round"
							strokeWidth={2}
							d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
						/>
					</svg>
				</div>
				<h2 className="text-xl font-semibold text-gray-900 dark:text-white mb-2">
					Access Restricted
				</h2>
				<p className="text-gray-600 dark:text-gray-400">
					Branding settings require admin or owner access.
				</p>
			</div>
		);
	}

	return (
		<div className="space-y-6">
			<div className="flex items-center justify-between">
				<div>
					<h1 className="text-2xl font-bold text-gray-900 dark:text-white">
						Branding
					</h1>
					<p className="text-gray-600 dark:text-gray-400 mt-1">
						Customize your organization's branding and appearance
					</p>
				</div>
				{!isEditing && (
					<button
						type="button"
						onClick={() => setIsEditing(true)}
						className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors"
					>
						Edit Branding
					</button>
				)}
			</div>

			{/* Enable/Disable Toggle */}
			<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
				<div className="flex items-center justify-between">
					<div>
						<h2 className="text-lg font-semibold text-gray-900 dark:text-white">
							Enable Custom Branding
						</h2>
						<p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
							When enabled, your custom branding will be applied throughout the
							application
						</p>
					</div>
					<label className="relative inline-flex items-center cursor-pointer">
						<input
							type="checkbox"
							checked={form.enabled}
							disabled={!isEditing}
							onChange={(e) => setForm({ ...form, enabled: e.target.checked })}
							className="sr-only peer"
						/>
						<div className="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-indigo-300 dark:peer-focus:ring-indigo-800 rounded-full peer dark:bg-gray-700 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all dark:border-gray-600 peer-checked:bg-indigo-600 peer-disabled:opacity-50" />
					</label>
				</div>
			</div>

			{/* Product Identity */}
			<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
				<div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
					<h2 className="text-lg font-semibold text-gray-900 dark:text-white">
						Product Identity
					</h2>
					<p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
						Customize the product name and company information
					</p>
				</div>
				<div className="p-6 space-y-4">
					<div className="grid grid-cols-2 gap-4">
						<div>
							<label
								htmlFor="product-name"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								Product Name
							</label>
							<input
								type="text"
								id="product-name"
								value={form.product_name}
								disabled={!isEditing}
								onChange={(e) =>
									setForm({ ...form, product_name: e.target.value })
								}
								placeholder="Keldris"
								className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-white disabled:opacity-50 disabled:bg-gray-100 dark:disabled:bg-gray-800"
							/>
						</div>
						<div>
							<label
								htmlFor="company-name"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								Company Name
							</label>
							<input
								type="text"
								id="company-name"
								value={form.company_name}
								disabled={!isEditing}
								onChange={(e) =>
									setForm({ ...form, company_name: e.target.value })
								}
								placeholder="Your Company"
								className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-white disabled:opacity-50 disabled:bg-gray-100 dark:disabled:bg-gray-800"
							/>
						</div>
					</div>

					<div>
						<label
							htmlFor="footer-text"
							className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
						>
							Footer Text
						</label>
						<textarea
							id="footer-text"
							value={form.footer_text}
							disabled={!isEditing}
							onChange={(e) =>
								setForm({ ...form, footer_text: e.target.value })
							}
							placeholder="Custom footer text..."
							rows={2}
							className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-white disabled:opacity-50 disabled:bg-gray-100 dark:disabled:bg-gray-800"
						/>
					</div>

					<div className="flex items-center gap-2">
						<input
							type="checkbox"
							id="hide-powered-by"
							checked={form.hide_powered_by}
							disabled={!isEditing}
							onChange={(e) =>
								setForm({ ...form, hide_powered_by: e.target.checked })
							}
							className="h-4 w-4 text-indigo-600 focus:ring-indigo-500 border-gray-300 rounded disabled:opacity-50"
						/>
						<label
							htmlFor="hide-powered-by"
							className="text-sm text-gray-700 dark:text-gray-300"
						>
							Hide "Powered by Keldris" text
						</label>
					</div>
				</div>
			</div>

			{/* Logos and Icons */}
			<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
				<div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
					<h2 className="text-lg font-semibold text-gray-900 dark:text-white">
						Logos and Icons
					</h2>
					<p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
						Customize the logo and favicon displayed in the application
					</p>
				</div>
				<div className="p-6 space-y-4">
					<div className="grid grid-cols-2 gap-4">
						<div>
							<label
								htmlFor="logo-url"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								Logo URL (Light Mode)
							</label>
							<input
								type="url"
								id="logo-url"
								value={form.logo_url}
								disabled={!isEditing}
								onChange={(e) => setForm({ ...form, logo_url: e.target.value })}
								placeholder="https://example.com/logo.png"
								className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-white disabled:opacity-50 disabled:bg-gray-100 dark:disabled:bg-gray-800"
							/>
							{form.logo_url && (
								<div className="mt-2 p-2 bg-gray-100 dark:bg-gray-700 rounded">
									<img
										src={form.logo_url}
										alt="Logo preview"
										className="h-8 object-contain"
										onError={(e) => {
											(e.target as HTMLImageElement).style.display = 'none';
										}}
									/>
								</div>
							)}
						</div>
						<div>
							<label
								htmlFor="logo-dark-url"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								Logo URL (Dark Mode)
							</label>
							<input
								type="url"
								id="logo-dark-url"
								value={form.logo_dark_url}
								disabled={!isEditing}
								onChange={(e) =>
									setForm({ ...form, logo_dark_url: e.target.value })
								}
								placeholder="https://example.com/logo-dark.png"
								className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-white disabled:opacity-50 disabled:bg-gray-100 dark:disabled:bg-gray-800"
							/>
							{form.logo_dark_url && (
								<div className="mt-2 p-2 bg-gray-800 rounded">
									<img
										src={form.logo_dark_url}
										alt="Dark logo preview"
										className="h-8 object-contain"
										onError={(e) => {
											(e.target as HTMLImageElement).style.display = 'none';
										}}
									/>
								</div>
							)}
						</div>
					</div>

					<div>
						<label
							htmlFor="favicon-url"
							className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
						>
							Favicon URL
						</label>
						<input
							type="url"
							id="favicon-url"
							value={form.favicon_url}
							disabled={!isEditing}
							onChange={(e) =>
								setForm({ ...form, favicon_url: e.target.value })
							}
							placeholder="https://example.com/favicon.ico"
							className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-white disabled:opacity-50 disabled:bg-gray-100 dark:disabled:bg-gray-800"
						/>
					</div>
				</div>
			</div>

			{/* Brand Colors */}
			<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
				<div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
					<h2 className="text-lg font-semibold text-gray-900 dark:text-white">
						Brand Colors
					</h2>
					<p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
						Customize the color scheme used throughout the application
					</p>
				</div>
				<div className="p-6">
					<div className="grid grid-cols-3 gap-4">
						<div>
							<label
								htmlFor="primary-color"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								Primary Color
							</label>
							<div className="flex gap-2">
								<input
									type="color"
									id="primary-color"
									value={form.primary_color}
									disabled={!isEditing}
									onChange={(e) =>
										setForm({ ...form, primary_color: e.target.value })
									}
									className="h-10 w-14 p-1 border border-gray-300 dark:border-gray-600 rounded cursor-pointer disabled:cursor-not-allowed"
								/>
								<input
									type="text"
									value={form.primary_color}
									disabled={!isEditing}
									onChange={(e) =>
										setForm({ ...form, primary_color: e.target.value })
									}
									placeholder="#4f46e5"
									className="flex-1 px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-white disabled:opacity-50 disabled:bg-gray-100 dark:disabled:bg-gray-800 font-mono text-sm"
								/>
							</div>
						</div>
						<div>
							<label
								htmlFor="secondary-color"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								Secondary Color
							</label>
							<div className="flex gap-2">
								<input
									type="color"
									id="secondary-color"
									value={form.secondary_color}
									disabled={!isEditing}
									onChange={(e) =>
										setForm({ ...form, secondary_color: e.target.value })
									}
									className="h-10 w-14 p-1 border border-gray-300 dark:border-gray-600 rounded cursor-pointer disabled:cursor-not-allowed"
								/>
								<input
									type="text"
									value={form.secondary_color}
									disabled={!isEditing}
									onChange={(e) =>
										setForm({ ...form, secondary_color: e.target.value })
									}
									placeholder="#64748b"
									className="flex-1 px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-white disabled:opacity-50 disabled:bg-gray-100 dark:disabled:bg-gray-800 font-mono text-sm"
								/>
							</div>
						</div>
						<div>
							<label
								htmlFor="accent-color"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								Accent Color
							</label>
							<div className="flex gap-2">
								<input
									type="color"
									id="accent-color"
									value={form.accent_color}
									disabled={!isEditing}
									onChange={(e) =>
										setForm({ ...form, accent_color: e.target.value })
									}
									className="h-10 w-14 p-1 border border-gray-300 dark:border-gray-600 rounded cursor-pointer disabled:cursor-not-allowed"
								/>
								<input
									type="text"
									value={form.accent_color}
									disabled={!isEditing}
									onChange={(e) =>
										setForm({ ...form, accent_color: e.target.value })
									}
									placeholder="#06b6d4"
									className="flex-1 px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-white disabled:opacity-50 disabled:bg-gray-100 dark:disabled:bg-gray-800 font-mono text-sm"
								/>
							</div>
						</div>
					</div>

					{/* Color Preview */}
					<div className="mt-6 p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg">
						<p className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">
							Color Preview
						</p>
						<div className="flex gap-4">
							<button
								type="button"
								style={{ backgroundColor: form.primary_color }}
								className="px-4 py-2 text-white rounded-lg text-sm"
							>
								Primary Button
							</button>
							<button
								type="button"
								style={{ backgroundColor: form.secondary_color }}
								className="px-4 py-2 text-white rounded-lg text-sm"
							>
								Secondary Button
							</button>
							<button
								type="button"
								style={{ backgroundColor: form.accent_color }}
								className="px-4 py-2 text-white rounded-lg text-sm"
							>
								Accent Button
							</button>
						</div>
					</div>
				</div>
			</div>

			{/* Login Page */}
			<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
				<div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
					<h2 className="text-lg font-semibold text-gray-900 dark:text-white">
						Login Page Customization
					</h2>
					<p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
						Customize the appearance of the login page
					</p>
				</div>
				<div className="p-6 space-y-4">
					<div className="grid grid-cols-2 gap-4">
						<div>
							<label
								htmlFor="login-title"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								Login Page Title
							</label>
							<input
								type="text"
								id="login-title"
								value={form.login_title}
								disabled={!isEditing}
								onChange={(e) =>
									setForm({ ...form, login_title: e.target.value })
								}
								placeholder="Welcome Back"
								className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-white disabled:opacity-50 disabled:bg-gray-100 dark:disabled:bg-gray-800"
							/>
						</div>
						<div>
							<label
								htmlFor="login-subtitle"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								Login Page Subtitle
							</label>
							<input
								type="text"
								id="login-subtitle"
								value={form.login_subtitle}
								disabled={!isEditing}
								onChange={(e) =>
									setForm({ ...form, login_subtitle: e.target.value })
								}
								placeholder="Sign in to your account"
								className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-white disabled:opacity-50 disabled:bg-gray-100 dark:disabled:bg-gray-800"
							/>
						</div>
					</div>

					<div>
						<label
							htmlFor="login-bg-url"
							className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
						>
							Login Background Image URL
						</label>
						<input
							type="url"
							id="login-bg-url"
							value={form.login_bg_url}
							disabled={!isEditing}
							onChange={(e) =>
								setForm({ ...form, login_bg_url: e.target.value })
							}
							placeholder="https://example.com/background.jpg"
							className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-white disabled:opacity-50 disabled:bg-gray-100 dark:disabled:bg-gray-800"
						/>
					</div>
				</div>
			</div>

			{/* Support Links */}
			<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
				<div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
					<h2 className="text-lg font-semibold text-gray-900 dark:text-white">
						Support and Legal
					</h2>
					<p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
						Configure support contact information and legal page links
					</p>
				</div>
				<div className="p-6 space-y-4">
					<div className="grid grid-cols-2 gap-4">
						<div>
							<label
								htmlFor="support-url"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								Support URL
							</label>
							<input
								type="url"
								id="support-url"
								value={form.support_url}
								disabled={!isEditing}
								onChange={(e) =>
									setForm({ ...form, support_url: e.target.value })
								}
								placeholder="https://support.example.com"
								className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-white disabled:opacity-50 disabled:bg-gray-100 dark:disabled:bg-gray-800"
							/>
						</div>
						<div>
							<label
								htmlFor="support-email"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								Support Email
							</label>
							<input
								type="email"
								id="support-email"
								value={form.support_email}
								disabled={!isEditing}
								onChange={(e) =>
									setForm({ ...form, support_email: e.target.value })
								}
								placeholder="support@example.com"
								className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-white disabled:opacity-50 disabled:bg-gray-100 dark:disabled:bg-gray-800"
							/>
						</div>
					</div>

					<div className="grid grid-cols-2 gap-4">
						<div>
							<label
								htmlFor="privacy-url"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								Privacy Policy URL
							</label>
							<input
								type="url"
								id="privacy-url"
								value={form.privacy_url}
								disabled={!isEditing}
								onChange={(e) =>
									setForm({ ...form, privacy_url: e.target.value })
								}
								placeholder="https://example.com/privacy"
								className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-white disabled:opacity-50 disabled:bg-gray-100 dark:disabled:bg-gray-800"
							/>
						</div>
						<div>
							<label
								htmlFor="terms-url"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								Terms of Service URL
							</label>
							<input
								type="url"
								id="terms-url"
								value={form.terms_url}
								disabled={!isEditing}
								onChange={(e) =>
									setForm({ ...form, terms_url: e.target.value })
								}
								placeholder="https://example.com/terms"
								className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-white disabled:opacity-50 disabled:bg-gray-100 dark:disabled:bg-gray-800"
							/>
						</div>
					</div>
				</div>
			</div>

			{/* Custom CSS */}
			<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
				<div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
					<h2 className="text-lg font-semibold text-gray-900 dark:text-white">
						Custom CSS
					</h2>
					<p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
						Add custom CSS to further customize the appearance (advanced)
					</p>
				</div>
				<div className="p-6">
					<textarea
						value={form.custom_css}
						disabled={!isEditing}
						onChange={(e) => setForm({ ...form, custom_css: e.target.value })}
						placeholder=".sidebar { background: #1a1a2e; }"
						rows={6}
						className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-white disabled:opacity-50 disabled:bg-gray-100 dark:disabled:bg-gray-800 font-mono text-sm"
					/>
					<p className="text-xs text-gray-500 dark:text-gray-400 mt-2">
						Maximum 10,000 characters. CSS is applied globally and should be
						used with caution.
					</p>
				</div>
			</div>

			{/* Save/Cancel Buttons */}
			{isEditing && (
				<div className="flex justify-end gap-3">
					<button
						type="button"
						onClick={handleReset}
						className="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
					>
						Cancel
					</button>
					<button
						type="button"
						onClick={handleSave}
						disabled={updateBranding.isPending}
						className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
					>
						{updateBranding.isPending ? 'Saving...' : 'Save Changes'}
					</button>
				</div>
			)}
		</div>
	);
}
