import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useCallback, useEffect, useRef, useState } from 'react';
import { activityApi } from '../lib/api';
import type { ActivityEvent, ActivityEventFilter } from '../lib/types';

// Activity event hooks

export function useActivity(filter?: ActivityEventFilter) {
	return useQuery({
		queryKey: ['activity', filter],
		queryFn: () => activityApi.list(filter),
		staleTime: 30 * 1000, // 30 seconds
	});
}

export function useRecentActivity(limit?: number) {
	return useQuery({
		queryKey: ['activity', 'recent', limit],
		queryFn: () => activityApi.recent(limit),
		staleTime: 10 * 1000, // 10 seconds - refresh more frequently for dashboard
		refetchInterval: 30 * 1000, // Refetch every 30 seconds
	});
}

export function useActivityCount(category?: string, type?: string) {
	return useQuery({
		queryKey: ['activity', 'count', category, type],
		queryFn: () => activityApi.count(category, type),
		staleTime: 30 * 1000,
	});
}

export function useActivityCategories() {
	return useQuery({
		queryKey: ['activity', 'categories'],
		queryFn: activityApi.categories,
		staleTime: 60 * 1000, // 1 minute
	});
}

export function useActivitySearch(query: string, limit?: number) {
	return useQuery({
		queryKey: ['activity', 'search', query, limit],
		queryFn: () => activityApi.search(query, limit),
		enabled: query.length > 0,
		staleTime: 30 * 1000,
	});
}

// WebSocket hook for real-time activity feed

interface UseActivityFeedOptions {
	enabled?: boolean;
	categories?: string[];
	types?: string[];
	agentIds?: string[];
	userIds?: string[];
	onEvent?: (event: ActivityEvent) => void;
	maxEvents?: number;
}

export function useActivityFeed(options: UseActivityFeedOptions = {}) {
	const {
		enabled = true,
		categories,
		types,
		agentIds,
		userIds,
		onEvent,
		maxEvents = 100,
	} = options;

	const queryClient = useQueryClient();
	const wsRef = useRef<WebSocket | null>(null);
	const reconnectTimeoutRef = useRef<number>();
	const [events, setEvents] = useState<ActivityEvent[]>([]);
	const [isConnected, setIsConnected] = useState(false);
	const [error, setError] = useState<string | null>(null);

	const connect = useCallback(() => {
		if (!enabled) return;

		// Construct WebSocket URL
		const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
		const wsUrl = `${protocol}//${window.location.host}/ws/activity`;

		try {
			const ws = new WebSocket(wsUrl);

			ws.onopen = () => {
				setIsConnected(true);
				setError(null);

				// Send filter update if filters are set
				if (categories || types || agentIds || userIds) {
					ws.send(
						JSON.stringify({
							type: 'filter',
							filter: {
								categories,
								types,
								agent_ids: agentIds,
								user_ids: userIds,
							},
						}),
					);
				}
			};

			ws.onmessage = (event) => {
				try {
					const activityEvent: ActivityEvent = JSON.parse(event.data);

					// Add to local state
					setEvents((prev) => {
						const newEvents = [activityEvent, ...prev];
						return newEvents.slice(0, maxEvents);
					});

					// Call event handler if provided
					onEvent?.(activityEvent);

					// Invalidate relevant queries to keep them fresh
					queryClient.invalidateQueries({ queryKey: ['activity', 'recent'] });
					queryClient.invalidateQueries({ queryKey: ['activity', 'count'] });
				} catch (e) {
					console.error('Failed to parse activity event:', e);
				}
			};

			ws.onclose = (event) => {
				setIsConnected(false);

				// Attempt to reconnect after a delay (with exponential backoff)
				if (event.code !== 1000 && enabled) {
					reconnectTimeoutRef.current = window.setTimeout(() => {
						connect();
					}, 5000);
				}
			};

			ws.onerror = () => {
				setError('WebSocket connection error');
			};

			wsRef.current = ws;
		} catch (e) {
			setError('Failed to connect to activity feed');
			console.error('WebSocket connection error:', e);
		}
	}, [
		enabled,
		categories,
		types,
		agentIds,
		userIds,
		onEvent,
		maxEvents,
		queryClient,
	]);

	// Update filter when options change
	useEffect(() => {
		if (wsRef.current?.readyState === WebSocket.OPEN) {
			wsRef.current.send(
				JSON.stringify({
					type: 'filter',
					filter: {
						categories,
						types,
						agent_ids: agentIds,
						user_ids: userIds,
					},
				}),
			);
		}
	}, [categories, types, agentIds, userIds]);

	// Connect on mount, disconnect on unmount
	useEffect(() => {
		connect();

		return () => {
			if (reconnectTimeoutRef.current) {
				clearTimeout(reconnectTimeoutRef.current);
			}
			if (wsRef.current) {
				wsRef.current.close(1000, 'Component unmounted');
				wsRef.current = null;
			}
		};
	}, [connect]);

	const clearEvents = useCallback(() => {
		setEvents([]);
	}, []);

	return {
		events,
		isConnected,
		error,
		clearEvents,
	};
}
