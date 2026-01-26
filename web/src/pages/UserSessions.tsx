import { useState } from 'react';
import {
	useRevokeAllSessions,
	useRevokeSession,
	useUserSessions,
} from '../hooks/useUserSessions';
import type { UserSession } from '../lib/types';

function formatUserAgent(userAgent?: string): string {
	if (!userAgent) return 'Unknown';

	// Extract browser and OS info from user agent string
	const browsers = [
		{ name: 'Chrome', pattern: /Chrome\/(\d+)/ },
		{ name: 'Firefox', pattern: /Firefox\/(\d+)/ },
		{ name: 'Safari', pattern: /Safari\/(\d+)/ },
		{ name: 'Edge', pattern: /Edg\/(\d+)/ },
	];

	const oses = [
		{ name: 'Windows', pattern: /Windows NT/ },
		{ name: 'macOS', pattern: /Macintosh/ },
		{ name: 'Linux', pattern: /Linux/ },
		{ name: 'iOS', pattern: /iPhone|iPad/ },
		{ name: 'Android', pattern: /Android/ },
	];

	let browser = 'Unknown browser';
	for (const b of browsers) {
		const match = userAgent.match(b.pattern);
		if (match) {
			browser = `${b.name} ${match[1]}`;
			break;
		}
	}

	let os = '';
	for (const o of oses) {
		if (o.pattern.test(userAgent)) {
			os = o.name;
			break;
		}
	}

	return os ? `${browser} on ${os}` : browser;
}

function formatRelativeTime(date: string): string {
	const now = new Date();
	const then = new Date(date);
	const diffMs = now.getTime() - then.getTime();
	const diffMins = Math.floor(diffMs / 60000);
	const diffHours = Math.floor(diffMs / 3600000);
	const diffDays = Math.floor(diffMs / 86400000);

	if (diffMins < 1) return 'Just now';
	if (diffMins < 60) return `${diffMins} minute${diffMins > 1 ? 's' : ''} ago`;
	if (diffHours < 24) return `${diffHours} hour${diffHours > 1 ? 's' : ''} ago`;
	if (diffDays < 7) return `${diffDays} day${diffDays > 1 ? 's' : ''} ago`;

	return then.toLocaleDateString();
}

function SessionCard({
	session,
	onRevoke,
	isRevoking,
}: {
	session: UserSession;
	onRevoke: (id: string) => void;
	isRevoking: boolean;
}) {
	return (
		<div
			className={`bg-white dark:bg-gray-800 rounded-lg border ${
				session.is_current
					? 'border-indigo-500 dark:border-indigo-400'
					: 'border-gray-200 dark:border-gray-700'
			} p-4`}
		>
			<div className="flex items-start justify-between">
				<div className="flex-1">
					<div className="flex items-center gap-2">
						<svg
							aria-hidden="true"
							className="w-5 h-5 text-gray-400"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
						>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"
							/>
						</svg>
						<span className="font-medium text-gray-900 dark:text-white">
							{formatUserAgent(session.user_agent)}
						</span>
						{session.is_current && (
							<span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-indigo-100 dark:bg-indigo-900 text-indigo-800 dark:text-indigo-200">
								Current session
							</span>
						)}
					</div>
					<div className="mt-2 text-sm text-gray-500 dark:text-gray-400 space-y-1">
						<div className="flex items-center gap-2">
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
									d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z"
								/>
								<path
									strokeLinecap="round"
									strokeLinejoin="round"
									strokeWidth={2}
									d="M15 11a3 3 0 11-6 0 3 3 0 016 0z"
								/>
							</svg>
							<span>{session.ip_address || 'Unknown IP'}</span>
						</div>
						<div className="flex items-center gap-2">
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
									d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
								/>
							</svg>
							<span>
								Last active: {formatRelativeTime(session.last_active_at)}
							</span>
						</div>
						<div className="flex items-center gap-2">
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
									d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"
								/>
							</svg>
							<span>
								Signed in: {new Date(session.created_at).toLocaleDateString()}{' '}
								at {new Date(session.created_at).toLocaleTimeString()}
							</span>
						</div>
					</div>
				</div>
				{!session.is_current && (
					<button
						type="button"
						onClick={() => onRevoke(session.id)}
						disabled={isRevoking}
						className="text-red-600 dark:text-red-400 hover:text-red-800 dark:hover:text-red-300 text-sm font-medium disabled:opacity-50"
					>
						{isRevoking ? 'Revoking...' : 'Revoke'}
					</button>
				)}
			</div>
		</div>
	);
}

export function UserSessions() {
	const { data: sessions, isLoading, isError } = useUserSessions();
	const revokeSession = useRevokeSession();
	const revokeAllSessions = useRevokeAllSessions();
	const [showRevokeAllConfirm, setShowRevokeAllConfirm] = useState(false);
	const [revokingId, setRevokingId] = useState<string | null>(null);

	const handleRevoke = async (id: string) => {
		setRevokingId(id);
		try {
			await revokeSession.mutateAsync(id);
		} finally {
			setRevokingId(null);
		}
	};

	const handleRevokeAll = async () => {
		try {
			await revokeAllSessions.mutateAsync();
			setShowRevokeAllConfirm(false);
		} catch {
			// Error handled by mutation
		}
	};

	const activeSessions = sessions?.filter((s) => !s.is_current) ?? [];
	const currentSession = sessions?.find((s) => s.is_current);

	if (isLoading) {
		return (
			<div className="space-y-6">
				<div>
					<div className="h-8 w-48 bg-gray-200 dark:bg-gray-700 rounded animate-pulse" />
					<div className="h-4 w-64 bg-gray-200 dark:bg-gray-700 rounded animate-pulse mt-2" />
				</div>
				<div className="space-y-4">
					{[1, 2, 3].map((i) => (
						<div
							key={i}
							className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-4"
						>
							<div className="h-5 w-48 bg-gray-200 dark:bg-gray-700 rounded animate-pulse" />
							<div className="h-4 w-32 bg-gray-200 dark:bg-gray-700 rounded animate-pulse mt-2" />
							<div className="h-4 w-40 bg-gray-200 dark:bg-gray-700 rounded animate-pulse mt-2" />
						</div>
					))}
				</div>
			</div>
		);
	}

	if (isError) {
		return (
			<div className="text-center py-12">
				<p className="text-red-500 dark:text-red-400">
					Failed to load sessions. Please try again.
				</p>
			</div>
		);
	}

	return (
		<div className="space-y-6">
			<div className="flex items-center justify-between">
				<div>
					<h1 className="text-2xl font-bold text-gray-900 dark:text-white">
						Active Sessions
					</h1>
					<p className="text-gray-600 dark:text-gray-400 mt-1">
						View and manage your active sessions across all devices
					</p>
				</div>
				{activeSessions.length > 0 && (
					<button
						type="button"
						onClick={() => setShowRevokeAllConfirm(true)}
						className="px-4 py-2 border border-red-300 dark:border-red-700 text-red-600 dark:text-red-400 rounded-lg hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors"
					>
						Revoke All Other Sessions
					</button>
				)}
			</div>

			{/* Current Session */}
			{currentSession && (
				<div>
					<h2 className="text-sm font-medium text-gray-500 dark:text-gray-400 mb-3">
						This Device
					</h2>
					<SessionCard
						session={currentSession}
						onRevoke={handleRevoke}
						isRevoking={revokingId === currentSession.id}
					/>
				</div>
			)}

			{/* Other Sessions */}
			{activeSessions.length > 0 && (
				<div>
					<h2 className="text-sm font-medium text-gray-500 dark:text-gray-400 mb-3">
						Other Sessions ({activeSessions.length})
					</h2>
					<div className="space-y-3">
						{activeSessions.map((session) => (
							<SessionCard
								key={session.id}
								session={session}
								onRevoke={handleRevoke}
								isRevoking={revokingId === session.id}
							/>
						))}
					</div>
				</div>
			)}

			{sessions?.length === 0 && (
				<div className="text-center py-12">
					<svg
						aria-hidden="true"
						className="mx-auto h-12 w-12 text-gray-400"
						fill="none"
						stroke="currentColor"
						viewBox="0 0 24 24"
					>
						<path
							strokeLinecap="round"
							strokeLinejoin="round"
							strokeWidth={2}
							d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"
						/>
					</svg>
					<h3 className="mt-2 text-sm font-medium text-gray-900 dark:text-white">
						No sessions found
					</h3>
					<p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
						Your active sessions will appear here.
					</p>
				</div>
			)}

			{activeSessions.length === 0 && currentSession && (
				<div className="text-center py-8 text-gray-500 dark:text-gray-400">
					<p>You have no other active sessions.</p>
				</div>
			)}

			{/* Revoke All Confirmation Modal */}
			{showRevokeAllConfirm && (
				<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
					<div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-md w-full mx-4">
						<div className="flex items-center gap-3 mb-4">
							<div className="p-2 bg-red-100 dark:bg-red-900/30 rounded-full">
								<svg
									aria-hidden="true"
									className="w-6 h-6 text-red-600 dark:text-red-400"
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
							<h3 className="text-lg font-semibold text-gray-900 dark:text-white">
								Revoke All Other Sessions
							</h3>
						</div>
						<p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
							This will sign you out of all other devices and browsers. Your
							current session will remain active.
						</p>
						<p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
							<strong>{activeSessions.length}</strong> session
							{activeSessions.length !== 1 ? 's' : ''} will be revoked.
						</p>
						{revokeAllSessions.isError && (
							<p className="text-sm text-red-600 dark:text-red-400 mb-4">
								Failed to revoke sessions. Please try again.
							</p>
						)}
						<div className="flex justify-end gap-3">
							<button
								type="button"
								onClick={() => setShowRevokeAllConfirm(false)}
								className="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
							>
								Cancel
							</button>
							<button
								type="button"
								onClick={handleRevokeAll}
								disabled={revokeAllSessions.isPending}
								className="px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 transition-colors disabled:opacity-50"
							>
								{revokeAllSessions.isPending
									? 'Revoking...'
									: 'Revoke All Sessions'}
							</button>
						</div>
					</div>
				</div>
			)}
		</div>
	);
}
