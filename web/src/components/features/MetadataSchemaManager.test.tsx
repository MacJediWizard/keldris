import { render } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

vi.mock('../../hooks/useMetadata', () => ({
	useMetadataSchemas: vi.fn(() => ({ data: [], isLoading: false })),
	useMetadataFieldTypes: vi.fn(() => ({ data: [] })),
	useMetadataEntityTypes: vi.fn(() => ({ data: [] })),
	useCreateMetadataSchema: vi.fn(() => ({ mutateAsync: vi.fn() })),
	useUpdateMetadataSchema: vi.fn(() => ({ mutateAsync: vi.fn() })),
	useDeleteMetadataSchema: vi.fn(() => ({ mutateAsync: vi.fn() })),
}));

import { MetadataSchemaManager } from './MetadataSchemaManager';

describe('MetadataSchemaManager', () => {
	it('renders without crashing', () => {
		const { container } = render(<MetadataSchemaManager entityType="agent" />);
		expect(container.firstChild).not.toBeNull();
	});
});
