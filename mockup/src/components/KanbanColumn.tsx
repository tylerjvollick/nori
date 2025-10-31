import { useDrop } from 'react-dnd';
import { Card } from './ui/card';
import { TaskCard } from './TaskCard';
import { Task, KanbanColumn as ColumnType } from '../types';
import { CheckCircle2, Circle, Clock } from 'lucide-react';

interface KanbanColumnProps {
  title: string;
  status: ColumnType;
  tasks: Task[];
  onTaskClick: (task: Task) => void;
  onTaskMove: (taskId: string, newStatus: ColumnType) => void;
}

const columnIcons = {
  todo: Circle,
  inProgress: Clock,
  done: CheckCircle2,
};

const columnColors = {
  todo: 'text-slate-500',
  inProgress: 'text-blue-500',
  done: 'text-emerald-500',
};

export function KanbanColumn({ title, status, tasks, onTaskClick, onTaskMove }: KanbanColumnProps) {
  const [{ isOver }, drop] = useDrop(() => ({
    accept: 'TASK',
    drop: (item: { id: string }) => {
      onTaskMove(item.id, status);
    },
    collect: (monitor) => ({
      isOver: !!monitor.isOver(),
    }),
  }));

  const Icon = columnIcons[status];

  return (
    <div className="flex flex-col flex-1 min-w-[320px]">
      <div className="flex items-center gap-2 mb-4 px-1">
        <Icon className={`w-5 h-5 ${columnColors[status]}`} />
        <h2 className="text-slate-700">{title}</h2>
        <span className="ml-auto text-slate-400">
          {tasks.length}
        </span>
      </div>
      <div
        ref={drop}
        className={`flex-1 p-4 rounded-lg transition-colors ${
          isOver ? 'bg-blue-50 border-2 border-blue-200' : 'bg-slate-50 border-2 border-transparent'
        }`}
      >
        <div className="space-y-3">
          {tasks.map((task) => (
            <TaskCard
              key={task.id}
              task={task}
              onClick={() => onTaskClick(task)}
            />
          ))}
          {tasks.length === 0 && (
            <div className="text-center py-12 text-slate-400">
              No tasks
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
