import { redirect, type Handle } from '@sveltejs/kit';
import type { User } from '$lib/api/auth';

// Server-side fetches must reach the API container by service name, not the
// host-facing URL the browser uses. INTERNAL_API_URL is set in docker-compose
// for containerized dev; fall back to VITE_API_URL for non-docker setups.
const API_BASE_URL =
	process.env.INTERNAL_API_URL || process.env.VITE_API_URL || 'http://localhost:8081';

/** Routes that do not require authentication. */
const PUBLIC_ROUTES = ['/login'];

/** Routes accessible even when mustChangePassword is true. */
const PASSWORD_EXEMPT_ROUTES = ['/login', '/change-password'];

/** Routes accessible even when the user has no spaces (onboarding flow). */
const NO_SPACES_EXEMPT_ROUTES = ['/login', '/change-password', '/onboarding'];

export const handle: Handle = async ({ event, resolve }) => {
	const { pathname } = event.url;

	// Allow public routes through without auth check
	if (PUBLIC_ROUTES.some((route) => pathname === route)) {
		event.locals.user = null;
		return resolve(event);
	}

	// Forward the nori_token cookie to the backend to validate the session
	const cookie = event.request.headers.get('cookie');
	let user: User | null = null;

	try {
		const response = await fetch(`${API_BASE_URL}/auth/me`, {
			headers: cookie ? { cookie } : {},
		});

		if (response.ok) {
			user = await response.json();
		}
	} catch {
		// Backend unreachable — treat as unauthenticated
	}

	// Unauthenticated → redirect to login
	if (!user) {
		const loginUrl = `/login?redirectTo=${encodeURIComponent(pathname)}`;
		throw redirect(302, loginUrl);
	}

	// Must change password → redirect to /change-password
	if (user.mustChangePassword && !PASSWORD_EXEMPT_ROUTES.some((route) => pathname === route)) {
		throw redirect(302, '/change-password');
	}

	// No accessible spaces → redirect to onboarding
	if (
		(!user.accessibleSpaces || user.accessibleSpaces.length === 0) &&
		!NO_SPACES_EXEMPT_ROUTES.some((route) => pathname === route)
	) {
		throw redirect(302, '/onboarding');
	}

	// Admin routes → require admin role (redirect non-admins to home)
	if (pathname.startsWith('/admin') && user.role !== 'admin') {
		throw redirect(302, '/');
	}

	event.locals.user = user;
	return resolve(event);
};
