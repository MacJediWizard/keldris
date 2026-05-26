import { renderHook } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

vi.mock('./useMaintenance', () => ({
	useActiveMaintenance: vi.fn(),
}));

import { useActiveMaintenance } from './useMaintenance';
import { useReadOnlyMode, useReadOnlyModeValue } from './useReadOnlyMode';

describe('useReadOnlyMode', () => {
	it('returns default not-readonly when no provider', () => {
		const { result } = renderHook(() => useReadOnlyMode());
		expect(result.current.isReadOnly).toBe(false);
	});
});

describe('useReadOnlyModeValue', () => {
	it('returns isReadOnly=false when no active maintenance', () => {
		vi.mocked(useActiveMaintenance).mockReturnValue({
			data: undefined,
		} as never);
		const { result } = renderHook(() => useReadOnlyModeValue());
		expect(result.current.isReadOnly).toBe(false);
	});

	it('reflects read_only_mode flag from active maintenance', () => {
		vi.mocked(useActiveMaintenance).mockReturnValue({
			data: {
				read_only_mode: true,
				active: { title: 'Maintenance', message: 'Coming back soon' },
			},
		} as never);
		const { result } = renderHook(() => useReadOnlyModeValue());
		expect(result.current.isReadOnly).toBe(true);
		expect(result.current.maintenanceTitle).toBe('Maintenance');
		expect(result.current.maintenanceMessage).toBe('Coming back soon');
	});

	it('returns undefined message when maintenance message is null', () => {
		vi.mocked(useActiveMaintenance).mockReturnValue({
			data: { read_only_mode: false, active: { title: 'x', message: null } },
		} as never);
		const { result } = renderHook(() => useReadOnlyModeValue());
		expect(result.current.maintenanceMessage).toBeUndefined();
	});
});
