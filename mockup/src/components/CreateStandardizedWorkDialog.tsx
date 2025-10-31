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
import { Textarea } from './ui/textarea';
import { Label } from './ui/label';
import { StandardizedWork, WorkElement, Material, KaizenSuggestion } from '../types';
import { Plus, X, Clock } from 'lucide-react';
import { ScrollArea } from './ui/scroll-area';

interface CreateStandardizedWorkDialogProps {
  open: boolean;
  onClose: () => void;
  onCreateStandardizedWork: (work: StandardizedWork) => void;
}

export function CreateStandardizedWorkDialog({
  open,
  onClose,
  onCreateStandardizedWork,
}: CreateStandardizedWorkDialogProps) {
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [category, setCategory] = useState('');
  const [pullSignal, setPullSignal] = useState('');
  const [workElements, setWorkElements] = useState<WorkElement[]>([
    {
      id: 'we-1',
      description: '',
      taktTime: 60,
      station: '',
      completed: false,
      inProgress: false,
    },
  ]);
  const [resources, setResources] = useState<Material[]>([
    { name: '', quantity: '', location: '' },
  ]);

  const addWorkElement = () => {
    setWorkElements([
      ...workElements,
      {
        id: `we-${workElements.length + 1}`,
        description: '',
        taktTime: 60,
        station: '',
        completed: false,
        inProgress: false,
      },
    ]);
  };

  const removeWorkElement = (index: number) => {
    if (workElements.length > 1) {
      setWorkElements(workElements.filter((_, i) => i !== index));
    }
  };

  const updateWorkElement = (index: number, field: keyof WorkElement, value: any) => {
    const updated = [...workElements];
    updated[index] = { ...updated[index], [field]: value };
    setWorkElements(updated);
  };

  const addResource = () => {
    setResources([...resources, { name: '', quantity: '', location: '' }]);
  };

  const removeResource = (index: number) => {
    if (resources.length > 1) {
      setResources(resources.filter((_, i) => i !== index));
    }
  };

  const updateResource = (index: number, field: keyof Material, value: any) => {
    const updated = [...resources];
    updated[index] = { ...updated[index], [field]: value };
    setResources(updated);
  };

  const handleCreate = () => {
    const totalTaktTime = workElements.reduce((sum, we) => sum + we.taktTime, 0);
    const now = Date.now();

    const newWork: StandardizedWork = {
      id: `sw-${now}`,
      name,
      description,
      category,
      pullSignal,
      workElements,
      resources,
      totalTaktTime,
      kaizen: [],
      createdAt: now,
      updatedAt: now,
    };

    onCreateStandardizedWork(newWork);
    resetForm();
    onClose();
  };

  const resetForm = () => {
    setName('');
    setDescription('');
    setCategory('');
    setPullSignal('');
    setWorkElements([
      {
        id: 'we-1',
        description: '',
        taktTime: 60,
        station: '',
        completed: false,
        inProgress: false,
      },
    ]);
    setResources([{ name: '', quantity: '', location: '' }]);
  };

  const totalTaktTime = workElements.reduce((sum, we) => sum + we.taktTime, 0);

  return (
    <Dialog open={open} onOpenChange={onClose}>
      <DialogContent className="max-w-[900px] max-h-[85vh]">
        <DialogHeader>
          <DialogTitle>Create Standardized Work</DialogTitle>
          <DialogDescription>
            Define work elements, resources, and takt time for a new standardized work template.
          </DialogDescription>
        </DialogHeader>

        <ScrollArea className="max-h-[60vh] pr-4">
          <div className="space-y-6">
            {/* Basic Info */}
            <div className="space-y-4">
              <div>
                <Label htmlFor="name">Work Name</Label>
                <Input
                  id="name"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="e.g., Calamari Prep Station Setup"
                />
              </div>

              <div>
                <Label htmlFor="description">Description</Label>
                <Textarea
                  id="description"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  placeholder="Brief description of this standardized work"
                  rows={2}
                />
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <Label htmlFor="category">Category</Label>
                  <Input
                    id="category"
                    value={category}
                    onChange={(e) => setCategory(e.target.value)}
                    placeholder="e.g., prep, service, cleanup"
                  />
                </div>
                <div>
                  <Label htmlFor="pullSignal">Pull Signal</Label>
                  <Input
                    id="pullSignal"
                    value={pullSignal}
                    onChange={(e) => setPullSignal(e.target.value)}
                    placeholder="e.g., When inventory < 2 units"
                  />
                </div>
              </div>
            </div>

            {/* Work Elements */}
            <div className="space-y-3">
              <div className="flex items-center justify-between">
                <Label>Work Elements (Sub-Tasks)</Label>
                <div className="flex items-center gap-2 text-sm text-slate-600">
                  <Clock className="w-4 h-4" />
                  Total: {Math.floor(totalTaktTime / 60)}m {totalTaktTime % 60}s
                </div>
              </div>
              {workElements.map((element, index) => (
                <div key={index} className="border rounded-lg p-4 space-y-3">
                  <div className="flex items-start gap-2">
                    <div className="flex-1 space-y-3">
                      <Input
                        value={element.description}
                        onChange={(e) =>
                          updateWorkElement(index, 'description', e.target.value)
                        }
                        placeholder="Work element description"
                      />
                      <div className="grid grid-cols-2 gap-3">
                        <div>
                          <Label className="text-xs text-slate-600">
                            Takt Time (seconds)
                          </Label>
                          <Input
                            type="number"
                            value={element.taktTime}
                            onChange={(e) =>
                              updateWorkElement(
                                index,
                                'taktTime',
                                parseInt(e.target.value) || 0
                              )
                            }
                            placeholder="60"
                          />
                        </div>
                        <div>
                          <Label className="text-xs text-slate-600">
                            Station/Person
                          </Label>
                          <Input
                            value={element.station || ''}
                            onChange={(e) =>
                              updateWorkElement(index, 'station', e.target.value)
                            }
                            placeholder="e.g., Prep station"
                          />
                        </div>
                      </div>
                    </div>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => removeWorkElement(index)}
                      disabled={workElements.length === 1}
                    >
                      <X className="w-4 h-4" />
                    </Button>
                  </div>
                </div>
              ))}
              <Button
                variant="outline"
                size="sm"
                onClick={addWorkElement}
                className="w-full"
              >
                <Plus className="w-4 h-4 mr-2" />
                Add Work Element
              </Button>
            </div>

            {/* Resources */}
            <div className="space-y-3">
              <Label>Resources (Tools, Materials, Equipment)</Label>
              {resources.map((resource, index) => (
                <div key={index} className="border rounded-lg p-3">
                  <div className="flex items-start gap-2">
                    <div className="flex-1 grid grid-cols-3 gap-2">
                      <Input
                        value={resource.name}
                        onChange={(e) => updateResource(index, 'name', e.target.value)}
                        placeholder="Name"
                      />
                      <Input
                        value={resource.quantity}
                        onChange={(e) => updateResource(index, 'quantity', e.target.value)}
                        placeholder="Quantity"
                      />
                      <Input
                        value={resource.location}
                        onChange={(e) => updateResource(index, 'location', e.target.value)}
                        placeholder="Location"
                      />
                    </div>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => removeResource(index)}
                      disabled={resources.length === 1}
                    >
                      <X className="w-4 h-4" />
                    </Button>
                  </div>
                </div>
              ))}
              <Button
                variant="outline"
                size="sm"
                onClick={addResource}
                className="w-full"
              >
                <Plus className="w-4 h-4 mr-2" />
                Add Resource
              </Button>
            </div>
          </div>
        </ScrollArea>

        <DialogFooter>
          <Button variant="outline" onClick={onClose}>
            Cancel
          </Button>
          <Button
            onClick={handleCreate}
            disabled={!name || workElements.every((we) => !we.description)}
            className="bg-emerald-600 hover:bg-emerald-700"
          >
            Create Standardized Work
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}