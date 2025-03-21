<script setup lang="ts">
import { computed } from 'vue'
import { NEllipsis } from 'naive-ui'
import { ItemIcon } from '@/components/common'
import { PanelPanelConfigStyleEnum } from '@/enums'

interface Prop {
  itemInfo?: Panel.ItemInfo
  size?: number // 默认70
  forceBackground?: string // 强制背景色
  iconTextColor?: string
  iconTextInfoHideDescription: boolean
  iconTextIconHideTitle: boolean
  style: PanelPanelConfigStyleEnum
}

const props = withDefaults(defineProps<Prop>(), {
  size: 70,
})

const defaultBackground = '#2a2a2a6b'

const calculateLuminance = (color: string) => {
  const hex = color.replace(/^#/, '')
  const r = parseInt(hex.substring(0, 2), 16)
  const g = parseInt(hex.substring(2, 4), 16)
  const b = parseInt(hex.substring(4, 6), 16)
  return (0.299 * r + 0.587 * g + 0.114 * b) / 255
}

const textColor = computed(() => {
  const luminance = calculateLuminance(props.itemInfo?.icon?.backgroundColor || defaultBackground)
  return luminance > 0.5 ? 'black' : 'white'
})

// 从描述中提取图标大小
const iconSize = computed(() => {
  const description = props.itemInfo?.description || ''
  const match = description.match(/##(\d+)px##/)
  return match ? parseInt(match[1]) : 50
})

// 从描述中提取图标位置偏移值
// 格式为 ##x,y## 如 ##-1,1## 表示水平向左偏移1，垂直向下偏移1
// x为水平偏移，负值向左，正值向右
// y为垂直偏移，负值向上，正值向下
const iconOffset = computed(() => {
  const description = props.itemInfo?.description || ''
  const match = description.match(/##([-\d\.]+),([-\d\.]+)##/)
  if (!match) return { x: 0, y: 0 } // 默认无偏移
  
  return {
    x: parseFloat(match[1]),
    y: parseFloat(match[2])
  }
})

// 清理描述中的标记
const cleanDescription = computed(() => {
  const description = props.itemInfo?.description || ''
  // 移除大小标记和偏移标记
  return description.replace(/##\d+px##/g, '').replace(/##[-\d\.]+,[-\d\.]+##/g, '')
})

// 生成图标位置样式
const getIconPositionStyle = computed(() => {
  const offset = iconOffset.value
  return { 
    // 基于居中位置进行偏移
    // transform用来移动图标，以图标大小作为基准进行百分比偏移
    transform: `translate(${offset.x * 50}%, ${offset.y * 50}%)`,
    // 保持居中对齐作为基准点
    justifyContent: 'center',
    alignItems: 'center'
  }
})
</script>

<template>
  <div class="app-icon w-full">
    <!-- 详情图标 -->
    <div v-if="style === PanelPanelConfigStyleEnum.info"
      class="app-icon-info w-full rounded-2xl  transition-all duration-200 hover:shadow-[0_0_20px_10px_rgba(0,0,0,0.2)] flex"
      :style="{ background: itemInfo?.icon?.backgroundColor || defaultBackground }">
      <!-- 图标 -->
      <div class="app-icon-info-icon w-[70px] h-[70px]">
        <div class="w-[70px] h-full flex items-center justify-center">
          <ItemIcon :item-icon="itemInfo?.icon" force-background="transparent" :size="iconSize"
            class="overflow-hidden rounded-xl"
            :style="getIconPositionStyle" />
        </div>
      </div>

      <!-- 文字 -->
      <!-- 如果为纯白色，将自动根据背景的明暗计算字体的黑白色 -->
      <div class="text-white flex items-center"
        :style="{ color: (iconTextColor === '#ffffff') ? textColor : iconTextColor, maxWidth: 'calc(100% - 80px)', flex: 1, position: 'relative' }">
        <transition name="fade">

          <div class="badge" v-if="itemInfo?.time">{{ itemInfo?.time }}</div> <!-- 这里的数字表示未读数 -->
        </transition>
        <div class="app-icon-info-text-box w-full">
          <div class="app-icon-info-text-box-title font-semibold w-full">
            <NEllipsis>
              {{ itemInfo?.title }}
            </NEllipsis>
          </div>
          <div v-if="!iconTextInfoHideDescription" class="app-icon-info-text-box-description">
            <NEllipsis :line-clamp="2" class="text-xs">
              {{ cleanDescription }}
            </NEllipsis>
          </div>
        </div>
      </div>
    </div>

    <!-- 极简(小)图标（APP） -->
    <div v-if="style === PanelPanelConfigStyleEnum.icon" class="app-icon-small" style="position: relative;">
      <transition name="fade">

        <div class="badge" v-if="itemInfo?.time" style="right: 12px;">{{ itemInfo?.time }}</div> <!-- 这里的数字表示未读数 -->
      </transition>

      <div
        class="app-icon-small-icon overflow-hidden rounded-2xl sunpanel w-[70px] h-[70px] mx-auto rounded-2xl transition-all duration-200 hover:shadow-[0_0_20px_10px_rgba(0,0,0,0.2)]"
        :title="cleanDescription"
        :style="{ background: itemInfo?.icon?.backgroundColor || defaultBackground }">
        <div class="w-[70px] h-full flex items-center justify-center">
          <ItemIcon :item-icon="itemInfo?.icon" force-background="transparent" :size="iconSize"
            class="overflow-hidden rounded-xl"
            :style="getIconPositionStyle" />
        </div>
      </div>
      <div v-if="!iconTextIconHideTitle"
        class="app-icon-small-title text-center app-icon-text-shadow cursor-pointer mt-[2px]"
        :style="{ color: iconTextColor }">
        <span>{{ itemInfo?.title }}</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.badge {
  position: absolute;
  top: 0;
  right: 0;
  color: white;
  width: 20px;
  /* 设置徽章的宽度 */
  height: 20px;
  /* 设置徽章的高度 */
  display: flex;
  justify-content: center;
  align-items: center;
  font-size: 12px;
  mix-blend-mode: difference;
  /* 设置文字与背景色反色 */

  /* 设置徽章内文本的大小 */
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.5s;
}

.fade-enter,
.fade-leave-to {
  opacity: 0;
}
</style>