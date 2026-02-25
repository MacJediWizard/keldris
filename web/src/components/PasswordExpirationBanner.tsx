import { useState } from 'react';
import { usePasswordExpiration } from '../hooks/usePasswordPolicy';
import { ChangePasswordForm } from './ChangePasswordForm';

export function PasswordExpirationBanner() {
	const { data: expirationInfo, isLoading } = usePasswordExpiration();
	const [showChangePassword, setShowChangePassword] = useState(false);
	const [dismissed, setDismissed] = useState(false);

	if (isLoading || dismissed) {
		return null;
	}

	if (!expirationInfo) {
		return null;
	}

	// Don't show anything if password doesn't expire or isn't close to expiring
	if (
		!expirationInfo.is_expired &&
		!expirationInfo.must_change_now &&
		(expirationInfo.days_until_expiry === undefined ||
			expirationInfo.days_until_expiry > expirationInfo.warn_days_remaining)
	) {
		return null;
	}

	// If password is expired or must change, show blocking modal
	if (expirationInfo.is_expired || expirationInfo.must_change_now) {
		return (
			<div className="fixed inset-0 bg-gray-500 dark:bg-gray-900 bg-opacity-75 dark:bg-opacity-75 flex items-center justify-center z-50">
				<div className="bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-lg w-full mx-4 p-6">
					<div className="flex items-center gap-3 text-red-600 dark:text-red-400 mb-4">
						<svg
							className="w-8 h-8"
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
						<h2 className="text-xl font-bold">
							{expirationInfo.is_expired
								? 'Password Expired'
								: 'Password Change Required'}
						</h2>
					</div>

					<p className="text-gray-600 dark:text-gray-400 mb-6">
						{expirationInfo.is_expired
							? 'Your password has expired. You must change your password before continuing.'
							: 'An administrator requires you to change your password before continuing.'}
					</p>

					<ChangePasswordForm
						showCancel={false}
						onSuccess={() => {
							// Reload the page to refresh the auth state
							window.location.reload();
						}}
					/>
				</div>
			</div>
		);
	}

	// Show warning banner for passwords expiring soon
	if (showChangePassword) {
		return (
			<div className="fixed inset-0 bg-gray-500 dark:bg-gray-900 bg-opacity-75 dark:bg-opacity-75 flex items-center justify-center z-50">
				<div className="bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-lg w-full mx-4 p-6">
					<div className="flex items-center justify-between mb-4">
						<h2 className="text-xl font-bold text-gray-900 dark:text-white">Change Password</h2>
						<button
							type="button"
							onClick={() => setShowChangePassword(false)}
							className="text-gray-400 hover:text-gray-600 dark:text-gray-500 dark:hover:text-gray-300"
						>
							<svg
								className="w-6 h-6"
								fill="none"
								stroke="currentColor"
								viewBox="0 0 24 24"
								aria-hidden="true"
							>
								<path
									strokeLinecap="round"
									strokeLinejoin="round"
									strokeWidth={2}
									d="M6 18L18 6M6 6l12 12"
								/>
							</svg>
						</button>
					</div>

					<ChangePasswordForm
						showCancel={true}
						onCancel={() => setShowChangePassword(false)}
						onSuccess={() => {
							setShowChangePassword(false);
							setDismissed(true);
						}}
					/>
				</div>
			</div>
		);
	}

	return (
		<div className="bg-amber-50 dark:bg-amber-900/20 border-b border-amber-200 dark:border-amber-800">
			<div className="max-w-7xl mx-auto py-3 px-4 sm:px-6 lg:px-8">
				<div className="flex items-center justify-between flex-wrap">
					<div className="flex-1 flex items-center">
						<span className="flex p-2 rounded-lg bg-amber-100 dark:bg-amber-800">
							<svg
								className="w-5 h-5 text-amber-600"
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
						</span>
						<p className="ml-3 font-medium text-amber-700 dark:text-amber-200">
							<span>
								Your password will expire in {expirationInfo.days_until_expiry}{' '}
								{expirationInfo.days_until_expiry === 1 ? 'day' : 'days'}.
							</span>
						</p>
					</div>
					<div className="flex gap-2">
						<button
							type="button"
							onClick={() => setShowChangePassword(true)}
							className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md text-amber-700 bg-amber-100 hover:bg-amber-200 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-amber-500"
						>
							Change Now
						</button>
						<button
							type="button"
							onClick={() => setDismissed(true)}
							className="inline-flex items-center px-4 py-2 text-sm font-medium text-amber-700 hover:text-amber-900"
						>
							Dismiss
						</button>
					</div>
				</div>
			</div>
		</div>
	);
}
