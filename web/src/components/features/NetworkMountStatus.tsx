import type { NetworkMount } from '../../lib/types';

interface NetworkMountStatusProps {
	mounts: NetworkMount[];
	compact?: boolean;
}

function getMountStatusColor(status: string) {
	switch (status) {
		case 'connected':
			return { bg: 'bg-green-100', text: 'text-green-800', dot: 'bg-green-500' };
		case 'stale':
			return { bg: 'bg-amber-100', text: 'text-amber-800', dot: 'bg-amber-500' };
		case 'disconnected':
			return { bg: 'bg-red-100', text: 'text-red-800', dot: 'bg-red-500' };
		default:
			return { bg: 'bg-gray-100', text: 'text-gray-800', dot: 'bg-gray-500' };
	}
}

export function NetworkMountStatus({
	mounts,
	compact = false,
}: NetworkMountStatusProps) {
	if (!mounts || mounts.length === 0) {
		return null;
	}

	const connectedCount = mounts.filter((m) => m.status === 'connected').length;
	const unavailableCount = mounts.length - connectedCount;

	if (compact) {
		return (
			<div className="flex items-center gap-2 flex-wrap">
				<span className="inline-flex items-center gap-1 px-2 py-0.5 text-xs bg-blue-50 text-blue-700 rounded">
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
							d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2"
						/>
					</svg>
					{mounts.length} mount{mounts.length !== 1 ? 's' : ''}
				</span>
				{connectedCount > 0 && (
					<span className="inline-flex items-center gap-1 px-2 py-0.5 text-xs bg-green-50 text-green-700 rounded">
						<span className="w-1.5 h-1.5 bg-green-500 rounded-full" />
						{connectedCount} connected
					</span>
				)}
				{unavailableCount > 0 && (
					<span className="inline-flex items-center gap-1 px-2 py-0.5 text-xs bg-amber-50 text-amber-700 rounded">
						<span className="w-1.5 h-1.5 bg-amber-500 rounded-full" />
						{unavailableCount} unavailable
					</span>
				)}
			</div>
		);
	}

	return (
		<div className="mt-4">
			<h4 className="text-sm font-medium text-gray-700 mb-2">Network Mounts</h4>
			<div className="space-y-2">
				{mounts.map((mount) => {
					const color = getMountStatusColor(mount.status);
					return (
						<div
							key={mount.path}
							className="flex items-center justify-between p-2 bg-gray-50 rounded-lg"
						>
							<div>
								<div className="text-sm font-medium text-gray-900">
									{mount.path}
								</div>
								<div className="text-xs text-gray-500">
									{mount.type.toUpperCase()} - {mount.remote}
								</div>
							</div>
							<div className="flex items-center gap-2">
								<span
									className={`inline-flex items-center gap-1 px-2 py-0.5 text-xs rounded-full ${color.bg} ${color.text}`}
								>
									<span className={`w-1.5 h-1.5 ${color.dot} rounded-full`} />
									{mount.status}
								</span>
							</div>
						</div>
					);
				})}
			</div>
		</div>
	);
}
