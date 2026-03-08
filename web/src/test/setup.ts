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

// Reset mocks between tests
afterEach(() => {
	vi.restoreAllMocks();
	vi.stubGlobal('fetch', fetchMock);
	locationMock.href = '';
});
