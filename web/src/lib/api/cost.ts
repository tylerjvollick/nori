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

/** Rolled-up recorded time for a job (own + all descendant tasks). */
export interface JobTimeSummary {
	jobId: string;
	/** Seconds logged directly on the job task itself. */
	ownSecs: number;
	/** ownSecs plus the time logged on every descendant task. */
	rollupSecs: number;
}

export const costApi = {
	getJobCostSummary(spaceId: string, jobId: string): Promise<CostSummary> {
		return apiClient.get<CostSummary>(`/api/v1/spaces/${spaceId}/jobs/${jobId}/cost-summary`);
	},
	getJobTimeSummary(spaceId: string, jobId: string): Promise<JobTimeSummary> {
		return apiClient.get<JobTimeSummary>(`/api/v1/spaces/${spaceId}/jobs/${jobId}/time-summary`);
	},
};
