import { redirect, type Handle } from '@sveltejs/kit';
import type { User } from '$lib/api/auth';

const API_BASE_URL = process.env.VITE_API_URL || 'http://localhost:8080';

/** Routes that do not require authentication. */
const PUBLIC_ROUTES = ['/login'];

/** Routes accessible even when mustChangePassword is true. */
const PASSWORD_EXEMPT_ROUTES = ['/login', '/change-password'];

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

	event.locals.user = user;
	return resolve(event);
};
