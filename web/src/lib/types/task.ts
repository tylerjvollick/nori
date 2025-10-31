import type { SOPTemplate } from '$lib/api/sop';

export type TaskStatus = 'todo' | 'inProgress' | 'done';

export interface Task {
  id: string;
  sopId: number;
  sop: SOPTemplate;
  status: TaskStatus;
  assignedTo?: string;
  createdAt: number;
  updatedAt: number;
  // Track progress through steps
  currentStepIndex: number;
  completedSteps: number[];
  stepTimers: Record<number, { startTime?: number; timeSpent: number }>;
  // Materials checklist
  materialsChecked: string[];
  equipmentChecked: string[];
}

export type KanbanColumn = 'todo' | 'inProgress' | 'done';
