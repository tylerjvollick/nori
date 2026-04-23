import { apiClient } from './client';

export interface CostBreakdown {
	laborHours: string;
	laborCost: string;
	total: string;
}

export interface CostVariance {
	laborHours: string;
	laborCost: string;
	total: string;
	percent: string;
}

export interface StationCostSummary {
	stationId: string;
	stationName: string;
	estimatedHours: string;
	actualHours: string;
	variance: string;
}

export interface CostSummary {
	jobId: string;
	estimated: CostBreakdown;
	actual: CostBreakdown;
	variance: CostVariance;
	byStation: StationCostSummary[];
}

export const costApi = {
	getJobCostSummary(jobId: string): Promise<CostSummary> {
		return apiClient.get<CostSummary>(`/api/v1/jobs/${jobId}/cost-summary`);
	},
};
