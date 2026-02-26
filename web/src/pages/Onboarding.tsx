import { useEffect, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { AgentDownloads } from '../components/features/AgentDownloads';
import { VerticalStepper } from '../components/ui/Stepper';
import { useAgents } from '../hooks/useAgents';
import {
	useActivateLicense,
	useLicense,
	useStartTrial,
} from '../hooks/useLicense';
import {
	useCompleteOnboardingStep,
	useOnboardingStatus,
	useSkipOnboarding,
} from '../hooks/useOnboarding';
import {
	useOrganizations,
	useUpdateOrganization,
} from '../hooks/useOrganizations';
import { useRepositories } from '../hooks/useRepositories';
import { useSchedules } from '../hooks/useSchedules';
import type { OnboardingStep } from '../lib/types';

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
	license: '/docs/licensing',
	organization: '/docs/organizations',
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
				<a
					href={DOCS_LINKS.license}
					target="_blank"
					rel="noopener noreferrer"
					className="text-sm text-indigo-600 hover:text-indigo-700 dark:text-indigo-400 dark:hover:text-indigo-300"
				>
					Learn about license tiers
				</a>
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
	const { data: organizations } = useOrganizations();
	const { data: license } = useLicense();
	const updateOrg = useUpdateOrganization();
	const hasOrganization = organizations && organizations.length > 0;

	// Check if the org still has the default name and can be updated from license
	const defaultOrg = organizations?.find(
		(o) =>
			o.name === 'Default' ||
			o.name === 'Default Organization' ||
			o.name === 'default',
	);
	const licenseOrgName = license?.customer_name;
	const canRename = defaultOrg && licenseOrgName && defaultOrg.name !== licenseOrgName;

	const [orgName, setOrgName] = useState('');
	const [renameError, setRenameError] = useState('');

	// Pre-fill org name from license when available
	useEffect(() => {
		if (licenseOrgName && !orgName) {
			setOrgName(licenseOrgName);
		}
	}, [licenseOrgName, orgName]);

	const handleRename = async () => {
		if (!defaultOrg || !orgName.trim()) return;
		setRenameError('');
		try {
			await updateOrg.mutateAsync({
				id: defaultOrg.id,
				data: { name: orgName.trim() },
			});
		} catch (err) {
			setRenameError(
				err instanceof Error ? err.message : 'Failed to rename organization',
			);
		}
	};

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
							<span className="font-medium">
								Organization ready!
							</span>
						</div>
						<p className="mt-1 text-sm text-green-700 dark:text-green-400">
							You're part of{' '}
							<strong>{organizations.map((o) => o.name).join(', ')}</strong>.
						</p>
					</div>

					{canRename && (
						<div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-4">
							<p className="text-sm font-medium text-blue-800 dark:text-blue-300 mb-2">
								Update organization name
							</p>
							<p className="text-sm text-blue-700 dark:text-blue-400 mb-3">
								Your organization is currently named "{defaultOrg.name}". Would you like to rename it?
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
									disabled={!orgName.trim() || updateOrg.isPending}
									className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
								>
									{updateOrg.isPending ? 'Renaming...' : 'Rename'}
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
				<a
					href={DOCS_LINKS.organization}
					target="_blank"
					rel="noopener noreferrer"
					className="text-sm text-indigo-600 hover:text-indigo-700 dark:text-indigo-400 dark:hover:text-indigo-300"
				>
					Learn more about organizations
				</a>
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

function SMTPStep({ onComplete, onSkip, isLoading }: StepProps) {
	return (
		<div className="py-4">
			<h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100 mb-2">
				Configure Email Notifications
			</h2>
			<p className="text-gray-600 dark:text-gray-400 mb-6">
				Set up email notifications to stay informed about your backups. This
				step is optional and can be configured later.
			</p>

			<div className="bg-gray-50 dark:bg-gray-700/50 border border-gray-200 dark:border-gray-600 rounded-lg p-4 mb-6">
				<h3 className="font-medium text-gray-900 dark:text-gray-100 mb-2">
					Email notifications can alert you when:
				</h3>
				<ul className="text-sm text-gray-600 dark:text-gray-400 space-y-1">
					<li>- Backups complete successfully</li>
					<li>- Backups fail or encounter errors</li>
					<li>- Agents go offline</li>
					<li>- Storage reaches capacity thresholds</li>
				</ul>
			</div>

			<div className="flex justify-between items-center">
				<div className="flex items-center gap-4">
					<a
						href={DOCS_LINKS.smtp}
						target="_blank"
						rel="noopener noreferrer"
						className="text-sm text-indigo-600 hover:text-indigo-700 dark:text-indigo-400 dark:hover:text-indigo-300"
					>
						Learn about email setup
					</a>
					<Link
						to="/notifications"
						className="text-sm text-indigo-600 hover:text-indigo-700 dark:text-indigo-400 dark:hover:text-indigo-300"
					>
						Configure email now
					</Link>
				</div>
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

function RepositoryStep({ onComplete, isLoading }: StepProps) {
	const { data: repositories } = useRepositories();
	const hasRepository = repositories && repositories.length > 0;

	return (
		<div className="py-4">
			<h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100 mb-2">
				Create a Backup Repository
			</h2>
			<p className="text-gray-600 dark:text-gray-400 mb-6">
				Repositories are where your backups are stored. Create one using local
				storage, S3, or other supported backends.
			</p>

			{hasRepository ? (
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
			) : (
				<div className="bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg p-4 mb-6">
					<p className="text-sm text-yellow-800 dark:text-yellow-300 mb-2">
						You need to create a repository to store your backups.
					</p>
					<Link
						to="/repositories"
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
						Create Repository
					</Link>
				</div>
			)}

			<div className="flex justify-between items-center">
				<a
					href={DOCS_LINKS.repository}
					target="_blank"
					rel="noopener noreferrer"
					className="text-sm text-indigo-600 hover:text-indigo-700 dark:text-indigo-400 dark:hover:text-indigo-300"
				>
					Learn about repository types
				</a>
				<button
					type="button"
					onClick={onComplete}
					disabled={!hasRepository || isLoading}
					className="px-6 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
				>
					{isLoading ? 'Saving...' : 'Continue'}
				</button>
			</div>
		</div>
	);
}

function AgentStep({ onComplete, isLoading }: StepProps) {
	const { data: agents } = useAgents();
	const hasActiveAgent = agents?.some(
		(a) => a.status === 'active' || a.status === 'pending',
	);

	return (
		<div className="py-4">
			<h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100 mb-2">
				Install a Backup Agent
			</h2>
			<p className="text-gray-600 dark:text-gray-400 mb-6">
				Install the Keldris agent on the systems you want to back up.
			</p>

			{hasActiveAgent ? (
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
						<span className="font-medium">Agent installed!</span>
					</div>
					<p className="mt-1 text-sm text-green-700 dark:text-green-400">
						You have {agents?.length} agent(s) registered.
					</p>
				</div>
			) : (
				<div className="mb-6">
					<AgentDownloads showInstallCommands={true} />
				</div>
			)}

			<div className="flex justify-between items-center">
				<a
					href={DOCS_LINKS.agent}
					target="_blank"
					rel="noopener noreferrer"
					className="text-sm text-indigo-600 hover:text-indigo-700 dark:text-indigo-400 dark:hover:text-indigo-300"
				>
					View installation guide
				</a>
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
				<a
					href={DOCS_LINKS.schedule}
					target="_blank"
					rel="noopener noreferrer"
					className="text-sm text-indigo-600 hover:text-indigo-700 dark:text-indigo-400 dark:hover:text-indigo-300"
				>
					Learn about backup schedules
				</a>
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
				<a
					href={DOCS_LINKS.verify}
					target="_blank"
					rel="noopener noreferrer"
					className="text-sm text-indigo-600 hover:text-indigo-700 dark:text-indigo-400 dark:hover:text-indigo-300"
				>
					Learn about backup verification
				</a>
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

	const handleCompleteStep = (step: OnboardingStep) => {
		completeStep.mutate(step);
	};

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
						isLoading={isLoading}
					/>
				);
			case 'agent':
				return (
					<AgentStep
						onComplete={() => handleCompleteStep('agent')}
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
