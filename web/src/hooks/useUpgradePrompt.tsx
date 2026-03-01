import { useCallback, useEffect, useState } from 'react';
import type { ReactNode } from 'react';
import { useLocation } from 'react-router-dom';
import { UpgradePrompt } from '../components/features/UpgradePrompt';
import type { UpgradeEvent } from '../lib/api';
import { onUpgradeRequired } from '../lib/api';
import type { UpgradeFeature } from '../lib/types';

const PUBLIC_PATHS = ['/setup', '/login', '/reset-password'];

export function UpgradePromptProvider({ children }: { children: ReactNode }) {
	const [info, setInfo] = useState<UpgradeEvent | null>(null);
	const location = useLocation();

	const isPublicPage = PUBLIC_PATHS.some(
		(p) => location.pathname === p || location.pathname.startsWith(`${p}/`),
	);

	useEffect(() => {
		return onUpgradeRequired((event) => {
			setInfo(event);
		});
	}, []);

	const handleClose = useCallback(() => {
		setInfo(null);
	}, []);

	if (!info || isPublicPage) return <>{children}</>;

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
