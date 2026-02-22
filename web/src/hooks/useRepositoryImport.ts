import { useMutation, useQueryClient } from '@tanstack/react-query';
import { repositoryImportApi } from '../lib/api';
import type {
	ImportPreviewRequest,
	ImportRepositoryRequest,
	VerifyImportAccessRequest,
} from '../lib/types';

export function useVerifyImportAccess() {
	return useMutation({
		mutationFn: (data: VerifyImportAccessRequest) =>
			repositoryImportApi.verifyAccess(data),
	});
}

export function useImportPreview() {
	return useMutation({
		mutationFn: (data: ImportPreviewRequest) =>
			repositoryImportApi.preview(data),
	});
}

export function useImportRepository() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: ImportRepositoryRequest) =>
			repositoryImportApi.import(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['repositories'] });
		},
	});
}
