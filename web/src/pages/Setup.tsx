import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { VerticalStepper } from '../components/ui/Stepper';
import {
	useActivateLicense,
	useCompleteSetup,
	useConfigureOIDC,
	useConfigureSMTP,
	useCreateFirstOrganization,
	useCreateSuperuser,
	useSetupStatus,
	useSkipOIDC,
	useSkipSMTP,
	useStartTrial,
	useTestDatabase,
} from '../hooks/useSetup';
import type { OIDCSettings, SMTPSettings } from '../lib/types';

const SETUP_STEPS = [
	{ id: 'database', label: 'Database', description: 'Verify database connection' },
	{ id: 'superuser', label: 'Superuser', description: 'Create admin account' },
	{ id: 'smtp', label: 'Email (Optional)', description: 'Configure SMTP' },
	{ id: 'oidc', label: 'SSO (Optional)', description: 'Configure OIDC' },
	{ id: 'license', label: 'License', description: 'Enter license or start trial' },
	{ id: 'organization', label: 'Organization', description: 'Create first organization' },
];

interface StepProps {
	onComplete: () => void;
	onSkip?: () => void;
	isLoading?: boolean;
}

function DatabaseStep({ onComplete, isLoading }: StepProps) {
	const testDatabase = useTestDatabase();
	const [tested, setTested] = useState(false);

	const handleTest = () => {
		testDatabase.mutate(undefined, {
			onSuccess: (data) => {
				if (data.ok) {
					setTested(true);
					onComplete();
				}
			},
		});
	};

	useEffect(() => {
		// Auto-test on mount
		handleTest();
	}, []);

	return (
		<div className="py-4">
			<h2 className="text-xl font-semibold text-gray-900 mb-2">
				Database Connection
			</h2>
			<p className="text-gray-600 mb-6">
				Verifying the database connection to ensure the server can store data.
			</p>

			{testDatabase.isPending && (
				<div className="flex items-center gap-3 p-4 bg-blue-50 border border-blue-200 rounded-lg mb-6">
					<div className="w-5 h-5 border-2 border-blue-200 border-t-blue-600 rounded-full animate-spin" />
					<span className="text-blue-800">Testing database connection...</span>
				</div>
			)}

			{testDatabase.isError && (
				<div className="p-4 bg-red-50 border border-red-200 rounded-lg mb-6">
					<div className="flex items-center gap-2 text-red-800">
						<svg aria-hidden="true" className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
							<path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
						</svg>
						<span className="font-medium">Connection failed</span>
					</div>
					<p className="mt-1 text-sm text-red-700">
						{testDatabase.error instanceof Error ? testDatabase.error.message : 'Unknown error'}
					</p>
				</div>
			)}

			{tested && testDatabase.data?.ok && (
				<div className="p-4 bg-green-50 border border-green-200 rounded-lg mb-6">
					<div className="flex items-center gap-2 text-green-800">
						<svg aria-hidden="true" className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
							<path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
						</svg>
						<span className="font-medium">Database connection successful</span>
					</div>
				</div>
			)}

			<div className="flex justify-end">
				<button
					type="button"
					onClick={handleTest}
					disabled={testDatabase.isPending || isLoading}
					className="px-6 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
				>
					{testDatabase.isPending ? 'Testing...' : tested ? 'Continue' : 'Test Connection'}
				</button>
			</div>
		</div>
	);
}

function SuperuserStep({ onComplete, isLoading }: StepProps) {
	const createSuperuser = useCreateSuperuser();
	const [email, setEmail] = useState('');
	const [name, setName] = useState('');
	const [password, setPassword] = useState('');
	const [confirmPassword, setConfirmPassword] = useState('');
	const [error, setError] = useState('');

	const handleSubmit = (e: React.FormEvent) => {
		e.preventDefault();
		setError('');

		if (password !== confirmPassword) {
			setError('Passwords do not match');
			return;
		}

		if (password.length < 8) {
			setError('Password must be at least 8 characters');
			return;
		}

		createSuperuser.mutate(
			{ email, password, name },
			{
				onSuccess: () => onComplete(),
				onError: (err) => setError(err instanceof Error ? err.message : 'Failed to create superuser'),
			},
		);
	};

	return (
		<div className="py-4">
			<h2 className="text-xl font-semibold text-gray-900 mb-2">
				Create Superuser Account
			</h2>
			<p className="text-gray-600 mb-6">
				Create the administrator account that will have full access to manage the server.
			</p>

			<form onSubmit={handleSubmit} className="space-y-4">
				<div>
					<label htmlFor="name" className="block text-sm font-medium text-gray-700 mb-1">
						Name <span className="text-red-500">*</span>
					</label>
					<input
						id="name"
						type="text"
						value={name}
						onChange={(e) => setName(e.target.value)}
						required
						className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						placeholder="Admin User"
					/>
				</div>

				<div>
					<label htmlFor="email" className="block text-sm font-medium text-gray-700 mb-1">
						Email <span className="text-red-500">*</span>
					</label>
					<input
						id="email"
						type="email"
						value={email}
						onChange={(e) => setEmail(e.target.value)}
						required
						className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						placeholder="admin@example.com"
					/>
				</div>

				<div>
					<label htmlFor="password" className="block text-sm font-medium text-gray-700 mb-1">
						Password <span className="text-red-500">*</span>
					</label>
					<input
						id="password"
						type="password"
						value={password}
						onChange={(e) => setPassword(e.target.value)}
						required
						minLength={8}
						className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						placeholder="Minimum 8 characters"
					/>
				</div>

				<div>
					<label htmlFor="confirmPassword" className="block text-sm font-medium text-gray-700 mb-1">
						Confirm Password <span className="text-red-500">*</span>
					</label>
					<input
						id="confirmPassword"
						type="password"
						value={confirmPassword}
						onChange={(e) => setConfirmPassword(e.target.value)}
						required
						className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
					/>
				</div>

				{(error || createSuperuser.isError) && (
					<div className="p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700">
						{error || (createSuperuser.error instanceof Error ? createSuperuser.error.message : 'An error occurred')}
					</div>
				)}

				<div className="flex justify-end pt-4">
					<button
						type="submit"
						disabled={createSuperuser.isPending || isLoading}
						className="px-6 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
					>
						{createSuperuser.isPending ? 'Creating...' : 'Create Account'}
					</button>
				</div>
			</form>
		</div>
	);
}

function SMTPStepComponent({ onComplete, onSkip, isLoading }: StepProps) {
	const configureSMTP = useConfigureSMTP();
	const skipSMTP = useSkipSMTP();
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
		configureSMTP.mutate(settings, {
			onSuccess: () => onComplete(),
		});
	};

	const handleSkip = () => {
		skipSMTP.mutate(undefined, {
			onSuccess: () => onSkip?.(),
		});
	};

	return (
		<div className="py-4">
			<h2 className="text-xl font-semibold text-gray-900 mb-2">
				Email Configuration (Optional)
			</h2>
			<p className="text-gray-600 mb-6">
				Configure SMTP settings to enable email notifications. You can skip this and configure it later.
			</p>

			<form onSubmit={handleSubmit} className="space-y-4">
				<div className="grid grid-cols-2 gap-4">
					<div>
						<label htmlFor="smtp-host" className="block text-sm font-medium text-gray-700 mb-1">
							SMTP Host
						</label>
						<input
							id="smtp-host"
							type="text"
							value={settings.host}
							onChange={(e) => setSettings({ ...settings, host: e.target.value })}
							className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
							placeholder="smtp.example.com"
						/>
					</div>

					<div>
						<label htmlFor="smtp-port" className="block text-sm font-medium text-gray-700 mb-1">
							Port
						</label>
						<input
							id="smtp-port"
							type="number"
							value={settings.port}
							onChange={(e) => setSettings({ ...settings, port: parseInt(e.target.value, 10) })}
							className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						/>
					</div>
				</div>

				<div className="grid grid-cols-2 gap-4">
					<div>
						<label htmlFor="smtp-username" className="block text-sm font-medium text-gray-700 mb-1">
							Username
						</label>
						<input
							id="smtp-username"
							type="text"
							value={settings.username}
							onChange={(e) => setSettings({ ...settings, username: e.target.value })}
							className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						/>
					</div>

					<div>
						<label htmlFor="smtp-password" className="block text-sm font-medium text-gray-700 mb-1">
							Password
						</label>
						<input
							id="smtp-password"
							type="password"
							value={settings.password}
							onChange={(e) => setSettings({ ...settings, password: e.target.value })}
							className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						/>
					</div>
				</div>

				<div className="grid grid-cols-2 gap-4">
					<div>
						<label htmlFor="smtp-from-email" className="block text-sm font-medium text-gray-700 mb-1">
							From Email
						</label>
						<input
							id="smtp-from-email"
							type="email"
							value={settings.from_email}
							onChange={(e) => setSettings({ ...settings, from_email: e.target.value })}
							className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
							placeholder="noreply@example.com"
						/>
					</div>

					<div>
						<label htmlFor="smtp-from-name" className="block text-sm font-medium text-gray-700 mb-1">
							From Name
						</label>
						<input
							id="smtp-from-name"
							type="text"
							value={settings.from_name}
							onChange={(e) => setSettings({ ...settings, from_name: e.target.value })}
							className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
							placeholder="Keldris Backups"
						/>
					</div>
				</div>

				<div>
					<label htmlFor="smtp-encryption" className="block text-sm font-medium text-gray-700 mb-1">
						Encryption
					</label>
					<select
						id="smtp-encryption"
						value={settings.encryption}
						onChange={(e) => setSettings({ ...settings, encryption: e.target.value as 'none' | 'tls' | 'starttls' })}
						className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
					>
						<option value="starttls">STARTTLS</option>
						<option value="tls">TLS</option>
						<option value="none">None</option>
					</select>
				</div>

				{configureSMTP.isError && (
					<div className="p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700">
						{configureSMTP.error instanceof Error ? configureSMTP.error.message : 'Failed to configure SMTP'}
					</div>
				)}

				<div className="flex justify-between pt-4">
					<button
						type="button"
						onClick={handleSkip}
						disabled={skipSMTP.isPending || isLoading}
						className="px-6 py-2 text-gray-700 border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors disabled:opacity-50"
					>
						{skipSMTP.isPending ? 'Skipping...' : 'Skip for now'}
					</button>
					<button
						type="submit"
						disabled={configureSMTP.isPending || isLoading}
						className="px-6 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
					>
						{configureSMTP.isPending ? 'Saving...' : 'Save & Continue'}
					</button>
				</div>
			</form>
		</div>
	);
}

function OIDCStepComponent({ onComplete, onSkip, isLoading }: StepProps) {
	const configureOIDC = useConfigureOIDC();
	const skipOIDC = useSkipOIDC();
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
		configureOIDC.mutate(settings, {
			onSuccess: () => onComplete(),
		});
	};

	const handleSkip = () => {
		skipOIDC.mutate(undefined, {
			onSuccess: () => onSkip?.(),
		});
	};

	return (
		<div className="py-4">
			<h2 className="text-xl font-semibold text-gray-900 mb-2">
				Single Sign-On (Optional)
			</h2>
			<p className="text-gray-600 mb-6">
				Configure OpenID Connect (OIDC) for single sign-on authentication. You can skip this and configure it later.
			</p>

			<form onSubmit={handleSubmit} className="space-y-4">
				<div>
					<label htmlFor="oidc-issuer" className="block text-sm font-medium text-gray-700 mb-1">
						Issuer URL
					</label>
					<input
						id="oidc-issuer"
						type="url"
						value={settings.issuer}
						onChange={(e) => setSettings({ ...settings, issuer: e.target.value })}
						className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						placeholder="https://auth.example.com"
					/>
				</div>

				<div className="grid grid-cols-2 gap-4">
					<div>
						<label htmlFor="oidc-client-id" className="block text-sm font-medium text-gray-700 mb-1">
							Client ID
						</label>
						<input
							id="oidc-client-id"
							type="text"
							value={settings.client_id}
							onChange={(e) => setSettings({ ...settings, client_id: e.target.value })}
							className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						/>
					</div>

					<div>
						<label htmlFor="oidc-client-secret" className="block text-sm font-medium text-gray-700 mb-1">
							Client Secret
						</label>
						<input
							id="oidc-client-secret"
							type="password"
							value={settings.client_secret}
							onChange={(e) => setSettings({ ...settings, client_secret: e.target.value })}
							className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						/>
					</div>
				</div>

				<div>
					<label htmlFor="oidc-redirect" className="block text-sm font-medium text-gray-700 mb-1">
						Redirect URL
					</label>
					<input
						id="oidc-redirect"
						type="url"
						value={settings.redirect_url}
						onChange={(e) => setSettings({ ...settings, redirect_url: e.target.value })}
						className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						placeholder="https://your-domain.com/auth/callback"
					/>
					<p className="mt-1 text-xs text-gray-500">
						The URL that the OIDC provider will redirect to after authentication.
					</p>
				</div>

				{configureOIDC.isError && (
					<div className="p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700">
						{configureOIDC.error instanceof Error ? configureOIDC.error.message : 'Failed to configure OIDC'}
					</div>
				)}

				<div className="flex justify-between pt-4">
					<button
						type="button"
						onClick={handleSkip}
						disabled={skipOIDC.isPending || isLoading}
						className="px-6 py-2 text-gray-700 border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors disabled:opacity-50"
					>
						{skipOIDC.isPending ? 'Skipping...' : 'Skip for now'}
					</button>
					<button
						type="submit"
						disabled={configureOIDC.isPending || isLoading}
						className="px-6 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
					>
						{configureOIDC.isPending ? 'Saving...' : 'Save & Continue'}
					</button>
				</div>
			</form>
		</div>
	);
}

function LicenseStep({ onComplete, isLoading }: StepProps) {
	const activateLicense = useActivateLicense();
	const startTrial = useStartTrial();
	const [mode, setMode] = useState<'license' | 'trial'>('trial');
	const [licenseKey, setLicenseKey] = useState('');
	const [contactEmail, setContactEmail] = useState('');
	const [companyName, setCompanyName] = useState('');

	const handleActivateLicense = (e: React.FormEvent) => {
		e.preventDefault();
		activateLicense.mutate(
			{ license_key: licenseKey },
			{ onSuccess: () => onComplete() },
		);
	};

	const handleStartTrial = (e: React.FormEvent) => {
		e.preventDefault();
		startTrial.mutate(
			{ contact_email: contactEmail, company_name: companyName },
			{ onSuccess: () => onComplete() },
		);
	};

	return (
		<div className="py-4">
			<h2 className="text-xl font-semibold text-gray-900 mb-2">
				License
			</h2>
			<p className="text-gray-600 mb-6">
				Enter your license key or start a 14-day free trial.
			</p>

			<div className="flex gap-4 mb-6">
				<button
					type="button"
					onClick={() => setMode('trial')}
					className={`flex-1 py-3 px-4 rounded-lg border-2 transition-colors ${
						mode === 'trial'
							? 'border-indigo-600 bg-indigo-50 text-indigo-700'
							: 'border-gray-200 text-gray-700 hover:border-gray-300'
					}`}
				>
					<div className="font-medium">Start Free Trial</div>
					<div className="text-sm opacity-75">14 days, no credit card</div>
				</button>
				<button
					type="button"
					onClick={() => setMode('license')}
					className={`flex-1 py-3 px-4 rounded-lg border-2 transition-colors ${
						mode === 'license'
							? 'border-indigo-600 bg-indigo-50 text-indigo-700'
							: 'border-gray-200 text-gray-700 hover:border-gray-300'
					}`}
				>
					<div className="font-medium">Enter License Key</div>
					<div className="text-sm opacity-75">Already have a license</div>
				</button>
			</div>

			{mode === 'trial' ? (
				<form onSubmit={handleStartTrial} className="space-y-4">
					<div>
						<label htmlFor="trial-email" className="block text-sm font-medium text-gray-700 mb-1">
							Email <span className="text-red-500">*</span>
						</label>
						<input
							id="trial-email"
							type="email"
							value={contactEmail}
							onChange={(e) => setContactEmail(e.target.value)}
							required
							className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
							placeholder="you@example.com"
						/>
					</div>

					<div>
						<label htmlFor="trial-company" className="block text-sm font-medium text-gray-700 mb-1">
							Company Name
						</label>
						<input
							id="trial-company"
							type="text"
							value={companyName}
							onChange={(e) => setCompanyName(e.target.value)}
							className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						/>
					</div>

					<div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
						<h4 className="font-medium text-blue-900 mb-2">Trial includes:</h4>
						<ul className="text-sm text-blue-700 space-y-1">
							<li>Up to 5 agents</li>
							<li>Up to 2 repositories</li>
							<li>50 GB storage limit</li>
							<li>Full feature access for 14 days</li>
						</ul>
					</div>

					{startTrial.isError && (
						<div className="p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700">
							{startTrial.error instanceof Error ? startTrial.error.message : 'Failed to start trial'}
						</div>
					)}

					<div className="flex justify-end pt-4">
						<button
							type="submit"
							disabled={startTrial.isPending || isLoading}
							className="px-6 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
						>
							{startTrial.isPending ? 'Starting...' : 'Start Free Trial'}
						</button>
					</div>
				</form>
			) : (
				<form onSubmit={handleActivateLicense} className="space-y-4">
					<div>
						<label htmlFor="license-key" className="block text-sm font-medium text-gray-700 mb-1">
							License Key <span className="text-red-500">*</span>
						</label>
						<input
							id="license-key"
							type="text"
							value={licenseKey}
							onChange={(e) => setLicenseKey(e.target.value)}
							required
							className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 font-mono"
							placeholder="XXXX-XXXX-XXXX-XXXX"
						/>
					</div>

					{activateLicense.isError && (
						<div className="p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700">
							{activateLicense.error instanceof Error ? activateLicense.error.message : 'Failed to activate license'}
						</div>
					)}

					<div className="flex justify-end pt-4">
						<button
							type="submit"
							disabled={activateLicense.isPending || isLoading}
							className="px-6 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
						>
							{activateLicense.isPending ? 'Activating...' : 'Activate License'}
						</button>
					</div>
				</form>
			)}
		</div>
	);
}

function OrganizationStep({ onComplete, isLoading }: StepProps) {
	const createOrg = useCreateFirstOrganization();
	const [name, setName] = useState('');

	const handleSubmit = (e: React.FormEvent) => {
		e.preventDefault();
		createOrg.mutate(
			{ name: name || 'Default Organization' },
			{ onSuccess: () => onComplete() },
		);
	};

	return (
		<div className="py-4">
			<h2 className="text-xl font-semibold text-gray-900 mb-2">
				Create Organization
			</h2>
			<p className="text-gray-600 mb-6">
				Create your first organization. Organizations help you manage resources and team members.
			</p>

			<form onSubmit={handleSubmit} className="space-y-4">
				<div>
					<label htmlFor="org-name" className="block text-sm font-medium text-gray-700 mb-1">
						Organization Name
					</label>
					<input
						id="org-name"
						type="text"
						value={name}
						onChange={(e) => setName(e.target.value)}
						className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						placeholder="My Company"
					/>
					<p className="mt-1 text-xs text-gray-500">
						Leave empty to use "Default Organization"
					</p>
				</div>

				{createOrg.isError && (
					<div className="p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700">
						{createOrg.error instanceof Error ? createOrg.error.message : 'Failed to create organization'}
					</div>
				)}

				<div className="flex justify-end pt-4">
					<button
						type="submit"
						disabled={createOrg.isPending || isLoading}
						className="px-6 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
					>
						{createOrg.isPending ? 'Creating...' : 'Create & Continue'}
					</button>
				</div>
			</form>
		</div>
	);
}

function CompleteSetupStep() {
	const completeSetup = useCompleteSetup();
	const [showConfetti, setShowConfetti] = useState(false);

	const handleComplete = () => {
		setShowConfetti(true);
		completeSetup.mutate();
	};

	useEffect(() => {
		if (showConfetti) {
			const timer = setTimeout(() => setShowConfetti(false), 5000);
			return () => clearTimeout(timer);
		}
	}, [showConfetti]);

	return (
		<div className="text-center py-8 relative">
			{showConfetti && <Confetti />}

			<div className="w-20 h-20 bg-green-100 rounded-full flex items-center justify-center mx-auto mb-6">
				<svg
					aria-hidden="true"
					className="w-10 h-10 text-green-600"
					fill="none"
					stroke="currentColor"
					viewBox="0 0 24 24"
				>
					<path
						strokeLinecap="round"
						strokeLinejoin="round"
						strokeWidth={2}
						d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
					/>
				</svg>
			</div>

			<h2 className="text-2xl font-bold text-gray-900 mb-4">Setup Complete!</h2>

			<p className="text-gray-600 max-w-md mx-auto mb-8">
				Your Keldris server is now configured and ready to use. Click the button below to start using your backup system.
			</p>

			<div className="bg-green-50 border border-green-200 rounded-lg p-4 mb-8 max-w-md mx-auto text-left">
				<h3 className="text-sm font-semibold text-green-900 mb-2">What's next?</h3>
				<ul className="text-sm text-green-700 space-y-1">
					<li>- Log in with your superuser account</li>
					<li>- Create repositories to store backups</li>
					<li>- Install agents on systems to back up</li>
					<li>- Configure backup schedules</li>
				</ul>
			</div>

			{completeSetup.isError && (
				<div className="p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700 mb-4 max-w-md mx-auto">
					{completeSetup.error instanceof Error ? completeSetup.error.message : 'Failed to complete setup'}
				</div>
			)}

			<button
				type="button"
				onClick={handleComplete}
				disabled={completeSetup.isPending}
				className="inline-flex items-center gap-2 px-6 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
			>
				{completeSetup.isPending ? 'Completing...' : 'Go to Login'}
				<svg
					aria-hidden="true"
					className="w-4 h-4"
					fill="none"
					stroke="currentColor"
					viewBox="0 0 24 24"
				>
					<path
						strokeLinecap="round"
						strokeLinejoin="round"
						strokeWidth={2}
						d="M9 5l7 7-7 7"
					/>
				</svg>
			</button>
		</div>
	);
}

function Confetti() {
	const confettiPieces = Array.from({ length: 50 }, (_, i) => ({
		id: i,
		left: `${Math.random() * 100}%`,
		delay: `${Math.random() * 2}s`,
		duration: `${2 + Math.random() * 2}s`,
		color: ['#6366f1', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6'][
			Math.floor(Math.random() * 5)
		],
	}));

	return (
		<div className="fixed inset-0 pointer-events-none overflow-hidden z-50">
			{confettiPieces.map((piece) => (
				<div
					key={piece.id}
					className="absolute w-3 h-3 animate-confetti"
					style={{
						left: piece.left,
						top: '-12px',
						backgroundColor: piece.color,
						animationDelay: piece.delay,
						animationDuration: piece.duration,
					}}
				/>
			))}
			<style>{`
				@keyframes confetti {
					0% {
						transform: translateY(0) rotate(0deg);
						opacity: 1;
					}
					100% {
						transform: translateY(100vh) rotate(720deg);
						opacity: 0;
					}
				}
				.animate-confetti {
					animation: confetti linear forwards;
				}
			`}</style>
		</div>
	);
}

export function Setup() {
	const navigate = useNavigate();
	const { data: setupStatus, isLoading: statusLoading } = useSetupStatus();

	const currentStep = setupStatus?.current_step ?? 'database';
	const completedSteps = setupStatus?.completed_steps ?? [];

	// Redirect to dashboard if setup is already complete
	useEffect(() => {
		if (setupStatus?.setup_completed) {
			navigate('/');
		}
	}, [setupStatus?.setup_completed, navigate]);

	// Handle step completion by refetching status (which advances the step)
	const handleStepComplete = () => {
		// The hooks already invalidate the query, so the status will update automatically
	};

	if (statusLoading) {
		return (
			<div className="min-h-screen bg-gray-50 flex items-center justify-center">
				<div className="text-center">
					<div className="w-12 h-12 border-4 border-indigo-200 border-t-indigo-600 rounded-full animate-spin mx-auto mb-4" />
					<p className="text-gray-600">Loading setup...</p>
				</div>
			</div>
		);
	}

	const renderStep = () => {
		switch (currentStep) {
			case 'database':
				return <DatabaseStep onComplete={handleStepComplete} />;
			case 'superuser':
				return <SuperuserStep onComplete={handleStepComplete} />;
			case 'smtp':
				return (
					<SMTPStepComponent
						onComplete={handleStepComplete}
						onSkip={handleStepComplete}
					/>
				);
			case 'oidc':
				return (
					<OIDCStepComponent
						onComplete={handleStepComplete}
						onSkip={handleStepComplete}
					/>
				);
			case 'license':
				return <LicenseStep onComplete={handleStepComplete} />;
			case 'organization':
				return <OrganizationStep onComplete={handleStepComplete} />;
			case 'complete':
				return <CompleteSetupStep />;
			default:
				return null;
		}
	};

	return (
		<div className="min-h-screen bg-gray-50 flex items-center justify-center py-12 px-4">
			<div className="max-w-4xl w-full">
				{/* Header */}
				<div className="text-center mb-8">
					<div className="w-16 h-16 bg-indigo-100 rounded-full flex items-center justify-center mx-auto mb-4">
						<svg
							aria-hidden="true"
							className="w-8 h-8 text-indigo-600"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
						>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01"
							/>
						</svg>
					</div>
					<h1 className="text-3xl font-bold text-gray-900">Keldris Server Setup</h1>
					<p className="text-gray-600 mt-2">
						Configure your Keldris backup server
					</p>
				</div>

				<div className="flex gap-8">
					{/* Sidebar Stepper */}
					<div className="w-64 shrink-0">
						<div className="sticky top-6">
							<VerticalStepper
								steps={SETUP_STEPS}
								currentStep={currentStep}
								completedSteps={completedSteps}
							/>
						</div>
					</div>

					{/* Main Content */}
					<div className="flex-1">
						<div className="bg-white rounded-lg border border-gray-200 shadow-sm p-6">
							{renderStep()}
						</div>
					</div>
				</div>
			</div>
		</div>
	);
}
