import { useMutation, useQueryClient } from '@tanstack/react-query';
import { agentImportApi } from '../lib/api';
import type { AgentImportJobResult } from '../lib/types';

export interface AgentImportOptions {
	hasHeader?: boolean;
	hostnameCol?: number;
	groupCol?: number;
	tagsCol?: number;
	configCol?: number;
	createMissingGroups?: boolean;
	tokenExpiryHours?: number;
}

export function useAgentImportPreview() {
	return useMutation({
		mutationFn: ({
			file,
			options,
		}: {
			file: File;
			options?: AgentImportOptions;
		}) => agentImportApi.preview(file, options),
	});
}

export function useAgentImport() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({
			file,
			options,
		}: {
			file: File;
			options?: AgentImportOptions;
		}) => agentImportApi.import(file, options),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['agents'] });
			queryClient.invalidateQueries({ queryKey: ['agent-groups'] });
			queryClient.invalidateQueries({ queryKey: ['registration-codes'] });
		},
	});
}

export function useAgentImportTemplate() {
	return useMutation({
		mutationFn: (format?: 'json' | 'csv') => agentImportApi.getTemplate(format),
	});
}

export function useAgentImportTemplateDownload() {
	return useMutation({
		mutationFn: async () => {
			const blob = await agentImportApi.downloadTemplate();
			const url = window.URL.createObjectURL(blob);
			const a = document.createElement('a');
			a.href = url;
			a.download = 'agent_import_template.csv';
			document.body.appendChild(a);
			a.click();
			window.URL.revokeObjectURL(url);
			document.body.removeChild(a);
		},
	});
}

export function useAgentRegistrationScript() {
	return useMutation({
		mutationFn: ({
			hostname,
			registrationCode,
		}: {
			hostname: string;
			registrationCode: string;
		}) =>
			agentImportApi.generateScript({
				hostname,
				registration_code: registrationCode,
			}),
	});
}

export function useAgentImportTokensExport() {
	return useMutation({
		mutationFn: async (results: AgentImportJobResult[]) => {
			const blob = await agentImportApi.exportTokens(results);
			const url = window.URL.createObjectURL(blob);
			const a = document.createElement('a');
			a.href = url;
			a.download = 'agent_registration_tokens.csv';
			document.body.appendChild(a);
			a.click();
			window.URL.revokeObjectURL(url);
			document.body.removeChild(a);
		},
	});
}
