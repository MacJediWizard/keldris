import { useCallback, useEffect, useState } from 'react';
import type { ReactNode } from 'react';
import { useLocation } from 'react-router-dom';
import { UpgradePrompt } from '../components/features/UpgradePrompt';
import type { UpgradeEvent } from '../lib/api';
import { onUpgradeRequired } from '../lib/api';
import type { UpgradeFeature } from '../lib/types';

const SUPPRESSED_PATHS = ['/setup', '/login', '/reset-password', '/onboarding'];

export function UpgradePromptProvider({ children }: { children: ReactNode }) {
	const [info, setInfo] = useState<UpgradeEvent | null>(null);
	const location = useLocation();

	const isSuppressed = SUPPRESSED_PATHS.some(
		(p) => location.pathname === p || location.pathname.startsWith(`${p}/`),
	);

	useEffect(() => {
		return onUpgradeRequired((event) => {
			setInfo(event);
		});
	}, []);

	// Clear upgrade prompt when navigating to a suppressed page
	useEffect(() => {
		if (isSuppressed && info) {
			setInfo(null);
		}
	}, [isSuppressed, info]);

	const handleClose = useCallback(() => {
		setInfo(null);
	}, []);

	if (!info || isSuppressed) return <>{children}</>;

	return (
		<>
			{children}
			<UpgradePrompt
				feature={info.feature as UpgradeFeature}
				variant="modal"
				onDismiss={handleClose}
			/>
		</>
	);
}
