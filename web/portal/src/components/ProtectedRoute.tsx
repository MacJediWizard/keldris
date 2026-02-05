import { Navigate } from 'react-router-dom';
import { useMe } from '../hooks/useAuth';

interface ProtectedRouteProps {
	children: React.ReactNode;
}

export function ProtectedRoute({ children }: ProtectedRouteProps) {
	const { data: customer, isLoading, isError } = useMe();

	if (isLoading) {
		return (
			<div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-dark-bg">
				<div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" />
			</div>
		);
	}

	if (isError || !customer) {
		return <Navigate to="/login" replace />;
	}

	return <>{children}</>;
}
