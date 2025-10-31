import { useState } from 'react';
import { DndProvider } from 'react-dnd';
import { HTML5Backend } from 'react-dnd-html5-backend';
import { KanbanColumn } from './components/KanbanColumn';
import { TaskDetailDialog } from './components/TaskDetailDialog';
import { CreateStandardizedWorkDialog } from './components/CreateStandardizedWorkDialog';
import { CreateTaskDialog } from './components/CreateTaskDialog';
import { Button } from './components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from './components/ui/dropdown-menu';
import { Toaster } from './components/ui/sonner';
import { Task, KanbanColumn as ColumnType, StandardizedWork } from './types';
import { mockTasks, mockSOPs, mockStandardizedWorks } from './data/mockData';
import { Plus, Waves, FileText, ListChecks } from 'lucide-react';
import { toast } from 'sonner@2.0.3';

export default function App() {
  const [tasks, setTasks] = useState<Task[]>(mockTasks);
  const [standardizedWorks, setStandardizedWorks] = useState<Record<string, StandardizedWork>>(
    mockStandardizedWorks
  );
  const [selectedTask, setSelectedTask] = useState<Task | null>(null);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [createTaskDialogOpen, setCreateTaskDialogOpen] = useState(false);
  const [createStandardizedWorkDialogOpen, setCreateStandardizedWorkDialogOpen] = useState(false);

  const handleTaskClick = (task: Task) => {
    setSelectedTask(task);
    setDialogOpen(true);
  };

  const handleTaskMove = (taskId: string, newStatus: ColumnType) => {
    setTasks((prevTasks) =>
      prevTasks.map((task) =>
        task.id === taskId ? { ...task, status: newStatus } : task
      )
    );
  };

  const handleUpdateTask = (taskId: string, updates: Partial<Task>) => {
    setTasks((prevTasks) =>
      prevTasks.map((task) => (task.id === taskId ? { ...task, ...updates } : task))
    );
  };

  const handleCreatePrepTask = (sopId: string, title: string) => {
    const newTask: Task = {
      id: `task-${Date.now()}`,
      title,
      status: 'todo',
      sopId,
      syncToMaster: false,
      tags: ['prep'],
      createdAt: Date.now(),
    };
    setTasks((prevTasks) => [...prevTasks, newTask]);
    toast.success(`Prep task "${title}" added to To Do`);
  };

  const handleCreateStandardizedWork = (work: StandardizedWork) => {
    setStandardizedWorks((prev) => ({
      ...prev,
      [work.id]: work,
    }));
    toast.success(`Standardized work "${work.name}" created`);
  };

  const handleCreateTask = (task: Task) => {
    setTasks((prevTasks) => [...prevTasks, task]);
    toast.success(`Task "${task.title}" created from standardized work`);
  };

  const todoTasks = tasks.filter((t) => t.status === 'todo');
  const inProgressTasks = tasks.filter((t) => t.status === 'inProgress');
  const doneTasks = tasks.filter((t) => t.status === 'done');

  const selectedSOP = selectedTask ? mockSOPs[selectedTask.sopId] : null;

  return (
    <DndProvider backend={HTML5Backend}>
      <div className="min-h-screen bg-gradient-to-br from-slate-50 via-white to-slate-50">
        {/* Header */}
        <header className="border-b border-slate-200 bg-white/80 backdrop-blur-sm sticky top-0 z-10">
          <div className="max-w-7xl mx-auto px-6 py-4">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 bg-gradient-to-br from-emerald-500 to-teal-600 rounded-lg flex items-center justify-center">
                  <Waves className="w-6 h-6 text-white" />
                </div>
                <div>
                  <h1 className="text-slate-900">Nori</h1>
                  <p className="text-sm text-slate-600">A thin layer that holds everything together</p>
                </div>
              </div>
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button className="bg-emerald-600 hover:bg-emerald-700">
                    <Plus className="w-4 h-4 mr-2" />
                    Create
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end" className="w-56">
                  <DropdownMenuItem onClick={() => setCreateTaskDialogOpen(true)}>
                    <ListChecks className="w-4 h-4 mr-2" />
                    <div>
                      <div>Task</div>
                      <div className="text-xs text-slate-500">
                        From standardized work
                      </div>
                    </div>
                  </DropdownMenuItem>
                  <DropdownMenuItem onClick={() => setCreateStandardizedWorkDialogOpen(true)}>
                    <FileText className="w-4 h-4 mr-2" />
                    <div>
                      <div>Standardized Work</div>
                      <div className="text-xs text-slate-500">
                        Create new template
                      </div>
                    </div>
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </div>
          </div>
        </header>

        {/* Main Content */}
        <main className="max-w-7xl mx-auto px-6 py-8">
          <div className="flex gap-6 overflow-x-auto pb-4">
            <KanbanColumn
              title="To Do"
              status="todo"
              tasks={todoTasks}
              onTaskClick={handleTaskClick}
              onTaskMove={handleTaskMove}
            />
            <KanbanColumn
              title="In Progress"
              status="inProgress"
              tasks={inProgressTasks}
              onTaskClick={handleTaskClick}
              onTaskMove={handleTaskMove}
            />
            <KanbanColumn
              title="Done"
              status="done"
              tasks={doneTasks}
              onTaskClick={handleTaskClick}
              onTaskMove={handleTaskMove}
            />
          </div>
        </main>

        {/* Task Detail Dialog */}
        <TaskDetailDialog
          task={selectedTask}
          sop={selectedSOP}
          open={dialogOpen}
          onClose={() => setDialogOpen(false)}
          onUpdateTask={handleUpdateTask}
          onCreatePrepTask={handleCreatePrepTask}
        />

        {/* Create Task Dialog */}
        <CreateTaskDialog
          open={createTaskDialogOpen}
          onClose={() => setCreateTaskDialogOpen(false)}
          standardizedWorks={standardizedWorks}
          onCreateTask={handleCreateTask}
        />

        {/* Create Standardized Work Dialog */}
        <CreateStandardizedWorkDialog
          open={createStandardizedWorkDialogOpen}
          onClose={() => setCreateStandardizedWorkDialogOpen(false)}
          onCreateStandardizedWork={handleCreateStandardizedWork}
        />

        <Toaster position="bottom-right" />
      </div>
    </DndProvider>
  );
}