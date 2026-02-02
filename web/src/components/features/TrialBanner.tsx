import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useStartTrial, useTrialStatus } from '../../hooks/useTrial';
import type { TrialInfo } from '../../lib/types';

function getTrialBannerStyles(daysRemaining: number): {
	bg: string;
	borderColor: string;
	textColor: string;
} {
	if (daysRemaining <= 0) {
		return {
			bg: 'bg-red-50',
			borderColor: 'border-red-200',
			textColor: 'text-red-800',
		};
	}
	if (daysRemaining <= 3) {
		return {
			bg: 'bg-amber-50',
			borderColor: 'border-amber-200',
			textColor: 'text-amber-800',
		};
	}
	if (daysRemaining <= 7) {
		return {
			bg: 'bg-yellow-50',
			borderColor: 'border-yellow-200',
			textColor: 'text-yellow-800',
		};
	}
	return {
		bg: 'bg-blue-50',
		borderColor: 'border-blue-200',
		textColor: 'text-blue-800',
	};
}

function ActiveTrialBanner({ info }: { info: TrialInfo }) {
	const { t } = useTranslation();
	const styles = getTrialBannerStyles(info.days_remaining);

	const getMessage = () => {
		if (info.days_remaining <= 0) {
			return t(
				'trial.expiredMessage',
				'Your Pro trial has expired. Upgrade to continue using Pro features.',
			);
		}
		if (info.days_remaining === 1) {
			return t(
				'trial.lastDayMessage',
				'Your Pro trial expires tomorrow. Upgrade now to keep Pro features.',
			);
		}
		if (info.days_remaining <= 3) {
			return t(
				'trial.expiringMessage',
				'Your Pro trial expires in {{days}} days. Upgrade to keep Pro features.',
				{ days: info.days_remaining },
			);
		}
		return t('trial.activeMessage', 'Pro trial: {{days}} days remaining', {
			days: info.days_remaining,
		});
	};

	return (
		<div className={`${styles.bg} border-b ${styles.borderColor}`}>
			<div className="px-4 py-2">
				<div className="flex items-center justify-between gap-3">
					<div className="flex items-center gap-3 flex-1 min-w-0">
						<svg
							className={`w-5 h-5 ${styles.textColor} flex-shrink-0`}
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
							aria-hidden="true"
						>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
							/>
						</svg>
						<span className={`${styles.textColor} font-medium text-sm`}>
							{getMessage()}
						</span>
					</div>
					<a
						href="/settings/billing"
						className="flex-shrink-0 text-sm font-medium text-blue-600 hover:text-blue-500 dark:text-blue-400"
					>
						{t('trial.upgradeNow', 'Upgrade Now')}
					</a>
				</div>
			</div>
		</div>
	);
}

function ExpiredTrialBanner() {
	const { t } = useTranslation();

	return (
		<div className="bg-red-50 border-b border-red-200 dark:bg-red-900/20 dark:border-red-800">
			<div className="px-4 py-3">
				<div className="flex items-center justify-between gap-3">
					<div className="flex items-center gap-3 flex-1 min-w-0">
						<svg
							className="w-5 h-5 text-red-600 dark:text-red-400 flex-shrink-0"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
							aria-hidden="true"
						>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
							/>
						</svg>
						<div className="min-w-0">
							<span className="text-red-800 dark:text-red-200 font-medium text-sm block">
								{t('trial.expiredTitle', 'Pro Trial Expired')}
							</span>
							<span className="text-red-600 dark:text-red-300 text-xs">
								{t(
									'trial.expiredDescription',
									'Some features are now limited. Upgrade to restore full access.',
								)}
							</span>
						</div>
					</div>
					<div className="flex items-center gap-2 flex-shrink-0">
						<a
							href="/settings/billing"
							className="px-3 py-1.5 text-sm font-medium text-white bg-red-600 rounded-md hover:bg-red-700"
						>
							{t('trial.upgradeToPro', 'Upgrade to Pro')}
						</a>
					</div>
				</div>
			</div>
		</div>
	);
}

function StartTrialBanner() {
	const { t } = useTranslation();
	const [email, setEmail] = useState('');
	const [showForm, setShowForm] = useState(false);
	const startTrialMutation = useStartTrial();

	const handleStartTrial = (e: React.FormEvent) => {
		e.preventDefault();
		if (email) {
			startTrialMutation.mutate({ email });
		}
	};

	if (!showForm) {
		return (
			<div className="bg-gradient-to-r from-blue-50 to-indigo-50 border-b border-blue-200 dark:from-blue-900/20 dark:to-indigo-900/20 dark:border-blue-800">
				<div className="px-4 py-2">
					<div className="flex items-center justify-between gap-3">
						<div className="flex items-center gap-3 flex-1 min-w-0">
							<svg
								className="w-5 h-5 text-blue-600 dark:text-blue-400 flex-shrink-0"
								fill="none"
								stroke="currentColor"
								viewBox="0 0 24 24"
								aria-hidden="true"
							>
								<path
									strokeLinecap="round"
									strokeLinejoin="round"
									strokeWidth={2}
									d="M5 3v4M3 5h4M6 17v4m-2-2h4m5-16l2.286 6.857L21 12l-5.714 2.143L13 21l-2.286-6.857L5 12l5.714-2.143L13 3z"
								/>
							</svg>
							<span className="text-blue-800 dark:text-blue-200 font-medium text-sm">
								{t('trial.promoMessage', 'Try Pro features free for 30 days')}
							</span>
						</div>
						<button
							type="button"
							onClick={() => setShowForm(true)}
							className="flex-shrink-0 px-3 py-1 text-sm font-medium text-white bg-blue-600 rounded-md hover:bg-blue-700"
						>
							{t('trial.startFreeTrial', 'Start Free Trial')}
						</button>
					</div>
				</div>
			</div>
		);
	}

	return (
		<div className="bg-gradient-to-r from-blue-50 to-indigo-50 border-b border-blue-200 dark:from-blue-900/20 dark:to-indigo-900/20 dark:border-blue-800">
			<div className="px-4 py-3">
				<form onSubmit={handleStartTrial} className="flex items-center gap-3">
					<svg
						className="w-5 h-5 text-blue-600 dark:text-blue-400 flex-shrink-0"
						fill="none"
						stroke="currentColor"
						viewBox="0 0 24 24"
						aria-hidden="true"
					>
						<path
							strokeLinecap="round"
							strokeLinejoin="round"
							strokeWidth={2}
							d="M5 3v4M3 5h4M6 17v4m-2-2h4m5-16l2.286 6.857L21 12l-5.714 2.143L13 21l-2.286-6.857L5 12l5.714-2.143L13 3z"
						/>
					</svg>
					<span className="text-blue-800 dark:text-blue-200 font-medium text-sm">
						{t(
							'trial.enterEmail',
							'Enter your email to start your 30-day Pro trial:',
						)}
					</span>
					<input
						type="email"
						value={email}
						onChange={(e) => setEmail(e.target.value)}
						placeholder="you@example.com"
						required
						className="flex-1 max-w-xs px-3 py-1.5 text-sm border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-800 dark:border-gray-600 dark:text-white"
					/>
					<button
						type="submit"
						disabled={startTrialMutation.isPending || !email}
						className="px-3 py-1.5 text-sm font-medium text-white bg-blue-600 rounded-md hover:bg-blue-700 disabled:opacity-50"
					>
						{startTrialMutation.isPending
							? t('common.starting', 'Starting...')
							: t('trial.startTrial', 'Start Trial')}
					</button>
					<button
						type="button"
						onClick={() => setShowForm(false)}
						className="text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
						aria-label="Close trial form"
					>
						<svg
							className="w-5 h-5"
							fill="currentColor"
							viewBox="0 0 20 20"
							aria-hidden="true"
						>
							<path
								fillRule="evenodd"
								d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
								clipRule="evenodd"
							/>
						</svg>
					</button>
				</form>
				{startTrialMutation.isError && (
					<p className="mt-2 text-sm text-red-600 dark:text-red-400">
						{t('trial.startError', 'Failed to start trial. Please try again.')}
					</p>
				)}
			</div>
		</div>
	);
}

export function TrialBanner() {
	const { data: trialInfo, isLoading, isError } = useTrialStatus();

	// Don't show anything while loading or on error
	if (isLoading || isError) {
		return null;
	}

	// No trial info means we're not logged in or no org selected
	if (!trialInfo) {
		return null;
	}

	// Already on a paid plan - no banner needed
	if (trialInfo.plan_tier === 'pro' || trialInfo.plan_tier === 'enterprise') {
		return null;
	}

	// Trial is converted - no banner needed
	if (trialInfo.trial_status === 'converted') {
		return null;
	}

	// Trial expired
	if (trialInfo.trial_status === 'expired') {
		return <ExpiredTrialBanner />;
	}

	// Active trial - show days remaining
	if (trialInfo.trial_status === 'active' && trialInfo.is_trial_active) {
		return <ActiveTrialBanner info={trialInfo} />;
	}

	// No trial started yet - show start trial prompt
	if (trialInfo.trial_status === 'none') {
		return <StartTrialBanner />;
	}

	return null;
}
