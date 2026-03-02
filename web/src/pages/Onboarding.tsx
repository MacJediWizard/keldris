import { useQueryClient } from '@tanstack/react-query';
import { useCallback, useEffect, useRef, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { VerticalStepper } from '../components/ui/Stepper';
import { useCreateRegistrationCode } from '../hooks/useAgentRegistration';
import { useAgents } from '../hooks/useAgents';
import {
	useActivateLicense,
	useLicense,
	useStartTrial,
} from '../hooks/useLicense';
import {
	useCompleteOIDCStep,
	useCompleteOnboardingStep,
	useCompleteSMTPStep,
	useOnboardingStatus,
	useSkipOnboarding,
	useTestOnboardingOIDC,
	useTestOnboardingSMTP,
} from '../hooks/useOnboarding';
import {
	useOrganizations,
	useUpdateOrganization,
} from '../hooks/useOrganizations';
import {
	useCreateRepository,
	useRepositories,
	useTestConnection,
} from '../hooks/useRepositories';
import { useSchedules } from '../hooks/useSchedules';
import { AGENT_DOWNLOADS } from '../lib/constants';
import type {
	BackendConfig,
	CreateRegistrationCodeResponse,
	OnboardingStep,
	RepositoryType,
	TestRepositoryResponse,
} from '../lib/types';

const ONBOARDING_STEPS = [
	{
		id: 'welcome',
		label: 'Welcome',
		description: 'Get started with Keldris',
	},
	{
		id: 'license',
		label: 'License',
		description: 'Activate your license',
	},
	{
		id: 'organization',
		label: 'Organization',
		description: 'Create your first organization',
	},
	{
		id: 'oidc',
		label: 'Single Sign-On',
		description: 'Configure OIDC authentication (Pro+)',
	},
	{
		id: 'smtp',
		label: 'Email Setup',
		description: 'Configure email notifications (optional)',
	},
	{
		id: 'repository',
		label: 'Repository',
		description: 'Create a backup repository',
	},
	{
		id: 'agent',
		label: 'Install Agent',
		description: 'Install the backup agent',
	},
	{
		id: 'schedule',
		label: 'Schedule',
		description: 'Create your first backup schedule',
	},
	{
		id: 'verify',
		label: 'Verify',
		description: 'Verify your backup works',
	},
];

const DOCS_LINKS: Record<string, string> = {
	welcome: '/docs/getting-started',
	license: '/docs/getting-started',
	organization: '/docs/organizations',
	oidc: '/docs/getting-started',
	smtp: '/docs/notifications/email',
	repository: '/docs/repositories',
	agent: '/docs/agent-installation',
	schedule: '/docs/schedules',
	verify: '/docs/backup-verification',
};

interface StepProps {
	onComplete: () => void;
	onSkip?: () => void;
	isLoading?: boolean;
}

function WelcomeStep({ onComplete, onSkip, isLoading }: StepProps) {
	return (
		<div className="text-center py-8">
			<div className="w-20 h-20 bg-indigo-100 dark:bg-indigo-900/30 rounded-full flex items-center justify-center mx-auto mb-6">
				<svg
					aria-hidden="true"
					className="w-10 h-10 text-indigo-600"
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

			<h2 className="text-2xl font-bold text-gray-900 dark:text-gray-100 mb-4">
				Welcome to Keldris
			</h2>

			<p className="text-gray-600 dark:text-gray-400 max-w-md mx-auto mb-8">
				Keldris is your self-hosted backup solution. This wizard will guide you
				through setting up your first backup in just a few minutes.
			</p>

			<div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-4 mb-8 max-w-md mx-auto text-left">
				<h3 className="text-sm font-semibold text-blue-900 dark:text-blue-300 mb-2">
					What you'll set up:
				</h3>
				<ul className="text-sm text-blue-700 dark:text-blue-400 space-y-1">
					<li className="flex items-center gap-2">
						<svg
							aria-hidden="true"
							className="w-4 h-4"
							fill="currentColor"
							viewBox="0 0 20 20"
						>
							<path
								fillRule="evenodd"
								d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
								clipRule="evenodd"
							/>
						</svg>
						Organization for your team
					</li>
					<li className="flex items-center gap-2">
						<svg
							aria-hidden="true"
							className="w-4 h-4"
							fill="currentColor"
							viewBox="0 0 20 20"
						>
							<path
								fillRule="evenodd"
								d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
								clipRule="evenodd"
							/>
						</svg>
						Backup storage repository
					</li>
					<li className="flex items-center gap-2">
						<svg
							aria-hidden="true"
							className="w-4 h-4"
							fill="currentColor"
							viewBox="0 0 20 20"
						>
							<path
								fillRule="evenodd"
								d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
								clipRule="evenodd"
							/>
						</svg>
						Backup agent on your machine
					</li>
					<li className="flex items-center gap-2">
						<svg
							aria-hidden="true"
							className="w-4 h-4"
							fill="currentColor"
							viewBox="0 0 20 20"
						>
							<path
								fillRule="evenodd"
								d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
								clipRule="evenodd"
							/>
						</svg>
						Automated backup schedule
					</li>
				</ul>
			</div>

			<div className="flex justify-center gap-4">
				<button
					type="button"
					onClick={onSkip}
					className="px-4 py-2 text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-200"
				>
					Skip for now
				</button>
				<button
					type="button"
					onClick={onComplete}
					disabled={isLoading}
					className="px-6 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
				>
					{isLoading ? 'Loading...' : "Let's get started"}
				</button>
			</div>
		</div>
	);
}

function LicenseStep({ onComplete, onSkip, isLoading }: StepProps) {
	const { data: license } = useLicense();
	const activateMutation = useActivateLicense();
	const startTrialMutation = useStartTrial();
	const [licenseKey, setLicenseKey] = useState('');
	const [activateError, setActivateError] = useState('');
	const [trialEmail, setTrialEmail] = useState('');
	const [trialError, setTrialError] = useState('');

	const isActivated = license && license.tier !== 'free';

	const handleStartTrial = async () => {
		setTrialError('');
		try {
			await startTrialMutation.mutateAsync({ email: trialEmail, tier: 'pro' });
			setTrialEmail('');
			// Auto-advance after successful trial start
			onComplete();
		} catch (err) {
			setTrialError(
				err instanceof Error ? err.message : 'Failed to start trial',
			);
		}
	};

	const handleActivate = async () => {
		setActivateError('');
		try {
			await activateMutation.mutateAsync(licenseKey);
			setLicenseKey('');
			// Auto-advance after successful activation
			onComplete();
		} catch (err) {
			setActivateError(
				err instanceof Error ? err.message : 'Activation failed',
			);
		}
	};

	return (
		<div className="py-4">
			<h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100 mb-2">
				Activate Your License
			</h2>
			<p className="text-gray-600 dark:text-gray-400 mb-6">
				Enter your license key to unlock Pro or Enterprise features. You can
				also skip this step and use the free tier.
			</p>

			{isActivated ? (
				<div className="bg-green-50 border border-green-200 rounded-lg p-4 mb-6 dark:bg-green-900/20 dark:border-green-800">
					<div className="flex items-center gap-2 text-green-800 dark:text-green-300">
						<svg
							aria-hidden="true"
							className="w-5 h-5"
							fill="currentColor"
							viewBox="0 0 20 20"
						>
							<path
								fillRule="evenodd"
								d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
								clipRule="evenodd"
							/>
						</svg>
						<span className="font-medium">
							License activated! ({license.tier} tier)
						</span>
					</div>
				</div>
			) : (
				<div className="mb-6">
					<div className="rounded-lg border border-indigo-200 bg-indigo-50 p-4 dark:border-indigo-800 dark:bg-indigo-900/20">
						<div className="flex gap-3">
							<input
								type="text"
								value={licenseKey}
								onChange={(e) => setLicenseKey(e.target.value)}
								placeholder="Enter your license key..."
								className="flex-1 rounded-md border border-gray-300 px-3 py-2 text-sm font-mono focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100"
							/>
							<button
								type="button"
								onClick={handleActivate}
								disabled={!licenseKey.trim() || activateMutation.isPending}
								className="rounded-md bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50 disabled:cursor-not-allowed"
							>
								{activateMutation.isPending ? 'Activating...' : 'Activate'}
							</button>
						</div>
						{activateError && (
							<p className="mt-2 text-sm text-red-600 dark:text-red-400">
								{activateError}
							</p>
						)}
					</div>

					{/* Trial option */}
					<div className="mt-4 rounded-lg border border-emerald-200 bg-emerald-50 p-4 dark:border-emerald-800 dark:bg-emerald-900/20">
						<p className="font-medium text-emerald-800 dark:text-emerald-300 mb-2">
							Or start a free 14-day trial
						</p>
						<div className="flex gap-3">
							<input
								type="email"
								value={trialEmail}
								onChange={(e) => setTrialEmail(e.target.value)}
								placeholder="Enter your email..."
								className="flex-1 rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-emerald-500 focus:ring-1 focus:ring-emerald-500 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100"
							/>
							<button
								type="button"
								onClick={handleStartTrial}
								disabled={!trialEmail.trim() || startTrialMutation.isPending}
								className="rounded-md bg-emerald-600 px-4 py-2 text-sm font-medium text-white hover:bg-emerald-700 disabled:opacity-50 disabled:cursor-not-allowed"
							>
								{startTrialMutation.isPending ? 'Starting...' : 'Start Trial'}
							</button>
						</div>
						{trialError && (
							<p className="mt-2 text-sm text-red-600 dark:text-red-400">
								{trialError}
							</p>
						)}
					</div>

					<div className="mt-4 rounded-lg border border-gray-200 bg-gray-50 p-4 dark:border-gray-700 dark:bg-gray-800">
						<h3 className="font-medium text-gray-900 dark:text-gray-100 mb-2">
							Available tiers:
						</h3>
						<ul className="text-sm text-gray-600 dark:text-gray-400 space-y-1">
							<li>
								<strong>Free</strong> — Basic backups with limited agents and
								storage
							</li>
							<li>
								<strong>Pro</strong> — More agents, users, and advanced features
							</li>
							<li>
								<strong>Enterprise</strong> — Unlimited resources with SSO,
								audit logs, and more
							</li>
						</ul>
					</div>
				</div>
			)}

			<div className="flex justify-between items-center">
				<Link
					to={DOCS_LINKS.license}
					target="_blank"
					className="text-sm text-indigo-600 hover:text-indigo-700 dark:text-indigo-400 dark:hover:text-indigo-300"
				>
					Learn about license tiers
				</Link>
				<div className="flex gap-3">
					{!isActivated && (
						<button
							type="button"
							onClick={onSkip}
							className="px-4 py-2 text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-200"
						>
							Skip (use free tier)
						</button>
					)}
					<button
						type="button"
						onClick={onComplete}
						disabled={isLoading}
						className="px-6 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
					>
						{isLoading ? 'Saving...' : 'Continue'}
					</button>
				</div>
			</div>
		</div>
	);
}

function OrganizationStep({ onComplete, isLoading }: StepProps) {
	const queryClient = useQueryClient();
	const { data: organizations } = useOrganizations();
	const { data: license } = useLicense();
	const updateOrg = useUpdateOrganization();
	const hasOrganization = organizations && organizations.length > 0;

	// Find the current org (prefer default-slug org, fallback to first)
	const currentOrg =
		organizations?.find((o) => o.slug === 'default') ?? organizations?.[0];
	const licenseOrgName = license?.company || license?.customer_name;

	const [orgName, setOrgName] = useState('');
	const [renameError, setRenameError] = useState('');
	const [renamed, setRenamed] = useState(false);
	const [autoRenamed, setAutoRenamed] = useState(false);

	// Pre-fill org name from license
	useEffect(() => {
		if (!orgName && currentOrg) {
			setOrgName(licenseOrgName || currentOrg.name);
		}
	}, [licenseOrgName, currentOrg, orgName]);

	// Auto-rename the default org from license company name
	// biome-ignore lint/correctness/useExhaustiveDependencies: only run once when data is ready
	useEffect(() => {
		if (
			!autoRenamed &&
			currentOrg &&
			licenseOrgName &&
			currentOrg.name !== licenseOrgName
		) {
			setAutoRenamed(true);
			updateOrg
				.mutateAsync({
					id: currentOrg.id,
					data: { name: licenseOrgName },
				})
				.then(() => {
					setOrgName(licenseOrgName);
					setRenamed(true);
					queryClient.invalidateQueries({ queryKey: ['organizations'] });
				})
				.catch(() => {
					// Auto-rename failed silently; user can still rename manually
				});
		}
	}, [currentOrg, licenseOrgName, autoRenamed]);

	const handleRename = async () => {
		if (!currentOrg || !orgName.trim()) return;
		setRenameError('');
		try {
			await updateOrg.mutateAsync({
				id: currentOrg.id,
				data: { name: orgName.trim() },
			});
			setRenamed(true);
		} catch (err) {
			setRenameError(
				err instanceof Error ? err.message : 'Failed to rename organization',
			);
		}
	};

	// Display name: use the input value if changed, otherwise the current org name
	const displayName = renamed
		? orgName
		: currentOrg?.name || 'your organization';

	return (
		<div className="py-4">
			<h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100 mb-2">
				Your Organization
			</h2>
			<p className="text-gray-600 dark:text-gray-400 mb-6">
				Organizations help you manage backup resources and team access.
			</p>

			{hasOrganization ? (
				<div className="space-y-4 mb-6">
					<div className="bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg p-4">
						<div className="flex items-center gap-2 text-green-800 dark:text-green-300">
							<svg
								aria-hidden="true"
								className="w-5 h-5"
								fill="currentColor"
								viewBox="0 0 20 20"
							>
								<path
									fillRule="evenodd"
									d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
									clipRule="evenodd"
								/>
							</svg>
							<span className="font-medium">Organization ready!</span>
						</div>
						<p className="mt-1 text-sm text-green-700 dark:text-green-400">
							You're part of <strong>{displayName}</strong>.
						</p>
					</div>

					{currentOrg && !renamed && (
						<div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-4">
							<p className="text-sm font-medium text-blue-800 dark:text-blue-300 mb-2">
								Organization name
							</p>
							<p className="text-sm text-blue-700 dark:text-blue-400 mb-3">
								Set a name for your organization.
							</p>
							<div className="flex gap-3">
								<input
									type="text"
									value={orgName}
									onChange={(e) => setOrgName(e.target.value)}
									placeholder="Organization name..."
									className="flex-1 rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100"
								/>
								<button
									type="button"
									onClick={handleRename}
									disabled={
										!orgName.trim() ||
										orgName.trim() === currentOrg.name ||
										updateOrg.isPending
									}
									className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
								>
									{updateOrg.isPending ? 'Saving...' : 'Save'}
								</button>
							</div>
							{renameError && (
								<p className="mt-2 text-sm text-red-600 dark:text-red-400">
									{renameError}
								</p>
							)}
						</div>
					)}
				</div>
			) : (
				<div className="bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg p-4 mb-6">
					<p className="text-sm text-yellow-800 dark:text-yellow-300">
						You need to create an organization to continue. This is typically
						done automatically when you first sign in.
					</p>
				</div>
			)}

			<div className="flex justify-between items-center">
				<Link
					to={DOCS_LINKS.organization}
					target="_blank"
					className="text-sm text-indigo-600 hover:text-indigo-700 dark:text-indigo-400 dark:hover:text-indigo-300"
				>
					Learn more about organizations
				</Link>
				<button
					type="button"
					onClick={onComplete}
					disabled={!hasOrganization || isLoading}
					className="px-6 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
				>
					{isLoading ? 'Saving...' : 'Continue'}
				</button>
			</div>
		</div>
	);
}

interface OIDCStepProps extends StepProps {
	licenseTier?: string;
}

function OIDCStep({ onSkip, isLoading, licenseTier }: OIDCStepProps) {
	const completeOIDC = useCompleteOIDCStep();
	const testOIDC = useTestOnboardingOIDC();
	const [issuer, setIssuer] = useState('');
	const [clientId, setClientId] = useState('');
	const [clientSecret, setClientSecret] = useState('');
	const [redirectUrl, setRedirectUrl] = useState('');
	const [oidcError, setOidcError] = useState('');
	const [testResult, setTestResult] = useState<{
		success: boolean;
		message: string;
		provider_name?: string;
	} | null>(null);

	const isFree = !licenseTier || licenseTier === 'free';
	const skipAttempted = useRef(false);

	// Auto-skip if free tier (guard prevents infinite retry loop)
	useEffect(() => {
		if (isFree && onSkip && !skipAttempted.current) {
			skipAttempted.current = true;
			onSkip();
		}
	}, [isFree, onSkip]);

	const oidcData = {
		issuer,
		client_id: clientId,
		client_secret: clientSecret,
		redirect_url: redirectUrl,
	};

	const handleTest = async () => {
		setOidcError('');
		setTestResult(null);
		try {
			const result = await testOIDC.mutateAsync(oidcData);
			setTestResult(result);
			if (!result.success) {
				setOidcError(result.message);
			}
		} catch (err) {
			setOidcError(
				err instanceof Error ? err.message : 'Failed to test OIDC connection',
			);
		}
	};

	const handleSave = async () => {
		setOidcError('');
		try {
			await completeOIDC.mutateAsync(oidcData);
		} catch (err) {
			setOidcError(
				err instanceof Error ? err.message : 'Failed to configure OIDC',
			);
		}
	};

	// If free tier, show nothing while auto-skip runs
	if (isFree) {
		return (
			<div className="py-4 text-center">
				<p className="text-gray-600 dark:text-gray-400">
					OIDC is available on Pro and Enterprise tiers. Skipping...
				</p>
			</div>
		);
	}

	const canSave =
		issuer.trim() &&
		clientId.trim() &&
		clientSecret.trim() &&
		redirectUrl.trim();

	return (
		<div className="py-4">
			<h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100 mb-2">
				Configure Single Sign-On (OIDC)
			</h2>
			<p className="text-gray-600 dark:text-gray-400 mb-6">
				Set up OpenID Connect to enable SSO for your team. This step is optional
				and can be configured later.
			</p>

			<div className="space-y-4 mb-6">
				<div>
					<label
						htmlFor="oidc-issuer"
						className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
					>
						Issuer URL
					</label>
					<input
						id="oidc-issuer"
						type="url"
						value={issuer}
						onChange={(e) => setIssuer(e.target.value)}
						placeholder="https://accounts.google.com"
						className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100"
					/>
				</div>

				<div>
					<label
						htmlFor="oidc-client-id"
						className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
					>
						Client ID
					</label>
					<input
						id="oidc-client-id"
						type="text"
						value={clientId}
						onChange={(e) => setClientId(e.target.value)}
						placeholder="your-client-id"
						className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100"
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
						id="oidc-client-secret"
						type="password"
						value={clientSecret}
						onChange={(e) => setClientSecret(e.target.value)}
						placeholder="your-client-secret"
						className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100"
					/>
				</div>

				<div>
					<label
						htmlFor="oidc-redirect-url"
						className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
					>
						Redirect URL
					</label>
					<input
						id="oidc-redirect-url"
						type="url"
						value={redirectUrl}
						onChange={(e) => setRedirectUrl(e.target.value)}
						placeholder="https://keldris.example.com/auth/callback"
						className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100"
					/>
				</div>
			</div>

			{testResult?.success && (
				<div className="mb-4 rounded-lg border border-green-200 bg-green-50 p-3 dark:border-green-800 dark:bg-green-900/20">
					<p className="text-sm font-medium text-green-700 dark:text-green-400">
						Connection successful
					</p>
					<p className="text-sm text-green-600 dark:text-green-400">
						Provider: {testResult.provider_name || issuer}
					</p>
				</div>
			)}

			{oidcError && (
				<div className="mb-4 rounded-lg border border-red-200 bg-red-50 p-3 dark:border-red-800 dark:bg-red-900/20">
					<p className="text-sm text-red-600 dark:text-red-400">{oidcError}</p>
				</div>
			)}

			<div className="flex justify-between items-center">
				<Link
					to={DOCS_LINKS.oidc}
					target="_blank"
					className="text-sm text-indigo-600 hover:text-indigo-700 dark:text-indigo-400 dark:hover:text-indigo-300"
				>
					Learn about OIDC setup
				</Link>
				<div className="flex gap-3">
					<button
						type="button"
						onClick={onSkip}
						className="px-4 py-2 text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-200"
					>
						Skip for now
					</button>
					<button
						type="button"
						onClick={handleTest}
						disabled={!canSave || testOIDC.isPending}
						className="px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 transition-colors disabled:opacity-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-800"
					>
						{testOIDC.isPending ? 'Testing...' : 'Test Connection'}
					</button>
					<button
						type="button"
						onClick={handleSave}
						disabled={!canSave || completeOIDC.isPending || isLoading}
						className="px-6 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
					>
						{completeOIDC.isPending ? 'Saving...' : 'Save & Continue'}
					</button>
				</div>
			</div>
		</div>
	);
}

function SMTPStep({ onSkip, isLoading }: StepProps) {
	const completeSMTP = useCompleteSMTPStep();
	const testSMTP = useTestOnboardingSMTP();
	const [host, setHost] = useState('');
	const [port, setPort] = useState(587);
	const [username, setUsername] = useState('');
	const [password, setPassword] = useState('');
	const [fromEmail, setFromEmail] = useState('');
	const [fromName, setFromName] = useState('');
	const [encryption, setEncryption] = useState('starttls');
	const [smtpError, setSmtpError] = useState('');
	const [testResult, setTestResult] = useState<{
		success: boolean;
		message: string;
	} | null>(null);

	const smtpData = {
		host,
		port,
		username,
		password,
		from_email: fromEmail,
		from_name: fromName,
		encryption,
	};

	const handleTest = async () => {
		setSmtpError('');
		setTestResult(null);
		try {
			const result = await testSMTP.mutateAsync(smtpData);
			setTestResult(result);
			if (!result.success) {
				setSmtpError(result.message);
			}
		} catch (err) {
			setSmtpError(
				err instanceof Error ? err.message : 'Failed to test SMTP connection',
			);
		}
	};

	const handleSave = async () => {
		setSmtpError('');
		try {
			await completeSMTP.mutateAsync(smtpData);
		} catch (err) {
			setSmtpError(
				err instanceof Error ? err.message : 'Failed to configure SMTP',
			);
		}
	};

	const canSave = host.trim() && fromEmail.trim() && port > 0;

	return (
		<div className="py-4">
			<h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100 mb-2">
				Configure Email Notifications
			</h2>
			<p className="text-gray-600 dark:text-gray-400 mb-6">
				Set up SMTP to receive email notifications about your backups. This step
				is optional and can be configured later.
			</p>

			<div className="space-y-4 mb-6">
				<div className="grid grid-cols-2 gap-4">
					<div>
						<label
							htmlFor="smtp-host"
							className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
						>
							SMTP Host
						</label>
						<input
							id="smtp-host"
							type="text"
							value={host}
							onChange={(e) => setHost(e.target.value)}
							placeholder="smtp.gmail.com"
							className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100"
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
							id="smtp-port"
							type="number"
							value={port}
							onChange={(e) => setPort(Number(e.target.value))}
							placeholder="587"
							className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100"
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
							id="smtp-username"
							type="text"
							value={username}
							onChange={(e) => setUsername(e.target.value)}
							placeholder="user@example.com"
							className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100"
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
							id="smtp-password"
							type="password"
							value={password}
							onChange={(e) => setPassword(e.target.value)}
							placeholder="App password or SMTP password"
							className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100"
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
							id="smtp-from-email"
							type="email"
							value={fromEmail}
							onChange={(e) => setFromEmail(e.target.value)}
							placeholder="noreply@example.com"
							className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100"
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
							id="smtp-from-name"
							type="text"
							value={fromName}
							onChange={(e) => setFromName(e.target.value)}
							placeholder="Keldris Backups"
							className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100"
						/>
					</div>
				</div>

				<div>
					<label
						htmlFor="smtp-encryption"
						className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
					>
						Encryption
					</label>
					<select
						id="smtp-encryption"
						value={encryption}
						onChange={(e) => setEncryption(e.target.value)}
						className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100"
					>
						<option value="starttls">STARTTLS (port 587)</option>
						<option value="tls">TLS (port 465)</option>
						<option value="none">None (port 25)</option>
					</select>
				</div>
			</div>

			{testResult?.success && (
				<div className="mb-4 rounded-lg border border-green-200 bg-green-50 p-3 dark:border-green-800 dark:bg-green-900/20">
					<p className="text-sm font-medium text-green-700 dark:text-green-400">
						{testResult.message}
					</p>
				</div>
			)}

			{smtpError && (
				<div className="mb-4 rounded-lg border border-red-200 bg-red-50 p-3 dark:border-red-800 dark:bg-red-900/20">
					<p className="text-sm text-red-600 dark:text-red-400">{smtpError}</p>
				</div>
			)}

			<div className="flex justify-between items-center">
				<Link
					to={DOCS_LINKS.smtp}
					target="_blank"
					className="text-sm text-indigo-600 hover:text-indigo-700 dark:text-indigo-400 dark:hover:text-indigo-300"
				>
					Learn about email setup
				</Link>
				<div className="flex gap-3">
					<button
						type="button"
						onClick={onSkip}
						className="px-4 py-2 text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-200"
					>
						Skip for now
					</button>
					<button
						type="button"
						onClick={handleTest}
						disabled={!canSave || testSMTP.isPending}
						className="px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 transition-colors disabled:opacity-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-800"
					>
						{testSMTP.isPending ? 'Testing...' : 'Test Connection'}
					</button>
					<button
						type="button"
						onClick={handleSave}
						disabled={!canSave || completeSMTP.isPending || isLoading}
						className="px-6 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
					>
						{completeSMTP.isPending ? 'Saving...' : 'Save & Continue'}
					</button>
				</div>
			</div>
		</div>
	);
}

interface RepoFieldProps {
	label: string;
	id: string;
	value: string;
	onChange: (v: string) => void;
	placeholder?: string;
	required?: boolean;
	type?: 'text' | 'password' | 'number';
	helpText?: string;
}

function RepoField({
	label,
	id,
	value,
	onChange,
	placeholder,
	required,
	type = 'text',
	helpText,
}: RepoFieldProps) {
	return (
		<div>
			<label
				htmlFor={id}
				className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
			>
				{label}
				{required && <span className="text-red-500 ml-1">*</span>}
			</label>
			<input
				id={id}
				type={type}
				value={value}
				onChange={(e) => onChange(e.target.value)}
				placeholder={placeholder}
				className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100"
			/>
			{helpText && (
				<p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
					{helpText}
				</p>
			)}
		</div>
	);
}

function RepositoryStep({ onComplete, onSkip, isLoading }: StepProps) {
	const { data: repositories } = useRepositories();
	const createRepository = useCreateRepository();
	const testConnection = useTestConnection();
	const hasRepository = repositories && repositories.length > 0;

	const [name, setName] = useState('');
	const [type, setType] = useState<RepositoryType>('local');
	const [escrowEnabled, setEscrowEnabled] = useState(false);
	const [error, setError] = useState('');

	// Created result
	const [createdPassword, setCreatedPassword] = useState('');
	const [createdRepoName, setCreatedRepoName] = useState('');
	const [copied, setCopied] = useState(false);

	// Local
	const [localPath, setLocalPath] = useState('');

	// S3
	const [s3Bucket, setS3Bucket] = useState('');
	const [s3AccessKey, setS3AccessKey] = useState('');
	const [s3SecretKey, setS3SecretKey] = useState('');
	const [s3Region, setS3Region] = useState('');
	const [s3Endpoint, setS3Endpoint] = useState('');
	const [s3Prefix, setS3Prefix] = useState('');
	const [s3UseSsl, setS3UseSsl] = useState(true);

	// B2
	const [b2Bucket, setB2Bucket] = useState('');
	const [b2AccountId, setB2AccountId] = useState('');
	const [b2AppKey, setB2AppKey] = useState('');
	const [b2Prefix, setB2Prefix] = useState('');

	// SFTP
	const [sftpHost, setSftpHost] = useState('');
	const [sftpPort, setSftpPort] = useState('22');
	const [sftpUser, setSftpUser] = useState('');
	const [sftpPath, setSftpPath] = useState('');
	const [sftpPassword, setSftpPassword] = useState('');
	const [sftpPrivateKey, setSftpPrivateKey] = useState('');

	// REST
	const [restUrl, setRestUrl] = useState('');
	const [restUsername, setRestUsername] = useState('');
	const [restPassword, setRestPassword] = useState('');

	// Dropbox
	const [dropboxRemoteName, setDropboxRemoteName] = useState('');
	const [dropboxPath, setDropboxPath] = useState('');
	const [dropboxToken, setDropboxToken] = useState('');
	const [dropboxAppKey, setDropboxAppKey] = useState('');
	const [dropboxAppSecret, setDropboxAppSecret] = useState('');

	// Test connection
	const [testResult, setTestResult] = useState<TestRepositoryResponse | null>(
		null,
	);

	// biome-ignore lint/correctness/useExhaustiveDependencies: intentionally reset when type changes
	useEffect(() => {
		setTestResult(null);
	}, [type]);

	const buildConfig = (): BackendConfig => {
		switch (type) {
			case 'local':
				return { path: localPath };
			case 's3':
				return {
					endpoint: s3Endpoint || undefined,
					bucket: s3Bucket,
					prefix: s3Prefix || undefined,
					region: s3Region || undefined,
					access_key_id: s3AccessKey,
					secret_access_key: s3SecretKey,
					use_ssl: s3UseSsl,
				};
			case 'b2':
				return {
					bucket: b2Bucket,
					prefix: b2Prefix || undefined,
					account_id: b2AccountId,
					application_key: b2AppKey,
				};
			case 'sftp':
				return {
					host: sftpHost,
					port: sftpPort ? Number.parseInt(sftpPort, 10) : undefined,
					user: sftpUser,
					path: sftpPath,
					password: sftpPassword || undefined,
					private_key: sftpPrivateKey || undefined,
				};
			case 'rest':
				return {
					url: restUrl,
					username: restUsername || undefined,
					password: restPassword || undefined,
				};
			case 'dropbox':
				return {
					remote_name: dropboxRemoteName,
					path: dropboxPath || undefined,
					token: dropboxToken || undefined,
					app_key: dropboxAppKey || undefined,
					app_secret: dropboxAppSecret || undefined,
				};
			default:
				return { path: localPath };
		}
	};

	const canCreate = (() => {
		if (!name.trim()) return false;
		switch (type) {
			case 'local':
				return !!localPath.trim();
			case 's3':
				return (
					!!s3Bucket.trim() && !!s3AccessKey.trim() && !!s3SecretKey.trim()
				);
			case 'b2':
				return !!b2Bucket.trim() && !!b2AccountId.trim() && !!b2AppKey.trim();
			case 'sftp':
				return !!sftpHost.trim() && !!sftpUser.trim() && !!sftpPath.trim();
			case 'rest':
				return !!restUrl.trim();
			case 'dropbox':
				return !!dropboxRemoteName.trim();
			default:
				return false;
		}
	})();

	const handleTestConnection = async () => {
		setTestResult(null);
		setError('');
		try {
			const result = await testConnection.mutateAsync({
				type,
				config: buildConfig(),
			});
			setTestResult(result);
		} catch {
			setTestResult({
				success: false,
				message: 'Failed to test connection',
			});
		}
	};

	const handleCreate = async () => {
		setError('');
		try {
			const response = await createRepository.mutateAsync({
				name,
				type,
				config: buildConfig(),
				escrow_enabled: escrowEnabled,
			});
			setCreatedPassword(response.password);
			setCreatedRepoName(response.repository.name);
		} catch (err) {
			setError(
				err instanceof Error ? err.message : 'Failed to create repository',
			);
		}
	};

	const handleCopy = async () => {
		await navigator.clipboard.writeText(createdPassword);
		setCopied(true);
		setTimeout(() => setCopied(false), 2000);
	};

	// After creation: show password + continue
	if (createdPassword) {
		return (
			<div className="py-4">
				<h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100 mb-2">
					Repository Created
				</h2>

				<div className="bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg p-4 mb-4">
					<div className="flex items-center gap-2 text-green-800 dark:text-green-300">
						<svg
							aria-hidden="true"
							className="w-5 h-5"
							fill="currentColor"
							viewBox="0 0 20 20"
						>
							<path
								fillRule="evenodd"
								d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
								clipRule="evenodd"
							/>
						</svg>
						<span className="font-medium">
							Repository &ldquo;{createdRepoName}&rdquo; created successfully!
						</span>
					</div>
				</div>

				<div className="mb-4">
					<p className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
						Repository Password
					</p>
					<div className="bg-gray-50 dark:bg-gray-700 rounded-lg p-4">
						<div className="flex items-center justify-between gap-4">
							<code className="text-sm font-mono break-all flex-1">
								{createdPassword}
							</code>
							<button
								type="button"
								onClick={handleCopy}
								className="flex-shrink-0 px-3 py-1.5 text-sm bg-indigo-600 text-white rounded hover:bg-indigo-700 transition-colors"
							>
								{copied ? 'Copied!' : 'Copy'}
							</button>
						</div>
					</div>
				</div>

				<div className="bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-lg p-4 mb-6">
					<p className="text-sm text-amber-800 dark:text-amber-300">
						<strong>Important:</strong> This password is required to decrypt
						your backups. Store it securely — without it, your backup data
						cannot be recovered.
					</p>
				</div>

				<div className="flex justify-end">
					<button
						type="button"
						onClick={onComplete}
						disabled={isLoading}
						className="px-6 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
					>
						{isLoading ? 'Saving...' : 'Continue'}
					</button>
				</div>
			</div>
		);
	}

	// Already have a repository: show success + continue
	if (hasRepository) {
		return (
			<div className="py-4">
				<h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100 mb-2">
					Create a Backup Repository
				</h2>
				<p className="text-gray-600 dark:text-gray-400 mb-6">
					Repositories are where your backups are stored.
				</p>

				<div className="bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg p-4 mb-6">
					<div className="flex items-center gap-2 text-green-800 dark:text-green-300">
						<svg
							aria-hidden="true"
							className="w-5 h-5"
							fill="currentColor"
							viewBox="0 0 20 20"
						>
							<path
								fillRule="evenodd"
								d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
								clipRule="evenodd"
							/>
						</svg>
						<span className="font-medium">Repository configured!</span>
					</div>
					<p className="mt-1 text-sm text-green-700 dark:text-green-400">
						You have {repositories.length} repository(s) set up.
					</p>
				</div>

				<div className="flex justify-between items-center">
					<Link
						to={DOCS_LINKS.repository}
						target="_blank"
						className="text-sm text-indigo-600 hover:text-indigo-700 dark:text-indigo-400 dark:hover:text-indigo-300"
					>
						Learn about repository types
					</Link>
					<button
						type="button"
						onClick={onComplete}
						disabled={isLoading}
						className="px-6 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
					>
						{isLoading ? 'Saving...' : 'Continue'}
					</button>
				</div>
			</div>
		);
	}

	// Inline creation form
	const renderBackendFields = () => {
		switch (type) {
			case 'local':
				return (
					<RepoField
						label="Path"
						id="ob-local-path"
						value={localPath}
						onChange={setLocalPath}
						placeholder="/var/backups/restic"
						required
						helpText="Absolute path to the backup directory"
					/>
				);
			case 's3':
				return (
					<>
						<RepoField
							label="Bucket"
							id="ob-s3-bucket"
							value={s3Bucket}
							onChange={setS3Bucket}
							placeholder="my-backup-bucket"
							required
						/>
						<RepoField
							label="Access Key ID"
							id="ob-s3-access-key"
							value={s3AccessKey}
							onChange={setS3AccessKey}
							placeholder="AKIAIOSFODNN7EXAMPLE"
							required
						/>
						<RepoField
							label="Secret Access Key"
							id="ob-s3-secret-key"
							value={s3SecretKey}
							onChange={setS3SecretKey}
							type="password"
							required
						/>
						<RepoField
							label="Region"
							id="ob-s3-region"
							value={s3Region}
							onChange={setS3Region}
							placeholder="us-east-1"
							helpText="Required for AWS S3"
						/>
						<RepoField
							label="Endpoint"
							id="ob-s3-endpoint"
							value={s3Endpoint}
							onChange={setS3Endpoint}
							placeholder="minio.example.com:9000"
							helpText="For MinIO, Wasabi, or other S3-compatible services"
						/>
						<RepoField
							label="Prefix"
							id="ob-s3-prefix"
							value={s3Prefix}
							onChange={setS3Prefix}
							placeholder="backups/server1"
							helpText="Optional path prefix within the bucket"
						/>
						<div className="flex items-center gap-2">
							<input
								type="checkbox"
								id="ob-s3-use-ssl"
								checked={s3UseSsl}
								onChange={(e) => setS3UseSsl(e.target.checked)}
								className="rounded border-gray-300 text-indigo-600 focus:ring-indigo-500"
							/>
							<label
								htmlFor="ob-s3-use-ssl"
								className="text-sm text-gray-700 dark:text-gray-300"
							>
								Use SSL/TLS
							</label>
						</div>
					</>
				);
			case 'b2':
				return (
					<>
						<RepoField
							label="Bucket"
							id="ob-b2-bucket"
							value={b2Bucket}
							onChange={setB2Bucket}
							placeholder="my-backup-bucket"
							required
						/>
						<RepoField
							label="Account ID"
							id="ob-b2-account-id"
							value={b2AccountId}
							onChange={setB2AccountId}
							placeholder="0012345678abcdef"
							required
						/>
						<RepoField
							label="Application Key"
							id="ob-b2-app-key"
							value={b2AppKey}
							onChange={setB2AppKey}
							type="password"
							required
						/>
						<RepoField
							label="Prefix"
							id="ob-b2-prefix"
							value={b2Prefix}
							onChange={setB2Prefix}
							placeholder="backups/server1"
							helpText="Optional path prefix within the bucket"
						/>
					</>
				);
			case 'sftp':
				return (
					<>
						<div className="grid grid-cols-3 gap-3">
							<div className="col-span-2">
								<RepoField
									label="Host"
									id="ob-sftp-host"
									value={sftpHost}
									onChange={setSftpHost}
									placeholder="backup.example.com"
									required
								/>
							</div>
							<RepoField
								label="Port"
								id="ob-sftp-port"
								value={sftpPort}
								onChange={setSftpPort}
								placeholder="22"
								type="number"
							/>
						</div>
						<RepoField
							label="Username"
							id="ob-sftp-user"
							value={sftpUser}
							onChange={setSftpUser}
							placeholder="backup"
							required
						/>
						<RepoField
							label="Remote Path"
							id="ob-sftp-path"
							value={sftpPath}
							onChange={setSftpPath}
							placeholder="/var/backups/restic"
							required
							helpText="Absolute path on the remote server"
						/>
						<RepoField
							label="Password"
							id="ob-sftp-password"
							value={sftpPassword}
							onChange={setSftpPassword}
							type="password"
							helpText="Password or private key required"
						/>
						<div>
							<label
								htmlFor="ob-sftp-private-key"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								Private Key
							</label>
							<textarea
								id="ob-sftp-private-key"
								value={sftpPrivateKey}
								onChange={(e) => setSftpPrivateKey(e.target.value)}
								placeholder="-----BEGIN OPENSSH PRIVATE KEY-----"
								className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm font-mono focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100"
								rows={4}
							/>
							<p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
								Paste your SSH private key (PEM format)
							</p>
						</div>
					</>
				);
			case 'rest':
				return (
					<>
						<RepoField
							label="URL"
							id="ob-rest-url"
							value={restUrl}
							onChange={setRestUrl}
							placeholder="https://backup.example.com:8000"
							required
							helpText="URL of the Restic REST server"
						/>
						<RepoField
							label="Username"
							id="ob-rest-username"
							value={restUsername}
							onChange={setRestUsername}
							placeholder="backup"
							helpText="Optional authentication"
						/>
						<RepoField
							label="Password"
							id="ob-rest-password"
							value={restPassword}
							onChange={setRestPassword}
							type="password"
						/>
					</>
				);
			case 'dropbox':
				return (
					<>
						<RepoField
							label="Remote Name"
							id="ob-dropbox-remote-name"
							value={dropboxRemoteName}
							onChange={setDropboxRemoteName}
							placeholder="dropbox"
							required
							helpText="Name for the rclone remote configuration"
						/>
						<RepoField
							label="Path"
							id="ob-dropbox-path"
							value={dropboxPath}
							onChange={setDropboxPath}
							placeholder="/Backups/server1"
							helpText="Path within your Dropbox"
						/>
						<RepoField
							label="Token"
							id="ob-dropbox-token"
							value={dropboxToken}
							onChange={setDropboxToken}
							type="password"
							helpText="OAuth token from rclone config (optional if rclone is pre-configured)"
						/>
						<RepoField
							label="App Key"
							id="ob-dropbox-app-key"
							value={dropboxAppKey}
							onChange={setDropboxAppKey}
							helpText="Your Dropbox App Key (optional)"
						/>
						<RepoField
							label="App Secret"
							id="ob-dropbox-app-secret"
							value={dropboxAppSecret}
							onChange={setDropboxAppSecret}
							type="password"
							helpText="Your Dropbox App Secret (optional)"
						/>
						<p className="text-xs text-gray-500 dark:text-gray-400 bg-gray-50 dark:bg-gray-700 p-3 rounded-lg">
							Dropbox backend requires rclone to be installed on the agent. You
							can either pre-configure rclone with `rclone config` or provide
							the OAuth token here.
						</p>
					</>
				);
			default:
				return null;
		}
	};

	return (
		<div className="py-4">
			<h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100 mb-2">
				Create a Backup Repository
			</h2>
			<p className="text-gray-600 dark:text-gray-400 mb-6">
				Repositories are where your backups are stored. Configure your first
				repository using local storage, S3, or another supported backend.
			</p>

			<div className="space-y-4 mb-6">
				<RepoField
					label="Name"
					id="ob-repo-name"
					value={name}
					onChange={setName}
					placeholder="e.g., primary-backup"
					required
				/>

				<div>
					<label
						htmlFor="ob-repo-type"
						className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
					>
						Type
					</label>
					<select
						id="ob-repo-type"
						value={type}
						onChange={(e) => setType(e.target.value as RepositoryType)}
						className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100"
					>
						<option value="local">Local Filesystem</option>
						<option value="s3">Amazon S3 / MinIO / Wasabi</option>
						<option value="b2">Backblaze B2</option>
						<option value="sftp">SFTP</option>
						<option value="rest">Restic REST Server</option>
						<option value="dropbox">Dropbox (via rclone)</option>
					</select>
				</div>

				<hr className="border-gray-200 dark:border-gray-700" />

				{renderBackendFields()}

				<div className="flex items-start gap-3 pt-2">
					<input
						type="checkbox"
						id="ob-escrow-enabled"
						checked={escrowEnabled}
						onChange={(e) => setEscrowEnabled(e.target.checked)}
						className="mt-1 h-4 w-4 text-indigo-600 border-gray-300 rounded focus:ring-indigo-500"
					/>
					<div>
						<label
							htmlFor="ob-escrow-enabled"
							className="block text-sm font-medium text-gray-700 dark:text-gray-300"
						>
							Enable key escrow
						</label>
						<p className="text-xs text-gray-500 dark:text-gray-400">
							Store an encrypted copy of the password server-side for recovery
							by administrators
						</p>
					</div>
				</div>
			</div>

			{testResult && (
				<div
					className={`mb-4 p-3 rounded-lg text-sm ${
						testResult.success
							? 'bg-green-50 text-green-800 border border-green-200 dark:bg-green-900/20 dark:text-green-300 dark:border-green-800'
							: 'bg-red-50 text-red-800 border border-red-200 dark:bg-red-900/20 dark:text-red-300 dark:border-red-800'
					}`}
				>
					<div className="flex items-center gap-2">
						{testResult.success ? (
							<svg
								aria-hidden="true"
								className="w-5 h-5"
								fill="currentColor"
								viewBox="0 0 20 20"
							>
								<path
									fillRule="evenodd"
									d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
									clipRule="evenodd"
								/>
							</svg>
						) : (
							<svg
								aria-hidden="true"
								className="w-5 h-5"
								fill="currentColor"
								viewBox="0 0 20 20"
							>
								<path
									fillRule="evenodd"
									d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
									clipRule="evenodd"
								/>
							</svg>
						)}
						<span>{testResult.message}</span>
					</div>
				</div>
			)}

			{error && (
				<div className="mb-4 rounded-lg border border-red-200 bg-red-50 p-3 dark:border-red-800 dark:bg-red-900/20">
					<p className="text-sm text-red-600 dark:text-red-400">{error}</p>
				</div>
			)}

			<div className="flex justify-between items-center">
				<Link
					to={DOCS_LINKS.repository}
					target="_blank"
					className="text-sm text-indigo-600 hover:text-indigo-700 dark:text-indigo-400 dark:hover:text-indigo-300"
				>
					Learn about repository types
				</Link>
				<div className="flex gap-3">
					<button
						type="button"
						onClick={onSkip}
						className="px-4 py-2 text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-200"
					>
						Skip for now
					</button>
					<button
						type="button"
						onClick={handleTestConnection}
						disabled={!canCreate || testConnection.isPending}
						className="px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 transition-colors disabled:opacity-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-800"
					>
						{testConnection.isPending ? 'Testing...' : 'Test Connection'}
					</button>
					<button
						type="button"
						onClick={handleCreate}
						disabled={!canCreate || createRepository.isPending}
						className="px-6 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
					>
						{createRepository.isPending ? 'Creating...' : 'Create Repository'}
					</button>
				</div>
			</div>
		</div>
	);
}

function AgentStep({ onComplete, onSkip, isLoading }: StepProps) {
	const createCode = useCreateRegistrationCode();
	const [regCode, setRegCode] = useState<CreateRegistrationCodeResponse | null>(
		null,
	);
	const [activeTab, setActiveTab] = useState<'linux' | 'macos' | 'windows'>(
		'linux',
	);
	const [copied, setCopied] = useState(false);

	// Poll for agents once a code has been generated
	const { data: agents } = useAgents();
	const hasActiveAgent = agents?.some(
		(a) => a.status === 'active' || a.status === 'pending',
	);

	// Enable polling after code is generated
	const pollAgents = useAgents();
	// biome-ignore lint/correctness/useExhaustiveDependencies: re-fetch on interval when code is active
	useEffect(() => {
		if (!regCode) return;
		const interval = setInterval(() => {
			pollAgents.refetch();
		}, 5000);
		return () => clearInterval(interval);
	}, [regCode]);

	const handleGenerateCode = async () => {
		try {
			const result = await createCode.mutateAsync({});
			setRegCode(result);
		} catch {
			// mutation error handled by react-query
		}
	};

	const handleRegenerate = async () => {
		setRegCode(null);
		setCopied(false);
		handleGenerateCode();
	};

	const serverUrl = window.location.origin;

	const getInstallCommand = () => {
		if (!regCode) return '';
		switch (activeTab) {
			case 'linux':
				return `curl -fsSL ${AGENT_DOWNLOADS.installers.linux} | sudo KELDRIS_SERVER=${serverUrl} KELDRIS_CODE=${regCode.code} KELDRIS_ORG_ID=${regCode.org_id} bash`;
			case 'macos':
				return `curl -fsSL ${AGENT_DOWNLOADS.installers.macos} | KELDRIS_SERVER=${serverUrl} KELDRIS_CODE=${regCode.code} KELDRIS_ORG_ID=${regCode.org_id} bash`;
			case 'windows':
				return `$env:KELDRIS_SERVER='${serverUrl}'; $env:KELDRIS_CODE='${regCode.code}'; $env:KELDRIS_ORG_ID='${regCode.org_id}'; irm ${AGENT_DOWNLOADS.installers.windows} | iex`;
			default:
				return '';
		}
	};

	const handleCopy = async () => {
		await navigator.clipboard.writeText(getInstallCommand());
		setCopied(true);
		setTimeout(() => setCopied(false), 2000);
	};

	const getTimeRemaining = () => {
		if (!regCode) return '';
		const expires = new Date(regCode.expires_at);
		const now = new Date();
		const diffMs = expires.getTime() - now.getTime();
		if (diffMs <= 0) return 'Expired';
		const minutes = Math.ceil(diffMs / 60000);
		return `Code expires in ${minutes} minute${minutes !== 1 ? 's' : ''}`;
	};

	return (
		<div className="py-4">
			<h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100 mb-2">
				Install a Backup Agent
			</h2>
			<p className="text-gray-600 dark:text-gray-400 mb-6">
				Install the Keldris agent on the systems you want to back up. Generate
				an install command that will automatically register the agent.
			</p>

			{/* Agent detected banner */}
			{hasActiveAgent && (
				<div className="bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg p-4 mb-6">
					<div className="flex items-center gap-2 text-green-800 dark:text-green-300">
						<svg
							aria-hidden="true"
							className="w-5 h-5"
							fill="currentColor"
							viewBox="0 0 20 20"
						>
							<path
								fillRule="evenodd"
								d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
								clipRule="evenodd"
							/>
						</svg>
						<span className="font-medium">Agent detected!</span>
					</div>
					<p className="mt-1 text-sm text-green-700 dark:text-green-400">
						You have {agents?.length} agent(s) registered.
					</p>
				</div>
			)}

			{/* Generate Install Command section */}
			{!regCode && !hasActiveAgent && (
				<div className="mb-6">
					<button
						type="button"
						onClick={handleGenerateCode}
						disabled={createCode.isPending}
						className="px-6 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
					>
						{createCode.isPending
							? 'Generating...'
							: 'Generate Install Command'}
					</button>
				</div>
			)}

			{/* Platform-specific install commands */}
			{regCode && !hasActiveAgent && (
				<div className="mb-6">
					{/* Platform tabs */}
					<div className="flex border-b border-gray-200 dark:border-gray-700 mb-4">
						{(
							[
								{ key: 'linux', label: 'Linux' },
								{ key: 'macos', label: 'macOS' },
								{ key: 'windows', label: 'Windows' },
							] as const
						).map((tab) => (
							<button
								key={tab.key}
								type="button"
								onClick={() => {
									setActiveTab(tab.key);
									setCopied(false);
								}}
								className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
									activeTab === tab.key
										? 'border-indigo-600 text-indigo-600 dark:border-indigo-400 dark:text-indigo-400'
										: 'border-transparent text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300'
								}`}
							>
								{tab.label}
							</button>
						))}
					</div>

					{/* Command display */}
					<div className="relative bg-gray-900 rounded-lg p-4">
						<pre className="text-sm text-green-400 font-mono whitespace-pre-wrap break-all">
							{getInstallCommand()}
						</pre>
						<button
							type="button"
							onClick={handleCopy}
							className="absolute top-2 right-2 px-3 py-1 text-xs bg-gray-700 text-gray-300 rounded hover:bg-gray-600 transition-colors"
						>
							{copied ? 'Copied!' : 'Copy'}
						</button>
					</div>

					{/* Code expiry and regenerate */}
					<div className="flex items-center justify-between mt-3">
						<p className="text-sm text-gray-500 dark:text-gray-400">
							{getTimeRemaining()}
						</p>
						<button
							type="button"
							onClick={handleRegenerate}
							disabled={createCode.isPending}
							className="text-sm text-indigo-600 hover:text-indigo-700 dark:text-indigo-400 dark:hover:text-indigo-300"
						>
							{createCode.isPending ? 'Generating...' : 'Regenerate'}
						</button>
					</div>

					{/* Polling indicator */}
					<div className="mt-4 flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
						<div className="w-2 h-2 bg-indigo-500 rounded-full animate-pulse" />
						Waiting for agent to connect...
					</div>
				</div>
			)}

			<div className="flex justify-between items-center">
				<Link
					to={DOCS_LINKS.agent}
					target="_blank"
					className="text-sm text-indigo-600 hover:text-indigo-700 dark:text-indigo-400 dark:hover:text-indigo-300"
				>
					View installation guide
				</Link>
				<div className="flex gap-3">
					<button
						type="button"
						onClick={onSkip}
						className="px-4 py-2 text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-200"
					>
						Skip for now
					</button>
					<button
						type="button"
						onClick={onComplete}
						disabled={!hasActiveAgent || isLoading}
						className="px-6 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
					>
						{isLoading ? 'Saving...' : 'Continue'}
					</button>
				</div>
			</div>
		</div>
	);
}

function ScheduleStep({ onComplete, isLoading }: StepProps) {
	const { data: schedules } = useSchedules();
	const hasSchedule = schedules && schedules.length > 0;

	return (
		<div className="py-4">
			<h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100 mb-2">
				Create a Backup Schedule
			</h2>
			<p className="text-gray-600 dark:text-gray-400 mb-6">
				Schedules define when and what to back up. Set up automated backups to
				protect your data.
			</p>

			{hasSchedule ? (
				<div className="bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg p-4 mb-6">
					<div className="flex items-center gap-2 text-green-800 dark:text-green-300">
						<svg
							aria-hidden="true"
							className="w-5 h-5"
							fill="currentColor"
							viewBox="0 0 20 20"
						>
							<path
								fillRule="evenodd"
								d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
								clipRule="evenodd"
							/>
						</svg>
						<span className="font-medium">Schedule created!</span>
					</div>
					<p className="mt-1 text-sm text-green-700 dark:text-green-400">
						You have {schedules?.length} backup schedule(s) configured.
					</p>
				</div>
			) : (
				<div className="bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg p-4 mb-6">
					<p className="text-sm text-yellow-800 dark:text-yellow-300 mb-2">
						Create a schedule to start automated backups.
					</p>
					<Link
						to="/schedules"
						className="inline-flex items-center gap-2 text-sm font-medium text-indigo-600 hover:text-indigo-700 dark:text-indigo-400 dark:hover:text-indigo-300"
					>
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
								d="M12 4v16m8-8H4"
							/>
						</svg>
						Create Schedule
					</Link>
				</div>
			)}

			<div className="flex justify-between items-center">
				<Link
					to={DOCS_LINKS.schedule}
					target="_blank"
					className="text-sm text-indigo-600 hover:text-indigo-700 dark:text-indigo-400 dark:hover:text-indigo-300"
				>
					Learn about backup schedules
				</Link>
				<button
					type="button"
					onClick={onComplete}
					disabled={!hasSchedule || isLoading}
					className="px-6 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
				>
					{isLoading ? 'Saving...' : 'Continue'}
				</button>
			</div>
		</div>
	);
}

function VerifyStep({ onComplete, isLoading }: StepProps) {
	return (
		<div className="py-4">
			<h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100 mb-2">
				Verify Your Backup Works
			</h2>
			<p className="text-gray-600 dark:text-gray-400 mb-6">
				Run a test backup to make sure everything is configured correctly.
			</p>

			<div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-4 mb-6">
				<h3 className="font-medium text-blue-900 dark:text-blue-300 mb-2">
					To verify your backup:
				</h3>
				<ol className="text-sm text-blue-700 dark:text-blue-400 space-y-2 list-decimal list-inside">
					<li>Go to the Schedules page</li>
					<li>Click "Run Now" on your schedule to trigger a manual backup</li>
					<li>Check the Backups page to see the backup status</li>
					<li>Once successful, return here and click "Complete Setup"</li>
				</ol>
			</div>

			<div className="flex gap-3 mb-6">
				<Link
					to="/schedules"
					className="inline-flex items-center gap-2 px-4 py-2 text-sm font-medium text-indigo-600 border border-indigo-600 rounded-lg hover:bg-indigo-50 dark:text-indigo-400 dark:border-indigo-400 dark:hover:bg-indigo-900/30"
				>
					Go to Schedules
				</Link>
				<Link
					to="/backups"
					className="inline-flex items-center gap-2 px-4 py-2 text-sm font-medium text-gray-700 border border-gray-300 rounded-lg hover:bg-gray-50 dark:text-gray-200 dark:border-gray-600 dark:hover:bg-gray-700"
				>
					View Backups
				</Link>
			</div>

			<div className="flex justify-between items-center">
				<Link
					to={DOCS_LINKS.verify}
					target="_blank"
					className="text-sm text-indigo-600 hover:text-indigo-700 dark:text-indigo-400 dark:hover:text-indigo-300"
				>
					Learn about backup verification
				</Link>
				<button
					type="button"
					onClick={onComplete}
					disabled={isLoading}
					className="px-6 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
				>
					{isLoading ? 'Completing...' : 'Complete Setup'}
				</button>
			</div>
		</div>
	);
}

function CompleteStep() {
	const [showConfetti, setShowConfetti] = useState(true);

	useEffect(() => {
		const timer = setTimeout(() => setShowConfetti(false), 5000);
		return () => clearTimeout(timer);
	}, []);

	return (
		<div className="text-center py-8 relative">
			{showConfetti && <Confetti />}

			<div className="w-20 h-20 bg-green-100 dark:bg-green-900/30 rounded-full flex items-center justify-center mx-auto mb-6">
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

			<h2 className="text-2xl font-bold text-gray-900 dark:text-gray-100 mb-4">
				Setup Complete!
			</h2>

			<p className="text-gray-600 dark:text-gray-400 max-w-md mx-auto mb-8">
				Congratulations! You've successfully set up Keldris. Your backups are
				now configured and ready to protect your data.
			</p>

			<div className="bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg p-4 mb-8 max-w-md mx-auto text-left">
				<h3 className="text-sm font-semibold text-green-900 dark:text-green-300 mb-2">
					What's next?
				</h3>
				<ul className="text-sm text-green-700 dark:text-green-400 space-y-1">
					<li>- Monitor your backups on the Dashboard</li>
					<li>- Add more agents to back up additional systems</li>
					<li>- Configure alerts for backup failures</li>
					<li>- Invite team members to your organization</li>
				</ul>
			</div>

			<Link
				to="/"
				className="inline-flex items-center gap-2 px-6 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors"
			>
				Go to Dashboard
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
			</Link>
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

export function Onboarding() {
	const navigate = useNavigate();
	const { data: onboardingStatus, isLoading: statusLoading } =
		useOnboardingStatus();
	const completeStep = useCompleteOnboardingStep();
	const skipOnboarding = useSkipOnboarding();

	const currentStep = onboardingStatus?.current_step ?? 'welcome';
	const completedSteps = onboardingStatus?.completed_steps ?? [];

	const handleCompleteStep = useCallback(
		(step: OnboardingStep) => {
			completeStep.mutate(step);
		},
		[completeStep.mutate],
	);

	const handleSkip = () => {
		skipOnboarding.mutate(undefined, {
			onSuccess: () => navigate('/'),
		});
	};

	// Redirect to dashboard if onboarding is complete
	useEffect(() => {
		if (onboardingStatus?.is_complete && currentStep !== 'complete') {
			navigate('/');
		}
	}, [onboardingStatus?.is_complete, currentStep, navigate]);

	if (statusLoading) {
		return (
			<div className="min-h-screen flex items-center justify-center">
				<div className="w-8 h-8 border-4 border-indigo-200 border-t-indigo-600 rounded-full animate-spin" />
			</div>
		);
	}

	const renderStep = () => {
		const isLoading = completeStep.isPending;

		switch (currentStep) {
			case 'welcome':
				return (
					<WelcomeStep
						onComplete={() => handleCompleteStep('welcome')}
						onSkip={handleSkip}
						isLoading={isLoading}
					/>
				);
			case 'license':
				return (
					<LicenseStep
						onComplete={() => handleCompleteStep('license')}
						onSkip={() => handleCompleteStep('license')}
						isLoading={isLoading}
					/>
				);
			case 'organization':
				return (
					<OrganizationStep
						onComplete={() => handleCompleteStep('organization')}
						isLoading={isLoading}
					/>
				);
			case 'oidc':
				return (
					<OIDCStep
						onComplete={() => handleCompleteStep('oidc')}
						onSkip={() => handleCompleteStep('oidc')}
						isLoading={isLoading}
						licenseTier={onboardingStatus?.license_tier}
					/>
				);
			case 'smtp':
				return (
					<SMTPStep
						onComplete={() => handleCompleteStep('smtp')}
						onSkip={() => handleCompleteStep('smtp')}
						isLoading={isLoading}
					/>
				);
			case 'repository':
				return (
					<RepositoryStep
						onComplete={() => handleCompleteStep('repository')}
						onSkip={() => handleCompleteStep('repository')}
						isLoading={isLoading}
					/>
				);
			case 'agent':
				return (
					<AgentStep
						onComplete={() => handleCompleteStep('agent')}
						onSkip={() => handleCompleteStep('agent')}
						isLoading={isLoading}
					/>
				);
			case 'schedule':
				return (
					<ScheduleStep
						onComplete={() => handleCompleteStep('schedule')}
						isLoading={isLoading}
					/>
				);
			case 'verify':
				return (
					<VerifyStep
						onComplete={() => handleCompleteStep('verify')}
						isLoading={isLoading}
					/>
				);
			case 'complete':
				return <CompleteStep />;
			default:
				return null;
		}
	};

	return (
		<div className="min-h-[calc(100vh-8rem)]">
			<div className="max-w-4xl mx-auto">
				{/* Header */}
				<div className="mb-8">
					<h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">
						Getting Started
					</h1>
					<p className="text-gray-600 dark:text-gray-400 mt-1">
						Complete these steps to set up your first backup
					</p>
				</div>

				<div className="flex gap-8">
					{/* Sidebar Stepper */}
					<div className="w-64 shrink-0">
						<div className="sticky top-6">
							<VerticalStepper
								steps={ONBOARDING_STEPS}
								currentStep={currentStep}
								completedSteps={completedSteps}
							/>

							{currentStep !== 'complete' && (
								<div className="mt-8 pt-4 border-t border-gray-200 dark:border-gray-700">
									<button
										type="button"
										onClick={handleSkip}
										disabled={skipOnboarding.isPending}
										className="text-sm text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
									>
										{skipOnboarding.isPending
											? 'Skipping...'
											: 'Skip setup wizard'}
									</button>
								</div>
							)}
						</div>
					</div>

					{/* Main Content */}
					<div className="flex-1">
						<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
							{renderStep()}
						</div>
					</div>
				</div>
			</div>
		</div>
	);
}

export default Onboarding;
