import { post } from '@/utils/request'

export function getList<T>(group?: string) {
  return post<T>({
    url: '/file/getList',
    data: group ? { group } : undefined
  })
}

export function deletes<T>(ids: number[]) {
  return post<T>({
    url: '/file/deletes',
    data: { ids },
  })
}

/**
 * 重命名文件
 * @param id 文件ID
 * @param newFileName 新文件名
 * @param force 是否强制覆盖已存在的文件
 * @returns 
 */
export function rename<T>(id: number, newFileName: string, force: boolean = false) {
  return post<T>({
    url: '/file/rename',
    data: { id, fileName: newFileName, force }
  })
}

/**
 * 刷新文件列表
 * @returns 
 */
export function refreshFiles<T>() {
  return post<T>({
    url: '/file/refresh'
  })
}
