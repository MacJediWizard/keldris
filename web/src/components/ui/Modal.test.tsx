import { fireEvent, render, screen } from '@testing-library/react';
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
});

describe('ModalHeader', () => {
	it('renders heading text', () => {
		render(<ModalHeader>Title</ModalHeader>);
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
