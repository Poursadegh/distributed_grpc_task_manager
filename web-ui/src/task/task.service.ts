import { Injectable, HttpException, HttpStatus } from '@nestjs/common';
import axios from 'axios';
import { CreateTaskDto, UpdateTaskDto, TaskResponseDto, Priority, Status } from './dto/task.dto';

@Injectable()
export class TaskService {
  private readonly schedulerUrl = process.env.SCHEDULER_URL || 'http://localhost:8080';

  async createTask(createTaskDto: CreateTaskDto): Promise<TaskResponseDto> {
    try {
      const response = await axios.post(`${this.schedulerUrl}/api/v1/tasks`, {
        priority: createTaskDto.priority,
        payload: createTaskDto.payload,
      });
      return response.data.task;
    } catch (error) {
      throw new HttpException(
        `Failed to create task: ${error.response?.data?.error || error.message}`,
        HttpStatus.BAD_REQUEST,
      );
    }
  }

  async createTaskWithDependencies(createTaskDto: CreateTaskDto): Promise<TaskResponseDto> {
    try {
      const response = await axios.post(`${this.schedulerUrl}/api/v1/tasks/dependencies`, {
        priority: createTaskDto.priority,
        payload: createTaskDto.payload,
        dependencies: createTaskDto.dependencies || [],
      });
      return response.data.task;
    } catch (error) {
      throw new HttpException(
        `Failed to create task with dependencies: ${error.response?.data?.error || error.message}`,
        HttpStatus.BAD_REQUEST,
      );
    }
  }

  async createTaskWithTimeout(createTaskDto: CreateTaskDto): Promise<TaskResponseDto> {
    try {
      const response = await axios.post(`${this.schedulerUrl}/api/v1/tasks/timeout`, {
        priority: createTaskDto.priority,
        payload: createTaskDto.payload,
        timeout: createTaskDto.timeout || 30,
      });
      return response.data.task;
    } catch (error) {
      throw new HttpException(
        `Failed to create task with timeout: ${error.response?.data?.error || error.message}`,
        HttpStatus.BAD_REQUEST,
      );
    }
  }

  async getAllTasks(): Promise<TaskResponseDto[]> {
    try {
      const response = await axios.get(`${this.schedulerUrl}/api/v1/tasks`);
      return response.data.tasks || [];
    } catch (error) {
      throw new HttpException(
        `Failed to get tasks: ${error.response?.data?.error || error.message}`,
        HttpStatus.INTERNAL_SERVER_ERROR,
      );
    }
  }

  async getTask(id: string): Promise<TaskResponseDto> {
    try {
      const response = await axios.get(`${this.schedulerUrl}/api/v1/tasks/${id}`);
      return response.data.task;
    } catch (error) {
      throw new HttpException(
        `Failed to get task: ${error.response?.data?.error || error.message}`,
        HttpStatus.NOT_FOUND,
      );
    }
  }

  async getTasksByStatus(status: string): Promise<TaskResponseDto[]> {
    try {
      const response = await axios.get(`${this.schedulerUrl}/api/v1/tasks/status/${status}`);
      return response.data.tasks || [];
    } catch (error) {
      throw new HttpException(
        `Failed to get tasks by status: ${error.response?.data?.error || error.message}`,
        HttpStatus.INTERNAL_SERVER_ERROR,
      );
    }
  }

  async updateTask(id: string, updateTaskDto: UpdateTaskDto): Promise<TaskResponseDto> {
    try {
      const response = await axios.put(`${this.schedulerUrl}/api/v1/tasks/${id}`, updateTaskDto);
      return response.data.task;
    } catch (error) {
      throw new HttpException(
        `Failed to update task: ${error.response?.data?.error || error.message}`,
        HttpStatus.BAD_REQUEST,
      );
    }
  }

  async deleteTask(id: string): Promise<{ message: string }> {
    try {
      await axios.delete(`${this.schedulerUrl}/api/v1/tasks/${id}`);
      return { message: 'Task deleted successfully' };
    } catch (error) {
      throw new HttpException(
        `Failed to delete task: ${error.response?.data?.error || error.message}`,
        HttpStatus.INTERNAL_SERVER_ERROR,
      );
    }
  }
} 