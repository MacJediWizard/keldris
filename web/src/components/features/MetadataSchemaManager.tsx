import { useState } from 'react';
import {
	useCreateMetadataSchema,
	useDeleteMetadataSchema,
	useMetadataFieldTypes,
	useMetadataSchemas,
	useUpdateMetadataSchema,
} from '../../hooks/useMetadata';
import type {
	CreateMetadataSchemaRequest,
	MetadataEntityType,
	MetadataFieldType,
	MetadataSchema,
	MetadataSelectOption,
	MetadataValidationRules,
} from '../../lib/types';

interface MetadataSchemaManagerProps {
	entityType: MetadataEntityType;
}

export function MetadataSchemaManager({
	entityType,
}: MetadataSchemaManagerProps) {
	const { data: schemas, isLoading } = useMetadataSchemas(entityType);
	const { data: fieldTypesData } = useMetadataFieldTypes();
	const createSchema = useCreateMetadataSchema();
	const updateSchema = useUpdateMetadataSchema();
	const deleteSchema = useDeleteMetadataSchema();

	const [isCreating, setIsCreating] = useState(false);
	const [editingId, setEditingId] = useState<string | null>(null);

	const entityTypeLabel =
		entityType.charAt(0).toUpperCase() + entityType.slice(1);

	if (isLoading) {
		return <div className="text-sm text-gray-500">Loading schemas...</div>;
	}

	return (
		<div className="space-y-4">
			<div className="flex items-center justify-between">
				<h3 className="text-lg font-medium text-gray-900">
					{entityTypeLabel} Metadata Fields
				</h3>
				{!isCreating && (
					<button
						type="button"
						onClick={() => setIsCreating(true)}
						className="inline-flex items-center px-3 py-1.5 border border-transparent text-sm font-medium rounded-md shadow-sm text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
					>
						Add Field
					</button>
				)}
			</div>

			{isCreating && (
				<SchemaForm
					entityType={entityType}
					fieldTypes={fieldTypesData?.types || []}
					onSave={async (data) => {
						await createSchema.mutateAsync(data);
						setIsCreating(false);
					}}
					onCancel={() => setIsCreating(false)}
					isSaving={createSchema.isPending}
				/>
			)}

			{schemas && schemas.length > 0 ? (
				<div className="space-y-3">
					{schemas.map((schema) => (
						<div key={schema.id}>
							{editingId === schema.id ? (
								<SchemaForm
									entityType={entityType}
									schema={schema}
									fieldTypes={fieldTypesData?.types || []}
									onSave={async (data) => {
										await updateSchema.mutateAsync({
											id: schema.id,
											data,
										});
										setEditingId(null);
									}}
									onCancel={() => setEditingId(null)}
									isSaving={updateSchema.isPending}
								/>
							) : (
								<SchemaCard
									schema={schema}
									onEdit={() => setEditingId(schema.id)}
									onDelete={async () => {
										if (
											confirm(
												`Are you sure you want to delete the "${schema.name}" field?`,
											)
										) {
											await deleteSchema.mutateAsync({
												id: schema.id,
												entityType,
											});
										}
									}}
									isDeleting={deleteSchema.isPending}
								/>
							)}
						</div>
					))}
				</div>
			) : (
				!isCreating && (
					<div className="text-center py-6 bg-gray-50 rounded-lg border border-dashed border-gray-300">
						<p className="text-sm text-gray-500">
							No custom fields defined for {entityTypeLabel.toLowerCase()}s yet.
						</p>
						<button
							type="button"
							onClick={() => setIsCreating(true)}
							className="mt-2 text-sm text-blue-600 hover:text-blue-500"
						>
							Create your first custom field
						</button>
					</div>
				)
			)}
		</div>
	);
}

interface SchemaCardProps {
	schema: MetadataSchema;
	onEdit: () => void;
	onDelete: () => void;
	isDeleting: boolean;
}

function SchemaCard({ schema, onEdit, onDelete, isDeleting }: SchemaCardProps) {
	const fieldTypeLabels: Record<MetadataFieldType, string> = {
		text: 'Text',
		number: 'Number',
		date: 'Date',
		select: 'Select',
		boolean: 'Boolean',
	};

	return (
		<div className="bg-white border border-gray-200 rounded-lg p-4 flex items-center justify-between">
			<div className="flex-1">
				<div className="flex items-center gap-2">
					<span className="font-medium text-gray-900">{schema.name}</span>
					<span className="text-xs px-2 py-0.5 bg-gray-100 text-gray-600 rounded">
						{fieldTypeLabels[schema.field_type]}
					</span>
					{schema.required && (
						<span className="text-xs px-2 py-0.5 bg-red-100 text-red-600 rounded">
							Required
						</span>
					)}
				</div>
				<div className="text-sm text-gray-500 mt-1">
					Key:{' '}
				<code className="bg-gray-100 px-1 rounded">{schema.field_key}</code>
					{schema.description && <span> - {schema.description}</span>}
				</div>
				{schema.field_type === 'select' && schema.options && (
					<div className="text-xs text-gray-400 mt-1">
						Options: {schema.options.map((o) => o.label).join(', ')}
					</div>
				)}
			</div>
			<div className="flex items-center gap-2">
				<button
					type="button"
					onClick={onEdit}
					className="text-sm text-blue-600 hover:text-blue-500"
				>
					Edit
				</button>
				<button
					type="button"
					onClick={onDelete}
					disabled={isDeleting}
					className="text-sm text-red-600 hover:text-red-500 disabled:opacity-50"
				>
					{isDeleting ? 'Deleting...' : 'Delete'}
				</button>
			</div>
		</div>
	);
}

interface SchemaFormProps {
	entityType: MetadataEntityType;
	schema?: MetadataSchema;
	fieldTypes: { type: MetadataFieldType; label: string; description: string }[];
	onSave: (data: CreateMetadataSchemaRequest) => Promise<void>;
	onCancel: () => void;
	isSaving: boolean;
}

function SchemaForm({
	entityType,
	schema,
	fieldTypes,
	onSave,
	onCancel,
	isSaving,
}: SchemaFormProps) {
	const [name, setName] = useState(schema?.name || '');
	const [fieldKey, setFieldKey] = useState(schema?.field_key || '');
	const [fieldType, setFieldType] = useState<MetadataFieldType>(
		schema?.field_type || 'text',
	);
	const [description, setDescription] = useState(schema?.description || '');
	const [required, setRequired] = useState(schema?.required || false);
	const [options, setOptions] = useState<MetadataSelectOption[]>(
		schema?.options || [{ value: '', label: '' }],
	);
	const [validation, setValidation] = useState<MetadataValidationRules>(
		schema?.validation || {},
	);

	const handleNameChange = (newName: string) => {
		setName(newName);
		// Auto-generate field_key from name if not editing
		if (!schema) {
			const key = newName
				.toLowerCase()
				.replace(/[^a-z0-9]+/g, '_')
				.replace(/^_|_$/g, '')
				.substring(0, 100);
			setFieldKey(key);
		}
	};

	const handleAddOption = () => {
		setOptions([...options, { value: '', label: '' }]);
	};

	const handleRemoveOption = (index: number) => {
		setOptions(options.filter((_, i) => i !== index));
	};

	const handleOptionChange = (
		index: number,
		field: 'value' | 'label',
		value: string,
	) => {
		const newOptions = [...options];
		newOptions[index] = { ...newOptions[index], [field]: value };
		setOptions(newOptions);
	};

	const handleSubmit = async (e: React.FormEvent) => {
		e.preventDefault();

		const data: CreateMetadataSchemaRequest = {
			entity_type: entityType,
			name,
			field_key: fieldKey,
			field_type: fieldType,
			description: description || undefined,
			required,
			options:
				fieldType === 'select'
					? options.filter((o) => o.value && o.label)
					: undefined,
			validation:
				Object.keys(validation).length > 0 ? validation : undefined,
		};

		await onSave(data);
	};

	return (
		<form
			onSubmit={handleSubmit}
			className="bg-gray-50 border border-gray-200 rounded-lg p-4 space-y-4"
		>
			<div className="grid grid-cols-2 gap-4">
				<div>
					<label htmlFor="schema-field-name" className="block text-sm font-medium text-gray-700">
						Field Name <span className="text-red-500">*</span>
					</label>
					<input
						id="schema-field-name"
						type="text"
						value={name}
						onChange={(e) => handleNameChange(e.target.value)}
						required
						className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
						placeholder="e.g., Department"
					/>
				</div>
				<div>
					<label htmlFor="schema-field-key" className="block text-sm font-medium text-gray-700">
						Field Key <span className="text-red-500">*</span>
					</label>
					<input
						id="schema-field-key"
						type="text"
						value={fieldKey}
						onChange={(e) => setFieldKey(e.target.value)}
						required
						pattern="^[a-z][a-z0-9_-]*$"
						className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm font-mono"
						placeholder="e.g., department"
					/>
					<p className="mt-1 text-xs text-gray-500">
						Lowercase letters, numbers, underscores, and dashes only
					</p>
				</div>
			</div>

			<div className="grid grid-cols-2 gap-4">
				<div>
					<label htmlFor="schema-field-type" className="block text-sm font-medium text-gray-700">
						Field Type <span className="text-red-500">*</span>
					</label>
					<select
						id="schema-field-type"
						value={fieldType}
						onChange={(e) => setFieldType(e.target.value as MetadataFieldType)}
						className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
					>
						{fieldTypes.map((type) => (
							<option key={type.type} value={type.type}>
								{type.label} - {type.description}
							</option>
						))}
					</select>
				</div>
				<div>
					<label htmlFor="schema-description" className="block text-sm font-medium text-gray-700">
						Description
					</label>
					<input
						id="schema-description"
						type="text"
						value={description}
						onChange={(e) => setDescription(e.target.value)}
						className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
						placeholder="Help text for this field"
					/>
				</div>
			</div>

			{fieldType === 'select' && (
				<div>
					<span className="block text-sm font-medium text-gray-700 mb-2">
						Options <span className="text-red-500">*</span>
					</span>
					<div className="space-y-2">
						{options.map((option, index) => (
							<div key={`option-${index}-${option.value || 'empty'}`} className="flex items-center gap-2">
								<input
									type="text"
									value={option.value}
									onChange={(e) =>
										handleOptionChange(index, 'value', e.target.value)
									}
									placeholder="Value"
									aria-label={`Option ${index + 1} value`}
									className="flex-1 rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
								/>
								<input
									type="text"
									value={option.label}
									onChange={(e) =>
										handleOptionChange(index, 'label', e.target.value)
									}
									placeholder="Label"
									aria-label={`Option ${index + 1} label`}
									className="flex-1 rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
								/>
								{options.length > 1 && (
									<button
										type="button"
										onClick={() => handleRemoveOption(index)}
										className="text-red-500 hover:text-red-700"
									>
										Remove
									</button>
								)}
							</div>
						))}
					</div>
					<button
						type="button"
						onClick={handleAddOption}
						className="mt-2 text-sm text-blue-600 hover:text-blue-500"
					>
						+ Add Option
					</button>
				</div>
			)}

			{fieldType === 'text' && (
				<div className="grid grid-cols-2 gap-4">
					<div>
						<label htmlFor="schema-min-length" className="block text-sm font-medium text-gray-700">
							Min Length
						</label>
						<input
							id="schema-min-length"
							type="number"
							min="0"
							value={validation.min_length ?? ''}
							onChange={(e) =>
								setValidation({
									...validation,
									min_length: e.target.value
										? Number(e.target.value)
										: undefined,
								})
							}
							className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
						/>
					</div>
					<div>
						<label htmlFor="schema-max-length" className="block text-sm font-medium text-gray-700">
							Max Length
						</label>
						<input
							id="schema-max-length"
							type="number"
							min="0"
							value={validation.max_length ?? ''}
							onChange={(e) =>
								setValidation({
									...validation,
									max_length: e.target.value
										? Number(e.target.value)
										: undefined,
								})
							}
							className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
						/>
					</div>
				</div>
			)}

			{fieldType === 'number' && (
				<div className="grid grid-cols-2 gap-4">
					<div>
						<label htmlFor="schema-min-value" className="block text-sm font-medium text-gray-700">
							Minimum Value
						</label>
						<input
							id="schema-min-value"
							type="number"
							value={validation.min ?? ''}
							onChange={(e) =>
								setValidation({
									...validation,
									min: e.target.value ? Number(e.target.value) : undefined,
								})
							}
							className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
						/>
					</div>
					<div>
						<label htmlFor="schema-max-value" className="block text-sm font-medium text-gray-700">
							Maximum Value
						</label>
						<input
							id="schema-max-value"
							type="number"
							value={validation.max ?? ''}
							onChange={(e) =>
								setValidation({
									...validation,
									max: e.target.value ? Number(e.target.value) : undefined,
								})
							}
							className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
						/>
					</div>
				</div>
			)}

			<div className="flex items-center">
				<input
					type="checkbox"
					id="required"
					checked={required}
					onChange={(e) => setRequired(e.target.checked)}
					className="rounded border-gray-300 text-blue-600 shadow-sm focus:border-blue-500 focus:ring-blue-500"
				/>
				<label htmlFor="required" className="ml-2 text-sm text-gray-700">
					Required field
				</label>
			</div>

			<div className="flex justify-end gap-2 pt-2 border-t border-gray-200">
				<button
					type="button"
					onClick={onCancel}
					className="px-3 py-1.5 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50"
				>
					Cancel
				</button>
				<button
					type="submit"
					disabled={isSaving}
					className="px-3 py-1.5 text-sm font-medium text-white bg-blue-600 border border-transparent rounded-md hover:bg-blue-700 disabled:opacity-50"
				>
					{isSaving ? 'Saving...' : schema ? 'Update' : 'Create'}
				</button>
			</div>
		</form>
	);
}

export default MetadataSchemaManager;
