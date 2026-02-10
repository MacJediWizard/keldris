import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, type RenderOptions } from '@testing-library/react';
import type { ReactElement, ReactNode } from 'react';
import { BrowserRouter } from 'react-router-dom';

function createTestQueryClient() {
	return new QueryClient({
		defaultOptions: {
			queries: { retry: false, gcTime: 0 },
			mutations: { retry: false },
		},
	});
}

function AllProviders({ children }: { children: ReactNode }) {
	const queryClient = createTestQueryClient();
	return (
		<QueryClientProvider client={queryClient}>
			<BrowserRouter>{children}</BrowserRouter>
		</QueryClientProvider>
	);
}

export function renderWithProviders(
	ui: ReactElement,
	options?: Omit<RenderOptions, 'wrapper'>,
) {
	return render(ui, { wrapper: AllProviders, ...options });
}

export function createWrapper() {
	const queryClient = createTestQueryClient();
	return function Wrapper({ children }: { children: ReactNode }) {
		return (
			<QueryClientProvider client={queryClient}>
				{children}
			</QueryClientProvider>
		);
	};
}
