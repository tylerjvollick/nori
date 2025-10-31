import { writable } from 'svelte/store';
import type { Task, TaskStatus } from '$lib/types/task';
import type { SOPTemplate } from '$lib/api/sop';

interface TaskStore {
  tasks: Task[];
}

const STORAGE_KEY = 'nori_tasks';

function loadTasksFromStorage(): Task[] {
  if (typeof window === 'undefined') return [];
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    return stored ? JSON.parse(stored) : [];
  } catch (error) {
    console.error('Failed to load tasks from localStorage:', error);
    return [];
  }
}

function saveTasksToStorage(tasks: Task[]) {
  if (typeof window === 'undefined') return;
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(tasks));
  } catch (error) {
    console.error('Failed to save tasks to localStorage:', error);
  }
}

function createTaskStore() {
  const { subscribe, set, update } = writable<TaskStore>({
    tasks: loadTasksFromStorage()
  });

  return {
    subscribe,

    createTask(sop: SOPTemplate, assignedTo?: string): Task {
      const newTask: Task = {
        id: `task-${Date.now()}-${Math.random().toString(36).substring(2, 11)}`,
        sopId: sop.id,
        sop,
        status: 'todo',
        assignedTo,
        createdAt: Date.now(),
        updatedAt: Date.now(),
        currentStepIndex: 0,
        completedSteps: [],
        stepTimers: {},
        materialsChecked: [],
        equipmentChecked: []
      };

      update(state => {
        const tasks = [...state.tasks, newTask];
        saveTasksToStorage(tasks);
        return { tasks };
      });

      return newTask;
    },

    updateTaskStatus(taskId: string, newStatus: TaskStatus) {
      update(state => {
        const tasks = state.tasks.map(task => 
          task.id === taskId 
            ? { ...task, status: newStatus, updatedAt: Date.now() }
            : task
        );
        saveTasksToStorage(tasks);
        return { tasks };
      });
    },

    updateTask(taskId: string, updates: Partial<Task>) {
      update(state => {
        const tasks = state.tasks.map(task =>
          task.id === taskId
            ? { ...task, ...updates, updatedAt: Date.now() }
            : task
        );
        saveTasksToStorage(tasks);
        return { tasks };
      });
    },

    deleteTask(taskId: string) {
      update(state => {
        const tasks = state.tasks.filter(task => task.id !== taskId);
        saveTasksToStorage(tasks);
        return { tasks };
      });
    },

    completeStep(taskId: string, stepIndex: number) {
      update(state => {
        const tasks = state.tasks.map(task => {
          if (task.id === taskId) {
            const completedSteps = task.completedSteps.includes(stepIndex)
              ? task.completedSteps
              : [...task.completedSteps, stepIndex];
            
            return {
              ...task,
              completedSteps,
              currentStepIndex: stepIndex + 1,
              updatedAt: Date.now()
            };
          }
          return task;
        });
        saveTasksToStorage(tasks);
        return { tasks };
      });
    },

    uncompleteStep(taskId: string, stepIndex: number) {
      update(state => {
        const tasks = state.tasks.map(task => {
          if (task.id === taskId) {
            return {
              ...task,
              completedSteps: task.completedSteps.filter(i => i !== stepIndex),
              updatedAt: Date.now()
            };
          }
          return task;
        });
        saveTasksToStorage(tasks);
        return { tasks };
      });
    },

    startStepTimer(taskId: string, stepIndex: number) {
      update(state => {
        const tasks = state.tasks.map(task => {
          if (task.id === taskId) {
            return {
              ...task,
              stepTimers: {
                ...task.stepTimers,
                [stepIndex]: {
                  ...task.stepTimers[stepIndex],
                  startTime: Date.now(),
                  timeSpent: task.stepTimers[stepIndex]?.timeSpent || 0
                }
              },
              updatedAt: Date.now()
            };
          }
          return task;
        });
        saveTasksToStorage(tasks);
        return { tasks };
      });
    },

    stopStepTimer(taskId: string, stepIndex: number) {
      update(state => {
        const tasks = state.tasks.map(task => {
          if (task.id === taskId) {
            const timer = task.stepTimers[stepIndex];
            if (timer?.startTime) {
              const timeSpent = timer.timeSpent + (Date.now() - timer.startTime);
              return {
                ...task,
                stepTimers: {
                  ...task.stepTimers,
                  [stepIndex]: {
                    timeSpent,
                    startTime: undefined
                  }
                },
                updatedAt: Date.now()
              };
            }
          }
          return task;
        });
        saveTasksToStorage(tasks);
        return { tasks };
      });
    },

    toggleMaterial(taskId: string, material: string) {
      update(state => {
        const tasks = state.tasks.map(task => {
          if (task.id === taskId) {
            const materialsChecked = task.materialsChecked.includes(material)
              ? task.materialsChecked.filter(m => m !== material)
              : [...task.materialsChecked, material];
            
            return { ...task, materialsChecked, updatedAt: Date.now() };
          }
          return task;
        });
        saveTasksToStorage(tasks);
        return { tasks };
      });
    },

    toggleEquipment(taskId: string, equipment: string) {
      update(state => {
        const tasks = state.tasks.map(task => {
          if (task.id === taskId) {
            const equipmentChecked = task.equipmentChecked.includes(equipment)
              ? task.equipmentChecked.filter(e => e !== equipment)
              : [...task.equipmentChecked, equipment];
            
            return { ...task, equipmentChecked, updatedAt: Date.now() };
          }
          return task;
        });
        saveTasksToStorage(tasks);
        return { tasks };
      });
    },

    clearCompleted() {
      update(state => {
        const tasks = state.tasks.filter(task => task.status !== 'done');
        saveTasksToStorage(tasks);
        return { tasks };
      });
    },

    reset() {
      set({ tasks: [] });
      if (typeof window !== 'undefined') {
        localStorage.removeItem(STORAGE_KEY);
      }
    }
  };
}

export const taskStore = createTaskStore();
