<template>
  <div class="page">
    <h1>豆种库</h1>
    <SearchFilter @search="onSearch" @reset="onReset">
      <template #filters>
        <el-form-item label="产地">
          <el-select v-model="origin" clearable placeholder="全部产地" style="width: 150px" @change="load">
            <el-option v-for="o in ORIGINS" :key="o" :label="o" :value="o" />
          </el-select>
        </el-form-item>
        <el-form-item label="处理法">
          <el-select v-model="process" clearable placeholder="全部处理法" style="width: 150px" @change="load">
            <el-option v-for="(label, value) in ProcessMethodMap" :key="value" :label="label" :value="value" />
          </el-select>
        </el-form-item>
      </template>
    </SearchFilter>
    <el-row :gutter="16">
      <el-col v-for="b in beans" :key="b.id" :xs="24" :sm="12" :md="8">
        <el-card class="bean-card" shadow="hover">
          <h3>
            {{ b.name }} <el-tag size="small" type="warning">{{ ProcessMethodMap[b.process_method] }}</el-tag>
            <el-tag v-if="isLoggedIn && b.tracked" size="small" :type="b.track_status === 'tasted' ? 'success' : 'info'" class="track-tag">
              {{ b.track_status === 'tasted' ? `喝过 ${b.user_note_count} 篇` : '待喝' }}
            </el-tag>
          </h3>
          <div class="meta">{{ b.origin || '-' }} · 共 {{ b.note_total || 0 }} 篇品鉴</div>
          <FlavorTags :tags="b.flavor_tags" />
          <p class="desc">{{ b.description }}</p>
          <div class="card-actions">
            <template v-if="isLoggedIn">
              <el-button
                v-if="!b.tracked"
                size="small" type="primary" plain
                :loading="pendingId === b.id"
                @click="addToWant(b)"
              >加入待喝</el-button>
              <template v-else-if="b.track_status === 'want'">
                <el-tag size="small" type="info">待喝名单中</el-tag>
                <el-button
                  size="small" type="danger" plain
                  :loading="pendingId === b.id"
                  @click="removeFromWant(b)"
                >移出</el-button>
              </template>
              <el-tag v-else size="small" type="success">已喝 {{ b.user_note_count }} 篇</el-tag>
            </template>
            <el-button v-if="isAdmin" size="small" type="danger" plain @click="removeBean(b.id)">删除</el-button>
          </div>
        </el-card>
      </el-col>
    </el-row>
    <EmptyState v-if="!beans.length" description="暂无豆种" />
    <el-button v-if="isAdmin" type="primary" style="margin-top: 16px" @click="showAdd = true">新增豆种</el-button>
    <el-dialog v-model="showAdd" title="新增豆种" width="480px">
      <el-form label-width="80px">
        <el-form-item label="名称"><el-input v-model="addForm.name" /></el-form-item>
        <el-form-item label="产地"><el-input v-model="addForm.origin" /></el-form-item>
        <el-form-item label="处理法">
          <el-select v-model="addForm.process_method" style="width: 200px">
            <el-option v-for="(label, value) in ProcessMethodMap" :key="value" :label="label" :value="value" />
          </el-select>
        </el-form-item>
        <el-form-item label="风味标签"><el-input v-model="addForm.flavor_tags" placeholder='如 ["坚果","焦糖"]' /></el-form-item>
        <el-form-item label="描述"><el-input v-model="addForm.description" type="textarea" :rows="3" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showAdd = false">取消</el-button>
        <el-button type="primary" @click="addBean">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import SearchFilter from '@/components/common/SearchFilter.vue'
import FlavorTags from '@/components/common/FlavorTags.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import { useBeanStore } from '@/stores/useBeanStore'
import { useAuth } from '@/hooks/useAuth'
import { createBean, deleteBean, addBeanToWant, removeBeanFromWant } from '@/api/bean'
import { ProcessMethodMap, type ProcessMethod, type CoffeeBean } from '@/constants/bean'

const store = useBeanStore()
const { isAdmin, isLoggedIn } = useAuth()
const route = useRoute()
const beans = computed(() => store.beans)
const origin = ref('')
const process = ref('')
const keyword = ref('')
const showAdd = ref(false)
// The bean whose 待喝 request is in flight; failures leave the card unchanged.
const pendingId = ref(0)
const addForm = reactive({ name: '', origin: '', process_method: 'washed', flavor_tags: '[]', description: '' })

const ORIGINS = ['埃塞俄比亚', '哥伦比亚', '哥斯达黎加', '印度尼西亚']

onMounted(() => {
  if (typeof route.query.keyword === 'string') keyword.value = route.query.keyword
  load()
})

async function load() {
  await store.load({ page: 1, page_size: 20, origin: origin.value, process: process.value, keyword: keyword.value })
}
function onSearch(kw: string) {
  keyword.value = kw
  load()
}
function onReset() {
  origin.value = ''
  process.value = ''
  keyword.value = ''
  load()
}

// Add to 待喝: only mutate the card after the API succeeds.
async function addToWant(b: CoffeeBean) {
  pendingId.value = b.id
  try {
    await addBeanToWant(b.id)
    b.tracked = true
    b.track_status = 'want'
    b.user_note_count = 0
    ElMessage.success('已加入待喝名单')
  } catch {
    // request.ts surfaced the error; keep the original button state.
  } finally {
    pendingId.value = 0
  }
}

// Remove from 待喝: a 喝过 bean is rejected server-side, state is preserved on failure.
async function removeFromWant(b: CoffeeBean) {
  pendingId.value = b.id
  try {
    await removeBeanFromWant(b.id)
    b.tracked = false
    b.track_status = ''
    b.user_note_count = 0
    ElMessage.success('已移出待喝名单')
  } catch {
    // keep the card in 待喝 state
  } finally {
    pendingId.value = 0
  }
}
async function addBean() {
  if (!addForm.name) {
    ElMessage.warning('请填写名称')
    return
  }
  await createBean({ ...addForm, process_method: addForm.process_method as ProcessMethod })
  ElMessage.success('豆种已添加')
  showAdd.value = false
  await load()
}
async function removeBean(id: number) {
  await deleteBean(id)
  ElMessage.success('已删除')
  await load()
}
</script>

<style scoped>
.page { max-width: 1200px; margin: 0 auto; }
.bean-card { margin-bottom: 16px; }
.meta { color: #999; font-size: 12px; margin: 6px 0; }
.desc { color: #666; margin-top: 8px; }
.track-tag { margin-left: 6px; }
.card-actions { margin-top: 10px; display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
</style>
