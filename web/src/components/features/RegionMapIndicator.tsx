import { useMemo } from 'react';
import type { GeoRegion, GeoReplicationConfig } from '../../lib/types';

interface RegionMapIndicatorProps {
	configs: GeoReplicationConfig[];
	regions?: GeoRegion[];
	compact?: boolean;
}

// Region coordinates for the simplified world map
const REGION_POSITIONS: Record<string, { x: number; y: number }> = {
	'us-east-1': { x: 25, y: 35 },
	'us-west-2': { x: 12, y: 38 },
	'eu-west-1': { x: 45, y: 30 },
	'eu-central-1': { x: 52, y: 32 },
	'ap-southeast-1': { x: 78, y: 55 },
	'ap-northeast-1': { x: 88, y: 38 },
};

function getStatusColor(status: string): string {
	switch (status) {
		case 'synced':
			return '#10B981'; // green
		case 'syncing':
			return '#3B82F6'; // blue
		case 'pending':
			return '#9CA3AF'; // gray
		case 'failed':
			return '#EF4444'; // red
		case 'disabled':
			return '#6B7280'; // dark gray
		default:
			return '#9CA3AF';
	}
}

function RegionDot({
	region,
	status,
	isSource,
}: {
	region: GeoRegion;
	status: string;
	isSource: boolean;
}) {
	const position = REGION_POSITIONS[region.code] ?? { x: 50, y: 50 };
	const color = getStatusColor(status);

	return (
		<g>
			{/* Outer ring for source regions */}
			{isSource && (
				<circle
					cx={`${position.x}%`}
					cy={`${position.y}%`}
					r="8"
					fill="none"
					stroke={color}
					strokeWidth="2"
					opacity="0.5"
				/>
			)}
			{/* Main dot */}
			<circle
				cx={`${position.x}%`}
				cy={`${position.y}%`}
				r="5"
				fill={color}
				stroke="white"
				strokeWidth="1"
			/>
			{/* Pulse animation for syncing */}
			{status === 'syncing' && (
				<circle
					cx={`${position.x}%`}
					cy={`${position.y}%`}
					r="5"
					fill="none"
					stroke={color}
					strokeWidth="2"
					opacity="0.5"
				>
					<animate
						attributeName="r"
						values="5;12;5"
						dur="1.5s"
						repeatCount="indefinite"
					/>
					<animate
						attributeName="opacity"
						values="0.5;0;0.5"
						dur="1.5s"
						repeatCount="indefinite"
					/>
				</circle>
			)}
			{/* Label */}
			<text
				x={`${position.x}%`}
				y={`${position.y + 8}%`}
				textAnchor="middle"
				fontSize="8"
				fill="#6B7280"
				className="select-none"
			>
				{region.display_name}
			</text>
		</g>
	);
}

function ReplicationLine({
	source,
	target,
	status,
}: {
	source: GeoRegion;
	target: GeoRegion;
	status: string;
}) {
	const sourcePos = REGION_POSITIONS[source.code] ?? { x: 50, y: 50 };
	const targetPos = REGION_POSITIONS[target.code] ?? { x: 50, y: 50 };
	const color = getStatusColor(status);

	// Calculate control point for curved line
	const midX = (sourcePos.x + targetPos.x) / 2;
	const midY = (sourcePos.y + targetPos.y) / 2 - 10;

	return (
		<g>
			{/* Connection line */}
			<path
				d={`M ${sourcePos.x}% ${sourcePos.y}% Q ${midX}% ${midY}% ${targetPos.x}% ${targetPos.y}%`}
				fill="none"
				stroke={color}
				strokeWidth="2"
				strokeDasharray={status === 'syncing' ? '4 2' : 'none'}
				opacity="0.6"
			>
				{status === 'syncing' && (
					<animate
						attributeName="stroke-dashoffset"
						values="0;-12"
						dur="0.5s"
						repeatCount="indefinite"
					/>
				)}
			</path>
			{/* Arrow head */}
			<circle
				cx={`${targetPos.x}%`}
				cy={`${targetPos.y}%`}
				r="3"
				fill={color}
			/>
		</g>
	);
}

export function RegionMapIndicator({
	configs,
	compact = false,
}: RegionMapIndicatorProps) {
	const activeConfigs = useMemo(
		() => configs.filter((c) => c.enabled),
		[configs],
	);

	if (activeConfigs.length === 0) {
		return (
			<div className="bg-gray-50 rounded-lg p-4 text-center">
				<p className="text-sm text-gray-500">No geo-replication configured</p>
			</div>
		);
	}

	if (compact) {
		// Compact view - just show status indicators
		const syncedCount = activeConfigs.filter(
			(c) => c.status === 'synced',
		).length;
		const syncingCount = activeConfigs.filter(
			(c) => c.status === 'syncing',
		).length;
		const failedCount = activeConfigs.filter(
			(c) => c.status === 'failed',
		).length;

		return (
			<div className="flex items-center gap-3">
				<div className="flex items-center gap-1.5">
					<span className="w-2 h-2 rounded-full bg-green-500" />
					<span className="text-xs text-gray-600">{syncedCount} synced</span>
				</div>
				{syncingCount > 0 && (
					<div className="flex items-center gap-1.5">
						<span className="w-2 h-2 rounded-full bg-blue-500 animate-pulse" />
						<span className="text-xs text-gray-600">
							{syncingCount} syncing
						</span>
					</div>
				)}
				{failedCount > 0 && (
					<div className="flex items-center gap-1.5">
						<span className="w-2 h-2 rounded-full bg-red-500" />
						<span className="text-xs text-gray-600">{failedCount} failed</span>
					</div>
				)}
			</div>
		);
	}

	return (
		<div className="bg-white rounded-lg border border-gray-200 p-4">
			<h3 className="text-sm font-medium text-gray-900 mb-3">
				Geo-Replication Map
			</h3>
			<div className="relative bg-gray-100 rounded-lg overflow-hidden">
				{/* Simplified world map background */}
				<svg
					aria-hidden="true"
					viewBox="0 0 100 60"
					className="w-full h-auto"
					preserveAspectRatio="xMidYMid meet"
				>
					{/* Simple world outline */}
					<rect x="0" y="0" width="100" height="60" fill="#F3F4F6" />
					{/* Simplified continents */}
					<ellipse cx="22" cy="38" rx="15" ry="10" fill="#E5E7EB" />{' '}
					{/* North America */}
					<ellipse cx="28" cy="52" rx="8" ry="6" fill="#E5E7EB" />{' '}
					{/* South America */}
					<ellipse cx="50" cy="35" rx="12" ry="15" fill="#E5E7EB" />{' '}
					{/* Europe/Africa */}
					<ellipse cx="75" cy="40" rx="18" ry="15" fill="#E5E7EB" />{' '}
					{/* Asia */}
					<ellipse cx="88" cy="55" rx="8" ry="5" fill="#E5E7EB" />{' '}
					{/* Australia */}
					{/* Replication lines */}
					{activeConfigs.map((config) => (
						<ReplicationLine
							key={config.id}
							source={config.source_region}
							target={config.target_region}
							status={config.status}
						/>
					))}
					{/* Region dots */}
					{activeConfigs.map((config) => (
						<g key={`regions-${config.id}`}>
							<RegionDot
								region={config.source_region}
								status={config.status}
								isSource={true}
							/>
							<RegionDot
								region={config.target_region}
								status={config.status}
								isSource={false}
							/>
						</g>
					))}
				</svg>
			</div>

			{/* Legend */}
			<div className="mt-3 flex flex-wrap gap-3 text-xs text-gray-600">
				<div className="flex items-center gap-1.5">
					<span className="w-2 h-2 rounded-full bg-green-500" />
					<span>Synced</span>
				</div>
				<div className="flex items-center gap-1.5">
					<span className="w-2 h-2 rounded-full bg-blue-500" />
					<span>Syncing</span>
				</div>
				<div className="flex items-center gap-1.5">
					<span className="w-2 h-2 rounded-full bg-gray-400" />
					<span>Pending</span>
				</div>
				<div className="flex items-center gap-1.5">
					<span className="w-2 h-2 rounded-full bg-red-500" />
					<span>Failed</span>
				</div>
			</div>
		</div>
	);
}
