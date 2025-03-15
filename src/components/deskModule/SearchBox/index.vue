<script setup lang="ts">
import { defineEmits, onMounted, ref, watch } from 'vue'
import { NAvatar, NCheckbox } from 'naive-ui'
import { SvgIcon } from '@/components/common'
import { useModuleConfig } from '@/store/modules'
import { useAuthStore } from '@/store'
import { VisitMode } from '@/enums/auth'

import SvgSrcBaidu from '@/assets/search_engine_svg/baidu.svg'
import SvgSrcBing from '@/assets/search_engine_svg/bing.svg'
import SvgSrcGoogle from '@/assets/search_engine_svg/google.svg'
import SvgSrcMetaso from '@/assets/search_engine_svg/metaso.ico'
import SvgSrcBilibili from '@/assets/search_engine_svg/bilibili.ico'
import SvgSrcZhihu from '@/assets/search_engine_svg/zhihu.png'

withDefaults(defineProps<{
  background?: string
  textColor?: string
}>(), {
  background: '#2a2a2a6b',
  textColor: 'white',
})

const emits = defineEmits(['itemSearch'])

interface State {
  currentSearchEngine: DeskModule.SearchBox.SearchEngine
  searchEngineList: DeskModule.SearchBox.SearchEngine[]
  newWindowOpen: boolean
}

const moduleConfigName = 'deskModuleSearchBox'
const moduleConfig = useModuleConfig()
const authStore = useAuthStore()
const searchTerm = ref('')
const isFocused = ref(false)
const searchSelectListShow = ref(false)
const defaultSearchEngineList = ref<DeskModule.SearchBox.SearchEngine[]>([
  {
    iconSrc: SvgSrcBaidu,
    title: 'Baidu',
    url: 'https://www.baidu.com/s?wd=%s',
  },
  {
    iconSrc: SvgSrcGoogle,
    title: 'Google',
    url: 'https://www.google.com/search?q=%s',
  },
  {
    iconSrc: SvgSrcBing,
    title: 'Bing',
    url: 'https://www.bing.com/search?q=%s',
  },
  {
    iconSrc: SvgSrcMetaso,
    title: 'Metaso',
    url: 'https://metaso.cn/search?q=%s',
  },
  {
    iconSrc: SvgSrcBilibili,
    title: 'Bilibili',
    url: 'https://search.bilibili.com/all?keyword=%s',
  },
  {
    iconSrc: SvgSrcZhihu,
    title: 'Zhihu',
    url: 'https://www.zhihu.com/search?type=content&q=%s',
  },
])

const defaultState: State = {
  currentSearchEngine: defaultSearchEngineList.value[0],
  searchEngineList: [] || defaultSearchEngineList,
  newWindowOpen: false,
}

const state = ref<State>({ ...defaultState })

const searchSuggestions = ref<string[]>([])
const showSuggestions = ref(false)
const suggestionTimeout = ref<number | null>(null)
const activeIndex = ref(-1) // 当前选中建议的索引
// 保存用户原始输入的内容，避免方向键选择时覆盖
const originalTerm = ref('')
// 标记搜索词变化是否由方向键导致
const isArrowNavigation = ref(false)

const onFocus = (): void => {
  isFocused.value = true
  if (searchTerm.value)
    fetchSuggestions(searchTerm.value)
}

const onBlur = (): void => {
  // 添加短暂延时，以便用户可以点击搜索建议
  setTimeout(() => {
    isFocused.value = false
  }, 200)
}

// 处理键盘导航
function handleKeyDown(e: KeyboardEvent) {
  if (!showSuggestions.value || !searchSuggestions.value.length) 
    return

  // 按下方向键：下一个建议
  if (e.key === 'ArrowDown') {
    e.preventDefault() // 防止光标移动到文本末尾
    
    // 保存原始输入内容（仅在首次按下方向键时）
    if (activeIndex.value === -1) 
      originalTerm.value = searchTerm.value
    
    // 标记为方向键导航
    isArrowNavigation.value = true
    
    activeIndex.value = (activeIndex.value + 1) % searchSuggestions.value.length
    // 只在UI中显示当前选中项，不触发搜索建议更新
    searchTerm.value = searchSuggestions.value[activeIndex.value]
    
    // 重置标记，为下一次输入做准备
    setTimeout(() => {
      isArrowNavigation.value = false
    }, 50)
  }
  // 按上方向键：上一个建议
  else if (e.key === 'ArrowUp') {
    e.preventDefault() // 防止光标移动到文本开始
    
    // 保存原始输入内容（仅在首次按下方向键时）
    if (activeIndex.value === -1) 
      originalTerm.value = searchTerm.value
    
    // 标记为方向键导航
    isArrowNavigation.value = true
    
    activeIndex.value = activeIndex.value <= 0 
      ? searchSuggestions.value.length - 1 
      : activeIndex.value - 1
    // 只在UI中显示当前选中项，不触发搜索建议更新
    searchTerm.value = searchSuggestions.value[activeIndex.value]
    
    // 重置标记，为下一次输入做准备
    setTimeout(() => {
      isArrowNavigation.value = false
    }, 50)
  }
  // 按Escape键：恢复原始输入
  else if (e.key === 'Escape') {
    if (originalTerm.value && activeIndex.value !== -1) {
      searchTerm.value = originalTerm.value
      originalTerm.value = ''
      activeIndex.value = -1
      e.stopPropagation() // 阻止冒泡，避免触发外层的Escape处理
    }
  }
  // 按Enter键：直接执行搜索
  else if (e.key === 'Enter') {
    // 已经由外层事件处理，这里无需额外处理
  }
  // 任何其他按键：重置为原始状态，准备接受新输入
  else if (e.key.length === 1 && activeIndex.value !== -1) {
    originalTerm.value = ''
    activeIndex.value = -1
    // 不需要重置searchTerm，因为用户正在输入新内容
  }
}

function handleEngineClick() {
  // 访客模式不允许修改
  if (authStore.visitMode === VisitMode.VISIT_MODE_PUBLIC)
    return
  searchSelectListShow.value = !searchSelectListShow.value
}

function handleEngineUpdate(engine: DeskModule.SearchBox.SearchEngine) {
  state.value.currentSearchEngine = engine
  moduleConfig.saveToCloud(moduleConfigName, state.value)
  searchSelectListShow.value = false
}

function handleSearchClick() {
  const url = state.value.currentSearchEngine.url
  const keyword = searchTerm
  // 如果网址中存在 %s，则直接替换为关键字
  const fullUrl = replaceOrAppendKeywordToUrl(url, keyword.value)
  handleClearSearchTerm()
  if (state.value.newWindowOpen)
    window.open(fullUrl)
  else
    window.location.href = fullUrl
}

function replaceOrAppendKeywordToUrl(url: string, keyword: string) {
  // 如果网址中存在 %s，则直接替换为关键字
  if (url.includes('%s'))
    return url.replace('%s', encodeURIComponent(keyword))

  // 如果网址中不存在 %s，则将关键字追加到末尾
  return url + (keyword ? `${encodeURIComponent(keyword)}` : '')
}

const handleItemSearch = () => {
  emits('itemSearch', searchTerm.value)
}

function handleClearSearchTerm() {
  searchTerm.value = ''
  searchSuggestions.value = []
  showSuggestions.value = false
  emits('itemSearch', searchTerm.value)
}

// 将选择建议函数修改为填充搜索框，增加isMouseClick参数
function selectSuggestion(suggestion: string, isMouseClick: boolean = false) {
  searchTerm.value = suggestion
  
  // 如果是鼠标点击，则直接执行搜索
  if (isMouseClick) {
    handleSearchClick()
  } else {
    // 否则只执行搜索但保持下拉框打开，直到用户离开搜索框
    handleItemSearch()
  }
}

// 获取搜索建议
async function fetchSuggestions(query: string) {
  if (!query) {
    searchSuggestions.value = []
    showSuggestions.value = false
    activeIndex.value = -1 // 重置选中索引
    return
  }

  // 清除之前的超时
  if (suggestionTimeout.value)
    clearTimeout(suggestionTimeout.value)

  // 设置延迟，避免频繁请求
  suggestionTimeout.value = setTimeout(async () => {
    try {
      // 只使用百度搜索建议
      const suggestions = await fetchBaiduSuggestions(query)
      searchSuggestions.value = suggestions
      showSuggestions.value = suggestions.length > 0
      activeIndex.value = -1 // 重置选中索引
    }
    catch (error) {
      console.error('Failed to fetch suggestions:', error)
      // 如果API调用失败，回退到空数组
      searchSuggestions.value = []
      showSuggestions.value = false
    }
  }, 300) as unknown as number
}

// 通过JSONP方式获取百度搜索建议
function fetchBaiduSuggestions(query: string): Promise<string[]> {
  return new Promise((resolve) => {
    const callbackName = `baidu_suggestion_${Date.now()}`
    
    // 修复类型错误：正确定义全局回调函数
    // 使用类型断言解决window对象上动态添加属性的类型问题
    ;(window as any)[callbackName] = (data: any) => {
      const suggestions = data?.s || []
      resolve(suggestions)
      document.body.removeChild(script)
      // 同样使用类型断言移除属性
      delete (window as any)[callbackName]
    }
    
    // 创建script标签发送JSONP请求
    const script = document.createElement('script')
    script.src = `https://suggestion.baidu.com/su?wd=${encodeURIComponent(query)}&cb=${callbackName}`
    document.body.appendChild(script)
  })
}

// 监听搜索词变化
watch(searchTerm, (newValue, oldValue) => {
  // 只要不是由方向键导致的变化，都重新获取建议
  if (!isArrowNavigation.value) {
    if (newValue) {
      fetchSuggestions(newValue)
    }
    else {
      searchSuggestions.value = []
      showSuggestions.value = false
    }
  }
})

onMounted(() => {
  moduleConfig.getValueByNameFromCloud<State>('deskModuleSearchBox').then(({ code, data }) => {
    if (code === 0)
      state.value = data || defaultState
    else
      state.value = defaultState
  })
})
</script>

<template>
  <div class="search-box w-full relative" 
       @keydown.enter="handleSearchClick" 
       @keydown.esc="handleClearSearchTerm"
       @keydown="handleKeyDown">
    <div class="search-container flex rounded-2xl items-center justify-center text-white w-full" :style="{ background, color: textColor }" :class="{ focused: isFocused }">
      <div class="search-box-btn-engine w-[40px] flex justify-center cursor-pointer" @click="handleEngineClick">
        <NAvatar :src="state.currentSearchEngine.iconSrc" style="background-color: transparent;" :size="20" />
      </div>

      <input v-model="searchTerm" :placeholder="$t('deskModule.searchBox.inputPlaceholder')" @focus="onFocus" @blur="onBlur" @input="handleItemSearch">

      <div v-if="searchTerm !== ''" class="search-box-btn-clear w-[25px] mr-[10px] flex justify-center cursor-pointer" @click="handleClearSearchTerm">
        <SvgIcon style="width: 20px;height: 20px;" icon="line-md:close-small" />
      </div>
      <div class="search-box-btn-search w-[25px] flex justify-center cursor-pointer" @click="handleSearchClick">
        <SvgIcon style="width: 20px;height: 20px;" icon="iconamoon:search-fill" />
      </div>
    </div>

    <!-- 搜索建议 - 使用与搜索引擎选择相同的样式 -->
    <div v-if="showSuggestions && isFocused && !searchSelectListShow" class="w-full mt-[10px] rounded-xl p-[10px]" :style="{ background }">
      <div 
        v-for="(suggestion, index) in searchSuggestions" 
        :key="index"
        class="suggestion-item p-[8px] rounded-lg cursor-pointer flex items-center"
        :class="{'active-suggestion': index === activeIndex, 'hover:bg-opacity-20 hover:bg-white': index !== activeIndex}"
        @click="selectSuggestion(suggestion, true)"
        @mouseover="activeIndex = index"
      >
        <SvgIcon class="mr-[8px]" style="width: 16px;height: 16px;" icon="iconamoon:search" />
        <span :style="{ color: textColor }">{{ suggestion }}</span>
      </div>
    </div>

    <!-- 搜索引擎选择 -->
    <div v-if="searchSelectListShow" class="w-full mt-[10px] rounded-xl p-[10px]" :style="{ background }">
      <div class="flex items-center">
        <div class="flex items-center">
          <div
            v-for="item, index in defaultSearchEngineList"
            :key="index"
            :title="item.title"
            class="w-[40px] h-[40px] mr-[10px]  cursor-pointer bg-[#ffffff] flex items-center justify-center rounded-xl"
            @click="handleEngineUpdate(item)"
          >
            <NAvatar :src="item.iconSrc" style="background-color: transparent;" :size="20" />
          </div>
          <div class="w-[40px] h-[40px] ml-[10px] flex justify-center items-center cursor-pointer" @click="handleEngineClick">
            <NAvatar style="background-color: transparent;" :size="30">
              <SvgIcon icon="lets-icons:setting-alt-fill" style="font-size: 20px;" />
            </NAvatar>
          </div>
        </div>
      </div>

      <div class="mt-[10px]">
        <NCheckbox v-model:checked="state.newWindowOpen" @update-checked="moduleConfig.saveToCloud(moduleConfigName, state)">
          <span :style="{ color: textColor }">
            {{ $t('deskModule.searchBox.openWithNewOpen') }}
          </span>
        </NCheckbox>
      </div>
    </div>
  </div>
</template>

<style scoped>
.search-container {
  border: 1px solid #ccc;
  transition: box-shadow 0.5s,backdrop-filter 0.5s;
  padding: 2px 10px;
  backdrop-filter:blur(2px)
}

.focused, .search-container:hover {
  box-shadow: 0px 0px 30px -5px rgba(41, 41, 41, 0.45);
  -webkit-box-shadow: 0px 0px 30px -5px rgba(0, 0, 0, 0.45);
  -moz-box-shadow: 0px 0px 30px -5px rgba(0, 0, 0, 0.45);
  backdrop-filter:blur(5px)
}

.before {
  left: 10px;
}

.after {
  right: 10px;
}

input {
  background-color: transparent;
  box-sizing: border-box;
  width: 100%;
  height: 40px;
  padding: 10px 5px;
  border: none;
  outline: none;
  font-size: 17px;
}

.suggestion-item {
  transition: background-color 0.2s;
}

.active-suggestion {
  background-color: rgba(255, 255, 255, 0.3); /* 增强高亮效果 */
  border-radius: 8px;
  box-shadow: 0 0 3px rgba(255, 255, 255, 0.5); /* 添加轻微阴影增强效果 */
}
</style>
