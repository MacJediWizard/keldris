import { createContext, useContext } from 'react';
import { useActiveMaintenance } from './useMaintenance';

interface ReadOnlyModeContext {
	isReadOnly: boolean;
	maintenanceTitle?: string;
	maintenanceMessage?: string;
}

const ReadOnlyModeContext = createContext<ReadOnlyModeContext>({
	isReadOnly: false,
});

export function useReadOnlyMode(): ReadOnlyModeContext {
	return useContext(ReadOnlyModeContext);
}

export function useReadOnlyModeValue(): ReadOnlyModeContext {
	const { data } = useActiveMaintenance();

	return {
		isReadOnly: data?.read_only_mode ?? false,
		maintenanceTitle: data?.active?.title,
		maintenanceMessage: data?.active?.message ?? undefined,
	};
}

export { ReadOnlyModeContext };
