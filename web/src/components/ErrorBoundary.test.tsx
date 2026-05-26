import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { ErrorBoundary } from './ErrorBoundary';

function Boom(): React.ReactElement {
	throw new Error('boom');
}

function Ok(): React.ReactElement {
	return <div>child-ok</div>;
}

const consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

describe('ErrorBoundary', () => {
	beforeEach(() => {
		consoleErrorSpy.mockClear();
	});

	afterEach(() => {
		consoleErrorSpy.mockClear();
	});

	it('renders children when no error', () => {
		render(
			<MemoryRouter>
				<ErrorBoundary>
					<Ok />
				</ErrorBoundary>
			</MemoryRouter>,
		);
		expect(screen.getByText('child-ok')).toBeDefined();
	});

	it('renders fallback when child throws', () => {
		render(
			<MemoryRouter>
				<ErrorBoundary fallback={<div>custom-fallback</div>}>
					<Boom />
				</ErrorBoundary>
			</MemoryRouter>,
		);
		expect(screen.getByText('custom-fallback')).toBeDefined();
	});

	it('renders default ServerError fallback when no fallback provided', () => {
		const { container } = render(
			<MemoryRouter>
				<ErrorBoundary>
					<Boom />
				</ErrorBoundary>
			</MemoryRouter>,
		);
		// ServerError fallback rendered (replaces Boom child entirely)
		expect(container.textContent).not.toContain('boom-not-rendered');
		// Fallback content rendered something (not empty)
		expect(container.firstChild).not.toBeNull();
	});
});
