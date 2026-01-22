import { useState } from 'react';
import {
	useCreateTag,
	useDeleteTag,
	useTags,
	useUpdateTag,
} from '../hooks/useTags';
import type { Tag } from '../lib/types';
import { formatDate } from '../lib/utils';

const DEFAULT_COLORS = [
	'#6366f1', // indigo
	'#8b5cf6', // violet
	'#ec4899', // pink
	'#ef4444', // red
	'#f97316', // orange
	'#eab308', // yellow
	'#22c55e', // green
	'#14b8a6', // teal
	'#0ea5e9', // sky
	'#3b82f6', // blue
];

function LoadingRow() {
	return (
		<tr className="animate-pulse">
			<td className="px-6 py-4">
				<div className="flex items-center gap-2">
					<div className="w-4 h-4 rounded-full bg-gray-200" />
					<div className="h-4 w-24 bg-gray-200 rounded" />
				</div>
			</td>
			<td className="px-6 py-4">
				<div className="h-4 w-16 bg-gray-200 rounded" />
			</td>
			<td className="px-6 py-4">
				<div className="h-4 w-32 bg-gray-200 rounded" />
			</td>
			<td className="px-6 py-4 text-right">
				<div className="h-8 w-20 bg-gray-200 rounded inline-block" />
			</td>
		</tr>
	);
}

interface CreateTagModalProps {
	isOpen: boolean;
	onClose: () => void;
}

function CreateTagModal({ isOpen, onClose }: CreateTagModalProps) {
	const [name, setName] = useState('');
	const [color, setColor] = useState(DEFAULT_COLORS[0]);
	const createTag = useCreateTag();

	const handleSubmit = async (e: React.FormEvent) => {
		e.preventDefault();
		try {
			await createTag.mutateAsync({ name, color });
			setName('');
			setColor(DEFAULT_COLORS[0]);
			onClose();
		} catch {
			// Error handled by mutation
		}
	};

	if (!isOpen) return null;

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white rounded-lg p-6 max-w-md w-full mx-4">
				<h3 className="text-lg font-semibold text-gray-900 mb-4">
					Create New Tag
				</h3>
				<form onSubmit={handleSubmit}>
					<div className="mb-4">
						<label
							htmlFor="name"
							className="block text-sm font-medium text-gray-700 mb-1"
						>
							Name
						</label>
						<input
							type="text"
							id="name"
							value={name}
							onChange={(e) => setName(e.target.value)}
							placeholder="e.g., production, daily-backup"
							className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
							required
							maxLength={100}
						/>
					</div>
					<div className="mb-4">
						<span className="block text-sm font-medium text-gray-700 mb-2">
							Color
						</span>
						<div className="flex flex-wrap gap-2">
							{DEFAULT_COLORS.map((c) => (
								<button
									key={c}
									type="button"
									onClick={() => setColor(c)}
									className={`w-8 h-8 rounded-full border-2 transition-all ${
										color === c
											? 'border-gray-900 scale-110'
											: 'border-transparent hover:border-gray-300'
									}`}
									style={{ backgroundColor: c }}
								/>
							))}
						</div>
					</div>
					{createTag.isError && (
						<p className="text-sm text-red-600 mb-4">
							Failed to create tag. Please try again.
						</p>
					)}
					<div className="flex justify-end gap-3">
						<button
							type="button"
							onClick={onClose}
							className="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
						>
							Cancel
						</button>
						<button
							type="submit"
							disabled={createTag.isPending}
							className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
						>
							{createTag.isPending ? 'Creating...' : 'Create'}
						</button>
					</div>
				</form>
			</div>
		</div>
	);
}

interface EditTagModalProps {
	tag: Tag;
	onClose: () => void;
}

function EditTagModal({ tag, onClose }: EditTagModalProps) {
	const [name, setName] = useState(tag.name);
	const [color, setColor] = useState(tag.color);
	const updateTag = useUpdateTag();

	const handleSubmit = async (e: React.FormEvent) => {
		e.preventDefault();
		try {
			await updateTag.mutateAsync({ id: tag.id, data: { name, color } });
			onClose();
		} catch {
			// Error handled by mutation
		}
	};

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white rounded-lg p-6 max-w-md w-full mx-4">
				<h3 className="text-lg font-semibold text-gray-900 mb-4">Edit Tag</h3>
				<form onSubmit={handleSubmit}>
					<div className="mb-4">
						<label
							htmlFor="edit-name"
							className="block text-sm font-medium text-gray-700 mb-1"
						>
							Name
						</label>
						<input
							type="text"
							id="edit-name"
							value={name}
							onChange={(e) => setName(e.target.value)}
							className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
							required
							maxLength={100}
						/>
					</div>
					<div className="mb-4">
						<span className="block text-sm font-medium text-gray-700 mb-2">
							Color
						</span>
						<div className="flex flex-wrap gap-2">
							{DEFAULT_COLORS.map((c) => (
								<button
									key={c}
									type="button"
									onClick={() => setColor(c)}
									className={`w-8 h-8 rounded-full border-2 transition-all ${
										color === c
											? 'border-gray-900 scale-110'
											: 'border-transparent hover:border-gray-300'
									}`}
									style={{ backgroundColor: c }}
								/>
							))}
						</div>
					</div>
					{updateTag.isError && (
						<p className="text-sm text-red-600 mb-4">
							Failed to update tag. Please try again.
						</p>
					)}
					<div className="flex justify-end gap-3">
						<button
							type="button"
							onClick={onClose}
							className="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
						>
							Cancel
						</button>
						<button
							type="submit"
							disabled={updateTag.isPending}
							className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
						>
							{updateTag.isPending ? 'Saving...' : 'Save'}
						</button>
					</div>
				</form>
			</div>
		</div>
	);
}

interface DeleteTagModalProps {
	tag: Tag;
	onClose: () => void;
}

function DeleteTagModal({ tag, onClose }: DeleteTagModalProps) {
	const deleteTag = useDeleteTag();

	const handleDelete = async () => {
		try {
			await deleteTag.mutateAsync(tag.id);
			onClose();
		} catch {
			// Error handled by mutation
		}
	};

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white rounded-lg p-6 max-w-md w-full mx-4">
				<h3 className="text-lg font-semibold text-gray-900 mb-4">Delete Tag</h3>
				<p className="text-gray-600 mb-4">
					Are you sure you want to delete the tag{' '}
					<span
						className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium text-white"
						style={{ backgroundColor: tag.color }}
					>
						{tag.name}
					</span>
					? This will remove it from all associated backups.
				</p>
				{deleteTag.isError && (
					<p className="text-sm text-red-600 mb-4">
						Failed to delete tag. Please try again.
					</p>
				)}
				<div className="flex justify-end gap-3">
					<button
						type="button"
						onClick={onClose}
						className="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
					>
						Cancel
					</button>
					<button
						type="button"
						onClick={handleDelete}
						disabled={deleteTag.isPending}
						className="px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 transition-colors disabled:opacity-50"
					>
						{deleteTag.isPending ? 'Deleting...' : 'Delete'}
					</button>
				</div>
			</div>
		</div>
	);
}

interface TagRowProps {
	tag: Tag;
	onEdit: (tag: Tag) => void;
	onDelete: (tag: Tag) => void;
}

function TagRow({ tag, onEdit, onDelete }: TagRowProps) {
	return (
		<tr className="hover:bg-gray-50">
			<td className="px-6 py-4">
				<div className="flex items-center gap-2">
					<span
						className="w-4 h-4 rounded-full"
						style={{ backgroundColor: tag.color }}
					/>
					<span className="text-sm font-medium text-gray-900">{tag.name}</span>
				</div>
			</td>
			<td className="px-6 py-4">
				<code className="text-sm text-gray-500">{tag.color}</code>
			</td>
			<td className="px-6 py-4 text-sm text-gray-500">
				{formatDate(tag.created_at)}
			</td>
			<td className="px-6 py-4 text-right">
				<div className="flex items-center justify-end gap-2">
					<button
						type="button"
						onClick={() => onEdit(tag)}
						className="text-indigo-600 hover:text-indigo-800 text-sm font-medium"
					>
						Edit
					</button>
					<button
						type="button"
						onClick={() => onDelete(tag)}
						className="text-red-600 hover:text-red-800 text-sm font-medium"
					>
						Delete
					</button>
				</div>
			</td>
		</tr>
	);
}

export function Tags() {
	const [showCreateModal, setShowCreateModal] = useState(false);
	const [editingTag, setEditingTag] = useState<Tag | null>(null);
	const [deletingTag, setDeletingTag] = useState<Tag | null>(null);

	const { data: tags, isLoading, isError } = useTags();

	return (
		<div className="space-y-6">
			<div className="flex items-center justify-between">
				<div>
					<h1 className="text-2xl font-bold text-gray-900">Tags</h1>
					<p className="text-gray-600 mt-1">
						Organize and categorize your backups with tags
					</p>
				</div>
				<button
					type="button"
					onClick={() => setShowCreateModal(true)}
					className="flex items-center gap-2 px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors"
				>
					<svg
						aria-hidden="true"
						className="w-5 h-5"
						fill="none"
						stroke="currentColor"
						viewBox="0 0 24 24"
					>
						<path
							strokeLinecap="round"
							strokeLinejoin="round"
							strokeWidth={2}
							d="M12 4v16m8-8H4"
						/>
					</svg>
					Create Tag
				</button>
			</div>

			<div className="bg-white rounded-lg border border-gray-200">
				<div className="overflow-x-auto">
					{isError ? (
						<div className="p-12 text-center text-red-500">
							<p className="font-medium">Failed to load tags</p>
							<p className="text-sm">Please try refreshing the page</p>
						</div>
					) : isLoading ? (
						<table className="w-full">
							<thead className="bg-gray-50 border-b border-gray-200">
								<tr>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Name
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Color
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Created
									</th>
									<th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
										Actions
									</th>
								</tr>
							</thead>
							<tbody className="divide-y divide-gray-200">
								<LoadingRow />
								<LoadingRow />
								<LoadingRow />
							</tbody>
						</table>
					) : tags && tags.length > 0 ? (
						<table className="w-full">
							<thead className="bg-gray-50 border-b border-gray-200">
								<tr>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Name
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Color
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
										Created
									</th>
									<th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
										Actions
									</th>
								</tr>
							</thead>
							<tbody className="divide-y divide-gray-200">
								{tags.map((tag) => (
									<TagRow
										key={tag.id}
										tag={tag}
										onEdit={setEditingTag}
										onDelete={setDeletingTag}
									/>
								))}
							</tbody>
						</table>
					) : (
						<div className="p-12 text-center">
							<svg
								aria-hidden="true"
								className="w-12 h-12 mx-auto mb-3 text-gray-300"
								fill="none"
								stroke="currentColor"
								viewBox="0 0 24 24"
							>
								<path
									strokeLinecap="round"
									strokeLinejoin="round"
									strokeWidth={2}
									d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A1.994 1.994 0 013 12V7a4 4 0 014-4z"
								/>
							</svg>
							<p className="font-medium text-gray-900">No tags yet</p>
							<p className="text-sm text-gray-500">
								Create your first tag to organize backups
							</p>
						</div>
					)}
				</div>
			</div>

			<CreateTagModal
				isOpen={showCreateModal}
				onClose={() => setShowCreateModal(false)}
			/>

			{editingTag && (
				<EditTagModal tag={editingTag} onClose={() => setEditingTag(null)} />
			)}

			{deletingTag && (
				<DeleteTagModal
					tag={deletingTag}
					onClose={() => setDeletingTag(null)}
				/>
			)}
		</div>
	);
}
