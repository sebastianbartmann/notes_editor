import { apiRequest } from './client'
import type {
  ListFilesResponse,
  ReadFileResponse,
  SaveResponse,
  CreateFileRequest,
  SaveFileRequest,
  DeleteFileRequest,
  UnpinRequest,
} from './types'

export async function listFiles(path: string = '.'): Promise<ListFilesResponse> {
  const encodedPath = encodeURIComponent(path)
  return apiRequest<ListFilesResponse>(`/api/files/list?path=${encodedPath}`)
}

export async function readFile(path: string): Promise<ReadFileResponse> {
  const encodedPath = encodeURIComponent(path)
  return apiRequest<ReadFileResponse>(`/api/files/read?path=${encodedPath}`)
}

export async function createFile(data: CreateFileRequest): Promise<SaveResponse> {
  return apiRequest<SaveResponse>('/api/files/create', {
    method: 'POST',
    body: data,
  })
}

export async function saveFile(data: SaveFileRequest): Promise<SaveResponse> {
  return apiRequest<SaveResponse>('/api/files/save', {
    method: 'POST',
    body: data,
  })
}

export async function deleteFile(data: DeleteFileRequest): Promise<SaveResponse> {
  return apiRequest<SaveResponse>('/api/files/delete', {
    method: 'POST',
    body: data,
  })
}

export async function unpinEntry(data: UnpinRequest): Promise<SaveResponse> {
  return apiRequest<SaveResponse>('/api/files/unpin', {
    method: 'POST',
    body: data,
  })
}
