import { useDrag } from 'react-dnd';
import { Card } from './ui/card';
import { Badge } from './ui/badge';
import { Task } from '../types';
import { Clock, User } from 'lucide-react';

interface TaskCardProps {
  task: Task;
  onClick: () => void;
}

export function TaskCard({ task, onClick }: TaskCardProps) {
  const [{ isDragging }, drag] = useDrag(() => ({
    type: 'TASK',
    item: { id: task.id },
    collect: (monitor) => ({
      isDragging: !!monitor.isDragging(),
    }),
  }));

  const getTagColor = (tag: string) => {
    switch (tag) {
      case 'order':
        return 'bg-rose-100 text-rose-700 border-rose-200';
      case 'prep':
        return 'bg-blue-100 text-blue-700 border-blue-200';
      case 'practice':
        return 'bg-purple-100 text-purple-700 border-purple-200';
      case 'special':
        return 'bg-amber-100 text-amber-700 border-amber-200';
      case 'appetizer':
        return 'bg-emerald-100 text-emerald-700 border-emerald-200';
      default:
        return 'bg-slate-100 text-slate-700 border-slate-200';
    }
  };

  const timeAgo = (timestamp: number) => {
    const minutes = Math.floor((Date.now() - timestamp) / 60000);
    if (minutes < 1) return 'Just now';
    if (minutes < 60) return `${minutes}m ago`;
    const hours = Math.floor(minutes / 60);
    return `${hours}h ago`;
  };

  return (
    <Card
      ref={drag}
      className={`p-4 cursor-pointer hover:shadow-md transition-all border border-slate-200 ${
        isDragging ? 'opacity-50' : 'opacity-100'
      }`}
      onClick={onClick}
    >
      <div className="space-y-3">
        <div>
          <h3 className="text-slate-900 mb-1">{task.title}</h3>
          {task.tableNumber && (
            <div className="text-slate-500 text-sm">Table {task.tableNumber}</div>
          )}
        </div>

        {task.orderItems && task.orderItems.length > 0 && (
          <div className="text-sm text-slate-600 space-y-1">
            {task.orderItems.map((item, idx) => (
              <div key={idx} className="flex items-center gap-2">
                <div className="w-1 h-1 rounded-full bg-slate-400" />
                {item}
              </div>
            ))}
          </div>
        )}

        <div className="flex items-center gap-2 flex-wrap">
          {task.tags.map((tag) => (
            <Badge key={tag} variant="outline" className={getTagColor(tag)}>
              {tag}
            </Badge>
          ))}
        </div>

        <div className="flex items-center gap-4 text-sm text-slate-500">
          {task.assignedTo && (
            <div className="flex items-center gap-1.5">
              <User className="w-3.5 h-3.5" />
              <span>{task.assignedTo}</span>
            </div>
          )}
          <div className="flex items-center gap-1.5 ml-auto">
            <Clock className="w-3.5 h-3.5" />
            <span>{timeAgo(task.createdAt)}</span>
          </div>
        </div>
      </div>
    </Card>
  );
}
