import { useState } from 'react';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from './ui/dialog';
import { Button } from './ui/button';
import { Input } from './ui/input';
import { Label } from './ui/label';
import { StandardizedWork, Task } from '../types';
import { ScrollArea } from './ui/scroll-area';
import { Badge } from './ui/badge';
import { Clock, FileText, RefreshCw } from 'lucide-react';
import { Switch } from './ui/switch';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from './ui/select';

interface CreateTaskDialogProps {
  open: boolean;
  onClose: () => void;
  standardizedWorks: Record<string, StandardizedWork>;
  onCreateTask: (task: Task) => void;
}

export function CreateTaskDialog({
  open,
  onClose,
  standardizedWorks,
  onCreateTask,
}: CreateTaskDialogProps) {
  const [selectedWorkId, setSelectedWorkId] = useState<string>('');
  const [taskTitle, setTaskTitle] = useState('');
  const [assignedTo, setAssignedTo] = useState('');
  const [syncToMaster, setSyncToMaster] = useState(false);

  const selectedWork = selectedWorkId ? standardizedWorks[selectedWorkId] : null;

  const handleCreate = () => {
    if (!selectedWorkId || !taskTitle) return;

    const newTask: Task = {
      id: `task-${Date.now()}`,
      title: taskTitle,
      status: 'todo',
      sopId: selectedWorkId, // Using standardized work as the SOP
      standardizedWorkId: selectedWorkId,
      syncToMaster,
      assignedTo: assignedTo || undefined,
      tags: ['standardized-work'],
      createdAt: Date.now(),
      forkedFrom: selectedWorkId,
    };

    onCreateTask(newTask);
    resetForm();
    onClose();
  };

  const resetForm = () => {
    setSelectedWorkId('');
    setTaskTitle('');
    setAssignedTo('');
    setSyncToMaster(false);
  };

  return (
    <Dialog open={open} onOpenChange={onClose}>
      <DialogContent className="max-w-[700px]">
        <DialogHeader>
          <DialogTitle>Create Task from Standardized Work</DialogTitle>
          <DialogDescription>
            Select a standardized work template and create a task instance from it.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-6 py-4">
          {/* Task Basic Info */}
          <div className="space-y-4">
            <div>
              <Label htmlFor="taskTitle">Task Title</Label>
              <Input
                id="taskTitle"
                value={taskTitle}
                onChange={(e) => setTaskTitle(e.target.value)}
                placeholder="e.g., Morning Prep - Calamari Station"
              />
            </div>

            <div>
              <Label htmlFor="assignedTo">Assign To (Optional)</Label>
              <Input
                id="assignedTo"
                value={assignedTo}
                onChange={(e) => setAssignedTo(e.target.value)}
                placeholder="e.g., Chef Name"
              />
            </div>
          </div>

          {/* Select Standardized Work */}
          <div className="space-y-3">
            <Label>Select Standardized Work Template</Label>
            <ScrollArea className="max-h-[300px] border rounded-lg">
              <div className="p-2 space-y-2">
                {Object.values(standardizedWorks).length === 0 ? (
                  <div className="text-center py-8 text-slate-500">
                    <FileText className="w-12 h-12 mx-auto mb-2 opacity-50" />
                    <p>No standardized work templates available.</p>
                    <p className="text-sm">Create one first to use as a template.</p>
                  </div>
                ) : (
                  Object.values(standardizedWorks).map((work) => (
                    <button
                      key={work.id}
                      onClick={() => setSelectedWorkId(work.id)}
                      className={`w-full text-left p-3 rounded-lg border-2 transition-all ${
                        selectedWorkId === work.id
                          ? 'border-emerald-500 bg-emerald-50'
                          : 'border-slate-200 hover:border-slate-300 bg-white'
                      }`}
                    >
                      <div className="flex items-start justify-between gap-2">
                        <div className="flex-1">
                          <div className="flex items-center gap-2 mb-1">
                            <h4 className="text-slate-900">{work.name}</h4>
                            {work.category && (
                              <Badge variant="secondary" className="text-xs">
                                {work.category}
                              </Badge>
                            )}
                          </div>
                          {work.description && (
                            <p className="text-sm text-slate-600 mb-2">
                              {work.description}
                            </p>
                          )}
                          <div className="flex items-center gap-4 text-xs text-slate-500">
                            <span className="flex items-center gap-1">
                              <Clock className="w-3 h-3" />
                              {Math.floor(work.totalTaktTime / 60)}m{' '}
                              {work.totalTaktTime % 60}s
                            </span>
                            <span>{work.workElements.length} work elements</span>
                            <span>{work.resources.length} resources</span>
                          </div>
                        </div>
                      </div>
                    </button>
                  ))
                )}
              </div>
            </ScrollArea>
          </div>

          {/* Sync Option */}
          {selectedWork && (
            <div className="border rounded-lg p-4 space-y-3 bg-slate-50">
              <div className="flex items-start justify-between">
                <div className="flex-1">
                  <div className="flex items-center gap-2 mb-1">
                    <RefreshCw className="w-4 h-4 text-emerald-600" />
                    <Label htmlFor="syncToMaster" className="cursor-pointer">
                      Sync changes back to master
                    </Label>
                  </div>
                  <p className="text-sm text-slate-600">
                    When enabled, any improvements or changes made to this task will
                    automatically update the original standardized work template.
                  </p>
                </div>
                <Switch
                  id="syncToMaster"
                  checked={syncToMaster}
                  onCheckedChange={setSyncToMaster}
                />
              </div>
            </div>
          )}

          {/* Preview */}
          {selectedWork && (
            <div className="border rounded-lg p-4 space-y-2 bg-white">
              <Label className="text-xs text-slate-600">Preview</Label>
              <div className="space-y-1">
                <p className="text-sm">
                  <span className="text-slate-600">Task:</span>{' '}
                  <span className="text-slate-900">
                    {taskTitle || 'Untitled Task'}
                  </span>
                </p>
                <p className="text-sm">
                  <span className="text-slate-600">Based on:</span>{' '}
                  <span className="text-slate-900">{selectedWork.name}</span>
                </p>
                <p className="text-sm">
                  <span className="text-slate-600">Work Elements:</span>{' '}
                  <span className="text-slate-900">
                    {selectedWork.workElements.length} steps
                  </span>
                </p>
                <p className="text-sm">
                  <span className="text-slate-600">Estimated Time:</span>{' '}
                  <span className="text-slate-900">
                    {Math.floor(selectedWork.totalTaktTime / 60)}m{' '}
                    {selectedWork.totalTaktTime % 60}s
                  </span>
                </p>
              </div>
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onClose}>
            Cancel
          </Button>
          <Button
            onClick={handleCreate}
            disabled={!selectedWorkId || !taskTitle}
            className="bg-emerald-600 hover:bg-emerald-700"
          >
            Create Task
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}