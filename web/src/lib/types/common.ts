/**
 * Generic paginated response wrapper matching the backend pagination pattern.
 */
export interface PaginatedResponse<T> {
	items: T[];
	total: number;
	offset: number;
	limit: number;
}
