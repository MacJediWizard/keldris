import { useEffect, useState } from 'react';
import type { LicenseTier } from '../../lib/types';

interface UpgradePromptProps {
	feature: string;
	currentTier: LicenseTier;
	open: boolean;
	onClose: () => void;
}

export function UpgradePrompt({
	feature,
	currentTier,
	open,
	onClose,
}: UpgradePromptProps) {
	const [visible, setVisible] = useState(open);

	useEffect(() => {
		setVisible(open);
	}, [open]);

	if (!visible) return null;

	return (
		<div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
			<div className="w-full max-w-md rounded-lg bg-white p-6 shadow-xl">
				<div className="mb-4 flex items-center gap-3">
					<div className="flex h-10 w-10 items-center justify-center rounded-full bg-amber-100">
						<svg
							aria-hidden="true"
							className="h-5 w-5 text-amber-600"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
						>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"
							/>
						</svg>
					</div>
					<h3 className="text-lg font-semibold text-gray-900">
						Upgrade Required
					</h3>
				</div>

				<p className="mb-2 text-sm text-gray-600">
					The <span className="font-medium text-gray-900">{feature}</span>{' '}
					feature is not available on your current{' '}
					<span className="font-medium capitalize">{currentTier}</span> plan.
				</p>
				<p className="mb-6 text-sm text-gray-500">
					Upgrade your license to unlock this feature and more.
				</p>

				<div className="flex justify-end gap-3">
					<button
						type="button"
						onClick={onClose}
						className="rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50"
					>
						Close
					</button>
					<a
						href="/license"
						className="inline-flex items-center rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-500"
					>
						View License
					</a>
				</div>
			</div>
		</div>
	);
}
