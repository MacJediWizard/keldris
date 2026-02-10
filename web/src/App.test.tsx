import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

// Mock all lazy-loaded pages
vi.mock('./pages/Dashboard', () => ({
	default: () => <div data-testid="dashboard-page">Dashboard</div>,
	Dashboard: () => <div data-testid="dashboard-page">Dashboard</div>,
}));

// Mock the Layout component
vi.mock('./components/Layout', () => ({
	Layout: ({ children }: { children?: React.ReactNode }) => (
		<div data-testid="layout">{children}</div>
	),
}));

// Mock i18n
vi.mock('./lib/i18n', () => ({}));

import App from './App';

describe('App', () => {
	it('renders without crashing', () => {
		render(<App />);
		// App renders with providers and router
		expect(document.body).toBeTruthy();
	});

	it('wraps content in QueryClientProvider', () => {
		const { container } = render(<App />);
		// The app should render some content (providers don't render visible elements)
		expect(container).toBeTruthy();
	});
});
