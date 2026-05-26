import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import {
	DashboardWidgetHelp,
	FormLabelWithHelp,
	HelpTooltip,
	StatusBadgeWithHelp,
} from './HelpTooltip';

describe('HelpTooltip', () => {
	it('renders trigger button with default aria-label', () => {
		render(<HelpTooltip content="some help" />);
		expect(screen.getByRole('button', { name: 'Help' })).toBeDefined();
	});

	it('uses title as aria-label when provided', () => {
		render(<HelpTooltip content="some help" title="Backup Policy" />);
		expect(screen.getByRole('button', { name: 'Backup Policy' })).toBeDefined();
	});

	it('shows tooltip on mouseenter and hides on mouseleave', () => {
		render(<HelpTooltip content="hello world" title="Title" />);
		const btn = screen.getByRole('button', { name: 'Title' });
		fireEvent.mouseEnter(btn);
		expect(screen.getByRole('tooltip')).toBeDefined();
		expect(screen.getByText('hello world')).toBeDefined();
		fireEvent.mouseLeave(btn);
		expect(screen.queryByRole('tooltip')).toBeNull();
	});

	it('parses bold markdown', () => {
		render(<HelpTooltip content="this is **bold** text" />);
		fireEvent.mouseEnter(screen.getByRole('button'));
		expect(screen.getByText('bold').tagName).toBe('STRONG');
	});

	it('parses code markdown', () => {
		render(<HelpTooltip content="run `npm test` first" />);
		fireEvent.mouseEnter(screen.getByRole('button'));
		expect(screen.getByText('npm test').tagName).toBe('CODE');
	});

	it('renders docs link when docsUrl provided', () => {
		render(
			<HelpTooltip content="content" docsUrl="https://docs.example.com" />,
		);
		fireEvent.mouseEnter(screen.getByRole('button'));
		const link = screen.getByRole('link', { name: /Learn more/ });
		expect(link.getAttribute('href')).toBe('https://docs.example.com');
		expect(link.getAttribute('target')).toBe('_blank');
	});
});

describe('FormLabelWithHelp', () => {
	it('renders label and required indicator', () => {
		render(
			<FormLabelWithHelp
				htmlFor="my-field"
				label="My Field"
				helpContent="Help"
				required
			/>,
		);
		expect(screen.getByText('My Field')).toBeDefined();
		expect(screen.getByText('*')).toBeDefined();
	});

	it('hides required indicator when not required', () => {
		render(
			<FormLabelWithHelp
				htmlFor="my-field"
				label="My Field"
				helpContent="Help"
			/>,
		);
		expect(screen.queryByText('*')).toBeNull();
	});
});

describe('StatusBadgeWithHelp', () => {
	it('renders status and tooltip trigger', () => {
		render(
			<StatusBadgeWithHelp
				status="active"
				statusColor={{
					bg: 'bg-green-100',
					text: 'text-green-700',
					dot: 'bg-green-500',
				}}
				helpContent="Status info"
			/>,
		);
		expect(screen.getByText('active')).toBeDefined();
		expect(screen.getByRole('button', { name: 'Help' })).toBeDefined();
	});
});

describe('DashboardWidgetHelp', () => {
	it('renders title, help tooltip, and children', () => {
		render(
			<DashboardWidgetHelp title="My Widget" helpContent="info">
				<span>extra-child</span>
			</DashboardWidgetHelp>,
		);
		expect(screen.getByText('My Widget')).toBeDefined();
		expect(screen.getByText('extra-child')).toBeDefined();
		expect(screen.getByRole('button', { name: 'Help' })).toBeDefined();
	});
});
