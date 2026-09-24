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

    <el-card class="block">
      <template #header>
        <span class="group-title">🫘 待喝名单</span>
        <el-tag size="small" type="info" class="count-tag">{{ wantList.length }}</el-tag>
      </template>
      <div v-if="wantList.length" class="bean-grid">
        <div v-for="b in wantList" :key="b.id" class="bean-chip">
          <span class="chip-name" @click="$router.push('/beans')">{{ b.name }}</span>
          <span class="chip-meta">{{ b.origin || '-' }}</span>
          <el-button
            v-if="isOwner"
            size="small" text type="danger"
            :loading="removingId === b.id"
            @click="removeWant(b)"
          >移出</el-button>
        </div>
      </div>
      <EmptyState v-else description="还没有想喝的豆，去豆种库挑一挑吧" action-text="逛逛豆种库" @action="$router.push('/beans')" />
    </el-card>

    <el-card class="block">
      <template #header>
        <span class="group-title">✅ 喝过的豆</span>
        <el-tag size="small" type="success" class="count-tag">{{ tastedList.length }}</el-tag>
      </template>
      <div v-if="tastedList.length" class="tasted-list">
        <el-card v-for="b in tastedList" :key="b.id" class="tasted-card" shadow="never">
          <div class="tasted-head">
            <b>{{ b.name }}</b>
            <el-tag size="small" type="success">{{ b.note_count }} 篇笔记</el-tag>
          </div>
          <div class="tasted-notes">
            <el-link
              v-for="n in b.notes"
              :key="n.id"
              type="primary"
              :underline="false"
              class="note-link"
              @click="$router.push(`/note/${n.id}`)"
            >
              《{{ n.coffee_name }}》· {{ formatDate(n.created_at) }}
            </el-link>
          </div>
        </el-card>
      </div>
      <EmptyState v-else description="还没有喝过的豆，发布一篇品鉴笔记试试" />
    </el-card>

    <h3 class="history-title">品鉴历史</h3>
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
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import UserAvatar from '@/components/common/UserAvatar.vue'
import ScoreStars from '@/components/common/ScoreStars.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import { getUserProfile, followUser, unfollowUser } from '@/api/user'
import { removeBeanFromWant } from '@/api/bean'
import { useAuth } from '@/hooks/useAuth'
import { RoastLevelMap } from '@/constants/note'
import type { ProfileData } from '@/api/user'
import type { UserBeanGroupItem } from '@/constants/bean'
import { formatDate } from '@/utils/dateFormat'

const route = useRoute()
const { isLoggedIn, user } = useAuth()
const data = ref<ProfileData | null>(null)
const following = ref(false)
const removingId = ref(0)

const isOwner = computed(() => !!user.value && data.value?.user.id === user.value.id)
const wantList = computed(() => data.value?.bean_groups?.want ?? [])
const tastedList = computed(() => data.value?.bean_groups?.tasted ?? [])

onMounted(load)

async function load() {
  data.value = await getUserProfile(route.params.id as string)
}

// Move a 待喝 bean out. Only zero-note beans are removable server-side;
// on failure the list is left untouched.
async function removeWant(b: UserBeanGroupItem) {
  removingId.value = b.id
  try {
    await removeBeanFromWant(b.id)
    ElMessage.success('已移出待喝名单')
    await load()
  } catch {
    // keep the original list
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
.block { margin-bottom: 20px; }
.group-title { font-size: 16px; font-weight: 600; }
.count-tag { margin-left: 8px; }
.bean-grid { display: flex; flex-direction: column; gap: 10px; }
.bean-chip { display: flex; align-items: center; gap: 12px; padding: 6px 4px; border-bottom: 1px solid #f5f0e8; }
.chip-name { font-weight: 600; color: #7b4b2a; cursor: pointer; }
.chip-meta { color: #999; font-size: 12px; flex: 1; }
.tasted-list { display: flex; flex-direction: column; gap: 12px; }
.tasted-card { background: #faf6f0; }
.tasted-head { display: flex; align-items: center; gap: 10px; margin-bottom: 6px; }
.tasted-notes { display: flex; flex-direction: column; gap: 4px; }
.note-link { justify-content: flex-start; }
.history-title { margin: 24px 0 12px; }
.note-card { margin-bottom: 16px; cursor: pointer; }
.meta { color: #999; font-size: 12px; }
</style>
