import { useState } from 'react';
import { useMe } from '../../hooks/useAuth';
import {
	useCreateSnapshotComment,
	useDeleteSnapshotComment,
	useSnapshotComments,
} from '../../hooks/useSnapshotComments';
import type { SnapshotComment } from '../../lib/types';
import { formatDateTime } from '../../lib/utils';

interface SnapshotCommentsProps {
	snapshotId: string;
}

function CommentItem({
	comment,
	currentUserId,
	onDelete,
	isDeleting,
}: {
	comment: SnapshotComment;
	currentUserId?: string;
	onDelete: (id: string) => void;
	isDeleting: boolean;
}) {
	const canDelete = currentUserId === comment.user_id;

	return (
		<div className="border-b border-gray-100 last:border-0 py-4 first:pt-0 last:pb-0">
			<div className="flex items-start justify-between gap-2">
				<div className="flex items-center gap-2">
					<div className="w-8 h-8 bg-indigo-100 rounded-full flex items-center justify-center">
						<span className="text-sm font-medium text-indigo-600">
							{comment.user_name?.charAt(0).toUpperCase() ||
								comment.user_email?.charAt(0).toUpperCase() ||
								'?'}
						</span>
					</div>
					<div>
						<p className="text-sm font-medium text-gray-900">
							{comment.user_name || comment.user_email || 'Unknown User'}
						</p>
						<p className="text-xs text-gray-500">
							{formatDateTime(comment.created_at)}
						</p>
					</div>
				</div>
				{canDelete && (
					<button
						type="button"
						onClick={() => onDelete(comment.id)}
						disabled={isDeleting}
						className="text-xs text-gray-400 hover:text-red-600 disabled:opacity-50"
					>
						Delete
					</button>
				)}
			</div>
			<div className="mt-2 pl-10">
				<p className="text-sm text-gray-700 whitespace-pre-wrap">
					{comment.content}
				</p>
			</div>
		</div>
	);
}

export function SnapshotComments({ snapshotId }: SnapshotCommentsProps) {
	const [newComment, setNewComment] = useState('');
	const { data: user } = useMe();
	const {
		data: comments,
		isLoading,
		isError,
	} = useSnapshotComments(snapshotId);
	const createComment = useCreateSnapshotComment(snapshotId);
	const deleteComment = useDeleteSnapshotComment(snapshotId);

	const handleSubmit = (e: React.FormEvent) => {
		e.preventDefault();
		if (!newComment.trim()) return;

		createComment.mutate(
			{ content: newComment.trim() },
			{
				onSuccess: () => {
					setNewComment('');
				},
			},
		);
	};

	const handleDelete = (commentId: string) => {
		if (window.confirm('Are you sure you want to delete this comment?')) {
			deleteComment.mutate(commentId);
		}
	};

	return (
		<div className="space-y-4">
			<h4 className="font-medium text-gray-900 flex items-center gap-2">
				<svg
					aria-hidden="true"
					className="w-5 h-5 text-gray-400"
					fill="none"
					stroke="currentColor"
					viewBox="0 0 24 24"
				>
					<path
						strokeLinecap="round"
						strokeLinejoin="round"
						strokeWidth={2}
						d="M7 8h10M7 12h4m1 8l-4-4H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-3l-4 4z"
					/>
				</svg>
				Notes
				{comments && comments.length > 0 && (
					<span className="text-xs bg-gray-100 text-gray-600 px-2 py-0.5 rounded-full">
						{comments.length}
					</span>
				)}
			</h4>

			{isError && (
				<p className="text-sm text-red-600">Failed to load comments</p>
			)}

			{isLoading ? (
				<div className="animate-pulse space-y-3">
					<div className="h-12 bg-gray-100 rounded" />
					<div className="h-12 bg-gray-100 rounded" />
				</div>
			) : (
				<>
					{comments && comments.length > 0 ? (
						<div className="bg-gray-50 rounded-lg p-4">
							{comments.map((comment) => (
								<CommentItem
									key={comment.id}
									comment={comment}
									currentUserId={user?.id}
									onDelete={handleDelete}
									isDeleting={deleteComment.isPending}
								/>
							))}
						</div>
					) : (
						<p className="text-sm text-gray-500 text-center py-4">
							No notes yet. Add a note to document this snapshot.
						</p>
					)}

					<form onSubmit={handleSubmit} className="space-y-2">
						<textarea
							value={newComment}
							onChange={(e) => setNewComment(e.target.value)}
							placeholder="Add a note... (supports markdown)"
							rows={3}
							className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 text-sm resize-none"
						/>
						<div className="flex justify-end">
							<button
								type="submit"
								disabled={!newComment.trim() || createComment.isPending}
								className="px-4 py-2 bg-indigo-600 text-white text-sm rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
							>
								{createComment.isPending ? (
									<>
										<svg
											aria-hidden="true"
											className="animate-spin h-4 w-4"
											fill="none"
											viewBox="0 0 24 24"
										>
											<circle
												className="opacity-25"
												cx="12"
												cy="12"
												r="10"
												stroke="currentColor"
												strokeWidth="4"
											/>
											<path
												className="opacity-75"
												fill="currentColor"
												d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
											/>
										</svg>
										Saving...
									</>
								) : (
									'Add Note'
								)}
							</button>
						</div>
					</form>
				</>
			)}
		</div>
	);
}
