import { SOP, Task, StandardizedWork } from '../types';

export const mockStandardizedWorks: Record<string, StandardizedWork> = {
  'sw-calamari-prep': {
    id: 'sw-calamari-prep',
    name: 'Calamari Prep Station Setup',
    description: 'Standard work for preparing the calamari frying station',
    totalTaktTime: 900, // 15 minutes
    pullSignal: 'When <2 portions of pickled calamari remain',
    category: 'prep',
    createdAt: Date.now() - 86400000,
    updatedAt: Date.now() - 86400000,
    resources: [
      {
        name: 'Pickled Calamari',
        quantity: '2 jars',
        location: 'Top shelf fridge #1',
      },
      {
        name: 'Flour',
        quantity: '2 cups',
        location: 'Baking shelf',
      },
      {
        name: 'Panko Bread Crumbs',
        quantity: '2 cups',
        location: 'Baking shelf',
      },
      {
        name: 'Three Stainless Steel Bowls',
        quantity: '3',
        location: 'Kitchen station',
      },
    ],
    workElements: [
      {
        id: 'we-1',
        description: 'Drain pickled calamari and pat dry',
        taktTime: 180,
        station: 'Prep station',
        completed: false,
        inProgress: false,
      },
      {
        id: 'we-2',
        description: 'Set up three-bowl breading station (flour, egg wash, panko)',
        taktTime: 240,
        station: 'Prep station',
        completed: false,
        inProgress: false,
      },
      {
        id: 'we-3',
        description: 'Portion calamari into 8 serving sizes',
        taktTime: 300,
        station: 'Prep station',
        completed: false,
        inProgress: false,
      },
      {
        id: 'we-4',
        description: 'Store portioned calamari in labeled containers',
        taktTime: 180,
        station: 'Prep station',
        completed: false,
        inProgress: false,
      },
    ],
    kaizen: [
      {
        id: 'k-1',
        author: 'Chef Mike',
        suggestion: 'Add a timer reminder to check oil temperature before starting',
        status: 'approved',
        createdAt: Date.now() - 43200000,
      },
    ],
  },
};

export const mockSOPs: Record<string, SOP> = {
  'sop-pickled-calamari': {
    id: 'sop-pickled-calamari',
    name: 'Make Pickled Calamari',
    difficulty: 'easy',
    estimatedTime: 45,
    materials: [
      {
        name: 'Fresh Calamari',
        quantity: '2 lbs',
        location: 'Seafood fridge',
      },
      {
        name: 'White Vinegar',
        quantity: '2 cups',
        location: 'Baking shelf',
      },
      {
        name: 'Pickling Spices',
        quantity: '2 tbsp',
        location: 'Spice rack',
      },
      {
        name: 'Salt',
        quantity: '1 tbsp',
        location: 'Spice rack',
      },
      {
        name: 'Mason Jars',
        quantity: '3',
        location: 'Storage cabinet',
      },
    ],
    procedure: [
      {
        id: 'step-1',
        description: 'Clean and slice calamari into rings',
        completed: false,
        inProgress: false,
      },
      {
        id: 'step-2',
        description: 'Blanch calamari in boiling water for 2 minutes',
        completed: false,
        inProgress: false,
      },
      {
        id: 'step-3',
        description: 'Prepare pickling brine with vinegar, spices, and salt',
        completed: false,
        inProgress: false,
      },
      {
        id: 'step-4',
        description: 'Pack calamari into sterilized mason jars',
        completed: false,
        inProgress: false,
      },
      {
        id: 'step-5',
        description: 'Pour hot brine over calamari, seal jars',
        completed: false,
        inProgress: false,
      },
      {
        id: 'step-6',
        description: 'Refrigerate for 24 hours before use',
        completed: false,
        inProgress: false,
      },
    ],
  },
  'sop-calamari': {
    id: 'sop-calamari',
    name: 'Fried Pickled Calamari',
    difficulty: 'easy',
    estimatedTime: 15,
    materials: [
      {
        name: 'Pickled Calamari',
        quantity: '1 cup',
        location: 'Top shelf fridge #1',
        pullTrigger: '<2 jars',
        prepSopId: 'sop-pickled-calamari',
        prepTaskTitle: 'Make Pickled Calamari',
      },
      {
        name: 'Flour',
        quantity: '1/2 cup',
        location: 'Baking shelf',
        pullTrigger: '<4 bags triggers Order task',
        lowStock: false,
      },
      {
        name: 'Whole Milk',
        quantity: '1/4 cup',
        location: 'Fridge #2 middle shelf',
        pullTrigger: '<3 gallons',
      },
      {
        name: 'Egg',
        quantity: '1',
        location: 'Fridge #2 bottom shelf',
        pullTrigger: '<6 dozen',
      },
      {
        name: 'Panko Bread Crumbs',
        quantity: '1/2 cup',
        location: 'Baking shelf',
        pullTrigger: '<3 canisters',
      },
      {
        name: 'Fry Basket & Paper',
        quantity: '1 set',
        location: 'Underneath plating table',
      },
      {
        name: 'Stainless Steel Bowls',
        quantity: '3',
        location: 'Kitchen station',
      },
    ],
    procedure: [
      {
        id: 'step-1',
        description: 'Toss calamari in flour',
        completed: false,
        inProgress: false,
      },
      {
        id: 'step-2',
        description: 'Whisk egg and milk in second bowl',
        completed: false,
        inProgress: false,
      },
      {
        id: 'step-3',
        description: 'Coat calamari in egg mixture',
        completed: false,
        inProgress: false,
      },
      {
        id: 'step-4',
        description: 'Toss in panko crumbs',
        completed: false,
        inProgress: false,
      },
      {
        id: 'step-5',
        description: 'Fry for 5 minutes until golden',
        completed: false,
        inProgress: false,
        details: {
          images: ['https://images.unsplash.com/photo-1559477551-ee9f57f6e1d4?w=400'],
        },
      },
      {
        id: 'step-6',
        description: 'Prepare dipping sauce',
        completed: false,
        inProgress: false,
      },
      {
        id: 'step-7',
        description: 'Plate on fry paper',
        completed: false,
        inProgress: false,
      },
    ],
  },
  'sop-salmon-nigiri': {
    id: 'sop-salmon-nigiri',
    name: 'Fresh Salmon Nigiri',
    difficulty: 'medium',
    estimatedTime: 10,
    materials: [
      {
        name: 'Fresh Salmon',
        quantity: '2 oz',
        location: 'Sashimi fridge',
        lowStock: false,
      },
      {
        name: 'Sushi Rice',
        quantity: '2 portions',
        location: 'Rice warmer',
      },
      {
        name: 'Wasabi',
        quantity: '1 tsp',
        location: 'Condiment station',
      },
    ],
    procedure: [
      {
        id: 'step-1',
        description: 'Slice salmon at 45° angle into 2 pieces',
        completed: false,
        inProgress: false,
      },
      {
        id: 'step-2',
        description: 'Form rice into oblong shapes with wet hands',
        completed: false,
        inProgress: false,
      },
      {
        id: 'step-3',
        description: 'Add small dab of wasabi to rice',
        completed: false,
        inProgress: false,
      },
      {
        id: 'step-4',
        description: 'Place salmon on rice and gently press',
        completed: false,
        inProgress: false,
      },
      {
        id: 'step-5',
        description: 'Plate and garnish',
        completed: false,
        inProgress: false,
      },
    ],
  },
  'sop-tiger-roll': {
    id: 'sop-tiger-roll',
    name: 'Tiger Roll',
    difficulty: 'hard',
    estimatedTime: 20,
    materials: [
      {
        name: 'Shrimp Tempura',
        quantity: '4 pieces',
        location: 'Hot station',
      },
      {
        name: 'Sushi Rice',
        quantity: '1.5 cups',
        location: 'Rice warmer',
      },
      {
        name: 'Nori Sheet',
        quantity: '1',
        location: 'Dry storage',
      },
      {
        name: 'Avocado',
        quantity: '1/2',
        location: 'Prep station',
      },
      {
        name: 'Spicy Mayo',
        quantity: '2 tbsp',
        location: 'Sauce station',
      },
      {
        name: 'Eel Sauce',
        quantity: '1 tbsp',
        location: 'Sauce station',
      },
    ],
    procedure: [
      {
        id: 'step-1',
        description: 'Prepare shrimp tempura',
        completed: false,
        inProgress: false,
      },
      {
        id: 'step-2',
        description: 'Lay nori sheet on bamboo mat',
        completed: false,
        inProgress: false,
      },
      {
        id: 'step-3',
        description: 'Spread rice evenly on nori',
        completed: false,
        inProgress: false,
      },
      {
        id: 'step-4',
        description: 'Add shrimp and avocado',
        completed: false,
        inProgress: false,
      },
      {
        id: 'step-5',
        description: 'Roll tightly using bamboo mat',
        completed: false,
        inProgress: false,
      },
      {
        id: 'step-6',
        description: 'Slice into 8 pieces',
        completed: false,
        inProgress: false,
      },
      {
        id: 'step-7',
        description: 'Drizzle with spicy mayo and eel sauce',
        completed: false,
        inProgress: false,
      },
    ],
  },
  'sop-spicy-tuna': {
    id: 'sop-spicy-tuna',
    name: 'Spicy Tuna Roll',
    difficulty: 'medium',
    estimatedTime: 15,
    materials: [
      {
        name: 'Fresh Tuna',
        quantity: '3 oz',
        location: 'Sashimi fridge',
      },
      {
        name: 'Sriracha Mayo',
        quantity: '1 tbsp',
        location: 'Sauce station',
      },
      {
        name: 'Sushi Rice',
        quantity: '1 cup',
        location: 'Rice warmer',
      },
      {
        name: 'Nori Sheet',
        quantity: '1',
        location: 'Dry storage',
      },
      {
        name: 'Cucumber',
        quantity: '2 strips',
        location: 'Prep station',
      },
    ],
    procedure: [
      {
        id: 'step-1',
        description: 'Dice tuna into small cubes',
        completed: false,
        inProgress: false,
      },
      {
        id: 'step-2',
        description: 'Mix tuna with sriracha mayo',
        completed: false,
        inProgress: false,
      },
      {
        id: 'step-3',
        description: 'Lay nori and spread rice',
        completed: false,
        inProgress: false,
      },
      {
        id: 'step-4',
        description: 'Add spicy tuna and cucumber in center',
        completed: false,
        inProgress: false,
      },
      {
        id: 'step-5',
        description: 'Roll and slice into 6-8 pieces',
        completed: false,
        inProgress: false,
      },
    ],
  },
};

export const mockTasks: Task[] = [
  {
    id: 'task-1',
    title: 'Table 7 - Appetizer',
    status: 'todo',
    sopId: 'sop-calamari',
    syncToMaster: false,
    assignedTo: 'Sous Chef',
    tags: ['order', 'appetizer'],
    createdAt: Date.now() - 300000,
    tableNumber: '7',
    orderItems: ['Fried Pickled Calamari'],
  },
  {
    id: 'task-2',
    title: 'Table 7 - Main Order',
    status: 'todo',
    sopId: 'sop-salmon-nigiri',
    syncToMaster: false,
    assignedTo: 'Master Chef',
    tags: ['order', 'special'],
    createdAt: Date.now() - 240000,
    tableNumber: '7',
    orderItems: ['Fresh Salmon Nigiri (1)', 'Tiger Roll (2)', 'Spicy Tuna Roll (1)'],
  },
  {
    id: 'task-3',
    title: 'Daily Prep - Spicy Tuna Mix',
    status: 'todo',
    sopId: 'sop-spicy-tuna',
    syncToMaster: false,
    assignedTo: 'Line Chef',
    tags: ['prep'],
    createdAt: Date.now() - 180000,
  },
];