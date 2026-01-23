import {
	useActiveAnnouncements,
	useDismissAnnouncement,
} from '../../hooks/useAnnouncements';
import type { Announcement, AnnouncementType } from '../../lib/types';

function getTypeStyles(type: AnnouncementType): {
	bg: string;
	icon: React.ReactNode;
} {
	switch (type) {
		case 'critical':
			return {
				bg: 'bg-red-600',
				icon: (
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
							d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
						/>
					</svg>
				),
			};
		case 'warning':
			return {
				bg: 'bg-amber-500',
				icon: (
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
							d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
						/>
					</svg>
				),
			};
		default:
			return {
				bg: 'bg-blue-500',
				icon: (
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
							d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
						/>
					</svg>
				),
			};
	}
}

function AnnouncementItem({
	announcement,
	onDismiss,
	isDismissing,
}: {
	announcement: Announcement;
	onDismiss: (id: string) => void;
	isDismissing: boolean;
}) {
	const { bg, icon } = getTypeStyles(announcement.type);

	return (
		<div className={`px-4 py-3 ${bg} text-white`}>
			<div className="flex items-center justify-between gap-3">
				<div className="flex items-center gap-3 flex-1 min-w-0">
					{icon}
					<span className="font-medium truncate">{announcement.title}</span>
					{announcement.message && (
						<span className="opacity-90 truncate hidden sm:inline">
							- {announcement.message}
						</span>
					)}
				</div>
				{announcement.dismissible && (
					<button
						type="button"
						onClick={() => onDismiss(announcement.id)}
						disabled={isDismissing}
						className="text-white hover:text-gray-200 flex-shrink-0 disabled:opacity-50"
						aria-label="Dismiss announcement"
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
				)}
			</div>
		</div>
	);
}

export function AnnouncementBanner() {
	const { data: announcements } = useActiveAnnouncements();
	const dismissMutation = useDismissAnnouncement();

	if (!announcements || announcements.length === 0) {
		return null;
	}

	const handleDismiss = (id: string) => {
		dismissMutation.mutate(id);
	};

	return (
		<div className="space-y-0">
			{announcements.map((announcement) => (
				<AnnouncementItem
					key={announcement.id}
					announcement={announcement}
					onDismiss={handleDismiss}
					isDismissing={dismissMutation.isPending}
				/>
			))}
		</div>
	);
}
