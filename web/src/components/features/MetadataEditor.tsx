import { useEffect, useState } from 'react';
import { useMetadataSchemas } from '../../hooks/useMetadata';
import type { MetadataEntityType, MetadataSchema } from '../../lib/types';

interface MetadataEditorProps {
	entityType: MetadataEntityType;
	metadata: Record<string, unknown>;
	onChange: (metadata: Record<string, unknown>) => void;
	disabled?: boolean;
}

export function MetadataEditor({
	entityType,
	metadata,
	onChange,
	disabled = false,
}: MetadataEditorProps) {
	const { data: schemas, isLoading } = useMetadataSchemas(entityType);
	const [localMetadata, setLocalMetadata] = useState<Record<string, unknown>>(
		metadata || {},
	);

	useEffect(() => {
		setLocalMetadata(metadata || {});
	}, [metadata]);

	const handleFieldChange = (fieldKey: string, value: unknown) => {
		const newMetadata = { ...localMetadata, [fieldKey]: value };
		setLocalMetadata(newMetadata);
		onChange(newMetadata);
	};

	if (isLoading) {
		return (
			<div className="text-sm text-gray-500">Loading metadata fields...</div>
		);
	}

	if (!schemas || schemas.length === 0) {
		return (
			<div className="text-sm text-gray-500 italic">
				No custom metadata fields defined for this entity type.
			</div>
		);
	}

	return (
		<div className="space-y-4">
			{schemas.map((schema) => (
				<MetadataField
					key={schema.id}
					schema={schema}
					value={localMetadata[schema.field_key]}
					onChange={(value) => handleFieldChange(schema.field_key, value)}
					disabled={disabled}
				/>
			))}
		</div>
	);
}

interface MetadataFieldProps {
	schema: MetadataSchema;
	value: unknown;
	onChange: (value: unknown) => void;
	disabled?: boolean;
}

function MetadataField({
	schema,
	value,
	onChange,
	disabled = false,
}: MetadataFieldProps) {
	const fieldId = `metadata-${schema.id}`;

	const renderField = () => {
		switch (schema.field_type) {
			case 'text':
				return (
					<input
						id={fieldId}
						type="text"
						value={(value as string) || ''}
						onChange={(e) => onChange(e.target.value)}
						disabled={disabled}
						className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm disabled:bg-gray-100 disabled:cursor-not-allowed"
						placeholder={schema.description || ''}
					/>
				);

			case 'number':
				return (
					<input
						id={fieldId}
						type="number"
						value={(value as number) ?? ''}
						onChange={(e) =>
							onChange(e.target.value ? Number(e.target.value) : undefined)
						}
						disabled={disabled}
						min={schema.validation?.min}
						max={schema.validation?.max}
						className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm disabled:bg-gray-100 disabled:cursor-not-allowed"
						placeholder={schema.description || ''}
					/>
				);

			case 'date':
				return (
					<input
						id={fieldId}
						type="date"
						value={(value as string) || ''}
						onChange={(e) => onChange(e.target.value || undefined)}
						disabled={disabled}
						min={schema.validation?.min_date}
						max={schema.validation?.max_date}
						className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm disabled:bg-gray-100 disabled:cursor-not-allowed"
					/>
				);

			case 'select':
				return (
					<select
						id={fieldId}
						value={(value as string) || ''}
						onChange={(e) => onChange(e.target.value || undefined)}
						disabled={disabled}
						className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm disabled:bg-gray-100 disabled:cursor-not-allowed"
					>
						<option value="">Select...</option>
						{schema.options?.map((option) => (
							<option key={option.value} value={option.value}>
								{option.label}
							</option>
						))}
					</select>
				);

			case 'boolean':
				return (
					<div className="mt-1">
						<label className="inline-flex items-center">
							<input
								type="checkbox"
								checked={(value as boolean) || false}
								onChange={(e) => onChange(e.target.checked)}
								disabled={disabled}
								className="rounded border-gray-300 text-blue-600 shadow-sm focus:border-blue-500 focus:ring-blue-500 disabled:cursor-not-allowed"
							/>
							<span className="ml-2 text-sm text-gray-600">
								{schema.description || 'Enable'}
							</span>
						</label>
					</div>
				);

			default:
				return (
					<div className="text-sm text-gray-500">
						Unsupported field type: {schema.field_type}
					</div>
				);
		}
	};

	return (
		<div>
			<label htmlFor={fieldId} className="block text-sm font-medium text-gray-700">
				{schema.name}
				{schema.required && <span className="text-red-500 ml-1">*</span>}
			</label>
			{schema.field_type !== 'boolean' && schema.description && (
				<p className="text-xs text-gray-500 mt-0.5">{schema.description}</p>
			)}
			{renderField()}
		</div>
	);
}

// Export for use in other components
export default MetadataEditor;
