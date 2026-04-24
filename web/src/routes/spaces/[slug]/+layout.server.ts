import { error } from '@sveltejs/kit';
import type { LayoutServerLoad } from './$types';
import type { Space } from '$lib/api/space';

const API_BASE_URL = process.env.VITE_API_URL || 'http://localhost:8081';

export const load: LayoutServerLoad = async ({ params, request }) => {
	const cookie = request.headers.get('cookie');

	const res = await fetch(`${API_BASE_URL}/api/spaces/by-slug/${params.slug}`, {
		headers: cookie ? { cookie } : {},
	});

	if (!res.ok) {
		throw error(res.status === 404 ? 404 : 500, 'Space not found');
	}

	const space: Space = await res.json();

	// Record the visit (fire-and-forget)
	fetch(`${API_BASE_URL}/api/spaces/${space.id}/visit`, {
		method: 'POST',
		headers: cookie ? { cookie } : {},
	}).catch(() => {});

	return { space };
};
