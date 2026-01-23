import { useMemo, useState } from 'react';
import { useAgents } from '../../hooks/useAgents';
import { useBackupCalendar } from '../../hooks/useBackupCalendar';
import type {
	Backup,
	BackupCalendarDay,
	ScheduledBackup,
} from '../../lib/types';
import { formatDateTime, getBackupStatusColor } from '../../lib/utils';

interface BackupCalendarProps {
	onSelectBackup?: (backup: Backup) => void;
	compact?: boolean;
}

function getDaysInMonth(year: number, month: number): number {
	return new Date(year, month + 1, 0).getDate();
}

function getFirstDayOfMonth(year: number, month: number): number {
	return new Date(year, month, 1).getDay();
}

function formatMonthParam(date: Date): string {
	const year = date.getFullYear();
	const month = String(date.getMonth() + 1).padStart(2, '0');
	return `${year}-${month}`;
}

function CalendarDayCell({
	day,
	isCurrentMonth,
	isToday,
	calendarDay,
	scheduledBackups,
	onClick,
	compact,
}: {
	day: number;
	isCurrentMonth: boolean;
	isToday: boolean;
	calendarDay?: BackupCalendarDay;
	scheduledBackups: ScheduledBackup[];
	onClick: () => void;
	compact?: boolean;
}) {
	const hasBackups =
		calendarDay &&
		(calendarDay.completed > 0 ||
			calendarDay.failed > 0 ||
			calendarDay.running > 0);
	const hasScheduled = scheduledBackups.length > 0;

	return (
		<button
			type="button"
			onClick={onClick}
			disabled={!isCurrentMonth}
			className={`
				relative p-1 min-h-[${compact ? '32px' : '60px'}] text-left transition-colors rounded-lg
				${isCurrentMonth ? 'hover:bg-gray-100 dark:hover:bg-gray-700' : 'opacity-30 cursor-default'}
				${isToday ? 'ring-2 ring-indigo-500' : ''}
			`}
		>
			<span
				className={`text-sm ${isCurrentMonth ? 'text-gray-900 dark:text-white' : 'text-gray-400 dark:text-gray-600'}`}
			>
				{day}
			</span>
			{!compact && (
				<div className="flex flex-wrap gap-0.5 mt-1">
					{calendarDay && calendarDay.completed > 0 && (
						<span
							className="w-2 h-2 rounded-full bg-green-500"
							title={`${calendarDay.completed} completed`}
						/>
					)}
					{calendarDay && calendarDay.failed > 0 && (
						<span
							className="w-2 h-2 rounded-full bg-red-500"
							title={`${calendarDay.failed} failed`}
						/>
					)}
					{calendarDay && calendarDay.running > 0 && (
						<span
							className="w-2 h-2 rounded-full bg-blue-500 animate-pulse"
							title={`${calendarDay.running} running`}
						/>
					)}
					{hasScheduled && (
						<span
							className="w-2 h-2 rounded-full bg-gray-400"
							title={`${scheduledBackups.length} scheduled`}
						/>
					)}
				</div>
			)}
			{compact && (hasBackups || hasScheduled) && (
				<div className="absolute bottom-0.5 left-1/2 -translate-x-1/2 flex gap-0.5">
					{calendarDay && calendarDay.completed > 0 && (
						<span className="w-1.5 h-1.5 rounded-full bg-green-500" />
					)}
					{calendarDay && calendarDay.failed > 0 && (
						<span className="w-1.5 h-1.5 rounded-full bg-red-500" />
					)}
					{hasScheduled && (
						<span className="w-1.5 h-1.5 rounded-full bg-gray-400" />
					)}
				</div>
			)}
		</button>
	);
}

interface DayDetailsModalProps {
	date: string;
	calendarDay?: BackupCalendarDay;
	scheduledBackups: ScheduledBackup[];
	agentMap: Map<string, string>;
	onClose: () => void;
	onSelectBackup?: (backup: Backup) => void;
}

function DayDetailsModal({
	date,
	calendarDay,
	scheduledBackups,
	agentMap,
	onClose,
	onSelectBackup,
}: DayDetailsModalProps) {
	const formattedDate = new Date(date).toLocaleDateString('en-US', {
		weekday: 'long',
		year: 'numeric',
		month: 'long',
		day: 'numeric',
	});

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-lg w-full mx-4 max-h-[90vh] overflow-y-auto">
				<div className="flex items-center justify-between mb-4">
					<h3 className="text-lg font-semibold text-gray-900 dark:text-white">
						{formattedDate}
					</h3>
					<button
						type="button"
						onClick={onClose}
						className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
						aria-label="Close"
					>
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
								d="M6 18L18 6M6 6l12 12"
							/>
						</svg>
					</button>
				</div>

				{calendarDay && (
					<div className="mb-4">
						<div className="flex gap-4 mb-3">
							{calendarDay.completed > 0 && (
								<div className="flex items-center gap-1.5">
									<span className="w-3 h-3 rounded-full bg-green-500" />
									<span className="text-sm text-gray-600 dark:text-gray-400">
										{calendarDay.completed} completed
									</span>
								</div>
							)}
							{calendarDay.failed > 0 && (
								<div className="flex items-center gap-1.5">
									<span className="w-3 h-3 rounded-full bg-red-500" />
									<span className="text-sm text-gray-600 dark:text-gray-400">
										{calendarDay.failed} failed
									</span>
								</div>
							)}
							{calendarDay.running > 0 && (
								<div className="flex items-center gap-1.5">
									<span className="w-3 h-3 rounded-full bg-blue-500" />
									<span className="text-sm text-gray-600 dark:text-gray-400">
										{calendarDay.running} running
									</span>
								</div>
							)}
						</div>

						{calendarDay.backups && calendarDay.backups.length > 0 && (
							<div className="space-y-2">
								<h4 className="text-sm font-medium text-gray-700 dark:text-gray-300">
									Backups
								</h4>
								{calendarDay.backups.map((backup) => {
									const statusColor = getBackupStatusColor(backup.status);
									return (
										<button
											key={backup.id}
											type="button"
											onClick={() => onSelectBackup?.(backup)}
											className="w-full text-left p-3 bg-gray-50 dark:bg-gray-700 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-600 transition-colors"
										>
											<div className="flex items-center justify-between">
												<span className="text-sm font-medium text-gray-900 dark:text-white">
													{agentMap.get(backup.agent_id) ?? 'Unknown Agent'}
												</span>
												<span
													className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium ${statusColor.bg} ${statusColor.text}`}
												>
													<span
														className={`w-1.5 h-1.5 ${statusColor.dot} rounded-full`}
													/>
													{backup.status}
												</span>
											</div>
											<p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
												{formatDateTime(backup.started_at)}
											</p>
										</button>
									);
								})}
							</div>
						)}
					</div>
				)}

				{scheduledBackups.length > 0 && (
					<div>
						<h4 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
							Scheduled Backups
						</h4>
						<div className="space-y-2">
							{scheduledBackups.map((scheduled) => (
								<div
									key={`${scheduled.schedule_id}-${scheduled.scheduled_at}`}
									className="p-3 bg-gray-50 dark:bg-gray-700 rounded-lg"
								>
									<div className="flex items-center justify-between">
										<span className="text-sm font-medium text-gray-900 dark:text-white">
											{scheduled.schedule_name}
										</span>
										<span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-gray-200 dark:bg-gray-600 text-gray-700 dark:text-gray-300">
											<span className="w-1.5 h-1.5 bg-gray-400 rounded-full" />
											Scheduled
										</span>
									</div>
									<p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
										{scheduled.agent_name} -{' '}
										{formatDateTime(scheduled.scheduled_at)}
									</p>
								</div>
							))}
						</div>
					</div>
				)}

				{!calendarDay && scheduledBackups.length === 0 && (
					<p className="text-center text-gray-500 dark:text-gray-400 py-4">
						No backups or scheduled runs for this day
					</p>
				)}

				<div className="flex justify-end mt-6">
					<button
						type="button"
						onClick={onClose}
						className="px-4 py-2 bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-200 dark:hover:bg-gray-600 transition-colors"
					>
						Close
					</button>
				</div>
			</div>
		</div>
	);
}

export function BackupCalendar({
	onSelectBackup,
	compact,
}: BackupCalendarProps) {
	const [currentDate, setCurrentDate] = useState(() => new Date());
	const [selectedDate, setSelectedDate] = useState<string | null>(null);
	const { data: agents } = useAgents();

	const monthParam = formatMonthParam(currentDate);
	const { data: calendarData, isLoading } = useBackupCalendar({
		month: monthParam,
	});

	const agentMap = useMemo(
		() => new Map(agents?.map((a) => [a.id, a.hostname]) ?? []),
		[agents],
	);

	const year = currentDate.getFullYear();
	const month = currentDate.getMonth();
	const daysInMonth = getDaysInMonth(year, month);
	const firstDayOfMonth = getFirstDayOfMonth(year, month);

	const today = new Date();
	const isCurrentMonth =
		today.getFullYear() === year && today.getMonth() === month;
	const todayDate = today.getDate();

	const dayMap = useMemo(() => {
		const map = new Map<string, BackupCalendarDay>();
		if (calendarData?.days) {
			for (const day of calendarData.days) {
				map.set(day.date, day);
			}
		}
		return map;
	}, [calendarData?.days]);

	const scheduledByDate = useMemo(() => {
		const map = new Map<string, ScheduledBackup[]>();
		if (calendarData?.scheduled) {
			for (const s of calendarData.scheduled) {
				const date = s.scheduled_at.split('T')[0];
				if (!map.has(date)) {
					map.set(date, []);
				}
				map.get(date)?.push(s);
			}
		}
		return map;
	}, [calendarData?.scheduled]);

	const prevMonth = () => {
		setCurrentDate(new Date(year, month - 1, 1));
	};

	const nextMonth = () => {
		setCurrentDate(new Date(year, month + 1, 1));
	};

	const goToToday = () => {
		setCurrentDate(new Date());
	};

	const monthName = currentDate.toLocaleDateString('en-US', {
		month: 'long',
		year: 'numeric',
	});

	const weekDays = compact
		? ['S', 'M', 'T', 'W', 'T', 'F', 'S']
		: ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];

	// Static skeleton keys for loading state
	const skeletonKeys = useMemo(
		() => Array.from({ length: 35 }, (_, i) => `skeleton-${i}`),
		[],
	);

	// Generate calendar grid
	const calendarDays: {
		day: number;
		isCurrentMonth: boolean;
		dateString: string;
	}[] = [];

	// Previous month days
	const prevMonthDays = getDaysInMonth(year, month - 1);
	for (let i = firstDayOfMonth - 1; i >= 0; i--) {
		const day = prevMonthDays - i;
		const prevMonthDate = new Date(year, month - 1, day);
		calendarDays.push({
			day,
			isCurrentMonth: false,
			dateString: prevMonthDate.toISOString().split('T')[0],
		});
	}

	// Current month days
	for (let day = 1; day <= daysInMonth; day++) {
		const currentMonthDate = new Date(year, month, day);
		calendarDays.push({
			day,
			isCurrentMonth: true,
			dateString: currentMonthDate.toISOString().split('T')[0],
		});
	}

	// Next month days
	const remainingDays = 42 - calendarDays.length;
	for (let day = 1; day <= remainingDays; day++) {
		const nextMonthDate = new Date(year, month + 1, day);
		calendarDays.push({
			day,
			isCurrentMonth: false,
			dateString: nextMonthDate.toISOString().split('T')[0],
		});
	}

	const selectedCalendarDay = selectedDate
		? dayMap.get(selectedDate)
		: undefined;
	const selectedScheduled = selectedDate
		? (scheduledByDate.get(selectedDate) ?? [])
		: [];

	return (
		<div
			className={`bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 ${compact ? 'p-3' : 'p-6'}`}
		>
			<div className="flex items-center justify-between mb-4">
				<h2
					className={`font-semibold text-gray-900 dark:text-white ${compact ? 'text-sm' : 'text-lg'}`}
				>
					{monthName}
				</h2>
				<div className="flex items-center gap-2">
					{!compact && (
						<button
							type="button"
							onClick={goToToday}
							className="px-2 py-1 text-xs text-indigo-600 dark:text-indigo-400 hover:bg-indigo-50 dark:hover:bg-indigo-900/30 rounded"
						>
							Today
						</button>
					)}
					<button
						type="button"
						onClick={prevMonth}
						className="p-1 text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
						aria-label="Previous month"
					>
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
								d="M15 19l-7-7 7-7"
							/>
						</svg>
					</button>
					<button
						type="button"
						onClick={nextMonth}
						className="p-1 text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
						aria-label="Next month"
					>
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
								d="M9 5l7 7-7 7"
							/>
						</svg>
					</button>
				</div>
			</div>

			{!compact && (
				<div className="flex items-center gap-4 mb-4 text-xs">
					<div className="flex items-center gap-1.5">
						<span className="w-2.5 h-2.5 rounded-full bg-green-500" />
						<span className="text-gray-600 dark:text-gray-400">Completed</span>
					</div>
					<div className="flex items-center gap-1.5">
						<span className="w-2.5 h-2.5 rounded-full bg-red-500" />
						<span className="text-gray-600 dark:text-gray-400">Failed</span>
					</div>
					<div className="flex items-center gap-1.5">
						<span className="w-2.5 h-2.5 rounded-full bg-blue-500" />
						<span className="text-gray-600 dark:text-gray-400">Running</span>
					</div>
					<div className="flex items-center gap-1.5">
						<span className="w-2.5 h-2.5 rounded-full bg-gray-400" />
						<span className="text-gray-600 dark:text-gray-400">Scheduled</span>
					</div>
				</div>
			)}

			{isLoading ? (
				<div className="animate-pulse">
					<div className="grid grid-cols-7 gap-1 mb-1">
						{weekDays.map((day) => (
							<div
								key={day}
								className={`text-center text-gray-500 dark:text-gray-400 ${compact ? 'text-xs py-1' : 'text-sm py-2'}`}
							>
								{day}
							</div>
						))}
					</div>
					<div className="grid grid-cols-7 gap-1">
						{skeletonKeys.map((key) => (
							<div
								key={key}
								className={`${compact ? 'h-8' : 'h-16'} bg-gray-100 dark:bg-gray-700 rounded`}
							/>
						))}
					</div>
				</div>
			) : (
				<>
					<div className="grid grid-cols-7 gap-1 mb-1">
						{weekDays.map((day) => (
							<div
								key={day}
								className={`text-center text-gray-500 dark:text-gray-400 font-medium ${compact ? 'text-xs py-1' : 'text-sm py-2'}`}
							>
								{day}
							</div>
						))}
					</div>
					<div className="grid grid-cols-7 gap-1">
						{calendarDays.map(
							({ day, isCurrentMonth: isCurrent, dateString }) => (
								<CalendarDayCell
									key={dateString}
									day={day}
									isCurrentMonth={isCurrent}
									isToday={isCurrent && isCurrentMonth && day === todayDate}
									calendarDay={dayMap.get(dateString)}
									scheduledBackups={scheduledByDate.get(dateString) ?? []}
									onClick={() => isCurrent && setSelectedDate(dateString)}
									compact={compact}
								/>
							),
						)}
					</div>
				</>
			)}

			{selectedDate && (
				<DayDetailsModal
					date={selectedDate}
					calendarDay={selectedCalendarDay}
					scheduledBackups={selectedScheduled}
					agentMap={agentMap}
					onClose={() => setSelectedDate(null)}
					onSelectBackup={onSelectBackup}
				/>
			)}
		</div>
	);
}

export function MiniBackupCalendar() {
	return <BackupCalendar compact />;
}
