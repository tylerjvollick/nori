export interface Material {
  name: string;
  quantity: string;
  location: string;
  pullTrigger?: string;
  lowStock?: boolean;
  prepSopId?: string; // SOP ID for making this material
  prepTaskTitle?: string; // Custom title for the prep task
  verified?: boolean; // For tracking if material is verified/checked
}

export interface WorkElement {
  id: string;
  description: string;
  taktTime: number; // expected duration in seconds
  completed: boolean;
  inProgress: boolean;
  timeSpent?: number;
  startTime?: number;
  station?: string; // which person/station performs this
  details?: {
    images?: string[];
    videos?: string[];
    links?: string[];
  };
}

export interface ProcedureStep {
  id: string;
  description: string;
  completed: boolean;
  inProgress: boolean;
  timeSpent?: number;
  startTime?: number;
  details?: {
    images?: string[];
    videos?: string[];
    links?: string[];
  };
}

export interface SOP {
  id: string;
  name: string;
  materials: Material[];
  procedure: ProcedureStep[];
  estimatedTime?: number;
  difficulty?: 'easy' | 'medium' | 'hard' | 'master';
}

export interface StandardizedWork {
  id: string;
  name: string;
  description?: string;
  workElements: WorkElement[]; // smallest meaningful units of work
  resources: Material[]; // tools, materials, equipment
  totalTaktTime: number; // expected total duration
  pullSignal?: string; // conditions that trigger this work
  kaizen: KaizenSuggestion[]; // improvement feedback
  createdAt: number;
  updatedAt: number;
  category?: string;
}

export interface KaizenSuggestion {
  id: string;
  author: string;
  suggestion: string;
  createdAt: number;
  status: 'pending' | 'approved' | 'rejected' | 'implemented';
}

export interface Task {
  id: string;
  title: string;
  status: 'todo' | 'inProgress' | 'done';
  sopId: string;
  standardizedWorkId?: string; // Links to StandardizedWork if created from one
  syncToMaster: boolean; // If true, changes sync back to StandardizedWork
  assignedTo?: string;
  tags: string[];
  createdAt: number;
  tableNumber?: string;
  orderItems?: string[];
  forkedFrom?: string; // Original StandardizedWork ID if this is a fork
}

export type KanbanColumn = 'todo' | 'inProgress' | 'done';