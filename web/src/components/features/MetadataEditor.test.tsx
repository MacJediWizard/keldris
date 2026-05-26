import { render } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

vi.mock('../../hooks/useMetadata', () => ({
	useMetadataSchemas: vi.fn(() => ({ data: [] })),
}));

import { MetadataEditor } from './MetadataEditor';

describe('MetadataEditor', () => {
	it('renders without crashing', () => {
		const { container } = render(
			<MetadataEditor entityType="agent" value={{}} onChange={vi.fn()} />,
		);
		expect(container.firstChild).not.toBeNull();
	});

	it('handles pre-populated values', () => {
		const { container } = render(
			<MetadataEditor
				entityType="repository"
				value={{ team: 'platform' }}
				onChange={vi.fn()}
			/>,
		);
		expect(container.firstChild).not.toBeNull();
	});
});
