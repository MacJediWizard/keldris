import { useCallback, useEffect, useState } from 'react';
import type { ReactNode } from 'react';
import { UpgradePrompt } from '../components/features/UpgradePrompt';
import type { UpgradeEvent } from '../lib/api';
import { onUpgradeRequired } from '../lib/api';
import type { UpgradeFeature } from '../lib/types';

export function UpgradePromptProvider({ children }: { children: ReactNode }) {
	const [info, setInfo] = useState<UpgradeEvent | null>(null);

	useEffect(() => {
		return onUpgradeRequired((event) => {
			setInfo(event);
		});
	}, []);

	const handleClose = useCallback(() => {
		setInfo(null);
	}, []);

	if (!info) return <>{children}</>;

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
