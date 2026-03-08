import '@testing-library/jest-dom';

// Mock window.location for auth redirect tests
const locationMock = {
	...window.location,
	href: '',
	assign: vi.fn(),
	replace: vi.fn(),
	reload: vi.fn(),
};
Object.defineProperty(window, 'location', {
	value: locationMock,
	writable: true,
});

// Mock window.confirm
window.confirm = vi.fn(() => true);

// Mock IntersectionObserver
class MockIntersectionObserver {
	observe = vi.fn();
	unobserve = vi.fn();
	disconnect = vi.fn();
}
Object.defineProperty(window, 'IntersectionObserver', {
	value: MockIntersectionObserver,
});

// Mock matchMedia
Object.defineProperty(window, 'matchMedia', {
	value: vi.fn().mockImplementation((query: string) => ({
		matches: false,
		media: query,
		onchange: null,
		addListener: vi.fn(),
		removeListener: vi.fn(),
		addEventListener: vi.fn(),
		removeEventListener: vi.fn(),
		dispatchEvent: vi.fn(),
	})),
});

// Mock global fetch to prevent undici errors in CI
const fetchMock = vi.fn(() =>
	Promise.resolve(new Response('{}', { status: 200 })),
);
vi.stubGlobal('fetch', fetchMock);

// Suppress undici InvalidArgumentError in jsdom (Node 20+ CI issue)
// This error is thrown asynchronously by undici internals and is not
// related to application code. Vitest catches it as an unhandled error.
const originalEmit = process.emit.bind(process);
// biome-ignore lint/suspicious/noExplicitAny: process.emit override requires any
process.emit = ((event: string, ...args: any[]) => {
	if (event === 'unhandledRejection') {
		const error = args[0];
		if (error instanceof Error && error.message === 'invalid onError method') {
			return false;
		}
	}
	return originalEmit(event, ...args);
}) as typeof process.emit;

// Reset mocks between tests
afterEach(() => {
	vi.restoreAllMocks();
	vi.stubGlobal('fetch', fetchMock);
	locationMock.href = '';
});
