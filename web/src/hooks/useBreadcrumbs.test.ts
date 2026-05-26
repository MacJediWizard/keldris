import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { renderHook } from '@testing-library/react';
import { type ReactNode, createElement } from 'react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { useBreadcrumbs, useBreadcrumbsWithName } from './useBreadcrumbs';

function mockFetch(data: unknown, ok = true, status = 200) {
	const fn = vi.fn().mockResolvedValue({
		ok,
		status,
		json: () => Promise.resolve(data),
	} as unknown as Response);
	vi.stubGlobal('fetch', fn);
	return fn;
}

function makeWrapper(initialPath: string, routePattern: string) {
	const queryClient = new QueryClient({
		defaultOptions: {
			queries: { retry: false, gcTime: 0 },
			mutations: { retry: false },
		},
	});

	return function Wrapper({ children }: { children: ReactNode }) {
		return createElement(
			QueryClientProvider,
			{ client: queryClient },
			createElement(
				MemoryRouter,
				{ initialEntries: [initialPath] },
				createElement(
					Routes,
					null,
					createElement(Route, { path: routePattern, element: children }),
				),
			),
		);
	};
}

describe('useBreadcrumbs', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('returns Dashboard breadcrumb for root path', () => {
		mockFetch({});
		const { result } = renderHook(() => useBreadcrumbs(), {
			wrapper: makeWrapper('/', '/'),
		});

		expect(result.current.breadcrumbs).toHaveLength(1);
		expect(result.current.breadcrumbs[0]).toEqual({
			label: 'Dashboard',
			path: '/',
			isCurrentPage: true,
		});
		expect(result.current.isLoading).toBe(false);
	});

	it('builds breadcrumb trail for nested static routes', () => {
		mockFetch({});
		const { result } = renderHook(() => useBreadcrumbs(), {
			wrapper: makeWrapper('/agents', '/agents'),
		});

		expect(result.current.breadcrumbs).toHaveLength(2);
		expect(result.current.breadcrumbs[0].label).toBe('Dashboard');
		expect(result.current.breadcrumbs[1].label).toBe('Agents');
		expect(result.current.breadcrumbs[1].isCurrentPage).toBe(true);
	});
});

describe('useBreadcrumbsWithName', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('returns Dashboard breadcrumb for root path', () => {
		const { result } = renderHook(() => useBreadcrumbsWithName(), {
			wrapper: makeWrapper('/', '/'),
		});

		expect(result.current.breadcrumbs).toHaveLength(1);
		expect(result.current.breadcrumbs[0].label).toBe('Dashboard');
		expect(result.current.isLoading).toBe(false);
	});

	it('uses provided name when last segment is a UUID', () => {
		const uuid = '12345678-1234-1234-1234-123456789abc';
		const { result } = renderHook(() => useBreadcrumbsWithName('My Agent'), {
			wrapper: makeWrapper(`/agents/${uuid}`, '/agents/:id'),
		});

		const last =
			result.current.breadcrumbs[result.current.breadcrumbs.length - 1];
		expect(last.label).toBe('My Agent');
		expect(last.isCurrentPage).toBe(true);
	});
});
