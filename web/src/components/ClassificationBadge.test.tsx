import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import {
	ClassificationBadge,
	ClassificationLevelSelect,
	DataTypeMultiSelect,
} from './ClassificationBadge';

describe('ClassificationBadge', () => {
	it('renders Public label by default', () => {
		render(<ClassificationBadge />);
		expect(screen.getByText('Public')).toBeDefined();
	});

	it('renders Internal label', () => {
		render(<ClassificationBadge level="internal" />);
		expect(screen.getByText('Internal')).toBeDefined();
	});

	it('renders Confidential label', () => {
		render(<ClassificationBadge level="confidential" />);
		expect(screen.getByText('Confidential')).toBeDefined();
	});

	it('renders Restricted label', () => {
		render(<ClassificationBadge level="restricted" />);
		expect(screen.getByText('Restricted')).toBeDefined();
	});

	it('falls back to public colors for unknown level', () => {
		render(<ClassificationBadge level="bogus" />);
		expect(screen.getByText('bogus')).toBeDefined();
	});

	it('shows data type tags when showDataTypes is true', () => {
		render(
			<ClassificationBadge
				level="confidential"
				dataTypes={['pii', 'phi']}
				showDataTypes
			/>,
		);
		expect(screen.getByText('PII')).toBeDefined();
		expect(screen.getByText('PHI')).toBeDefined();
	});

	it('hides data type tags by default', () => {
		render(<ClassificationBadge level="restricted" dataTypes={['pii']} />);
		expect(screen.queryByText('PII')).toBeNull();
	});
});

describe('ClassificationLevelSelect', () => {
	it('renders 4 options', () => {
		const onChange = vi.fn();
		render(<ClassificationLevelSelect value="public" onChange={onChange} />);
		expect(screen.getByRole('option', { name: 'Public' })).toBeDefined();
		expect(screen.getByRole('option', { name: 'Internal' })).toBeDefined();
		expect(screen.getByRole('option', { name: 'Confidential' })).toBeDefined();
		expect(screen.getByRole('option', { name: 'Restricted' })).toBeDefined();
	});

	it('fires onChange when selection changes', () => {
		const onChange = vi.fn();
		render(<ClassificationLevelSelect value="public" onChange={onChange} />);
		fireEvent.change(screen.getByRole('combobox'), {
			target: { value: 'restricted' },
		});
		expect(onChange).toHaveBeenCalledWith('restricted');
	});

	it('renders disabled when disabled=true', () => {
		render(
			<ClassificationLevelSelect value="public" onChange={() => {}} disabled />,
		);
		expect(screen.getByRole('combobox')).toBeDisabled();
	});
});

describe('DataTypeMultiSelect', () => {
	it('renders all data type buttons', () => {
		render(<DataTypeMultiSelect value={[]} onChange={() => {}} />);
		expect(screen.getByRole('button', { name: 'PII' })).toBeDefined();
		expect(screen.getByRole('button', { name: 'PHI' })).toBeDefined();
		expect(screen.getByRole('button', { name: 'PCI' })).toBeDefined();
		expect(screen.getByRole('button', { name: 'Proprietary' })).toBeDefined();
		expect(screen.getByRole('button', { name: 'General' })).toBeDefined();
	});

	it('toggles a type on click', () => {
		const onChange = vi.fn();
		render(<DataTypeMultiSelect value={['general']} onChange={onChange} />);
		screen.getByRole('button', { name: 'PII' }).click();
		// adding pii should remove 'general' (per handleToggle logic)
		expect(onChange).toHaveBeenCalledWith(['pii']);
	});

	it('removes a type when already selected', () => {
		const onChange = vi.fn();
		render(<DataTypeMultiSelect value={['pii', 'phi']} onChange={onChange} />);
		screen.getByRole('button', { name: 'PII' }).click();
		expect(onChange).toHaveBeenCalledWith(['phi']);
	});

	it('falls back to general when last type removed', () => {
		const onChange = vi.fn();
		render(<DataTypeMultiSelect value={['pii']} onChange={onChange} />);
		screen.getByRole('button', { name: 'PII' }).click();
		expect(onChange).toHaveBeenCalledWith(['general']);
	});
});
