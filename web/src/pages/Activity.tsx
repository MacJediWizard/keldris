import { ActivityFeedFull } from '../components/features/ActivityFeed';

export function Activity() {
	return (
		<div className="space-y-6">
			<ActivityFeedFull enableRealtime={true} />
		</div>
	);
}
