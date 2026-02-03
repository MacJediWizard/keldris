import { useEffect, useState } from 'react';
import { useMe } from '../hooks/useAuth';
import {
	useSettingsAuditLog,
	useSystemSettings,
	useTestOIDC,
	useTestSMTP,
	useUpdateOIDCSettings,
	useUpdateSecuritySettings,
	useUpdateSMTPSettings,
	useUpdateStorageDefaultSettings,
} from '../hooks/useSystemSettings';
import type {
	OIDCSettings,
	OrgRole,
	SecuritySettings,
	SettingsAuditLog,
	SMTPSettings,
	StorageDefaultSettings,
} from '../lib/types';

type SettingsTab = 'smtp' | 'oidc' | 'storage' | 'security' | 'audit';

const tabLabels: Record<SettingsTab, string> = {
	smtp: 'Email (SMTP)',
	oidc: 'Single Sign-On',
	storage: 'Storage Defaults',
	security: 'Security',
	audit: 'Audit Log',
};

export function SystemSettings() {
	const { data: user } = useMe();
	const currentUserRole = (user?.current_org_role ?? 'member') as OrgRole;
	const canEdit = currentUserRole === 'owner' || currentUserRole === 'admin';

	const { data: settings, isLoading } = useSystemSettings();
	const { data: auditLogs } = useSettingsAuditLog(50, 0);

	const updateSMTP = useUpdateSMTPSettings();
	const testSMTP = useTestSMTP();
	const updateOIDC = useUpdateOIDCSettings();
	const testOIDC = useTestOIDC();
	const updateStorage = useUpdateStorageDefaultSettings();
	const updateSecurity = useUpdateSecuritySettings();

	const [activeTab, setActiveTab] = useState<SettingsTab>('smtp');
	const [editingSection, setEditingSection] = useState<string | null>(null);

	// SMTP form state
	const [smtpForm, setSmtpForm] = useState<SMTPSettings>({
		host: '',
		port: 587,
		username: '',
		password: '',
		from_email: '',
		from_name: '',
		encryption: 'starttls',
		enabled: false,
		skip_tls_verify: false,
		connection_timeout_seconds: 30,
	});
	const [testEmail, setTestEmail] = useState('');
	const [smtpTestResult, setSmtpTestResult] = useState<{
		success: boolean;
		message: string;
	} | null>(null);

	// OIDC form state
	const [oidcForm, setOidcForm] = useState<OIDCSettings>({
		enabled: false,
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
	const [oidcTestResult, setOidcTestResult] = useState<{
		success: boolean;
		message: string;
	} | null>(null);

	// Storage form state
	const [storageForm, setStorageForm] = useState<StorageDefaultSettings>({
		default_retention_days: 30,
		max_retention_days: 365,
		default_storage_backend: 'local',
		max_backup_size_gb: 100,
		enable_compression: true,
		compression_level: 6,
		default_encryption_method: 'aes256',
		prune_schedule: '0 2 * * *',
		auto_prune_enabled: true,
	});

	// Security form state
	const [securityForm, setSecurityForm] = useState<SecuritySettings>({
		session_timeout_minutes: 480,
		max_concurrent_sessions: 5,
		require_mfa: false,
		mfa_grace_period_days: 7,
		allowed_ip_ranges: [],
		blocked_ip_ranges: [],
		failed_login_lockout_attempts: 5,
		failed_login_lockout_minutes: 30,
		api_key_expiration_days: 0,
		enable_audit_logging: true,
		audit_log_retention_days: 90,
		force_https: true,
		allow_password_login: true,
	});

	// Load settings into forms
	useEffect(() => {
		if (settings) {
			setSmtpForm({
				...settings.smtp,
				password: '', // Don't display password
			});
			setOidcForm({
				...settings.oidc,
				client_secret: '', // Don't display secret
			});
			setStorageForm(settings.storage_defaults);
			setSecurityForm(settings.security);
		}
	}, [settings]);

	const handleSaveSMTP = async () => {
		try {
			const data = { ...smtpForm };
			if (!data.password) {
				(data as Record<string, unknown>).password = undefined;
			}
			await updateSMTP.mutateAsync(data);
			setEditingSection(null);
		} catch {
			// Error handled by mutation
		}
	};

	const handleTestSMTP = async () => {
		if (!testEmail) return;
		try {
			const result = await testSMTP.mutateAsync({ recipient_email: testEmail });
			setSmtpTestResult(result);
		} catch {
			setSmtpTestResult({ success: false, message: 'Test failed' });
		}
	};

	const handleSaveOIDC = async () => {
		try {
			const data = { ...oidcForm };
			if (!data.client_secret) {
				(data as Record<string, unknown>).client_secret = undefined;
			}
			await updateOIDC.mutateAsync(data);
			setEditingSection(null);
		} catch {
			// Error handled by mutation
		}
	};

	const handleTestOIDC = async () => {
		try {
			const result = await testOIDC.mutateAsync();
			setOidcTestResult(result);
		} catch {
			setOidcTestResult({ success: false, message: 'Test failed' });
		}
	};

	const handleSaveStorage = async () => {
		try {
			await updateStorage.mutateAsync(storageForm);
			setEditingSection(null);
		} catch {
			// Error handled by mutation
		}
	};

	const handleSaveSecurity = async () => {
		try {
			await updateSecurity.mutateAsync(securityForm);
			setEditingSection(null);
		} catch {
			// Error handled by mutation
		}
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
					System settings require admin or owner access.
				</p>
			</div>
		);
	}

	return (
		<div className="space-y-6">
			<div>
				<h1 className="text-2xl font-bold text-gray-900 dark:text-white">
					System Settings
				</h1>
				<p className="text-gray-600 dark:text-gray-400 mt-1">
					Configure email, authentication, storage, and security settings for
					your organization
				</p>
			</div>

			{/* Tabs */}
			<div className="border-b border-gray-200 dark:border-gray-700">
				<nav className="-mb-px flex space-x-8">
					{(Object.keys(tabLabels) as SettingsTab[]).map((tab) => (
						<button
							key={tab}
							type="button"
							onClick={() => setActiveTab(tab)}
							className={`pb-4 px-1 border-b-2 font-medium text-sm transition-colors ${
								activeTab === tab
									? 'border-indigo-600 text-indigo-600 dark:border-indigo-400 dark:text-indigo-400'
									: 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300 dark:text-gray-400 dark:hover:text-gray-300'
							}`}
						>
							{tabLabels[tab]}
						</button>
					))}
				</nav>
			</div>

			{/* SMTP Settings */}
			{activeTab === 'smtp' && (
				<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
					<div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between">
						<div>
							<h2 className="text-lg font-semibold text-gray-900 dark:text-white">
								SMTP Configuration
							</h2>
							<p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
								Configure email delivery for notifications and alerts
							</p>
						</div>
						{editingSection !== 'smtp' && (
							<button
								type="button"
								onClick={() => setEditingSection('smtp')}
								className="text-indigo-600 hover:text-indigo-800 dark:text-indigo-400 dark:hover:text-indigo-300 text-sm font-medium"
							>
								Edit
							</button>
						)}
					</div>
					<div className="p-6 space-y-4">
						<div className="flex items-center gap-2 mb-6">
							<input
								type="checkbox"
								id="smtp-enabled"
								checked={smtpForm.enabled}
								disabled={editingSection !== 'smtp'}
								onChange={(e) =>
									setSmtpForm({ ...smtpForm, enabled: e.target.checked })
								}
								className="h-4 w-4 text-indigo-600 focus:ring-indigo-500 border-gray-300 rounded disabled:opacity-50"
							/>
							<label
								htmlFor="smtp-enabled"
								className="text-sm font-medium text-gray-700 dark:text-gray-300"
							>
								Enable SMTP
							</label>
						</div>

						<div className="grid grid-cols-2 gap-4">
							<div>
								<label
									htmlFor="smtp-host"
									className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
								>
									SMTP Host
								</label>
								<input
									type="text"
									id="smtp-host"
									value={smtpForm.host}
									disabled={editingSection !== 'smtp'}
									onChange={(e) =>
										setSmtpForm({ ...smtpForm, host: e.target.value })
									}
									placeholder="smtp.example.com"
									className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-white disabled:opacity-50 disabled:bg-gray-100 dark:disabled:bg-gray-800"
								/>
							</div>
							<div>
								<label
									htmlFor="smtp-port"
									className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
								>
									Port
								</label>
								<input
									type="number"
									id="smtp-port"
									value={smtpForm.port}
									disabled={editingSection !== 'smtp'}
									onChange={(e) =>
										setSmtpForm({ ...smtpForm, port: Number(e.target.value) })
									}
									className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-white disabled:opacity-50 disabled:bg-gray-100 dark:disabled:bg-gray-800"
								/>
							</div>
						</div>

						<div className="grid grid-cols-2 gap-4">
							<div>
								<label
									htmlFor="smtp-username"
									className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
								>
									Username
								</label>
								<input
									type="text"
									id="smtp-username"
									value={smtpForm.username ?? ''}
									disabled={editingSection !== 'smtp'}
									onChange={(e) =>
										setSmtpForm({ ...smtpForm, username: e.target.value })
									}
									className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-white disabled:opacity-50 disabled:bg-gray-100 dark:disabled:bg-gray-800"
								/>
							</div>
							<div>
								<label
									htmlFor="smtp-password"
									className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
								>
									Password
								</label>
								<input
									type="password"
									id="smtp-password"
									value={smtpForm.password ?? ''}
									disabled={editingSection !== 'smtp'}
									onChange={(e) =>
										setSmtpForm({ ...smtpForm, password: e.target.value })
									}
									placeholder={
										editingSection === 'smtp'
											? 'Enter new password'
											: '********'
									}
									className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-white disabled:opacity-50 disabled:bg-gray-100 dark:disabled:bg-gray-800"
								/>
							</div>
						</div>

						<div className="grid grid-cols-2 gap-4">
							<div>
								<label
									htmlFor="smtp-from-email"
									className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
								>
									From Email
								</label>
								<input
									type="email"
									id="smtp-from-email"
									value={smtpForm.from_email}
									disabled={editingSection !== 'smtp'}
									onChange={(e) =>
										setSmtpForm({ ...smtpForm, from_email: e.target.value })
									}
									placeholder="noreply@example.com"
									className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-white disabled:opacity-50 disabled:bg-gray-100 dark:disabled:bg-gray-800"
								/>
							</div>
							<div>
								<label
									htmlFor="smtp-from-name"
									className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
								>
									From Name
								</label>
								<input
									type="text"
									id="smtp-from-name"
									value={smtpForm.from_name ?? ''}
									disabled={editingSection !== 'smtp'}
									onChange={(e) =>
										setSmtpForm({ ...smtpForm, from_name: e.target.value })
									}
									placeholder="Keldris Backup"
									className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-white disabled:opacity-50 disabled:bg-gray-100 dark:disabled:bg-gray-800"
								/>
							</div>
						</div>

						<div className="grid grid-cols-2 gap-4">
							<div>
								<label
									htmlFor="smtp-encryption"
									className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
								>
									Encryption
								</label>
								<select
									id="smtp-encryption"
									value={smtpForm.encryption}
									disabled={editingSection !== 'smtp'}
									onChange={(e) =>
										setSmtpForm({
											...smtpForm,
											encryption: e.target.value as 'none' | 'tls' | 'starttls',
										})
									}
									className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-white disabled:opacity-50 disabled:bg-gray-100 dark:disabled:bg-gray-800"
								>
									<option value="starttls">STARTTLS (Recommended)</option>
									<option value="tls">TLS</option>
									<option value="none">None</option>
								</select>
							</div>
							<div>
								<label
									htmlFor="smtp-timeout"
									className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
								>
									Connection Timeout (seconds)
								</label>
								<input
									type="number"
									id="smtp-timeout"
									value={smtpForm.connection_timeout_seconds}
									disabled={editingSection !== 'smtp'}
									onChange={(e) =>
										setSmtpForm({
											...smtpForm,
											connection_timeout_seconds: Number(e.target.value),
										})
									}
									min={5}
									max={120}
									className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-white disabled:opacity-50 disabled:bg-gray-100 dark:disabled:bg-gray-800"
								/>
							</div>
						</div>

						<div className="flex items-center gap-2">
							<input
								type="checkbox"
								id="skip-tls-verify"
								checked={smtpForm.skip_tls_verify}
								disabled={editingSection !== 'smtp'}
								onChange={(e) =>
									setSmtpForm({
										...smtpForm,
										skip_tls_verify: e.target.checked,
									})
								}
								className="h-4 w-4 text-indigo-600 focus:ring-indigo-500 border-gray-300 rounded disabled:opacity-50"
							/>
							<label
								htmlFor="skip-tls-verify"
								className="text-sm text-gray-700 dark:text-gray-300"
							>
								Skip TLS certificate verification (not recommended for
								production)
							</label>
						</div>

						{/* Test Section */}
						{smtpForm.enabled && (
							<div className="mt-6 p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg">
								<h3 className="text-sm font-medium text-gray-900 dark:text-white mb-3">
									Test SMTP Connection
								</h3>
								<div className="flex gap-3">
									<input
										type="email"
										value={testEmail}
										onChange={(e) => setTestEmail(e.target.value)}
										placeholder="test@example.com"
										className="flex-1 px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
									/>
									<button
										type="button"
										onClick={handleTestSMTP}
										disabled={testSMTP.isPending || !testEmail}
										className="px-4 py-2 bg-gray-600 text-white rounded-lg hover:bg-gray-700 transition-colors disabled:opacity-50"
									>
										{testSMTP.isPending ? 'Testing...' : 'Send Test Email'}
									</button>
								</div>
								{smtpTestResult && (
									<p
										className={`mt-2 text-sm ${smtpTestResult.success ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400'}`}
									>
										{smtpTestResult.message}
									</p>
								)}
							</div>
						)}

						{editingSection === 'smtp' && (
							<div className="flex justify-end gap-3 pt-4 border-t border-gray-200 dark:border-gray-700">
								<button
									type="button"
									onClick={() => {
										setEditingSection(null);
										if (settings) {
											setSmtpForm({ ...settings.smtp, password: '' });
										}
									}}
									className="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
								>
									Cancel
								</button>
								<button
									type="button"
									onClick={handleSaveSMTP}
									disabled={updateSMTP.isPending}
									className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
								>
									{updateSMTP.isPending ? 'Saving...' : 'Save Changes'}
								</button>
							</div>
						)}
					</div>
				</div>
			)}

			{/* OIDC Settings */}
			{activeTab === 'oidc' && (
				<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
					<div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between">
						<div>
							<h2 className="text-lg font-semibold text-gray-900 dark:text-white">
								OIDC / Single Sign-On
							</h2>
							<p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
								Configure OpenID Connect for single sign-on authentication
							</p>
						</div>
						{editingSection !== 'oidc' && (
							<button
								type="button"
								onClick={() => setEditingSection('oidc')}
								className="text-indigo-600 hover:text-indigo-800 dark:text-indigo-400 dark:hover:text-indigo-300 text-sm font-medium"
							>
								Edit
							</button>
						)}
					</div>
					<div className="p-6 space-y-4">
						<div className="flex items-center gap-2 mb-6">
							<input
								type="checkbox"
								id="oidc-enabled"
								checked={oidcForm.enabled}
								disabled={editingSection !== 'oidc'}
								onChange={(e) =>
									setOidcForm({ ...oidcForm, enabled: e.target.checked })
								}
								className="h-4 w-4 text-indigo-600 focus:ring-indigo-500 border-gray-300 rounded disabled:opacity-50"
							/>
							<label
								htmlFor="oidc-enabled"
								className="text-sm font-medium text-gray-700 dark:text-gray-300"
							>
								Enable OIDC Authentication
							</label>
						</div>

						<div>
							<label
								htmlFor="oidc-issuer"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								Issuer URL
							</label>
							<input
								type="url"
								id="oidc-issuer"
								value={oidcForm.issuer}
								disabled={editingSection !== 'oidc'}
								onChange={(e) =>
									setOidcForm({ ...oidcForm, issuer: e.target.value })
								}
								placeholder="https://accounts.google.com"
								className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-white disabled:opacity-50 disabled:bg-gray-100 dark:disabled:bg-gray-800"
							/>
						</div>

						<div className="grid grid-cols-2 gap-4">
							<div>
								<label
									htmlFor="oidc-client-id"
									className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
								>
									Client ID
								</label>
								<input
									type="text"
									id="oidc-client-id"
									value={oidcForm.client_id}
									disabled={editingSection !== 'oidc'}
									onChange={(e) =>
										setOidcForm({ ...oidcForm, client_id: e.target.value })
									}
									className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-white disabled:opacity-50 disabled:bg-gray-100 dark:disabled:bg-gray-800"
								/>
							</div>
							<div>
								<label
									htmlFor="oidc-client-secret"
									className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
								>
									Client Secret
								</label>
								<input
									type="password"
									id="oidc-client-secret"
									value={oidcForm.client_secret ?? ''}
									disabled={editingSection !== 'oidc'}
									onChange={(e) =>
										setOidcForm({ ...oidcForm, client_secret: e.target.value })
									}
									placeholder={
										editingSection === 'oidc' ? 'Enter new secret' : '********'
									}
									className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-white disabled:opacity-50 disabled:bg-gray-100 dark:disabled:bg-gray-800"
								/>
							</div>
						</div>

						<div>
							<label
								htmlFor="oidc-redirect-url"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								Redirect URL
							</label>
							<input
								type="url"
								id="oidc-redirect-url"
								value={oidcForm.redirect_url}
								disabled={editingSection !== 'oidc'}
								onChange={(e) =>
									setOidcForm({ ...oidcForm, redirect_url: e.target.value })
								}
								placeholder="https://your-domain.com/auth/callback"
								className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-white disabled:opacity-50 disabled:bg-gray-100 dark:disabled:bg-gray-800"
							/>
						</div>

						<div className="grid grid-cols-2 gap-4">
							<div>
								<label
									htmlFor="oidc-default-role"
									className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
								>
									Default Role for New Users
								</label>
								<select
									id="oidc-default-role"
									value={oidcForm.default_role}
									disabled={editingSection !== 'oidc'}
									onChange={(e) =>
										setOidcForm({
											...oidcForm,
											default_role: e.target.value as 'member' | 'readonly',
										})
									}
									className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-white disabled:opacity-50 disabled:bg-gray-100 dark:disabled:bg-gray-800"
								>
									<option value="member">Member</option>
									<option value="readonly">Read-only</option>
								</select>
							</div>
							<div>
								<label
									htmlFor="oidc-allowed-domains"
									className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
								>
									Allowed Email Domains (comma-separated)
								</label>
								<input
									type="text"
									id="oidc-allowed-domains"
									value={oidcForm.allowed_domains?.join(', ') ?? ''}
									disabled={editingSection !== 'oidc'}
									onChange={(e) =>
										setOidcForm({
											...oidcForm,
											allowed_domains: e.target.value
												.split(',')
												.map((d) => d.trim())
												.filter(Boolean),
										})
									}
									placeholder="example.com, company.org"
									className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-white disabled:opacity-50 disabled:bg-gray-100 dark:disabled:bg-gray-800"
								/>
							</div>
						</div>

						<div className="space-y-2">
							<div className="flex items-center gap-2">
								<input
									type="checkbox"
									id="auto-create-users"
									checked={oidcForm.auto_create_users}
									disabled={editingSection !== 'oidc'}
									onChange={(e) =>
										setOidcForm({
											...oidcForm,
											auto_create_users: e.target.checked,
										})
									}
									className="h-4 w-4 text-indigo-600 focus:ring-indigo-500 border-gray-300 rounded disabled:opacity-50"
								/>
								<label
									htmlFor="auto-create-users"
									className="text-sm text-gray-700 dark:text-gray-300"
								>
									Automatically create user accounts on first login
								</label>
							</div>
							<div className="flex items-center gap-2">
								<input
									type="checkbox"
									id="require-email-verification"
									checked={oidcForm.require_email_verification}
									disabled={editingSection !== 'oidc'}
									onChange={(e) =>
										setOidcForm({
											...oidcForm,
											require_email_verification: e.target.checked,
										})
									}
									className="h-4 w-4 text-indigo-600 focus:ring-indigo-500 border-gray-300 rounded disabled:opacity-50"
								/>
								<label
									htmlFor="require-email-verification"
									className="text-sm text-gray-700 dark:text-gray-300"
								>
									Require email verification from identity provider
								</label>
							</div>
						</div>

						{/* Test Section */}
						{oidcForm.enabled && (
							<div className="mt-6 p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg">
								<h3 className="text-sm font-medium text-gray-900 dark:text-white mb-3">
									Test OIDC Configuration
								</h3>
								<button
									type="button"
									onClick={handleTestOIDC}
									disabled={testOIDC.isPending}
									className="px-4 py-2 bg-gray-600 text-white rounded-lg hover:bg-gray-700 transition-colors disabled:opacity-50"
								>
									{testOIDC.isPending ? 'Testing...' : 'Test Configuration'}
								</button>
								{oidcTestResult && (
									<p
										className={`mt-2 text-sm ${oidcTestResult.success ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400'}`}
									>
										{oidcTestResult.message}
									</p>
								)}
							</div>
						)}

						{editingSection === 'oidc' && (
							<div className="flex justify-end gap-3 pt-4 border-t border-gray-200 dark:border-gray-700">
								<button
									type="button"
									onClick={() => {
										setEditingSection(null);
										if (settings) {
											setOidcForm({ ...settings.oidc, client_secret: '' });
										}
									}}
									className="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
								>
									Cancel
								</button>
								<button
									type="button"
									onClick={handleSaveOIDC}
									disabled={updateOIDC.isPending}
									className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
								>
									{updateOIDC.isPending ? 'Saving...' : 'Save Changes'}
								</button>
							</div>
						)}
					</div>
				</div>
			)}

			{/* Storage Settings */}
			{activeTab === 'storage' && (
				<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
					<div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between">
						<div>
							<h2 className="text-lg font-semibold text-gray-900 dark:text-white">
								Storage Defaults
							</h2>
							<p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
								Configure default storage and retention settings
							</p>
						</div>
						{editingSection !== 'storage' && (
							<button
								type="button"
								onClick={() => setEditingSection('storage')}
								className="text-indigo-600 hover:text-indigo-800 dark:text-indigo-400 dark:hover:text-indigo-300 text-sm font-medium"
							>
								Edit
							</button>
						)}
					</div>
					<div className="p-6 space-y-4">
						<div className="grid grid-cols-2 gap-4">
							<div>
								<label
									htmlFor="storage-retention"
									className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
								>
									Default Retention (days)
								</label>
								<input
									type="number"
									id="storage-retention"
									value={storageForm.default_retention_days}
									disabled={editingSection !== 'storage'}
									onChange={(e) =>
										setStorageForm({
											...storageForm,
											default_retention_days: Number(e.target.value),
										})
									}
									min={1}
									max={3650}
									className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-white disabled:opacity-50 disabled:bg-gray-100 dark:disabled:bg-gray-800"
								/>
							</div>
							<div>
								<label
									htmlFor="storage-max-retention"
									className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
								>
									Maximum Retention (days)
								</label>
								<input
									type="number"
									id="storage-max-retention"
									value={storageForm.max_retention_days}
									disabled={editingSection !== 'storage'}
									onChange={(e) =>
										setStorageForm({
											...storageForm,
											max_retention_days: Number(e.target.value),
										})
									}
									min={1}
									max={3650}
									className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-white disabled:opacity-50 disabled:bg-gray-100 dark:disabled:bg-gray-800"
								/>
							</div>
						</div>

						<div className="grid grid-cols-2 gap-4">
							<div>
								<label
									htmlFor="storage-backend"
									className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
								>
									Default Storage Backend
								</label>
								<select
									id="storage-backend"
									value={storageForm.default_storage_backend}
									disabled={editingSection !== 'storage'}
									onChange={(e) =>
										setStorageForm({
											...storageForm,
											default_storage_backend: e.target.value as
												| 'local'
												| 's3'
												| 'b2'
												| 'sftp'
												| 'rest'
												| 'dropbox',
										})
									}
									className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-white disabled:opacity-50 disabled:bg-gray-100 dark:disabled:bg-gray-800"
								>
									<option value="local">Local</option>
									<option value="s3">Amazon S3</option>
									<option value="b2">Backblaze B2</option>
									<option value="sftp">SFTP</option>
									<option value="rest">REST Server</option>
									<option value="dropbox">Dropbox</option>
								</select>
							</div>
							<div>
								<label
									htmlFor="storage-max-size"
									className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
								>
									Maximum Backup Size (GB)
								</label>
								<input
									type="number"
									id="storage-max-size"
									value={storageForm.max_backup_size_gb}
									disabled={editingSection !== 'storage'}
									onChange={(e) =>
										setStorageForm({
											...storageForm,
											max_backup_size_gb: Number(e.target.value),
										})
									}
									min={1}
									max={10000}
									className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-white disabled:opacity-50 disabled:bg-gray-100 dark:disabled:bg-gray-800"
								/>
							</div>
						</div>

						<div className="grid grid-cols-2 gap-4">
							<div>
								<label
									htmlFor="storage-encryption"
									className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
								>
									Default Encryption
								</label>
								<select
									id="storage-encryption"
									value={storageForm.default_encryption_method}
									disabled={editingSection !== 'storage'}
									onChange={(e) =>
										setStorageForm({
											...storageForm,
											default_encryption_method: e.target.value as
												| 'aes256'
												| 'none',
										})
									}
									className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-white disabled:opacity-50 disabled:bg-gray-100 dark:disabled:bg-gray-800"
								>
									<option value="aes256">AES-256 (Recommended)</option>
									<option value="none">None</option>
								</select>
							</div>
							<div>
								<label
									htmlFor="storage-compression-level"
									className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
								>
									Compression Level (1-9)
								</label>
								<input
									type="number"
									id="storage-compression-level"
									value={storageForm.compression_level}
									disabled={editingSection !== 'storage'}
									onChange={(e) =>
										setStorageForm({
											...storageForm,
											compression_level: Number(e.target.value),
										})
									}
									min={1}
									max={9}
									className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-white disabled:opacity-50 disabled:bg-gray-100 dark:disabled:bg-gray-800"
								/>
							</div>
						</div>

						<div>
							<label
								htmlFor="storage-prune-schedule"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								Auto-Prune Schedule (cron expression)
							</label>
							<input
								type="text"
								id="storage-prune-schedule"
								value={storageForm.prune_schedule}
								disabled={editingSection !== 'storage'}
								onChange={(e) =>
									setStorageForm({
										...storageForm,
										prune_schedule: e.target.value,
									})
								}
								placeholder="0 2 * * *"
								className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-white disabled:opacity-50 disabled:bg-gray-100 dark:disabled:bg-gray-800"
							/>
							<p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
								Default: 0 2 * * * (2 AM daily)
							</p>
						</div>

						<div className="space-y-2">
							<div className="flex items-center gap-2">
								<input
									type="checkbox"
									id="enable-compression"
									checked={storageForm.enable_compression}
									disabled={editingSection !== 'storage'}
									onChange={(e) =>
										setStorageForm({
											...storageForm,
											enable_compression: e.target.checked,
										})
									}
									className="h-4 w-4 text-indigo-600 focus:ring-indigo-500 border-gray-300 rounded disabled:opacity-50"
								/>
								<label
									htmlFor="enable-compression"
									className="text-sm text-gray-700 dark:text-gray-300"
								>
									Enable compression by default
								</label>
							</div>
							<div className="flex items-center gap-2">
								<input
									type="checkbox"
									id="auto-prune-enabled"
									checked={storageForm.auto_prune_enabled}
									disabled={editingSection !== 'storage'}
									onChange={(e) =>
										setStorageForm({
											...storageForm,
											auto_prune_enabled: e.target.checked,
										})
									}
									className="h-4 w-4 text-indigo-600 focus:ring-indigo-500 border-gray-300 rounded disabled:opacity-50"
								/>
								<label
									htmlFor="auto-prune-enabled"
									className="text-sm text-gray-700 dark:text-gray-300"
								>
									Enable automatic pruning
								</label>
							</div>
						</div>

						{editingSection === 'storage' && (
							<div className="flex justify-end gap-3 pt-4 border-t border-gray-200 dark:border-gray-700">
								<button
									type="button"
									onClick={() => {
										setEditingSection(null);
										if (settings) {
											setStorageForm(settings.storage_defaults);
										}
									}}
									className="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
								>
									Cancel
								</button>
								<button
									type="button"
									onClick={handleSaveStorage}
									disabled={updateStorage.isPending}
									className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
								>
									{updateStorage.isPending ? 'Saving...' : 'Save Changes'}
								</button>
							</div>
						)}
					</div>
				</div>
			)}

			{/* Security Settings */}
			{activeTab === 'security' && (
				<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
					<div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between">
						<div>
							<h2 className="text-lg font-semibold text-gray-900 dark:text-white">
								Security Settings
							</h2>
							<p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
								Configure session management, MFA, and access controls
							</p>
						</div>
						{editingSection !== 'security' && (
							<button
								type="button"
								onClick={() => setEditingSection('security')}
								className="text-indigo-600 hover:text-indigo-800 dark:text-indigo-400 dark:hover:text-indigo-300 text-sm font-medium"
							>
								Edit
							</button>
						)}
					</div>
					<div className="p-6 space-y-6">
						<div>
							<h3 className="text-sm font-medium text-gray-900 dark:text-white mb-3">
								Session Management
							</h3>
							<div className="grid grid-cols-2 gap-4">
								<div>
									<label
										htmlFor="security-session-timeout"
										className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
									>
										Session Timeout (minutes)
									</label>
									<input
										type="number"
										id="security-session-timeout"
										value={securityForm.session_timeout_minutes}
										disabled={editingSection !== 'security'}
										onChange={(e) =>
											setSecurityForm({
												...securityForm,
												session_timeout_minutes: Number(e.target.value),
											})
										}
										min={5}
										max={10080}
										className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-white disabled:opacity-50 disabled:bg-gray-100 dark:disabled:bg-gray-800"
									/>
									<p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
										Max: 10080 (7 days)
									</p>
								</div>
								<div>
									<label
										htmlFor="security-max-sessions"
										className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
									>
										Max Concurrent Sessions
									</label>
									<input
										type="number"
										id="security-max-sessions"
										value={securityForm.max_concurrent_sessions}
										disabled={editingSection !== 'security'}
										onChange={(e) =>
											setSecurityForm({
												...securityForm,
												max_concurrent_sessions: Number(e.target.value),
											})
										}
										min={1}
										max={100}
										className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-white disabled:opacity-50 disabled:bg-gray-100 dark:disabled:bg-gray-800"
									/>
								</div>
							</div>
						</div>

						<div>
							<h3 className="text-sm font-medium text-gray-900 dark:text-white mb-3">
								Multi-Factor Authentication
							</h3>
							<div className="space-y-3">
								<div className="flex items-center gap-2">
									<input
										type="checkbox"
										id="require-mfa"
										checked={securityForm.require_mfa}
										disabled={editingSection !== 'security'}
										onChange={(e) =>
											setSecurityForm({
												...securityForm,
												require_mfa: e.target.checked,
											})
										}
										className="h-4 w-4 text-indigo-600 focus:ring-indigo-500 border-gray-300 rounded disabled:opacity-50"
									/>
									<label
										htmlFor="require-mfa"
										className="text-sm text-gray-700 dark:text-gray-300"
									>
										Require MFA for all users
									</label>
								</div>
								{securityForm.require_mfa && (
									<div className="ml-6">
										<label
											htmlFor="security-mfa-grace"
											className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
										>
											MFA Grace Period (days)
										</label>
										<input
											type="number"
											id="security-mfa-grace"
											value={securityForm.mfa_grace_period_days}
											disabled={editingSection !== 'security'}
											onChange={(e) =>
												setSecurityForm({
													...securityForm,
													mfa_grace_period_days: Number(e.target.value),
												})
											}
											min={0}
											max={30}
											className="w-32 px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-white disabled:opacity-50 disabled:bg-gray-100 dark:disabled:bg-gray-800"
										/>
									</div>
								)}
							</div>
						</div>

						<div>
							<h3 className="text-sm font-medium text-gray-900 dark:text-white mb-3">
								Login Protection
							</h3>
							<div className="grid grid-cols-2 gap-4">
								<div>
									<label
										htmlFor="security-lockout-attempts"
										className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
									>
										Failed Login Lockout Attempts
									</label>
									<input
										type="number"
										id="security-lockout-attempts"
										value={securityForm.failed_login_lockout_attempts}
										disabled={editingSection !== 'security'}
										onChange={(e) =>
											setSecurityForm({
												...securityForm,
												failed_login_lockout_attempts: Number(e.target.value),
											})
										}
										min={1}
										max={20}
										className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-white disabled:opacity-50 disabled:bg-gray-100 dark:disabled:bg-gray-800"
									/>
								</div>
								<div>
									<label
										htmlFor="security-lockout-duration"
										className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
									>
										Lockout Duration (minutes)
									</label>
									<input
										type="number"
										id="security-lockout-duration"
										value={securityForm.failed_login_lockout_minutes}
										disabled={editingSection !== 'security'}
										onChange={(e) =>
											setSecurityForm({
												...securityForm,
												failed_login_lockout_minutes: Number(e.target.value),
											})
										}
										min={1}
										max={1440}
										className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-white disabled:opacity-50 disabled:bg-gray-100 dark:disabled:bg-gray-800"
									/>
								</div>
							</div>
						</div>

						<div>
							<h3 className="text-sm font-medium text-gray-900 dark:text-white mb-3">
								API & Audit
							</h3>
							<div className="grid grid-cols-2 gap-4">
								<div>
									<label
										htmlFor="security-api-expiration"
										className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
									>
										API Key Expiration (days, 0 = no expiration)
									</label>
									<input
										type="number"
										id="security-api-expiration"
										value={securityForm.api_key_expiration_days}
										disabled={editingSection !== 'security'}
										onChange={(e) =>
											setSecurityForm({
												...securityForm,
												api_key_expiration_days: Number(e.target.value),
											})
										}
										min={0}
										max={365}
										className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-white disabled:opacity-50 disabled:bg-gray-100 dark:disabled:bg-gray-800"
									/>
								</div>
								<div>
									<label
										htmlFor="security-audit-retention"
										className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
									>
										Audit Log Retention (days)
									</label>
									<input
										type="number"
										id="security-audit-retention"
										value={securityForm.audit_log_retention_days}
										disabled={editingSection !== 'security'}
										onChange={(e) =>
											setSecurityForm({
												...securityForm,
												audit_log_retention_days: Number(e.target.value),
											})
										}
										min={7}
										max={3650}
										className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-white disabled:opacity-50 disabled:bg-gray-100 dark:disabled:bg-gray-800"
									/>
								</div>
							</div>
						</div>

						<div className="space-y-2">
							<div className="flex items-center gap-2">
								<input
									type="checkbox"
									id="enable-audit-logging"
									checked={securityForm.enable_audit_logging}
									disabled={editingSection !== 'security'}
									onChange={(e) =>
										setSecurityForm({
											...securityForm,
											enable_audit_logging: e.target.checked,
										})
									}
									className="h-4 w-4 text-indigo-600 focus:ring-indigo-500 border-gray-300 rounded disabled:opacity-50"
								/>
								<label
									htmlFor="enable-audit-logging"
									className="text-sm text-gray-700 dark:text-gray-300"
								>
									Enable audit logging
								</label>
							</div>
							<div className="flex items-center gap-2">
								<input
									type="checkbox"
									id="force-https"
									checked={securityForm.force_https}
									disabled={editingSection !== 'security'}
									onChange={(e) =>
										setSecurityForm({
											...securityForm,
											force_https: e.target.checked,
										})
									}
									className="h-4 w-4 text-indigo-600 focus:ring-indigo-500 border-gray-300 rounded disabled:opacity-50"
								/>
								<label
									htmlFor="force-https"
									className="text-sm text-gray-700 dark:text-gray-300"
								>
									Force HTTPS connections
								</label>
							</div>
							<div className="flex items-center gap-2">
								<input
									type="checkbox"
									id="allow-password-login"
									checked={securityForm.allow_password_login}
									disabled={editingSection !== 'security'}
									onChange={(e) =>
										setSecurityForm({
											...securityForm,
											allow_password_login: e.target.checked,
										})
									}
									className="h-4 w-4 text-indigo-600 focus:ring-indigo-500 border-gray-300 rounded disabled:opacity-50"
								/>
								<label
									htmlFor="allow-password-login"
									className="text-sm text-gray-700 dark:text-gray-300"
								>
									Allow password-based login (in addition to SSO)
								</label>
							</div>
						</div>

						{editingSection === 'security' && (
							<div className="flex justify-end gap-3 pt-4 border-t border-gray-200 dark:border-gray-700">
								<button
									type="button"
									onClick={() => {
										setEditingSection(null);
										if (settings) {
											setSecurityForm(settings.security);
										}
									}}
									className="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
								>
									Cancel
								</button>
								<button
									type="button"
									onClick={handleSaveSecurity}
									disabled={updateSecurity.isPending}
									className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
								>
									{updateSecurity.isPending ? 'Saving...' : 'Save Changes'}
								</button>
							</div>
						)}
					</div>
				</div>
			)}

			{/* Audit Log */}
			{activeTab === 'audit' && (
				<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
					<div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
						<h2 className="text-lg font-semibold text-gray-900 dark:text-white">
							Settings Change History
						</h2>
						<p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
							View all changes made to system settings
						</p>
					</div>
					<div className="p-6">
						{auditLogs?.logs && auditLogs.logs.length > 0 ? (
							<table className="w-full">
								<thead>
									<tr className="text-left text-sm text-gray-500 dark:text-gray-400 border-b border-gray-200 dark:border-gray-700">
										<th className="pb-3 font-medium">Timestamp</th>
										<th className="pb-3 font-medium">Setting</th>
										<th className="pb-3 font-medium">Changed By</th>
										<th className="pb-3 font-medium">IP Address</th>
									</tr>
								</thead>
								<tbody>
									{auditLogs.logs.map((log: SettingsAuditLog) => (
										<tr
											key={log.id}
											className="border-b border-gray-100 dark:border-gray-700"
										>
											<td className="py-3 text-sm text-gray-600 dark:text-gray-400">
												{new Date(log.changed_at).toLocaleString()}
											</td>
											<td className="py-3">
												<span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-indigo-100 text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-400">
													{log.setting_key}
												</span>
											</td>
											<td className="py-3 text-sm text-gray-900 dark:text-white">
												{log.changed_by_email || log.changed_by}
											</td>
											<td className="py-3 text-sm text-gray-600 dark:text-gray-400 font-mono">
												{log.ip_address || '-'}
											</td>
										</tr>
									))}
								</tbody>
							</table>
						) : (
							<div className="text-center py-8">
								<p className="text-gray-500 dark:text-gray-400">
									No settings changes recorded yet
								</p>
							</div>
						)}
					</div>
				</div>
			)}
		</div>
	);
}
