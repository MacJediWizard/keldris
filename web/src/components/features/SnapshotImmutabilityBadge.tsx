import { useSnapshotImmutabilityStatus } from '../../hooks/useImmutability';

interface SnapshotImmutabilityBadgeProps {
	snapshotId: string;
	repositoryId: string;
	showDetails?: boolean;
}

export function SnapshotImmutabilityBadge({
	snapshotId,
	repositoryId,
	showDetails = false,
}: SnapshotImmutabilityBadgeProps) {
	const { data: status, isLoading } = useSnapshotImmutabilityStatus(
		snapshotId,
		repositoryId
	);

	if (isLoading || !status?.is_locked) {
		return null;
	}

	return (
		<span
			className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-amber-100 text-amber-800"
			title={`Locked until ${status.locked_until ? new Date(status.locked_until).toLocaleDateString() : 'N/A'}${status.reason ? ` - ${status.reason}` : ''}`}
		>
			<svg
				className="w-3 h-3"
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
			{showDetails && status.remaining_days !== undefined && (
				<span>{status.remaining_days}d</span>
			)}
		</span>
	);
}

interface LockIconProps {
	isLocked: boolean;
	lockedUntil?: string;
	remainingDays?: number;
	reason?: string;
	size?: 'sm' | 'md' | 'lg';
}

export function LockIcon({
	isLocked,
	lockedUntil,
	remainingDays,
	reason,
	size = 'md',
}: LockIconProps) {
	if (!isLocked) {
		return null;
	}

	const sizeClasses = {
		sm: 'w-3 h-3',
		md: 'w-4 h-4',
		lg: 'w-5 h-5',
	};

	const title = `Locked until ${lockedUntil ? new Date(lockedUntil).toLocaleDateString() : 'N/A'}${remainingDays !== undefined ? ` (${remainingDays} days remaining)` : ''}${reason ? ` - ${reason}` : ''}`;

	return (
		<span
			className="inline-flex items-center text-amber-600"
			title={title}
		>
			<svg
				className={sizeClasses[size]}
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
		</span>
	);
}

interface ImmutabilityWarningProps {
	isLocked: boolean;
	lockedUntil?: string;
	remainingDays?: number;
}

export function ImmutabilityDeleteWarning({
	isLocked,
	lockedUntil,
	remainingDays,
}: ImmutabilityWarningProps) {
	if (!isLocked) {
		return null;
	}

	return (
		<div className="rounded-md bg-amber-50 p-4 mb-4">
			<div className="flex">
				<div className="flex-shrink-0">
					<svg
						className="h-5 w-5 text-amber-400"
						viewBox="0 0 20 20"
						fill="currentColor"
						aria-hidden="true"
					>
						<path
							fillRule="evenodd"
							d="M8.485 2.495c.673-1.167 2.357-1.167 3.03 0l6.28 10.875c.673 1.167-.17 2.625-1.516 2.625H3.72c-1.347 0-2.189-1.458-1.515-2.625L8.485 2.495zM10 5a.75.75 0 01.75.75v3.5a.75.75 0 01-1.5 0v-3.5A.75.75 0 0110 5zm0 9a1 1 0 100-2 1 1 0 000 2z"
							clipRule="evenodd"
						/>
					</svg>
				</div>
				<div className="ml-3">
					<h3 className="text-sm font-medium text-amber-800">
						Snapshot is immutable
					</h3>
					<div className="mt-2 text-sm text-amber-700">
						<p>
							This snapshot is protected by an immutability lock and cannot be deleted
							until{' '}
							<strong>
								{lockedUntil
									? new Date(lockedUntil).toLocaleDateString()
									: 'the lock expires'}
							</strong>
							{remainingDays !== undefined && remainingDays > 0 && (
								<> ({remainingDays} days remaining)</>
							)}
							.
						</p>
					</div>
				</div>
			</div>
		</div>
	);
}
