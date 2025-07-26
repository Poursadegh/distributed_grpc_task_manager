import { Controller, Get, Post, Body, Param, Delete, Put } from '@nestjs/common';
import { ApiTags, ApiOperation, ApiResponse } from '@nestjs/swagger';
import { TaskService } from './task.service';
import { CreateTaskDto, UpdateTaskDto, TaskResponseDto } from './dto/task.dto';

@ApiTags('tasks')
@Controller('tasks')
export class TaskController {
  constructor(private readonly taskService: TaskService) {}

  @Post()
  @ApiOperation({ summary: 'Submit a new task' })
  @ApiResponse({ status: 201, description: 'Task created successfully' })
  async createTask(@Body() createTaskDto: CreateTaskDto): Promise<TaskResponseDto> {
    return this.taskService.createTask(createTaskDto);
  }

  @Get()
  @ApiOperation({ summary: 'Get all tasks' })
  @ApiResponse({ status: 200, description: 'Tasks retrieved successfully' })
  async getAllTasks(): Promise<TaskResponseDto[]> {
    return this.taskService.getAllTasks();
  }

  @Get(':id')
  @ApiOperation({ summary: 'Get task by ID' })
  @ApiResponse({ status: 200, description: 'Task retrieved successfully' })
  async getTask(@Param('id') id: string): Promise<TaskResponseDto> {
    return this.taskService.getTask(id);
  }

  @Get('status/:status')
  @ApiOperation({ summary: 'Get tasks by status' })
  @ApiResponse({ status: 200, description: 'Tasks retrieved successfully' })
  async getTasksByStatus(@Param('status') status: string): Promise<TaskResponseDto[]> {
    return this.taskService.getTasksByStatus(status);
  }

  @Put(':id')
  @ApiOperation({ summary: 'Update task' })
  @ApiResponse({ status: 200, description: 'Task updated successfully' })
  async updateTask(@Param('id') id: string, @Body() updateTaskDto: UpdateTaskDto): Promise<TaskResponseDto> {
    return this.taskService.updateTask(id, updateTaskDto);
  }

  @Delete(':id')
  @ApiOperation({ summary: 'Delete task' })
  @ApiResponse({ status: 200, description: 'Task deleted successfully' })
  async deleteTask(@Param('id') id: string): Promise<{ message: string }> {
    return this.taskService.deleteTask(id);
  }

  @Post('dependencies')
  @ApiOperation({ summary: 'Submit task with dependencies' })
  @ApiResponse({ status: 201, description: 'Task with dependencies created successfully' })
  async createTaskWithDependencies(@Body() createTaskDto: CreateTaskDto): Promise<TaskResponseDto> {
    return this.taskService.createTaskWithDependencies(createTaskDto);
  }

  @Post('timeout')
  @ApiOperation({ summary: 'Submit task with timeout' })
  @ApiResponse({ status: 201, description: 'Task with timeout created successfully' })
  async createTaskWithTimeout(@Body() createTaskDto: CreateTaskDto): Promise<TaskResponseDto> {
    return this.taskService.createTaskWithTimeout(createTaskDto);
  }
} 