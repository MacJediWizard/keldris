import { createContext, useContext } from 'react';
import { useActiveMaintenance } from './useMaintenance';

interface ReadOnlyModeContextValue {
	isReadOnly: boolean;
	maintenanceTitle?: string;
	maintenanceMessage?: string;
}

const ReadOnlyModeContext = createContext<ReadOnlyModeContextValue>({
	isReadOnly: false,
});

export function useReadOnlyMode(): ReadOnlyModeContextValue {
	return useContext(ReadOnlyModeContext);
}

export function useReadOnlyModeValue(): ReadOnlyModeContextValue {
	const { data } = useActiveMaintenance();

	return {
		isReadOnly: data?.read_only_mode ?? false,
		maintenanceTitle: data?.active?.title,
		maintenanceMessage: data?.active?.message ?? undefined,
	};
}

export { ReadOnlyModeContext };
