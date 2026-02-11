import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { Select } from './Select';

const options = [
	{ value: 'a', label: 'Option A' },
	{ value: 'b', label: 'Option B' },
	{ value: 'c', label: 'Option C', disabled: true },
];

describe('Select', () => {
	it('renders a select element', () => {
		render(<Select options={options} />);
		expect(screen.getByRole('combobox')).toBeInTheDocument();
	});

	it('renders all options', () => {
		render(<Select options={options} />);
		expect(screen.getByText('Option A')).toBeInTheDocument();
		expect(screen.getByText('Option B')).toBeInTheDocument();
		expect(screen.getByText('Option C')).toBeInTheDocument();
	});

	it('renders with label', () => {
		render(<Select label="Category" options={options} />);
		expect(screen.getByLabelText('Category')).toBeInTheDocument();
	});

	it('renders placeholder option', () => {
		render(<Select options={options} placeholder="Select one..." />);
		expect(screen.getByText('Select one...')).toBeInTheDocument();
	});

	it('shows error message', () => {
		render(<Select options={options} error="Required" />);
		expect(screen.getByText('Required')).toBeInTheDocument();
	});

	it('sets aria-invalid when error is present', () => {
		render(<Select options={options} label="Type" error="Required" />);
		expect(screen.getByRole('combobox')).toHaveAttribute(
			'aria-invalid',
			'true',
		);
	});

	it('disables individual options', () => {
		render(<Select options={options} />);
		const disabledOption = screen
			.getByText('Option C')
			.closest('option') as HTMLOptionElement;
		expect(disabledOption.disabled).toBe(true);
	});

	it('fires onChange handler', () => {
		const onChange = vi.fn();
		render(<Select options={options} onChange={onChange} />);
		fireEvent.change(screen.getByRole('combobox'), {
			target: { value: 'b' },
		});
		expect(onChange).toHaveBeenCalled();
	});
});
