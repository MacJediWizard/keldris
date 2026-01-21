import { useEffect, useState } from 'react';
import { useActiveMaintenance } from '../../hooks/useMaintenance';

function formatTimeLeft(targetDate: Date): string {
	const now = new Date();
	const diff = targetDate.getTime() - now.getTime();

	if (diff <= 0) {
		return '0s';
	}

	const hours = Math.floor(diff / (1000 * 60 * 60));
	const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60));
	const seconds = Math.floor((diff % (1000 * 60)) / 1000);

	if (hours > 0) {
		return `${hours}h ${minutes}m`;
	}
	if (minutes > 0) {
		return `${minutes}m ${seconds}s`;
	}
	return `${seconds}s`;
}

export function MaintenanceBanner() {
	const { data } = useActiveMaintenance();
	const [timeLeft, setTimeLeft] = useState<string>('');

	useEffect(() => {
		if (!data?.active && !data?.upcoming) {
			setTimeLeft('');
			return;
		}

		const updateCountdown = () => {
			const target = data.active
				? new Date(data.active.ends_at)
				: new Date(data.upcoming!.starts_at);

			setTimeLeft(formatTimeLeft(target));
		};

		updateCountdown();
		const interval = setInterval(updateCountdown, 1000);
		return () => clearInterval(interval);
	}, [data]);

	if (!data?.active && !data?.upcoming) {
		return null;
	}

	const isActive = !!data.active;
	const window = data.active || data.upcoming!;

	return (
		<div
			className={`px-4 py-3 ${isActive ? 'bg-amber-500' : 'bg-blue-500'} text-white`}
		>
			<div className="flex items-center justify-center gap-3">
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
				<span className="font-medium">
					{isActive ? 'Maintenance in progress' : 'Scheduled maintenance'}:
				</span>
				<span>{window.title}</span>
				{window.message && (
					<span className="opacity-90">- {window.message}</span>
				)}
				<span className="font-mono bg-white/20 px-2 py-0.5 rounded">
					{isActive ? `Ends in ${timeLeft}` : `Starts in ${timeLeft}`}
				</span>
			</div>
		</div>
	);
}
