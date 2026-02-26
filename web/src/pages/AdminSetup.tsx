import { useState } from 'react';
import {
	useRerunConfigureOIDC,
	useRerunConfigureSMTP,
	useRerunStatus,
	useRerunUpdateLicense,
} from '../hooks/useSetup';
import type { OIDCSettings, SMTPSettings } from '../lib/types';

type ConfigSection = 'smtp' | 'oidc' | 'license' | null;

export function AdminSetup() {
	const { data: status, isLoading } = useRerunStatus();
	const [activeSection, setActiveSection] = useState<ConfigSection>(null);

	if (isLoading) {
		return (
			<div className="flex items-center justify-center py-12">
				<div className="w-8 h-8 border-4 border-indigo-200 border-t-indigo-600 rounded-full animate-spin" />
			</div>
		);
	}

	return (
		<div>
			{/* Header */}
			<div className="mb-8">
				<h1 className="text-2xl font-bold text-gray-900 dark:text-white">
					Server Setup
				</h1>
				<p className="text-gray-600 dark:text-gray-400 mt-1">
					Reconfigure server settings. These changes apply to the entire server.
				</p>
			</div>

			{/* License Status */}
			{status?.license && (
				<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6 mb-6">
					<h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
						Current License
					</h2>
					<div className="grid grid-cols-2 md:grid-cols-4 gap-4">
						<div>
							<dt className="text-sm text-gray-500 dark:text-gray-400">Type</dt>
							<dd className="text-sm font-medium text-gray-900 dark:text-white capitalize">
								{status.license.license_type}
							</dd>
						</div>
						<div>
							<dt className="text-sm text-gray-500 dark:text-gray-400">
								Status
							</dt>
							<dd
								className={`text-sm font-medium capitalize ${
									status.license.status === 'active'
										? 'text-green-600'
										: 'text-red-600'
								}`}
							>
								{status.license.status}
							</dd>
						</div>
						{status.license.expires_at && (
							<div>
								<dt className="text-sm text-gray-500 dark:text-gray-400">
									Expires
								</dt>
								<dd className="text-sm font-medium text-gray-900 dark:text-white">
									{new Date(status.license.expires_at).toLocaleDateString()}
								</dd>
							</div>
						)}
						{status.license.company_name && (
							<div>
								<dt className="text-sm text-gray-500 dark:text-gray-400">
									Company
								</dt>
								<dd className="text-sm font-medium text-gray-900 dark:text-white">
									{status.license.company_name}
								</dd>
							</div>
						)}
					</div>
				</div>
			)}

			{/* Configuration Sections */}
			<div className="space-y-4">
				<ConfigCard
					title="Email (SMTP)"
					description="Configure SMTP settings for email notifications"
					isOpen={activeSection === 'smtp'}
					onToggle={() =>
						setActiveSection(activeSection === 'smtp' ? null : 'smtp')
					}
				>
					<SMTPConfigForm onSuccess={() => setActiveSection(null)} />
				</ConfigCard>

				<ConfigCard
					title="Single Sign-On (OIDC)"
					description="Configure OpenID Connect for SSO authentication"
					isOpen={activeSection === 'oidc'}
					onToggle={() =>
						setActiveSection(activeSection === 'oidc' ? null : 'oidc')
					}
				>
					<OIDCConfigForm onSuccess={() => setActiveSection(null)} />
				</ConfigCard>

				<ConfigCard
					title="License"
					description="Update your license key"
					isOpen={activeSection === 'license'}
					onToggle={() =>
						setActiveSection(activeSection === 'license' ? null : 'license')
					}
				>
					<LicenseConfigForm onSuccess={() => setActiveSection(null)} />
				</ConfigCard>
			</div>
		</div>
	);
}

interface ConfigCardProps {
	title: string;
	description: string;
	isOpen: boolean;
	onToggle: () => void;
	children: React.ReactNode;
}

function ConfigCard({
	title,
	description,
	isOpen,
	onToggle,
	children,
}: ConfigCardProps) {
	return (
		<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 overflow-hidden">
			<button
				type="button"
				onClick={onToggle}
				className="w-full px-6 py-4 flex items-center justify-between text-left hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
			>
				<div>
					<h3 className="text-lg font-medium text-gray-900 dark:text-white">
						{title}
					</h3>
					<p className="text-sm text-gray-500 dark:text-gray-400">
						{description}
					</p>
				</div>
				<svg
					aria-hidden="true"
					className={`w-5 h-5 text-gray-400 dark:text-gray-500 transition-transform ${isOpen ? 'rotate-180' : ''}`}
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
			{isOpen && (
				<div className="px-6 pb-6 border-t border-gray-200 dark:border-gray-700 pt-4">
					{children}
				</div>
			)}
		</div>
	);
}

function SMTPConfigForm({ onSuccess }: { onSuccess: () => void }) {
	const configureSMTP = useRerunConfigureSMTP();
	const [settings, setSettings] = useState<SMTPSettings>({
		host: '',
		port: 587,
		username: '',
		password: '',
		from_email: '',
		from_name: '',
		encryption: 'starttls',
		enabled: true,
		skip_tls_verify: false,
		connection_timeout_seconds: 30,
	});

	const handleSubmit = (e: React.FormEvent) => {
		e.preventDefault();
		configureSMTP.mutate(settings, { onSuccess });
	};

	return (
		<form onSubmit={handleSubmit} className="space-y-4">
			<div className="grid grid-cols-2 gap-4">
				<div>
					<label
						htmlFor="admin-smtp-host"
						className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
					>
						SMTP Host
					</label>
					<input
						id="admin-smtp-host"
						type="text"
						value={settings.host}
						onChange={(e) => setSettings({ ...settings, host: e.target.value })}
						className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 dark:bg-gray-800 dark:text-white dark:placeholder-gray-500"
						placeholder="smtp.example.com"
					/>
				</div>
				<div>
					<label
						htmlFor="admin-smtp-port"
						className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
					>
						Port
					</label>
					<input
						id="admin-smtp-port"
						type="number"
						value={settings.port}
						onChange={(e) =>
							setSettings({
								...settings,
								port: Number.parseInt(e.target.value, 10),
							})
						}
						className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 dark:bg-gray-800 dark:text-white dark:placeholder-gray-500"
					/>
				</div>
			</div>

			<div className="grid grid-cols-2 gap-4">
				<div>
					<label
						htmlFor="admin-smtp-username"
						className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
					>
						Username
					</label>
					<input
						id="admin-smtp-username"
						type="text"
						value={settings.username}
						onChange={(e) =>
							setSettings({ ...settings, username: e.target.value })
						}
						className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 dark:bg-gray-800 dark:text-white dark:placeholder-gray-500"
					/>
				</div>
				<div>
					<label
						htmlFor="admin-smtp-password"
						className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
					>
						Password
					</label>
					<input
						id="admin-smtp-password"
						type="password"
						value={settings.password}
						onChange={(e) =>
							setSettings({ ...settings, password: e.target.value })
						}
						className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 dark:bg-gray-800 dark:text-white dark:placeholder-gray-500"
					/>
				</div>
			</div>

			<div className="grid grid-cols-2 gap-4">
				<div>
					<label
						htmlFor="admin-smtp-from-email"
						className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
					>
						From Email
					</label>
					<input
						id="admin-smtp-from-email"
						type="email"
						value={settings.from_email}
						onChange={(e) =>
							setSettings({ ...settings, from_email: e.target.value })
						}
						className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 dark:bg-gray-800 dark:text-white dark:placeholder-gray-500"
						placeholder="noreply@example.com"
					/>
				</div>
				<div>
					<label
						htmlFor="admin-smtp-from-name"
						className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
					>
						From Name
					</label>
					<input
						id="admin-smtp-from-name"
						type="text"
						value={settings.from_name}
						onChange={(e) =>
							setSettings({ ...settings, from_name: e.target.value })
						}
						className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 dark:bg-gray-800 dark:text-white dark:placeholder-gray-500"
						placeholder="Keldris Backups"
					/>
				</div>
			</div>

			<div>
				<label
					htmlFor="admin-smtp-encryption"
					className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
				>
					Encryption
				</label>
				<select
					id="admin-smtp-encryption"
					value={settings.encryption}
					onChange={(e) =>
						setSettings({
							...settings,
							encryption: e.target.value as 'none' | 'tls' | 'starttls',
						})
					}
					className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 dark:bg-gray-800 dark:text-white dark:placeholder-gray-500"
				>
					<option value="starttls">STARTTLS</option>
					<option value="tls">TLS</option>
					<option value="none">None</option>
				</select>
			</div>

			{configureSMTP.isError && (
				<div className="p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg text-sm text-red-700 dark:text-red-400">
					{configureSMTP.error instanceof Error
						? configureSMTP.error.message
						: 'Failed to configure SMTP'}
				</div>
			)}

			{configureSMTP.isSuccess && (
				<div className="p-3 bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg text-sm text-green-700 dark:text-green-400">
					SMTP settings saved successfully
				</div>
			)}

			<div className="flex justify-end">
				<button
					type="submit"
					disabled={configureSMTP.isPending}
					className="px-6 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
				>
					{configureSMTP.isPending ? 'Saving...' : 'Save SMTP Settings'}
				</button>
			</div>
		</form>
	);
}

function OIDCConfigForm({ onSuccess }: { onSuccess: () => void }) {
	const configureOIDC = useRerunConfigureOIDC();
	const [settings, setSettings] = useState<OIDCSettings>({
		enabled: true,
		issuer: '',
		client_id: '',
		client_secret: '',
		redirect_url: '',
		scopes: ['openid', 'profile', 'email'],
		auto_create_users: false,
		default_role: 'member',
		allowed_domains: [],
		require_email_verification: true,
	});

	const handleSubmit = (e: React.FormEvent) => {
		e.preventDefault();
		configureOIDC.mutate(settings, { onSuccess });
	};

	return (
		<form onSubmit={handleSubmit} className="space-y-4">
			<div>
				<label
					htmlFor="admin-oidc-issuer"
					className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
				>
					Issuer URL
				</label>
				<input
					id="admin-oidc-issuer"
					type="url"
					value={settings.issuer}
					onChange={(e) => setSettings({ ...settings, issuer: e.target.value })}
					className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 dark:bg-gray-800 dark:text-white dark:placeholder-gray-500"
					placeholder="https://auth.example.com"
				/>
			</div>

			<div className="grid grid-cols-2 gap-4">
				<div>
					<label
						htmlFor="admin-oidc-client-id"
						className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
					>
						Client ID
					</label>
					<input
						id="admin-oidc-client-id"
						type="text"
						value={settings.client_id}
						onChange={(e) =>
							setSettings({ ...settings, client_id: e.target.value })
						}
						className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 dark:bg-gray-800 dark:text-white dark:placeholder-gray-500"
					/>
				</div>
				<div>
					<label
						htmlFor="admin-oidc-client-secret"
						className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
					>
						Client Secret
					</label>
					<input
						id="admin-oidc-client-secret"
						type="password"
						value={settings.client_secret}
						onChange={(e) =>
							setSettings({ ...settings, client_secret: e.target.value })
						}
						className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 dark:bg-gray-800 dark:text-white dark:placeholder-gray-500"
					/>
				</div>
			</div>

			<div>
				<label
					htmlFor="admin-oidc-redirect"
					className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
				>
					Redirect URL
				</label>
				<input
					id="admin-oidc-redirect"
					type="url"
					value={settings.redirect_url}
					onChange={(e) =>
						setSettings({ ...settings, redirect_url: e.target.value })
					}
					className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 dark:bg-gray-800 dark:text-white dark:placeholder-gray-500"
					placeholder="https://your-domain.com/auth/callback"
				/>
			</div>

			<div className="flex items-center gap-2">
				<input
					id="admin-oidc-auto-create"
					type="checkbox"
					checked={settings.auto_create_users}
					onChange={(e) =>
						setSettings({ ...settings, auto_create_users: e.target.checked })
					}
					className="w-4 h-4 text-indigo-600 border-gray-300 dark:border-gray-600 rounded focus:ring-indigo-500 dark:bg-gray-800"
				/>
				<label
					htmlFor="admin-oidc-auto-create"
					className="text-sm text-gray-700 dark:text-gray-300"
				>
					Auto-create users on first login
				</label>
			</div>

			{configureOIDC.isError && (
				<div className="p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg text-sm text-red-700 dark:text-red-400">
					{configureOIDC.error instanceof Error
						? configureOIDC.error.message
						: 'Failed to configure OIDC'}
				</div>
			)}

			{configureOIDC.isSuccess && (
				<div className="p-3 bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg text-sm text-green-700 dark:text-green-400">
					OIDC settings saved successfully
				</div>
			)}

			<div className="flex justify-end">
				<button
					type="submit"
					disabled={configureOIDC.isPending}
					className="px-6 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
				>
					{configureOIDC.isPending ? 'Saving...' : 'Save OIDC Settings'}
				</button>
			</div>
		</form>
	);
}

function LicenseConfigForm({ onSuccess }: { onSuccess: () => void }) {
	const updateLicense = useRerunUpdateLicense();
	const [licenseKey, setLicenseKey] = useState('');

	const handleSubmit = (e: React.FormEvent) => {
		e.preventDefault();
		updateLicense.mutate({ license_key: licenseKey }, { onSuccess });
	};

	return (
		<form onSubmit={handleSubmit} className="space-y-4">
			<div>
				<label
					htmlFor="admin-license-key"
					className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
				>
					License Key
				</label>
				<input
					id="admin-license-key"
					type="text"
					value={licenseKey}
					onChange={(e) => setLicenseKey(e.target.value)}
					required
					className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 dark:bg-gray-800 dark:text-white dark:placeholder-gray-500 font-mono"
					placeholder="XXXX-XXXX-XXXX-XXXX"
				/>
			</div>

			{updateLicense.isError && (
				<div className="p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg text-sm text-red-700 dark:text-red-400">
					{updateLicense.error instanceof Error
						? updateLicense.error.message
						: 'Failed to update license'}
				</div>
			)}

			{updateLicense.isSuccess && (
				<div className="p-3 bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg text-sm text-green-700 dark:text-green-400">
					License updated successfully
				</div>
			)}

			<div className="flex justify-end">
				<button
					type="submit"
					disabled={updateLicense.isPending}
					className="px-6 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
				>
					{updateLicense.isPending ? 'Updating...' : 'Update License'}
				</button>
			</div>
		</form>
	);
}
