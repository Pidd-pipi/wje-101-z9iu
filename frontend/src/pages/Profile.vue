<template>
  <div class="page" v-if="data">
    <el-card class="head">
      <div class="user-line">
        <UserAvatar :name="data.user.username" :size="64" />
        <div>
          <h2>{{ data.user.username }} <el-tag size="small">{{ data.user.role === 'admin' ? '管理员' : '咖啡爱好者' }}</el-tag></h2>
          <p class="bio">{{ data.user.bio || '这个人很懒，什么都没写' }}</p>
        </div>
      </div>
      <div class="stats">
        <div class="stat"><b>{{ data.note_count }}</b><span>品鉴次数</span></div>
        <div class="stat"><b>{{ data.avg_score.toFixed(1) }}</b><span>平均分</span></div>
        <div class="stat"><b>{{ data.likes_received }}</b><span>收到点赞</span></div>
        <div class="stat"><b>{{ data.followers }}</b><span>粉丝</span></div>
        <div class="stat"><b>{{ data.following }}</b><span>关注</span></div>
      </div>
      <div class="origins" v-if="data.top_origins.length">
        最爱产地 TOP3：<el-tag v-for="o in data.top_origins" :key="o" size="small" class="origin-tag">{{ o }}</el-tag>
      </div>
      <el-button v-if="isLoggedIn && user?.id !== Number($route.params.id)" :type="following ? 'default' : 'primary'" @click="toggleFollow">
        {{ following ? '已关注' : '关注' }}
      </el-button>
    </el-card>

    <h3>待喝名单（{{ groups?.want.length || 0 }}）</h3>
    <el-row :gutter="16">
      <el-col v-for="b in groups?.want" :key="`want-${b.id}`" :xs="24" :sm="12" :md="8">
        <el-card class="bean-card want" shadow="hover">
          <div class="bean-head">
            <span class="bean-name">{{ b.name }}</span>
            <el-tag size="small" type="warning">{{ ProcessMethodMap[b.process_method] }}</el-tag>
          </div>
          <div class="meta">{{ b.origin || '-' }} · 站内共 {{ b.note_count }} 篇笔记</div>
          <el-button v-if="isOwner" size="small" type="info" plain :loading="removingId === b.id" @click="moveOut(b)">移出待喝</el-button>
        </el-card>
      </el-col>
    </el-row>
    <EmptyState v-if="groups && !groups.want.length" description="待喝名单还是空的，去豆种库挑挑想喝的豆子吧" action-text="逛逛豆种库" @action="goBeans" />

    <h3 style="margin-top: 24px">喝过的豆（{{ groups?.drunk.length || 0 }}）</h3>
    <el-row :gutter="16">
      <el-col v-for="b in groups?.drunk" :key="`drunk-${b.id}`" :xs="24" :sm="12" :md="8">
        <el-card class="bean-card drunk" shadow="hover">
          <div class="bean-head">
            <span class="bean-name">{{ b.name }}</span>
            <el-tag size="small" type="success">喝过 {{ b.my_note_count }} 篇</el-tag>
          </div>
          <div class="meta">{{ b.origin || '-' }} · {{ ProcessMethodMap[b.process_method] }}</div>
          <div class="note-links">
            <el-button
              v-for="n in notesOfBean(b.name)"
              :key="n.id"
              size="small"
              text
              type="primary"
              @click="$router.push(`/note/${n.id}`)"
            >
              #{{ n.id }} {{ formatDateTime(n.created_at) }} · {{ n.overall_score.toFixed(1) }} 分
            </el-button>
          </div>
        </el-card>
      </el-col>
    </el-row>
    <EmptyState v-if="groups && !groups.drunk.length" description="还没有喝过的豆，发布一篇品鉴笔记试试吧" />

    <h3 style="margin-top: 24px">品鉴历史</h3>
    <el-row :gutter="16">
      <el-col v-for="n in data.notes" :key="n.id" :xs="24" :sm="12" :md="8">
        <el-card class="note-card" shadow="hover" @click="$router.push(`/note/${n.id}`)">
          <h4>{{ n.coffee_name }}</h4>
          <div class="meta">{{ n.origin }} · {{ RoastLevelMap[n.roast_level] }}</div>
          <ScoreStars :model-value="n.overall_score" />
        </el-card>
      </el-col>
    </el-row>
    <EmptyState v-if="!data.notes.length" description="暂无品鉴记录" />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import UserAvatar from '@/components/common/UserAvatar.vue'
import ScoreStars from '@/components/common/ScoreStars.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import { getUserProfile, followUser, unfollowUser } from '@/api/user'
import { getUserBeans, removeBeanFromList } from '@/api/bean'
import { useAuth } from '@/hooks/useAuth'
import { RoastLevelMap, type TastingNote } from '@/constants/note'
import { ProcessMethodMap, type UserBeanGroups } from '@/constants/bean'
import type { ProfileData } from '@/api/user'
import { formatDateTime } from '@/utils/dateFormat'

const route = useRoute()
const router = useRouter()
const { isLoggedIn, user } = useAuth()
const data = ref<ProfileData | null>(null)
const groups = ref<UserBeanGroups | null>(null)
const following = ref(false)
const removingId = ref(0)

const isOwner = computed(() => isLoggedIn.value && user.value?.id === Number(route.params.id))

onMounted(loadProfile)

async function loadProfile() {
  data.value = await getUserProfile(route.params.id as string)
  groups.value = await getUserBeans(route.params.id as string)
}

function notesOfBean(coffeeName: string): TastingNote[] {
  return data.value?.notes.filter((n) => n.coffee_name === coffeeName) || []
}
function goBeans() {
  router.push('/beans')
}
async function moveOut(b: { id: number }) {
  const target = groups.value?.want.find((x) => x.id === b.id)
  if (!target) return
  removingId.value = b.id
  try {
    await removeBeanFromList(b.id)
    groups.value!.want = groups.value!.want.filter((x) => x.id !== b.id)
    ElMessage.success('已移出待喝名单')
  } catch {
    // request layer already surfaced the error; keep the item in place
  } finally {
    removingId.value = 0
  }
}

async function toggleFollow() {
  if (!isLoggedIn.value) {
    ElMessage.warning('请先登录')
    return
  }
  if (following.value) {
    await unfollowUser(data.value!.user.id)
    following.value = false
    ElMessage.success('已取消关注')
  } else {
    await followUser(data.value!.user.id)
    following.value = true
    ElMessage.success('关注成功')
  }
}
</script>

<style scoped>
.page { max-width: 1000px; margin: 0 auto; }
.head { margin-bottom: 20px; }
.user-line { display: flex; gap: 16px; align-items: center; }
.bio { color: #999; }
.stats { display: flex; gap: 32px; margin: 16px 0; }
.stat { display: flex; flex-direction: column; }
.stat b { font-size: 22px; color: #7b4b2a; }
.stat span { color: #999; font-size: 12px; }
.origins { margin: 12px 0; }
.origin-tag { margin-right: 6px; }
.bean-card { margin-bottom: 16px; }
.bean-card.want { border-left: 4px solid #e6a23c; }
.bean-card.drunk { border-left: 4px solid #67c23a; }
.bean-head { display: flex; align-items: center; justify-content: space-between; gap: 8px; }
.bean-name { font-weight: 600; font-size: 15px; }
.note-links { display: flex; flex-wrap: wrap; margin-top: 4px; }
.note-card { margin-bottom: 16px; cursor: pointer; }
.meta { color: #999; font-size: 12px; margin: 6px 0; }
</style>
