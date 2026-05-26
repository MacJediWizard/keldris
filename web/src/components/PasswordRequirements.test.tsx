import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

vi.mock('../hooks/usePasswordPolicy', () => ({
	usePasswordRequirements: vi.fn(),
	useValidatePassword: vi.fn(),
}));

import {
	usePasswordRequirements,
	useValidatePassword,
} from '../hooks/usePasswordPolicy';
import { PasswordRequirements } from './PasswordRequirements';

function setRequirements(
	data: unknown,
	{ isLoading = false }: { isLoading?: boolean } = {},
) {
	vi.mocked(usePasswordRequirements).mockReturnValue({
		data,
		isLoading,
	} as never);
	vi.mocked(useValidatePassword).mockReturnValue({
		mutate: vi.fn(),
		isPending: false,
	} as never);
}

describe('PasswordRequirements', () => {
	it('renders skeleton when loading', () => {
		setRequirements(undefined, { isLoading: true });
		const { container } = render(<PasswordRequirements password="" />);
		expect(container.querySelector('.animate-pulse')).not.toBeNull();
	});

	it('renders nothing when requirements undefined', () => {
		setRequirements(undefined);
		const { container } = render(<PasswordRequirements password="" />);
		expect(container.firstChild).toBeNull();
	});

	it('lists active requirements', () => {
		setRequirements({
			min_length: 8,
			require_uppercase: true,
			require_lowercase: true,
			require_number: true,
			require_special: false,
		});
		render(<PasswordRequirements password="" />);
		expect(screen.getByText('Password Requirements')).toBeDefined();
		expect(screen.getByText('At least 8 characters')).toBeDefined();
		expect(screen.getByText('Uppercase letter (A-Z)')).toBeDefined();
		expect(screen.getByText('Lowercase letter (a-z)')).toBeDefined();
		expect(screen.getByText('Number (0-9)')).toBeDefined();
		expect(screen.queryByText('Special character (!@#$%^&*...)')).toBeNull();
	});

	it('always includes min length even if other rules disabled', () => {
		setRequirements({
			min_length: 12,
			require_uppercase: false,
			require_lowercase: false,
			require_number: false,
			require_special: false,
		});
		render(<PasswordRequirements password="abc" />);
		expect(screen.getByText('At least 12 characters')).toBeDefined();
	});
});
