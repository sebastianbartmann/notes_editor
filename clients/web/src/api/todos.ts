import { apiRequest } from './client'
import type { SaveResponse, AddTodoRequest, ToggleTodoRequest } from './types'

export async function addTodo(data: AddTodoRequest): Promise<SaveResponse> {
  return apiRequest<SaveResponse>('/api/todos/add', {
    method: 'POST',
    body: data,
  })
}

export async function toggleTodo(data: ToggleTodoRequest): Promise<SaveResponse> {
  return apiRequest<SaveResponse>('/api/todos/toggle', {
    method: 'POST',
    body: data,
  })
}
