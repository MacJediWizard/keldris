import { useEffect, useState } from 'react';
import { useMe } from '../../hooks/useAuth';
import {
	useActiveMaintenance,
	useEmergencyOverride,
} from '../../hooks/useMaintenance';

function formatTimeLeft(targetDate: Date): string {
	const now = new Date();
	const diff = targetDate.getTime() - now.getTime();

	if (diff <= 0) {
		return '0s';
	}

	const days = Math.floor(diff / (1000 * 60 * 60 * 24));
	const hours = Math.floor((diff % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60));
	const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60));
	const seconds = Math.floor((diff % (1000 * 60)) / 1000);

	if (days > 0) {
		return `${days}d ${hours}h`;
	}
	if (hours > 0) {
		return `${hours}h ${minutes}m`;
	}
	if (minutes > 0) {
		return `${minutes}m ${seconds}s`;
	}
	return `${seconds}s`;
}

function formatCountdownDisplay(targetDate: Date): {
	display: string;
	isUrgent: boolean;
} {
	const now = new Date();
	const diff = targetDate.getTime() - now.getTime();

	if (diff <= 0) {
		return { display: '0:00', isUrgent: true };
	}

	const totalMinutes = Math.floor(diff / (1000 * 60));
	const hours = Math.floor(totalMinutes / 60);
	const minutes = totalMinutes % 60;
	const seconds = Math.floor((diff % (1000 * 60)) / 1000);

	const isUrgent = totalMinutes < 5;

	if (hours > 0) {
		return {
			display: `${hours}:${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`,
			isUrgent,
		};
	}
	return {
		display: `${minutes}:${seconds.toString().padStart(2, '0')}`,
		isUrgent,
	};
}

export function MaintenanceCountdown() {
	const { data: user } = useMe();
	const { data } = useActiveMaintenance();
	const emergencyOverride = useEmergencyOverride();
	const [timeLeft, setTimeLeft] = useState<string>('');
	const [countdown, setCountdown] = useState<{
		display: string;
		isUrgent: boolean;
	} | null>(null);
	const [showOverrideConfirm, setShowOverrideConfirm] = useState(false);

	const isAdmin =
		user?.current_org_role === 'owner' || user?.current_org_role === 'admin';

	useEffect(() => {
		if (!data?.active && !data?.upcoming && !data?.show_countdown) {
			setTimeLeft('');
			setCountdown(null);
			return;
		}

		const updateCountdown = () => {
			if (data.active) {
				setTimeLeft(formatTimeLeft(new Date(data.active.ends_at)));
			} else if (data.upcoming) {
				setTimeLeft(formatTimeLeft(new Date(data.upcoming.starts_at)));
			}

			if (data.show_countdown && data.countdown_to) {
				setCountdown(formatCountdownDisplay(new Date(data.countdown_to)));
			} else {
				setCountdown(null);
			}
		};

		updateCountdown();
		const interval = setInterval(updateCountdown, 1000);
		return () => clearInterval(interval);
	}, [data]);

	const handleEmergencyOverride = () => {
		if (!data?.active) return;
		emergencyOverride.mutate(
			{ id: data.active.id, override: true },
			{
				onSuccess: () => setShowOverrideConfirm(false),
			},
		);
	};

	if (!data?.active && !data?.upcoming) {
		return null;
	}

	const isActive = !!data.active;
	const window = data.active ?? data.upcoming;
	if (!window) return null;

	const isReadOnly = data.read_only_mode;
	const showCountdown = data.show_countdown && countdown;

	// Determine banner color based on state
	let bannerClass = 'bg-blue-500'; // Default: upcoming
	if (isActive) {
		if (isReadOnly) {
			bannerClass = 'bg-red-600'; // Active read-only
		} else {
			bannerClass = 'bg-amber-500'; // Active but not read-only
		}
	}
	if (showCountdown && countdown?.isUrgent) {
		bannerClass = isActive ? 'bg-red-700' : 'bg-amber-600';
	}

	return (
		<>
			<div className={`px-4 py-3 ${bannerClass} text-white`}>
				<div className="flex items-center justify-center gap-3">
					{/* Icon */}
					{isReadOnly ? (
						<svg
							className="w-5 h-5"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
							aria-hidden="true"
						>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"
							/>
						</svg>
					) : (
						<svg
							className="w-5 h-5"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
							aria-hidden="true"
						>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"
							/>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
							/>
						</svg>
					)}

					{/* Status text */}
					<span className="font-medium">
						{isReadOnly
							? 'Read-only mode active'
							: isActive
								? 'Maintenance in progress'
								: 'Scheduled maintenance'}
						:
					</span>

					{/* Title */}
					<span>{window.title}</span>

					{/* Message */}
					{window.message && (
						<span className="opacity-90">- {window.message}</span>
					)}

					{/* Countdown display */}
					{showCountdown && countdown && (
						<span
							className={`font-mono px-2 py-0.5 rounded ${
								countdown.isUrgent ? 'bg-white/30 animate-pulse' : 'bg-white/20'
							}`}
						>
							{countdown.display}
						</span>
					)}

					{/* Time remaining */}
					<span className="font-mono bg-white/20 px-2 py-0.5 rounded">
						{isActive ? `Ends in ${timeLeft}` : `Starts in ${timeLeft}`}
					</span>

					{/* Emergency override button for admins */}
					{isReadOnly && isAdmin && (
						<button
							type="button"
							onClick={() => setShowOverrideConfirm(true)}
							className="ml-2 px-3 py-1 bg-white/20 hover:bg-white/30 rounded text-sm font-medium transition-colors"
						>
							Emergency Override
						</button>
					)}
				</div>
			</div>

			{/* Emergency Override Confirmation Modal */}
			{showOverrideConfirm && (
				<div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
					<div className="bg-white rounded-lg shadow-xl max-w-md w-full mx-4">
						<div className="p-6">
							<div className="flex items-center gap-3 mb-4">
								<div className="p-2 bg-red-100 rounded-full">
									<svg
										className="w-6 h-6 text-red-600"
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
								</div>
								<h3 className="text-lg font-semibold text-gray-900">
									Emergency Override
								</h3>
							</div>

							<p className="text-gray-600 mb-4">
								This will disable read-only mode and allow write operations
								during the maintenance window. This action will be logged.
							</p>

							<p className="text-sm text-gray-500 mb-6">
								Maintenance window: <strong>{window.title}</strong>
							</p>

							<div className="flex justify-end gap-3">
								<button
									type="button"
									onClick={() => setShowOverrideConfirm(false)}
									className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50"
								>
									Cancel
								</button>
								<button
									type="button"
									onClick={handleEmergencyOverride}
									disabled={emergencyOverride.isPending}
									className="px-4 py-2 text-sm font-medium text-white bg-red-600 border border-transparent rounded-md hover:bg-red-700 disabled:opacity-50"
								>
									{emergencyOverride.isPending
										? 'Overriding...'
										: 'Confirm Override'}
								</button>
							</div>
						</div>
					</div>
				</div>
			)}
		</>
	);
}
