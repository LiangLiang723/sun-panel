<script setup lang="ts">
import { NButton, NButtonGroup, NCard, NEllipsis, NGrid, NGridItem, NImage, NImageGroup, NInput, NRadioButton, NRadioGroup, NSpin, NUpload, useDialog, useMessage } from 'naive-ui'
import type { UploadFileInfo } from 'naive-ui'
import { computed, onMounted, ref } from 'vue'
import { deletes, getList, refreshFiles, rename } from '@/api/system/file'
import { RoundCardModal, SvgIcon } from '@/components/common'
import { copyToClipboard, timeFormat } from '@/utils/cmn'
import { t } from '@/locales'
import { useAuthStore } from '@/store'
import { apiRespErrMsg } from '@/utils/request/apiMessage'

// 定义组件属性
const props = defineProps<{
  visible: boolean
}>()

// 定义事件
const emit = defineEmits<{
  (e: 'update:visible', visible: boolean): void
  (e: 'select', url: string): void
}>()

// 复用UploadFileManager的状态和接口定义
interface InfoModalState {
  title: string
  show: boolean
  fileInfo: File.Info | null
}

interface RenameModalState {
  show: boolean
  fileInfo: File.Info | null
  newFileName: string
  newFileExt: string
}

interface RenameResponseData {
  conflict: boolean;
  message?: string;
  targetPath?: string;
}

interface RenameResponse {
  code: number;
  msg?: string;
  data?: RenameResponseData;
}

interface DeleteResponseData {
  warnings?: string[];
  message?: string;
}

interface DeleteResponse {
  code: number;
  msg?: string;
  data?: DeleteResponseData;
}

// 状态变量
const imageList = ref<File.Info[]>([])
const searchQuery = ref('')
const ms = useMessage()
const dialog = useDialog()
const loading = ref(false)
const activeGroup = ref('all')
const infoModalState = ref<InfoModalState>({
  show: false,
  title: '',
  fileInfo: null,
})
const renameModalState = ref<RenameModalState>({
  show: false,
  fileInfo: null,
  newFileName: '',
  newFileExt: '',
})

// 添加authStore引用
const authStore = useAuthStore()

// 获取不带后缀名的文件名
function getFileNameWithoutExtension(fileName: string): string {
  const lastDotIndex = fileName.lastIndexOf('.');
  if (lastDotIndex > 0) {
    return fileName.substring(0, lastDotIndex);
  }
  return fileName; // 如果没有后缀名，返回原始文件名
}

// 过滤和分组后的图片列表
const groupedImageList = computed(() => {
  if (!searchQuery.value) {
    if (activeGroup.value === 'all') {
      return imageList.value
    } else if (activeGroup.value === 'renamed') {
      return imageList.value.filter(item => item.src.includes('/managed_user'))
    } else if (activeGroup.value === 'original') {
      return imageList.value.filter(item => !item.src.includes('/managed_user'))
    }
  }
  
  // 如果有搜索词，先按搜索词过滤（不考虑后缀名），再按分组过滤
  const searchQueryLower = searchQuery.value.toLowerCase();
  let filteredList = imageList.value.filter(item => {
    // 获取不带后缀的文件名并转为小写
    const fileNameWithoutExt = getFileNameWithoutExtension(item.fileName).toLowerCase();
    
    // 同时检查完整文件名和不带后缀的文件名
    return fileNameWithoutExt.includes(searchQueryLower) || 
           item.fileName.toLowerCase().includes(searchQueryLower);
  });
  
  if (activeGroup.value === 'all') {
    return filteredList
  } else if (activeGroup.value === 'renamed') {
    return filteredList.filter(item => item.src.includes('/managed_user'))
  } else if (activeGroup.value === 'original') {
    return filteredList.filter(item => !item.src.includes('/managed_user'))
  }
  
  return filteredList
})

// 模态框可见性控制
const show = computed({
  get: () => props.visible,
  set: (visible: boolean) => {
    emit('update:visible', visible)
  },
})

// 文件操作功能
async function getFileList() {
  loading.value = true
  try {
    const { data } = await getList<Common.ListResponse<File.Info[]>>()
    imageList.value = data.list
  } catch (error) {
    console.error('获取文件列表失败:', error)
    ms.error('获取文件列表失败')
  } finally {
    loading.value = false
  }
}

async function copyImageUrl(text: string) {
  const res = await copyToClipboard(text)
  if (res)
    ms.success(t('apps.uploadsFileManager.copySuccess'))
  else
    ms.error(t('apps.uploadsFileManager.copyFailed'))
}

function handleDelete(id: number) {
  dialog.warning({
    title: t('common.warning'),
    content: t('apps.uploadsFileManager.deleteWarningText'),
    positiveText: t('common.confirm'),
    negativeText: t('common.cancel'),
    onPositiveClick: () => {
      deletesImges(id)
    },
  })
}

async function deletesImges(id: number) {
  try {
    loading.value = true
    const response = await deletes<DeleteResponse>([id])
    const { code, msg, data } = response
    
    if (code === 0) {
      // 刷新文件列表
      await getFileList()
      
      // 使用更精确的类型检查
      const responseData = data as DeleteResponseData | undefined;
      if (responseData?.warnings && responseData.warnings.length > 0) {
        // 显示警告，但文件已从列表中移除
        ms.warning(t('apps.uploadsFileManager.deletePartialSuccess'))
      } else {
        ms.success(t('common.success'))
      }
    } else {
      ms.error(`${t('common.failed')}: ${msg}`)
    }
  } catch (error) {
    ms.error(t('common.failed'))
  } finally {
    loading.value = false
  }
}

function handleInfoClick(fileInfo: File.Info) {
  infoModalState.value.fileInfo = fileInfo
  infoModalState.value.show = true
}

function handleRenameClick(fileInfo: File.Info) {
  // 分离文件名和扩展名
  const fileName = fileInfo.fileName;
  const lastDotIndex = fileName.lastIndexOf('.');
  
  if (lastDotIndex > 0) {
    renameModalState.value.fileInfo = fileInfo;
    renameModalState.value.newFileName = fileName.substring(0, lastDotIndex);
    renameModalState.value.newFileExt = fileName.substring(lastDotIndex + 1);
  } else {
    renameModalState.value.fileInfo = fileInfo;
    renameModalState.value.newFileName = fileName;
    renameModalState.value.newFileExt = '';
  }
  
  renameModalState.value.show = true;
}

async function submitRename() {
  if (!renameModalState.value.fileInfo || !renameModalState.value.newFileName.trim()) {
    ms.error(t('common.invalidInput'))
    return
  }
  
  // 检查扩展名是否有效
  const fileExt = renameModalState.value.newFileExt.trim();
  if (fileExt && (!fileExt.match(/^[a-zA-Z0-9]+$/) || fileExt.length > 10)) {
    ms.error(t('apps.uploadsFileManager.invalidExtension'))
    return
  }
  
  // 合并文件名和扩展名
  const fullFileName = fileExt ? 
    `${renameModalState.value.newFileName}.${fileExt}` : 
    renameModalState.value.newFileName;
  
  try {
    const response = await rename<RenameResponse>(
      renameModalState.value.fileInfo.id as number, 
      fullFileName
    )
    
    const { code, msg, data } = response
    
    if (code === 0) {
      // 使用更明确的类型检查
      if (data && 'conflict' in data && data.conflict) {
        // 文件名冲突，询问用户是否覆盖
        dialog.warning({
          title: t('common.warning'),
          content: t('apps.uploadsFileManager.fileNameConflict'),
          positiveText: t('apps.uploadsFileManager.overwrite'),
          negativeText: t('common.cancel'),
          onPositiveClick: async () => {
            // 用户选择覆盖，发送强制覆盖请求
            const { code, msg } = await rename<RenameResponse>(
              renameModalState.value.fileInfo!.id as number, 
              fullFileName,
              true // 强制覆盖
            )
            
            if (code === 0) {
              ms.success(t('common.success'))
              renameModalState.value.show = false
              getFileList()
            } else {
              ms.error(`${t('common.failed')}: ${msg}`)
            }
          }
        })
      } else {
        // 操作成功
        ms.success(t('common.success'))
        renameModalState.value.show = false
        getFileList()
      }
    } else {
      ms.error(`${t('common.failed')}: ${msg}`)
    }
  } catch (error) {
    ms.error(t('common.failed'))
  }
}

// 刷新文件列表
async function handleRefreshFiles() {
  loading.value = true
  try {
    const { code, msg } = await refreshFiles()
    if (code === 0) {
      await getFileList() // 重新获取列表
      ms.success(t('apps.uploadsFileManager.refreshSuccess'))
    } else {
      ms.error(`${t('common.failed')}: ${msg}`)
    }
  } catch (error) {
    ms.error(t('common.failed'))
  } finally {
    loading.value = false
  }
}

function clearSearch() {
  searchQuery.value = ''
}

// 切换分组
function handleGroupChange(value: string) {
  activeGroup.value = value
}

// 选择文件 - 这是FileSelector特有的功能
function selectFile(item: File.Info) {
  emit('select', item.src)
  show.value = false
}

// 添加文件上传完成处理函数
const handleUploadFinish = ({
  file,
  event,
}: {
  file: UploadFileInfo
  event?: ProgressEvent
}) => {
  if (!event?.target) return file
  
  try {
    const res = JSON.parse((event.target as XMLHttpRequest).response)
    if (res.code === 0) {
      ms.success(t('common.success'))
      // 上传成功后刷新文件列表
      getFileList()
    }
    else {
      apiRespErrMsg(res)
    }
  } catch (error) {
    ms.error(t('common.uploadFail'))
  }

  return file
}

// 组件挂载时获取文件列表
onMounted(() => {
  getFileList()
})
</script>

<template>
  <RoundCardModal v-model:show="show" style="max-width: 900px;" size="small" :title="$t('iconItem.selectFromUploaded')">
    <div class="bg-slate-200 dark:bg-zinc-900 p-2 h-full file-selector-container">
      <NSpin v-show="loading" size="small" />
      
      <div class="flex flex-wrap justify-between items-center mt-2 mb-3 gap-3">
        <!-- 分组选择器 -->
        <div class="flex gap-2 items-center">
          <NRadioGroup v-model:value="activeGroup" @update:value="handleGroupChange" size="small">
            <NRadioButton value="all">{{ $t('apps.uploadsFileManager.allFiles') }}</NRadioButton>
            <NRadioButton value="original">{{ $t('apps.uploadsFileManager.originalFiles') }}</NRadioButton>
            <NRadioButton value="renamed">{{ $t('apps.uploadsFileManager.renamedFiles') }}</NRadioButton>
          </NRadioGroup>
          
          <!-- 刷新按钮 -->
          <NButton size="small" 
                  tertiary 
                  :title="$t('apps.uploadsFileManager.refreshFiles')" 
                  @click="handleRefreshFiles"
                  :loading="loading">
            <template #icon>
              <SvgIcon icon="mdi-refresh" />
            </template>
          </NButton>
          
          <!-- 添加上传按钮 -->
          <NUpload
            action="/api/file/uploadImg"
            :show-file-list="false"
            name="imgfile"
            :headers="{
              token: authStore.token as string,
            }"
            @finish="handleUploadFinish"
          >
            <NButton size="small" type="primary" :loading="loading">
              {{ $t('iconItem.selectUpload') }}
            </NButton>
          </NUpload>
        </div>
        
        <!-- 搜索功能 -->
        <div class="flex flex-1 max-w-xs">
          <NInput v-model:value="searchQuery" 
                  :placeholder="$t('common.search')" 
                  clearable
                  @clear="clearSearch">
            <template #prefix>
              <SvgIcon icon="ion-search" />
            </template>
          </NInput>
        </div>
      </div>

      <div class="file-list-container">
        <div class="flex justify-center mt-2">
          <div v-if="groupedImageList.length === 0 && !loading" class="flex">
            <template v-if="searchQuery">
              {{ $t('common.noSearchResults') }}
            </template>
            <template v-else-if="activeGroup === 'renamed'">
              {{ $t('apps.uploadsFileManager.noRenamedFiles') }}
            </template>
            <template v-else-if="activeGroup === 'original'">
              {{ $t('apps.uploadsFileManager.noOriginalFiles') }}
            </template>
            <template v-else>
              {{ $t('apps.uploadsFileManager.nothingText') }}
            </template>
          </div>
          <NImageGroup v-else>
            <NGrid cols="2 300:2 600:4 900:6 1100:9" :x-gap="5" :y-gap="5">
              <NGridItem v-for="(item, index) in groupedImageList" :key="index">
                <NCard size="small" style="border-radius: 5px;" :bordered="true" hover-style="cursor: pointer;" @click="selectFile(item)">
                  <template #cover>
                    <div class="card transparent-grid">
                      <NImage :lazy="true" style="object-fit: contain;height: 100%;" :src="item.src" />
                    </div>
                  </template>
                  <template #footer>
                    <span class="text-xs">
                      <NEllipsis>
                        {{ item.fileName }}
                      </NEllipsis>
                    </span>
                    <div class="flex justify-center mt-[10px]">
                      <NButtonGroup>
                        <NButton size="tiny" tertiary style="cursor: pointer;" :title="$t('apps.uploadsFileManager.copyLink')" @click.stop="copyImageUrl(item.src)">
                          <template #icon>
                            <SvgIcon icon="ion-copy" />
                          </template>
                        </NButton>
                        <NButton size="tiny" tertiary style="cursor: pointer;" :title="$t('common.rename')" @click.stop="handleRenameClick(item)">
                          <template #icon>
                            <SvgIcon icon="mdi-pencil-outline" />
                          </template>
                        </NButton>
                        <NButton size="tiny" tertiary style="cursor: pointer;" :title="timeFormat(item.createTime)" @click.stop="handleInfoClick(item)">
                          <template #icon>
                            <SvgIcon icon="mdi-information-box-outline" />
                          </template>
                        </NButton>
                        <NButton size="tiny" tertiary type="error" style="cursor: pointer;" :title="$t('common.delete')" @click.stop="handleDelete(item.id as number)">
                          <template #icon>
                            <SvgIcon icon="material-symbols-delete" />
                          </template>
                        </NButton>
                      </NButtonGroup>
                    </div>
                  </template>
                </NCard>
              </NGridItem>
            </NGrid>
          </NImageGroup>
        </div>
      </div>
    </div>

    <!-- 文件信息模态框 -->
    <RoundCardModal v-model:show="infoModalState.show" style="max-width: 300px;" size="small" :title="$t('apps.uploadsFileManager.infoTitle')">
      <div>
        <div>
          <div class="mb-2">
            <span class="text-slate-500">
              {{ $t('apps.uploadsFileManager.fileName') }}
            </span>
            <div class="text-xs">
              {{ infoModalState.fileInfo?.fileName }}
            </div>
          </div>
          <div class="mb-2">
            <span class="text-slate-500">
              {{ $t('apps.uploadsFileManager.path') }}
            </span>
            <div class="text-xs">
              {{ infoModalState.fileInfo?.src }}
            </div>
          </div>
          <div class="mb-2">
            <span class="text-slate-500">
              {{ $t('apps.uploadsFileManager.uploadTime') }}
            </span>
            <div class="text-xs">
              {{ timeFormat(infoModalState.fileInfo?.createTime) }}
            </div>
          </div>
        </div>
      </div>
    </RoundCardModal>

    <!-- 重命名模态框 -->
    <RoundCardModal v-model:show="renameModalState.show" style="max-width: 350px;" size="small" :title="$t('common.rename')">
      <div>
        <div class="mb-4">
          <label class="block mb-1 text-sm">{{ $t('apps.uploadsFileManager.fileNameLabel') }}</label>
          <NInput v-model:value="renameModalState.newFileName" :placeholder="$t('apps.uploadsFileManager.enterNewFilename')" />
        </div>
        
        <div class="mb-4">
          <label class="block mb-1 text-sm">{{ $t('apps.uploadsFileManager.extensionLabel') }}</label>
          <div class="flex items-center">
            <span class="mr-1">.</span>
            <NInput v-model:value="renameModalState.newFileExt" :placeholder="$t('apps.uploadsFileManager.enterExtension')" />
          </div>
        </div>
        
        <div class="flex justify-end gap-2 mt-4">
          <NButton @click="renameModalState.show = false">{{ $t('common.cancel') }}</NButton>
          <NButton type="primary" @click="submitRename">{{ $t('common.confirm') }}</NButton>
        </div>
      </div>
    </RoundCardModal>
  </RoundCardModal>
</template>

<style scoped>
.card {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 80px;
}

.transparent-grid {
  background-image: linear-gradient(45deg, #f0f0f0 25%, transparent 25%, transparent 75%, #f0f0f0 75%),
    linear-gradient(45deg, #f0f0f0 25%, transparent 25%, transparent 75%, #f0f0f0 75%);
  background-size: 16px 16px;
  background-position: 0 0, 8px 8px;
}

/* 固定高度容器样式 */
.file-selector-container {
  display: flex;
  flex-direction: column;
  height: 500px; /* 固定总高度 */
}

.file-list-container {
  flex: 1;
  overflow-y: auto;
  min-height: 400px; /* 确保有足够空间显示内容 */
}
</style>
