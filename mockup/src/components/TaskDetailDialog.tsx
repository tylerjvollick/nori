import { useState, useEffect } from 'react';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from './ui/dialog';
import { Button } from './ui/button';
import { Badge } from './ui/badge';
import { Progress } from './ui/progress';
import { Card } from './ui/card';
import { Separator } from './ui/separator';
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from './ui/collapsible';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from './ui/select';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from './ui/dropdown-menu';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from './ui/popover';
import { Textarea } from './ui/textarea';
import { Input } from './ui/input';
import { Task, SOP, ProcedureStep } from '../types';
import {
  Play,
  Pause,
  CheckCircle2,
  Circle,
  Clock,
  Package,
  AlertTriangle,
  ChevronDown,
  ChevronRight,
  ArrowDown,
  Check,
  Wrench,
  User,
  MoreVertical,
  Maximize2,
  MessageSquare,
  Send,
  Edit3,
} from 'lucide-react';
import { ImageWithFallback } from './figma/ImageWithFallback';

interface TaskDetailDialogProps {
  task: Task | null;
  sop: SOP | null;
  open: boolean;
  onClose: () => void;
  onUpdateTask: (taskId: string, updates: Partial<Task>) => void;
  onCreatePrepTask: (sopId: string, title: string) => void;
}

export function TaskDetailDialog({ task, sop, open, onClose, onUpdateTask, onCreatePrepTask }: TaskDetailDialogProps) {
  const [steps, setSteps] = useState<ProcedureStep[]>([]);
  const [expandedSteps, setExpandedSteps] = useState<Set<string>>(new Set());
  const [expandedMaterials, setExpandedMaterials] = useState<Set<number>>(new Set());
  const [checkedMaterials, setCheckedMaterials] = useState<Set<number>>(new Set());
  const [timePopoverOpen, setTimePopoverOpen] = useState<string | null>(null);
  const [customTime, setCustomTime] = useState('');
  const [stepDescriptions, setStepDescriptions] = useState<Record<string, string>>({});
  const [stepComments, setStepComments] = useState<Record<string, string>>({});
  const [newComment, setNewComment] = useState<Record<string, string>>({});
  const [fullScreenStep, setFullScreenStep] = useState<string | null>(null);

  useEffect(() => {
    if (sop) {
      const initialSteps = JSON.parse(JSON.stringify(sop.procedure));
      setSteps(initialSteps);
      
      // Initialize descriptions
      const descriptions: Record<string, string> = {};
      initialSteps.forEach((step: ProcedureStep) => {
        descriptions[step.id] = 'Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris.';
      });
      setStepDescriptions(descriptions);
    }
  }, [sop]);

  if (!task || !sop) return null;

  const completedSteps = steps.filter((s) => s.completed).length;
  const progress = (completedSteps / steps.length) * 100;

  const handleStepAction = (stepId: string, action: 'start' | 'pause' | 'complete') => {
    const step = steps.find(s => s.id === stepId);
    
    // If completing without starting, show time popover
    if (action === 'complete' && !step?.startTime && !step?.timeSpent) {
      setTimePopoverOpen(stepId);
      return;
    }

    setSteps((prevSteps) =>
      prevSteps.map((step) => {
        if (step.id === stepId) {
          if (action === 'start') {
            return { ...step, inProgress: true, startTime: Date.now() };
          } else if (action === 'pause') {
            const timeSpent = step.timeSpent || 0;
            const additionalTime = step.startTime ? Date.now() - step.startTime : 0;
            return {
              ...step,
              inProgress: false,
              timeSpent: timeSpent + additionalTime,
              startTime: undefined,
            };
          } else if (action === 'complete') {
            const timeSpent = step.timeSpent || 0;
            const additionalTime = step.startTime ? Date.now() - step.startTime : 0;
            const totalTime = timeSpent + additionalTime;
            
            return {
              ...step,
              completed: true,
              inProgress: false,
              timeSpent: totalTime,
              startTime: undefined,
            };
          }
        }
        // Auto-start next step when completing current
        if (action === 'complete') {
          const completedIndex = prevSteps.findIndex((s) => s.id === stepId);
          const currentIndex = prevSteps.findIndex((s) => s.id === step.id);
          if (currentIndex === completedIndex + 1 && !step.completed) {
            return { ...step, inProgress: true, startTime: Date.now() };
          }
        }
        return step;
      })
    );
  };

  const handleQuickTimeComplete = (stepId: string, timeMs: number | null) => {
    setSteps((prevSteps) =>
      prevSteps.map((step, idx) => {
        if (step.id === stepId) {
          return {
            ...step,
            completed: true,
            inProgress: false,
            timeSpent: timeMs || 0,
            startTime: undefined,
          };
        }
        // Auto-start next step
        const completedIndex = prevSteps.findIndex((s) => s.id === stepId);
        if (idx === completedIndex + 1 && !step.completed) {
          return { ...step, inProgress: true, startTime: Date.now() };
        }
        return step;
      })
    );
    setTimePopoverOpen(null);
    setCustomTime('');
  };

  const toggleStepExpand = (stepId: string) => {
    setExpandedSteps((prev) => {
      const next = new Set(prev);
      if (next.has(stepId)) {
        next.delete(stepId);
      } else {
        next.add(stepId);
      }
      return next;
    });
  };

  const toggleMaterialExpand = (index: number) => {
    setExpandedMaterials((prev) => {
      const next = new Set(prev);
      if (next.has(index)) {
        next.delete(index);
      } else {
        next.add(index);
      }
      return next;
    });
  };

  const toggleMaterialCheck = (index: number) => {
    setCheckedMaterials((prev) => {
      const next = new Set(prev);
      if (next.has(index)) {
        next.delete(index);
      } else {
        next.add(index);
      }
      return next;
    });
  };

  const formatTime = (ms?: number) => {
    if (!ms) return '0s';
    const seconds = Math.floor(ms / 1000);
    const minutes = Math.floor(seconds / 60);
    if (minutes > 0) {
      return `${minutes}m ${seconds % 60}s`;
    }
    return `${seconds}s`;
  };

  const getDifficultyColor = (difficulty?: string) => {
    switch (difficulty) {
      case 'easy':
        return 'bg-emerald-100 text-emerald-700 border-emerald-200';
      case 'medium':
        return 'bg-blue-100 text-blue-700 border-blue-200';
      case 'hard':
        return 'bg-amber-100 text-amber-700 border-amber-200';
      case 'master':
        return 'bg-rose-100 text-rose-700 border-rose-200';
      default:
        return 'bg-slate-100 text-slate-700 border-slate-200';
    }
  };

  const totalTimeSpent = steps.reduce((acc, step) => acc + (step.timeSpent || 0), 0);

  const handleStatusChange = (newStatus: string) => {
    onUpdateTask(task.id, { status: newStatus as Task['status'] });
  };

  const handleAddComment = (stepId: string) => {
    const comment = newComment[stepId];
    if (!comment?.trim()) return;
    
    setStepComments(prev => ({
      ...prev,
      [stepId]: (prev[stepId] || '') + (prev[stepId] ? '\n\n' : '') + `💬 ${comment}`
    }));
    
    setNewComment(prev => ({ ...prev, [stepId]: '' }));
  };

  const quickTimeOptions = [
    { label: '30s', value: 30000 },
    { label: '1m', value: 60000 },
    { label: '5m', value: 300000 },
    { label: '10m', value: 600000 },
    { label: '30m', value: 1800000 },
    { label: '1h', value: 3600000 },
  ];

  // Placeholder images for steps
  const placeholderImages = [
    'https://images.unsplash.com/photo-1556910103-1c02745aae4d?w=400&h=300&fit=crop',
    'https://images.unsplash.com/photo-1559847844-5315695dadae?w=400&h=300&fit=crop',
  ];

  return (
    <Dialog open={open} onOpenChange={onClose}>
      <DialogContent className="w-[98vw] max-w-[2400px] max-h-[90vh] overflow-hidden flex flex-col">
        <DialogHeader className="sr-only">
          <DialogTitle>{task.title}</DialogTitle>
          <DialogDescription>
            Task details for {task.title} - {sop.name}
          </DialogDescription>
        </DialogHeader>

        {/* Two Column Layout */}
        <div className="grid grid-cols-[1fr_400px] gap-6 flex-1 min-h-0">
          {/* Left Column - Main Content */}
          <div className="space-y-4 overflow-y-auto pr-2">
            {/* Task Title and Badges */}
            <div className="space-y-3">
              <h2 className="text-slate-900">{task.title}</h2>
              <div className="flex items-center gap-2 flex-wrap">
                <Badge variant="outline" className="bg-slate-100 text-slate-700 border-slate-200">
                  {sop.name}
                </Badge>
                {sop.difficulty && (
                  <Badge variant="outline" className={getDifficultyColor(sop.difficulty)}>
                    {sop.difficulty}
                  </Badge>
                )}
              </div>
            </div>

            {/* Progress */}
            <div className="space-y-2">
              <div className="flex items-center justify-between text-sm">
                <span className="text-slate-600">Progress</span>
                <span className="text-slate-900">
                  {completedSteps} / {steps.length} steps
                </span>
              </div>
              <Progress value={progress} className="h-2" />
            </div>

            <Separator />

            {/* Procedure */}
            <div className="space-y-3">
              <h3 className="text-slate-900">Procedure</h3>
              <div className="space-y-1.5">
                {steps.map((step, idx) => {
                  const isExpanded = expandedSteps.has(step.id);

                  return (
                    <Card
                      key={step.id}
                      className={`p-2 border transition-colors ${
                        step.completed
                          ? 'border-emerald-200 bg-emerald-50'
                          : step.inProgress
                          ? 'border-blue-200 bg-blue-50'
                          : 'border-slate-200'
                      }`}
                    >
                      <div className="flex items-center justify-between gap-4">
                        <div className="flex items-center gap-2 flex-1">
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => toggleStepExpand(step.id)}
                            className="h-5 w-5 p-0"
                          >
                            {isExpanded ? (
                              <ChevronDown className="w-3.5 h-3.5" />
                            ) : (
                              <ChevronRight className="w-3.5 h-3.5" />
                            )}
                          </Button>
                          <div className="flex items-center gap-2 flex-1 text-sm">
                            {step.completed ? (
                              <CheckCircle2 className="w-4 h-4 text-emerald-600" />
                            ) : step.inProgress ? (
                              <Clock className="w-4 h-4 text-blue-600" />
                            ) : (
                              <Circle className="w-4 h-4 text-slate-400" />
                            )}
                            <span className="text-slate-900">
                              {idx + 1}. {step.description}
                            </span>
                          </div>
                        </div>
                        <div className="flex items-center gap-1">
                          {!step.completed && (
                            <>
                              {!step.inProgress && (
                                <Button
                                  size="sm"
                                  onClick={() => handleStepAction(step.id, 'start')}
                                  className="bg-blue-600 hover:bg-blue-700 h-7 w-7 p-0"
                                  title="Start"
                                >
                                  <Play className="w-3 h-3" />
                                </Button>
                              )}
                              {step.inProgress && (
                                <Button
                                  size="sm"
                                  variant="outline"
                                  onClick={() => handleStepAction(step.id, 'pause')}
                                  className="h-7 w-7 p-0"
                                  title="Pause"
                                >
                                  <Pause className="w-3 h-3" />
                                </Button>
                              )}
                              <Popover open={timePopoverOpen === step.id} onOpenChange={(open) => !open && setTimePopoverOpen(null)}>
                                <PopoverTrigger asChild>
                                  <Button
                                    size="sm"
                                    onClick={() => {
                                      if (step.inProgress || step.timeSpent) {
                                        handleStepAction(step.id, 'complete');
                                      } else {
                                        setTimePopoverOpen(step.id);
                                      }
                                    }}
                                    className="bg-emerald-600 hover:bg-emerald-700 h-7 w-7 p-0"
                                    title="Complete"
                                  >
                                    <Check className="w-3 h-3" />
                                  </Button>
                                </PopoverTrigger>
                                <PopoverContent className="w-64">
                                  <div className="space-y-3">
                                    <h4 className="text-sm">How long did this step take?</h4>
                                    <div className="grid grid-cols-3 gap-2">
                                      {quickTimeOptions.map((option) => (
                                        <Button
                                          key={option.label}
                                          size="sm"
                                          variant="outline"
                                          onClick={() => handleQuickTimeComplete(step.id, option.value)}
                                          className="text-xs"
                                        >
                                          {option.label}
                                        </Button>
                                      ))}
                                    </div>
                                    <div className="space-y-2">
                                      <Input
                                        placeholder="Custom (e.g., 45m)"
                                        value={customTime}
                                        onChange={(e) => setCustomTime(e.target.value)}
                                        className="text-sm"
                                      />
                                      <div className="flex gap-2">
                                        <Button
                                          size="sm"
                                          onClick={() => handleQuickTimeComplete(step.id, null)}
                                          variant="ghost"
                                          className="flex-1 text-xs"
                                        >
                                          Skip
                                        </Button>
                                        <Button
                                          size="sm"
                                          onClick={() => {
                                            // Parse custom time (simplified)
                                            const match = customTime.match(/(\d+)([smh])/);
                                            if (match) {
                                              const value = parseInt(match[1]);
                                              const unit = match[2];
                                              const multiplier = unit === 's' ? 1000 : unit === 'm' ? 60000 : 3600000;
                                              handleQuickTimeComplete(step.id, value * multiplier);
                                            }
                                          }}
                                          className="flex-1 text-xs"
                                        >
                                          Set
                                        </Button>
                                      </div>
                                    </div>
                                  </div>
                                </PopoverContent>
                              </Popover>
                            </>
                          )}
                          <DropdownMenu>
                            <DropdownMenuTrigger asChild>
                              <Button
                                size="sm"
                                variant="ghost"
                                className="h-7 w-7 p-0"
                              >
                                <MoreVertical className="w-3 h-3" />
                              </Button>
                            </DropdownMenuTrigger>
                            <DropdownMenuContent align="end">
                              <DropdownMenuItem>Edit Step</DropdownMenuItem>
                              <DropdownMenuItem>Reset Timer</DropdownMenuItem>
                              <DropdownMenuItem>Delete Step</DropdownMenuItem>
                            </DropdownMenuContent>
                          </DropdownMenu>
                        </div>
                      </div>
                      
                      {isExpanded && (
                        <div className="mt-3 pl-7 space-y-3">
                          {/* Expand to full view button */}
                          <div className="flex justify-end">
                            <Button
                              size="sm"
                              variant="ghost"
                              onClick={() => setFullScreenStep(step.id)}
                              className="h-7 text-xs"
                            >
                              <Maximize2 className="w-3 h-3 mr-1" />
                              Expand
                            </Button>
                          </div>

                          {/* Editable Description */}
                          <div className="space-y-2">
                            <div className="flex items-center gap-2">
                              <Edit3 className="w-3 h-3 text-slate-400" />
                              <span className="text-xs text-slate-500">Description</span>
                            </div>
                            <Textarea
                              value={stepDescriptions[step.id] || ''}
                              onChange={(e) => setStepDescriptions(prev => ({ ...prev, [step.id]: e.target.value }))}
                              className="text-xs min-h-[60px]"
                              placeholder="Add step description..."
                            />
                          </div>

                          {/* Photos */}
                          <div className="space-y-2">
                            <span className="text-xs text-slate-500">Photos</span>
                            <div className="grid grid-cols-2 gap-2">
                              {placeholderImages.map((img, imgIdx) => (
                                <ImageWithFallback
                                  key={imgIdx}
                                  src={img}
                                  alt={`Step ${idx + 1} detail ${imgIdx + 1}`}
                                  className="rounded-md w-full h-32 object-cover"
                                />
                              ))}
                            </div>
                          </div>

                          {step.timeSpent && (
                            <div className="text-xs text-slate-600">
                              Time spent: {formatTime(step.timeSpent)}
                            </div>
                          )}

                          <Separator />

                          {/* Comments Section */}
                          <div className="space-y-2">
                            <div className="flex items-center gap-2">
                              <MessageSquare className="w-3 h-3 text-slate-400" />
                              <span className="text-xs text-slate-500">Comments & Suggestions</span>
                            </div>
                            
                            {stepComments[step.id] && (
                              <div className="text-xs text-slate-600 bg-slate-50 p-2 rounded whitespace-pre-wrap">
                                {stepComments[step.id]}
                              </div>
                            )}
                            
                            <div className="flex gap-2">
                              <Textarea
                                value={newComment[step.id] || ''}
                                onChange={(e) => setNewComment(prev => ({ ...prev, [step.id]: e.target.value }))}
                                placeholder="Add a comment or suggestion..."
                                className="text-xs min-h-[50px] flex-1"
                              />
                              <Button
                                size="sm"
                                onClick={() => handleAddComment(step.id)}
                                className="h-[50px] w-[50px] p-0"
                              >
                                <Send className="w-3 h-3" />
                              </Button>
                            </div>
                          </div>
                        </div>
                      )}
                    </Card>
                  );
                })}
              </div>
            </div>
          </div>

          {/* Right Column - Sidebar */}
          <div className="space-y-4 overflow-y-auto pr-2">
            {/* Status Picker */}
            <div className="space-y-2">
              <label className="text-sm text-slate-600">Status</label>
              <Select value={task.status} onValueChange={handleStatusChange}>
                <SelectTrigger className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="todo">To Do</SelectItem>
                  <SelectItem value="inProgress">In Progress</SelectItem>
                  <SelectItem value="done">Done</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <Separator />

            {/* Materials - Collapsible */}
            <Collapsible defaultOpen>
              <CollapsibleTrigger className="flex items-center justify-between w-full group">
                <div className="flex items-center gap-2">
                  <Package className="w-4 h-4 text-slate-600" />
                  <h3 className="text-sm text-slate-900">Materials ({sop.materials.length})</h3>
                </div>
                <ChevronDown className="w-4 h-4 text-slate-400 transition-transform group-data-[state=closed]:-rotate-90" />
              </CollapsibleTrigger>
              <CollapsibleContent className="mt-3">
                <div className="space-y-1.5">
                  {sop.materials.map((material, idx) => {
                    const isExpanded = expandedMaterials.has(idx);
                    const isChecked = checkedMaterials.has(idx);
                    const hasDetails = material.location || material.pullTrigger;

                    return (
                      <Card key={idx} className={`p-2 border transition-colors ${
                        isChecked ? 'border-emerald-200 bg-emerald-50' : 'border-slate-200'
                      }`}>
                        <div className="flex items-center justify-between gap-2">
                          <div className="flex items-center gap-1.5 flex-1 min-w-0">
                            {hasDetails && (
                              <Button
                                variant="ghost"
                                size="sm"
                                onClick={() => toggleMaterialExpand(idx)}
                                className="h-5 w-5 p-0 shrink-0"
                              >
                                {isExpanded ? (
                                  <ChevronDown className="w-3 h-3" />
                                ) : (
                                  <ChevronRight className="w-3 h-3" />
                                )}
                              </Button>
                            )}
                            <div className="flex flex-col gap-0.5 flex-1 min-w-0">
                              <div className="text-xs text-slate-900 truncate">{material.name}</div>
                              <div className="text-xs text-slate-500">{material.quantity}</div>
                            </div>
                          </div>
                          <div className="flex items-center gap-1 shrink-0">
                            {material.lowStock && (
                              <AlertTriangle className="w-3.5 h-3.5 text-amber-500" />
                            )}
                            {material.prepSopId && material.pullTrigger && (
                              <Button
                                size="sm"
                                variant="outline"
                                onClick={() => onCreatePrepTask(material.prepSopId!, material.prepTaskTitle || `Make ${material.name}`)}
                                className="h-6 w-6 p-0 border-emerald-300 text-emerald-700 hover:bg-emerald-50"
                              >
                                <ArrowDown className="w-3 h-3" />
                              </Button>
                            )}
                            <Button
                              size="sm"
                              variant="outline"
                              onClick={() => toggleMaterialCheck(idx)}
                              className={`h-6 w-6 p-0 ${
                                isChecked 
                                  ? 'border-emerald-300 bg-emerald-100 text-emerald-700 hover:bg-emerald-200' 
                                  : 'border-slate-300 text-slate-600 hover:bg-slate-50'
                              }`}
                            >
                              <Check className="w-3 h-3" />
                            </Button>
                          </div>
                        </div>
                        {isExpanded && hasDetails && (
                          <div className="mt-2 pl-6 space-y-0.5">
                            {material.location && (
                              <div className="text-xs text-slate-500">📍 {material.location}</div>
                            )}
                            {material.pullTrigger && (
                              <div className="text-xs text-amber-600">⚠️ {material.pullTrigger}</div>
                            )}
                          </div>
                        )}
                      </Card>
                    );
                  })}
                </div>
              </CollapsibleContent>
            </Collapsible>

            {/* Equipment - Collapsible */}
            <Collapsible defaultOpen={false}>
              <CollapsibleTrigger className="flex items-center justify-between w-full group">
                <div className="flex items-center gap-2">
                  <Wrench className="w-4 h-4 text-slate-600" />
                  <h3 className="text-sm text-slate-900">Equipment (0)</h3>
                </div>
                <ChevronDown className="w-4 h-4 text-slate-400 transition-transform group-data-[state=closed]:-rotate-90" />
              </CollapsibleTrigger>
              <CollapsibleContent className="mt-3">
                <div className="text-xs text-slate-500 text-center py-4">
                  No equipment listed
                </div>
              </CollapsibleContent>
            </Collapsible>

            <Separator />

            {/* Meta Fields */}
            <div className="space-y-3">
              <h3 className="text-sm text-slate-900">Details</h3>
              <div className="space-y-2 text-sm">
                {sop.estimatedTime && (
                  <div className="flex items-center justify-between">
                    <span className="text-slate-600">Takt Time</span>
                    <span className="text-slate-900">{sop.estimatedTime} min</span>
                  </div>
                )}
                <div className="flex items-center justify-between">
                  <span className="text-slate-600">Time Spent</span>
                  <span className="text-slate-900">{formatTime(totalTimeSpent)}</span>
                </div>
                {task.assignedTo && (
                  <div className="flex items-center justify-between">
                    <span className="text-slate-600">Assignee</span>
                    <span className="text-slate-900 flex items-center gap-1">
                      <User className="w-3.5 h-3.5" />
                      {task.assignedTo}
                    </span>
                  </div>
                )}
                {task.tableNumber && (
                  <div className="flex items-center justify-between">
                    <span className="text-slate-600">Table</span>
                    <span className="text-slate-900">#{task.tableNumber}</span>
                  </div>
                )}
                {task.tags && task.tags.length > 0 && (
                  <div className="flex items-start justify-between">
                    <span className="text-slate-600">Tags</span>
                    <div className="flex gap-1 flex-wrap justify-end">
                      {task.tags.map((tag, idx) => (
                        <Badge key={idx} variant="outline" className="text-xs">
                          {tag}
                        </Badge>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            </div>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
