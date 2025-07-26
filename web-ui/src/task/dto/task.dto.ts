import { ApiProperty } from '@nestjs/swagger';

export enum Priority {
  LOW = 'low',
  MEDIUM = 'medium',
  HIGH = 'high',
}

export enum Status {
  PENDING = 'pending',
  RUNNING = 'running',
  COMPLETED = 'completed',
  FAILED = 'failed',
  TIMEOUT = 'timeout',
}

export class CreateTaskDto {
  @ApiProperty({ enum: Priority, description: 'Task priority' })
  priority: Priority;

  @ApiProperty({ description: 'Task payload' })
  payload: any;

  @ApiProperty({ description: 'Task dependencies', required: false })
  dependencies?: string[];

  @ApiProperty({ description: 'Task timeout in seconds', required: false })
  timeout?: number;
}

export class UpdateTaskDto {
  @ApiProperty({ enum: Priority, description: 'Task priority', required: false })
  priority?: Priority;

  @ApiProperty({ description: 'Task payload', required: false })
  payload?: any;

  @ApiProperty({ enum: Status, description: 'Task status', required: false })
  status?: Status;

  @ApiProperty({ description: 'Task dependencies', required: false })
  dependencies?: string[];

  @ApiProperty({ description: 'Task timeout in seconds', required: false })
  timeout?: number;
}

export class TaskResponseDto {
  @ApiProperty({ description: 'Task ID' })
  id: string;

  @ApiProperty({ enum: Priority, description: 'Task priority' })
  priority: Priority;

  @ApiProperty({ description: 'Task payload' })
  payload: any;

  @ApiProperty({ description: 'Task creation timestamp' })
  createdAt: string;

  @ApiProperty({ enum: Status, description: 'Task status' })
  status: Status;

  @ApiProperty({ description: 'Task start timestamp', required: false })
  startedAt?: string;

  @ApiProperty({ description: 'Task completion timestamp', required: false })
  completedAt?: string;

  @ApiProperty({ description: 'Worker ID', required: false })
  workerId?: string;

  @ApiProperty({ description: 'Error message', required: false })
  error?: string;

  @ApiProperty({ description: 'Task dependencies', required: false })
  dependencies?: string[];

  @ApiProperty({ description: 'Task timeout timestamp', required: false })
  timeout?: string;

  @ApiProperty({ description: 'Maximum duration in seconds', required: false })
  maxDuration?: number;
} 