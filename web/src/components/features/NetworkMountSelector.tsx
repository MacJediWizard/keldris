import type { NetworkMount } from '../../lib/types';

interface NetworkMountSelectorProps {
	mounts: NetworkMount[];
	selectedPaths: string[];
	onPathsChange: (paths: string[]) => void;
}

export function NetworkMountSelector({
	mounts,
	selectedPaths,
	onPathsChange,
}: NetworkMountSelectorProps) {
	const connectedMounts = mounts.filter((m) => m.status === 'connected');
	const unavailableMounts = mounts.filter((m) => m.status !== 'connected');

	const toggleMount = (path: string) => {
		if (selectedPaths.includes(path)) {
			onPathsChange(selectedPaths.filter((p) => p !== path));
		} else {
			onPathsChange([...selectedPaths, path]);
		}
	};

	return (
		<div className="space-y-3">
			<label className="block text-sm font-medium text-gray-700">
				Network Mounts
			</label>

			{connectedMounts.length > 0 && (
				<div className="space-y-2">
					{connectedMounts.map((mount) => (
						<label
							key={mount.path}
							className="flex items-center gap-3 p-3 border border-gray-200 rounded-lg hover:bg-gray-50 cursor-pointer"
						>
							<input
								type="checkbox"
								checked={selectedPaths.includes(mount.path)}
								onChange={() => toggleMount(mount.path)}
								className="h-4 w-4 text-indigo-600 rounded"
							/>
							<div className="flex-1">
								<div className="font-medium text-gray-900">{mount.path}</div>
								<div className="text-sm text-gray-500">{mount.remote}</div>
							</div>
							<span className="inline-flex items-center gap-1 px-2 py-0.5 text-xs bg-green-100 text-green-800 rounded-full">
								<span className="w-1.5 h-1.5 bg-green-500 rounded-full" />
								{mount.type.toUpperCase()}
							</span>
						</label>
					))}
				</div>
			)}

			{unavailableMounts.length > 0 && (
				<div className="mt-3">
					<p className="text-sm text-amber-600 mb-2">Unavailable mounts:</p>
					{unavailableMounts.map((mount) => (
						<div
							key={mount.path}
							className="flex items-center gap-3 p-3 border border-amber-200 bg-amber-50 rounded-lg opacity-60"
						>
							<div className="flex-1">
								<div className="font-medium text-gray-700">{mount.path}</div>
								<div className="text-sm text-gray-500">{mount.remote}</div>
							</div>
							<span className="inline-flex items-center gap-1 px-2 py-0.5 text-xs bg-amber-100 text-amber-800 rounded-full">
								{mount.status}
							</span>
						</div>
					))}
				</div>
			)}

			{mounts.length === 0 && (
				<p className="text-sm text-gray-500">
					No network mounts detected on this agent.
				</p>
			)}
		</div>
	);
}
