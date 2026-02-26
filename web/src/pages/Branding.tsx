import { useEffect, useState } from 'react';
import { useMe } from '../hooks/useAuth';
import {
	useBranding,
	useResetBranding,
	useUpdateBranding,
} from '../hooks/useBranding';
import type { OrgRole } from '../lib/types';

function ColorPreview({ color, label }: { color: string; label: string }) {
	return (
		<div className="flex items-center gap-2">
			<div
				className="w-8 h-8 rounded border border-gray-300 dark:border-gray-600"
				style={{ backgroundColor: color || '#e5e7eb' }}
			/>
			<span className="text-sm text-gray-600 dark:text-gray-400">
				{color || label}
			</span>
		</div>
	);
}

export function Branding() {
	const { data: user } = useMe();
	const { data: branding, isLoading, error } = useBranding();
	const updateBranding = useUpdateBranding();
	const resetBranding = useResetBranding();

	const [logoUrl, setLogoUrl] = useState('');
	const [faviconUrl, setFaviconUrl] = useState('');
	const [productName, setProductName] = useState('');
	const [primaryColor, setPrimaryColor] = useState('');
	const [secondaryColor, setSecondaryColor] = useState('');
	const [supportUrl, setSupportUrl] = useState('');
	const [customCss, setCustomCss] = useState('');
	const [showResetConfirm, setShowResetConfirm] = useState(false);

	const currentUserRole = (user?.current_org_role ?? 'member') as OrgRole;
	const canEdit = currentUserRole === 'owner' || currentUserRole === 'admin';

	useEffect(() => {
		if (branding) {
			setLogoUrl(branding.logo_url);
			setFaviconUrl(branding.favicon_url);
			setProductName(branding.product_name);
			setPrimaryColor(branding.primary_color);
			setSecondaryColor(branding.secondary_color);
			setSupportUrl(branding.support_url);
			setCustomCss(branding.custom_css);
		}
	}, [branding]);

	const handleSave = async (e: React.FormEvent) => {
		e.preventDefault();
		try {
			await updateBranding.mutateAsync({
				logo_url: logoUrl,
				favicon_url: faviconUrl,
				product_name: productName,
				primary_color: primaryColor,
				secondary_color: secondaryColor,
				support_url: supportUrl,
				custom_css: customCss,
			});
		} catch {
			// Error handled by mutation
		}
	};

	const handleReset = async () => {
		try {
			await resetBranding.mutateAsync();
			setShowResetConfirm(false);
		} catch {
			// Error handled by mutation
		}
	};

	if (isLoading) {
		return (
			<div className="flex items-center justify-center h-64">
				<div className="w-8 h-8 border-4 border-indigo-200 border-t-indigo-600 rounded-full animate-spin" />
			</div>
		);
	}

	if (error) {
		const isFeatureGated =
			error instanceof Error &&
			'status' in error &&
			(error as { status: number }).status === 402;
		if (isFeatureGated) {
			return (
				<div className="p-6">
					<h1 className="text-2xl font-bold text-gray-900 dark:text-white mb-4">
						White Label Branding
					</h1>
					<div className="bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg p-6 text-center">
						<svg
							aria-hidden="true"
							className="w-12 h-12 text-yellow-500 mx-auto mb-3"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
						>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"
							/>
						</svg>
						<h2 className="text-lg font-semibold text-yellow-800 dark:text-yellow-300">
							Enterprise Feature
						</h2>
						<p className="text-yellow-700 dark:text-yellow-300 mt-1">
							White label branding is available on the Enterprise plan. Upgrade
							to customize your branding.
						</p>
					</div>
				</div>
			);
		}
		return (
			<div className="p-6">
				<h1 className="text-2xl font-bold text-gray-900 dark:text-white mb-4">
					White Label Branding
				</h1>
				<div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4 text-red-700 dark:text-red-300">
					Failed to load branding settings.
				</div>
			</div>
		);
	}

	return (
		<div className="p-6 max-w-4xl">
			<div className="flex items-center justify-between mb-6">
				<div>
					<h1 className="text-2xl font-bold text-gray-900 dark:text-white">
						White Label Branding
					</h1>
					<p className="text-gray-500 dark:text-gray-400 mt-1">
						Customize the appearance of your Keldris instance.
					</p>
				</div>
				{canEdit && (
					<button
						type="button"
						onClick={() => setShowResetConfirm(true)}
						className="px-4 py-2 text-sm text-red-600 dark:text-red-400 border border-red-300 dark:border-red-700 rounded-lg hover:bg-red-50 dark:hover:bg-red-900/20"
					>
						Reset to Defaults
					</button>
				)}
			</div>

			<form onSubmit={handleSave} className="space-y-8">
				{/* Product Identity */}
				<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
					<h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
						Product Identity
					</h2>
					<div className="space-y-4">
						<div>
							<label
								htmlFor="productName"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								Product Name
							</label>
							<input
								id="productName"
								type="text"
								value={productName}
								onChange={(e) => setProductName(e.target.value)}
								disabled={!canEdit}
								placeholder="Keldris"
								className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 dark:text-white disabled:bg-gray-100 dark:disabled:bg-gray-800"
							/>
							<p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
								Replaces the app name throughout the interface.
							</p>
						</div>
						<div>
							<label
								htmlFor="supportUrl"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								Support URL
							</label>
							<input
								id="supportUrl"
								type="url"
								value={supportUrl}
								onChange={(e) => setSupportUrl(e.target.value)}
								disabled={!canEdit}
								placeholder="https://support.example.com"
								className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 dark:text-white disabled:bg-gray-100 dark:disabled:bg-gray-800"
							/>
						</div>
					</div>
				</div>

				{/* Logo & Favicon */}
				<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
					<h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
						Logo & Favicon
					</h2>
					<div className="space-y-4">
						<div>
							<label
								htmlFor="logoUrl"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								Logo URL
							</label>
							<input
								id="logoUrl"
								type="url"
								value={logoUrl}
								onChange={(e) => setLogoUrl(e.target.value)}
								disabled={!canEdit}
								placeholder="https://example.com/logo.png"
								className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 dark:text-white disabled:bg-gray-100 dark:disabled:bg-gray-800"
							/>
							{logoUrl && (
								<div className="mt-2 p-2 bg-gray-50 dark:bg-gray-900 rounded border dark:border-gray-700">
									<img
										src={logoUrl}
										alt="Logo preview"
										className="h-10 object-contain"
										onError={(e) => {
											(e.target as HTMLImageElement).style.display = 'none';
										}}
									/>
								</div>
							)}
						</div>
						<div>
							<label
								htmlFor="faviconUrl"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								Favicon URL
							</label>
							<input
								id="faviconUrl"
								type="url"
								value={faviconUrl}
								onChange={(e) => setFaviconUrl(e.target.value)}
								disabled={!canEdit}
								placeholder="https://example.com/favicon.ico"
								className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 dark:text-white disabled:bg-gray-100 dark:disabled:bg-gray-800"
							/>
						</div>
					</div>
				</div>

				{/* Colors */}
				<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
					<h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
						Brand Colors
					</h2>
					<div className="grid grid-cols-1 md:grid-cols-2 gap-4">
						<div>
							<label
								htmlFor="primaryColor"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								Primary Color
							</label>
							<div className="flex gap-2">
								<input
									id="primaryColor"
									type="text"
									value={primaryColor}
									onChange={(e) => setPrimaryColor(e.target.value)}
									disabled={!canEdit}
									placeholder="#4F46E5"
									pattern="^#([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$"
									className="flex-1 px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 dark:text-white disabled:bg-gray-100 dark:disabled:bg-gray-800"
								/>
								<input
									type="color"
									value={primaryColor || '#4F46E5'}
									onChange={(e) => setPrimaryColor(e.target.value)}
									disabled={!canEdit}
									className="w-10 h-10 rounded border border-gray-300 dark:border-gray-600 cursor-pointer disabled:cursor-not-allowed"
								/>
							</div>
							<ColorPreview color={primaryColor} label="Default (Indigo)" />
						</div>
						<div>
							<label
								htmlFor="secondaryColor"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								Secondary Color
							</label>
							<div className="flex gap-2">
								<input
									id="secondaryColor"
									type="text"
									value={secondaryColor}
									onChange={(e) => setSecondaryColor(e.target.value)}
									disabled={!canEdit}
									placeholder="#6366F1"
									pattern="^#([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$"
									className="flex-1 px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 dark:text-white disabled:bg-gray-100 dark:disabled:bg-gray-800"
								/>
								<input
									type="color"
									value={secondaryColor || '#6366F1'}
									onChange={(e) => setSecondaryColor(e.target.value)}
									disabled={!canEdit}
									className="w-10 h-10 rounded border border-gray-300 dark:border-gray-600 cursor-pointer disabled:cursor-not-allowed"
								/>
							</div>
							<ColorPreview
								color={secondaryColor}
								label="Default (Indigo Light)"
							/>
						</div>
					</div>
				</div>

				{/* Custom CSS */}
				<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
					<h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
						Custom CSS
					</h2>
					<div>
						<label
							htmlFor="customCss"
							className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
						>
							Additional CSS
						</label>
						<textarea
							id="customCss"
							value={customCss}
							onChange={(e) => setCustomCss(e.target.value)}
							disabled={!canEdit}
							rows={6}
							placeholder=":root { --custom-var: #000; }"
							className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 dark:text-white disabled:bg-gray-100 dark:disabled:bg-gray-800 font-mono text-sm"
						/>
						<p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
							Custom CSS injected into the application. Use CSS variables to
							override colors.
						</p>
					</div>
				</div>

				{/* Save Button */}
				{canEdit && (
					<div className="flex justify-end">
						<button
							type="submit"
							disabled={updateBranding.isPending}
							className="px-6 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 disabled:opacity-50"
						>
							{updateBranding.isPending ? 'Saving...' : 'Save Changes'}
						</button>
					</div>
				)}

				{updateBranding.isSuccess && (
					<div className="bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg p-3 text-green-700 dark:text-green-300 text-sm">
						Branding settings saved successfully.
					</div>
				)}

				{updateBranding.isError && (
					<div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-3 text-red-700 dark:text-red-300 text-sm">
						Failed to save branding settings. Please check your input and try
						again.
					</div>
				)}
			</form>

			{/* Reset Confirmation Modal */}
			{showResetConfirm && (
				<div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
					<div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-sm mx-4">
						<h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-2">
							Reset Branding
						</h3>
						<p className="text-gray-600 dark:text-gray-400 mb-4">
							This will remove all custom branding and revert to the default
							Keldris appearance. This action cannot be undone.
						</p>
						<div className="flex justify-end gap-3">
							<button
								type="button"
								onClick={() => setShowResetConfirm(false)}
								className="px-4 py-2 text-sm text-gray-700 dark:text-gray-300 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700"
							>
								Cancel
							</button>
							<button
								type="button"
								onClick={handleReset}
								disabled={resetBranding.isPending}
								className="px-4 py-2 text-sm text-white bg-red-600 rounded-lg hover:bg-red-700 disabled:opacity-50"
							>
								{resetBranding.isPending ? 'Resetting...' : 'Reset'}
							</button>
						</div>
					</div>
				</div>
			)}
		</div>
	);
}

export default Branding;
