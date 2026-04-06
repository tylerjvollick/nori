/** Matches backend dtos.StationResponse exactly. */
export interface StationResponse {
	id: string;
	name: string;
	description?: string | null;
	displayOrder: number;
	wipLimit: number;
	wipCount: number;
	bufferSize: number;
	isActive: boolean;
}

export interface CreateStationRequest {
	name: string;
	description?: string;
	wipLimit?: number;
	bufferSize?: number;
}

export interface UpdateStationRequest {
	name?: string;
	description?: string;
	wipLimit?: number;
	bufferSize?: number;
	isActive?: boolean;
}
