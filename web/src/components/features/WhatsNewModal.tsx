import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import type { ChangelogEntry } from '../../lib/types';

interface WhatsNewModalProps {
	entry: ChangelogEntry | null;
	currentVersion: string | undefined;
	onDismiss: () => void;
}

const SEEN_VERSION_KEY = 'keldris_seen_version';

function ChangeList({
	title,
	items,
	icon,
}: { title: string; items: string[]; icon: string }) {
	if (!items || items.length === 0) return null;

	return (
		<div className="mb-4">
			<h4 className="text-sm font-medium text-gray-700 mb-2 flex items-center gap-2">
				<span>{icon}</span>
				{title}
			</h4>
			<ul className="space-y-1 pl-6">
				{items.slice(0, 5).map((item) => (
					<li key={item} className="text-sm text-gray-600 list-disc">
						{item}
					</li>
				))}
				{items.length > 5 && (
					<li className="text-sm text-gray-400 italic list-none">
						...and {items.length - 5} more
					</li>
				)}
			</ul>
		</div>
	);
}

export function WhatsNewModal({
	entry,
	currentVersion,
	onDismiss,
}: WhatsNewModalProps) {
	const [isOpen, setIsOpen] = useState(false);

	useEffect(() => {
		if (!entry || !currentVersion) return;

		// Check if user has seen this version
		const seenVersion = localStorage.getItem(SEEN_VERSION_KEY);
		if (seenVersion !== currentVersion) {
			setIsOpen(true);
		}
	}, [entry, currentVersion]);

	const handleDismiss = () => {
		if (currentVersion) {
			localStorage.setItem(SEEN_VERSION_KEY, currentVersion);
		}
		setIsOpen(false);
		onDismiss();
	};

	if (!isOpen || !entry) return null;

	const hasChanges =
		(entry.added?.length ?? 0) +
			(entry.changed?.length ?? 0) +
			(entry.fixed?.length ?? 0) >
		0;

	return (
		<div className="fixed inset-0 z-50 flex items-center justify-center">
			{/* Backdrop */}
			<div
				className="absolute inset-0 bg-black/50"
				onClick={handleDismiss}
				onKeyDown={(e) => e.key === 'Escape' && handleDismiss()}
				role="button"
				tabIndex={0}
				aria-label="Close modal"
			/>

			{/* Modal */}
			<div className="relative bg-white rounded-xl shadow-2xl w-full max-w-lg mx-4 overflow-hidden">
				{/* Header */}
				<div className="bg-gradient-to-r from-indigo-600 to-purple-600 px-6 py-8 text-white">
					<div className="flex items-center justify-between">
						<div>
							<p className="text-indigo-100 text-sm font-medium">What's New</p>
							<h2 className="text-2xl font-bold mt-1">
								Keldris v{entry.version}
							</h2>
							{entry.date && (
								<p className="text-indigo-200 text-sm mt-1">{entry.date}</p>
							)}
						</div>
						<div className="w-16 h-16 bg-white/10 rounded-full flex items-center justify-center">
							<svg
								className="w-8 h-8"
								fill="none"
								stroke="currentColor"
								viewBox="0 0 24 24"
								aria-hidden="true"
							>
								<path
									strokeLinecap="round"
									strokeLinejoin="round"
									strokeWidth={2}
									d="M5 3v4M3 5h4M6 17v4m-2-2h4m5-16l2.286 6.857L21 12l-5.714 2.143L13 21l-2.286-6.857L5 12l5.714-2.143L13 3z"
								/>
							</svg>
						</div>
					</div>
				</div>

				{/* Content */}
				<div className="px-6 py-6 max-h-80 overflow-y-auto">
					{hasChanges ? (
						<>
							<ChangeList
								title="New Features"
								items={entry.added ?? []}
								icon="+"
							/>
							<ChangeList
								title="Improvements"
								items={entry.changed ?? []}
								icon="~"
							/>
							<ChangeList
								title="Bug Fixes"
								items={entry.fixed ?? []}
								icon="*"
							/>
						</>
					) : (
						<p className="text-gray-500 text-center py-4">
							This release includes various improvements and bug fixes.
						</p>
					)}
				</div>

				{/* Footer */}
				<div className="px-6 py-4 bg-gray-50 border-t border-gray-200 flex items-center justify-between">
					<Link
						to="/changelog"
						onClick={handleDismiss}
						className="text-sm text-indigo-600 hover:text-indigo-800 font-medium"
					>
						View full changelog
					</Link>
					<button
						type="button"
						onClick={handleDismiss}
						className="px-4 py-2 bg-indigo-600 text-white text-sm font-medium rounded-lg hover:bg-indigo-700 transition-colors"
					>
						Got it
					</button>
				</div>
			</div>
		</div>
	);
}

// Hook to manage "What's New" modal state
export function useWhatsNew() {
	const [dismissed, setDismissed] = useState(false);

	const checkShouldShow = (currentVersion: string | undefined) => {
		if (!currentVersion || dismissed) return false;
		const seenVersion = localStorage.getItem(SEEN_VERSION_KEY);
		return seenVersion !== currentVersion;
	};

	const markAsSeen = (version: string) => {
		localStorage.setItem(SEEN_VERSION_KEY, version);
		setDismissed(true);
	};

	return { checkShouldShow, markAsSeen, dismissed, setDismissed };
}
