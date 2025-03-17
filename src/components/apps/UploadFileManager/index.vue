<script setup lang="ts">
import { NButton, NButtonGroup, NCard, NEllipsis, NGrid, NGridItem, NImage, NImageGroup, NInput, NRadioButton, NRadioGroup, NSpin, useDialog, useMessage } from 'naive-ui'
import { computed, onMounted, ref } from 'vue'
import { deletes, getList, refreshFiles, rename } from '@/api/system/file'
import { set as savePanelConfig } from '@/api/panel/userConfig'
import { RoundCardModal, SvgIcon } from '@/components/common'
import { copyToClipboard, timeFormat } from '@/utils/cmn'
import { t } from '@/locales'
import { usePanelState } from '@/store'

interface InfoModalState {
  title: string
  show: boolean
  fileInfo: File.Info | null
}

interface RenameModalState {
  show: boolean
  fileInfo: File.Info | null
  newFileName: string
}

// 定义文件重命名响应数据接口
interface RenameResponseData {
  conflict: boolean;
  message?: string;
  targetPath?: string;
}

// 定义文件重命名响应接口
interface RenameResponse {
  code: number;
  msg?: string;
  data?: RenameResponseData;
}

const imageList = ref<File.Info[]>([])
const searchQuery = ref('')
const ms = useMessage()
const dialog = useDialog()
const panelStore = usePanelState()
const loading = ref(false)
const activeGroup = ref('all') // 当前激活的分组
const infoModalState = ref<InfoModalState>({
  show: false,
  title: '',
  fileInfo: null,
})
const renameModalState = ref<RenameModalState>({
  show: false,
  fileInfo: null,
  newFileName: '',
})

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
  
  // 如果有搜索词，先按搜索词过滤，再按分组过滤
  let filteredList = imageList.value.filter(item => 
    item.fileName.toLowerCase().includes(searchQuery.value.toLowerCase())
  )
  
  if (activeGroup.value === 'all') {
    return filteredList
  } else if (activeGroup.value === 'renamed') {
    return filteredList.filter(item => item.src.includes('/managed_user'))
  } else if (activeGroup.value === 'original') {
    return filteredList.filter(item => !item.src.includes('/managed_user'))
  }
  
  return filteredList
})

async function getFileList() {
  loading.value = true
  const { data } = await getList<Common.ListResponse<File.Info[]>>()
  imageList.value = data.list
  loading.value = false
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
    const { code, msg } = await deletes([id])
    if (code === 0) {
      getFileList()
      ms.success(t('common.success'))
    }
    else {
      ms.error(`${t('common.failed')}:${msg}`)
    }
  }
  catch (error) {
    ms.error(t('common.failed'))
  }
}

function handleInfoClick(fileInfo: File.Info) {
  infoModalState.value.fileInfo = fileInfo
  infoModalState.value.show = true
}

function handleSetWallpaper(imgSrc: string) {
  panelStore.panelConfig.backgroundImageSrc = imgSrc
  savePanelConfig({ panel: panelStore.panelConfig })
}

function handleRenameClick(fileInfo: File.Info) {
  renameModalState.value.fileInfo = fileInfo
  renameModalState.value.newFileName = fileInfo.fileName
  renameModalState.value.show = true
}

async function submitRename() {
  if (!renameModalState.value.fileInfo || !renameModalState.value.newFileName.trim()) {
    ms.error(t('common.invalidInput'))
    return
  }
  
  try {
    const response = await rename<RenameResponse>(
      renameModalState.value.fileInfo.id as number, 
      renameModalState.value.newFileName
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
              renameModalState.value.newFileName,
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

onMounted(() => {
  getFileList()
})
</script>

<template>
  <div class="bg-slate-200 dark:bg-zinc-900 p-2 h-full">
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
            <NCard size="small" style="border-radius: 5px;" :bordered="true">
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
                    <NButton size="tiny" tertiary style="cursor: pointer;" :title="$t('apps.uploadsFileManager.copyLink')" @click="copyImageUrl(item.src)">
                      <template #icon>
                        <SvgIcon icon="ion-copy" />
                      </template>
                    </NButton>
                    <NButton size="tiny" tertiary style="cursor: pointer;" :title="$t('common.rename')" @click="handleRenameClick(item)">
                      <template #icon>
                        <SvgIcon icon="mdi-pencil-outline" />
                      </template>
                    </NButton>
                    <NButton size="tiny" tertiary style="cursor: pointer;" :title="timeFormat(item.createTime)" @click="handleInfoClick(item)">
                      <template #icon>
                        <SvgIcon icon="mdi-information-box-outline" />
                      </template>
                    </NButton>
                    <NButton size="tiny" tertiary style="cursor: pointer;" :title="$t('apps.uploadsFileManager.setWallpaper')" @click="handleSetWallpaper(item.src)">
                      <template #icon>
                        <SvgIcon icon="lucide:wallpaper" />
                      </template>
                    </NButton>
                    <NButton size="tiny" tertiary type="error" style="cursor: pointer;" :title="$t('common.delete')" @click="handleDelete(item.id as number)">
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
        <NInput v-model:value="renameModalState.newFileName" :placeholder="$t('apps.uploadsFileManager.enterNewFilename')" />
        <div class="flex justify-end gap-2 mt-4">
          <NButton @click="renameModalState.show = false">{{ $t('common.cancel') }}</NButton>
          <NButton type="primary" @click="submitRename">{{ $t('common.confirm') }}</NButton>
        </div>
      </div>
    </RoundCardModal>
  </div>
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
</style>
