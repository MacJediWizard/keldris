import { act, fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { Modal, ModalBody, ModalFooter, ModalHeader } from './Modal';

describe('Modal', () => {
	it('renders nothing when closed', () => {
		const { container } = render(
			<Modal open={false} onClose={vi.fn()}>
				Content
			</Modal>,
		);
		expect(container).toBeEmptyDOMElement();
	});

	it('renders children when open', () => {
		render(
			<Modal open={true} onClose={vi.fn()}>
				Modal content
			</Modal>,
		);
		expect(screen.getByText('Modal content')).toBeInTheDocument();
	});

	it('renders dialog with aria-modal', () => {
		render(
			<Modal open={true} onClose={vi.fn()}>
				Content
			</Modal>,
		);
		expect(screen.getByRole('dialog')).toHaveAttribute('aria-modal', 'true');
	});

	it('calls onClose when overlay is clicked', () => {
		const onClose = vi.fn();
		render(
			<Modal open={true} onClose={onClose}>
				Content
			</Modal>,
		);
		fireEvent.click(screen.getByTestId('modal-overlay'));
		expect(onClose).toHaveBeenCalledTimes(1);
	});

	it('calls onClose when close button is clicked', () => {
		const onClose = vi.fn();
		render(
			<Modal open={true} onClose={onClose}>
				Content
			</Modal>,
		);
		fireEvent.click(screen.getByLabelText('Close'));
		expect(onClose).toHaveBeenCalledTimes(1);
	});

	it('calls onClose on Escape key', () => {
		const onClose = vi.fn();
		render(
			<Modal open={true} onClose={onClose}>
				Content
			</Modal>,
		);
		fireEvent.keyDown(document, { key: 'Escape' });
		expect(onClose).toHaveBeenCalledTimes(1);
	});

	it('overlay does not have role=button or tabIndex', () => {
		render(
			<Modal open={true} onClose={vi.fn()}>
				Content
			</Modal>,
		);
		const overlay = screen.getByTestId('modal-overlay');
		expect(overlay).not.toHaveAttribute('role');
		expect(overlay).not.toHaveAttribute('tabindex');
	});

	it('has aria-labelledby pointing to the ModalHeader title', () => {
		render(
			<Modal open={true} onClose={vi.fn()}>
				<ModalHeader>Test Title</ModalHeader>
				<ModalBody>Body</ModalBody>
			</Modal>,
		);
		const dialog = screen.getByRole('dialog');
		const labelledBy = dialog.getAttribute('aria-labelledby');
		expect(labelledBy).toBeTruthy();

		const heading = screen.getByText('Test Title');
		expect(heading.id).toBe(labelledBy);
	});

	it('moves focus into modal on mount', () => {
		render(
			<Modal open={true} onClose={vi.fn()}>
				<ModalBody>
					<button type="button">First</button>
					<button type="button">Second</button>
				</ModalBody>
			</Modal>,
		);
		// The close button is the first focusable element
		expect(document.activeElement).toBe(screen.getByLabelText('Close'));
	});

	it('traps Tab within the modal', () => {
		render(
			<Modal open={true} onClose={vi.fn()}>
				<ModalBody>
					<button type="button">Action A</button>
					<button type="button">Action B</button>
				</ModalBody>
			</Modal>,
		);

		const closeBtn = screen.getByLabelText('Close');
		const actionB = screen.getByText('Action B');

		// Focus is on close button (first focusable)
		expect(document.activeElement).toBe(closeBtn);

		// Tab from last element wraps to first
		act(() => actionB.focus());
		expect(document.activeElement).toBe(actionB);
		fireEvent.keyDown(document, { key: 'Tab' });
		expect(document.activeElement).toBe(closeBtn);

		// Shift+Tab from first element wraps to last
		act(() => closeBtn.focus());
		fireEvent.keyDown(document, { key: 'Tab', shiftKey: true });
		expect(document.activeElement).toBe(actionB);
	});

	it('restores focus to previously focused element on unmount', () => {
		const trigger = document.createElement('button');
		trigger.textContent = 'Open Modal';
		document.body.appendChild(trigger);
		trigger.focus();
		expect(document.activeElement).toBe(trigger);

		const { unmount } = render(
			<Modal open={true} onClose={vi.fn()}>
				<ModalBody>Content</ModalBody>
			</Modal>,
		);

		// Focus moved into modal
		expect(document.activeElement).not.toBe(trigger);

		unmount();

		// Focus restored to trigger
		expect(document.activeElement).toBe(trigger);
		document.body.removeChild(trigger);
	});
});

describe('ModalHeader', () => {
	it('renders heading text', () => {
		render(
			<Modal open={true} onClose={vi.fn()}>
				<ModalHeader>Title</ModalHeader>
			</Modal>,
		);
		expect(screen.getByText('Title')).toBeInTheDocument();
		expect(screen.getByText('Title').tagName).toBe('H3');
	});
});

describe('ModalBody', () => {
	it('renders body content', () => {
		render(<ModalBody>Body content</ModalBody>);
		expect(screen.getByText('Body content')).toBeInTheDocument();
	});
});

describe('ModalFooter', () => {
	it('renders footer content', () => {
		render(<ModalFooter>Footer actions</ModalFooter>);
		expect(screen.getByText('Footer actions')).toBeInTheDocument();
	});
});
