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
          <h3>{{ b.name }} <el-tag size="small" type="warning">{{ ProcessMethodMap[b.process_method] }}</el-tag></h3>
          <div class="meta">{{ b.origin || '-' }}</div>
          <FlavorTags :tags="b.flavor_tags" />
          <p class="desc">{{ b.description }}</p>
          <div class="count-line">
            共 <b>{{ b.note_count }}</b> 篇品鉴笔记
            <el-tag v-if="isLoggedIn && b.my_note_count > 0" type="success" size="small">我喝过 {{ b.my_note_count }} 篇</el-tag>
            <el-tag v-else-if="isLoggedIn && b.in_list" type="info" size="small">待喝中</el-tag>
          </div>
          <div class="actions">
            <template v-if="isLoggedIn">
              <el-button v-if="b.my_note_count > 0" size="small" type="success" plain disabled>
                已喝 {{ b.my_note_count }} 篇
              </el-button>
              <el-button
                v-else-if="b.in_list"
                size="small"
                type="info"
                plain
                :loading="pendingId === b.id"
                @click="moveOutOfList(b)"
              >
                移出待喝
              </el-button>
              <el-button
                v-else
                size="small"
                type="primary"
                :loading="pendingId === b.id"
                @click="addToList(b)"
              >
                加入待喝
              </el-button>
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
import { ElMessage } from 'element-plus'
import SearchFilter from '@/components/common/SearchFilter.vue'
import FlavorTags from '@/components/common/FlavorTags.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import { useBeanStore } from '@/stores/useBeanStore'
import { useAuth } from '@/hooks/useAuth'
import { addBeanToList, createBean, deleteBean, removeBeanFromList } from '@/api/bean'
import { ProcessMethodMap, type BeanListItem, type ProcessMethod } from '@/constants/bean'

const store = useBeanStore()
const { isLoggedIn, isAdmin } = useAuth()
const beans = computed(() => store.beans)
const origin = ref('')
const process = ref('')
const keyword = ref('')
const showAdd = ref(false)
const pendingId = ref(0)
const addForm = reactive({ name: '', origin: '', process_method: 'washed', flavor_tags: '[]', description: '' })

const ORIGINS = ['埃塞俄比亚', '哥伦比亚', '哥斯达黎加', '印度尼西亚']

onMounted(() => load())

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

// Optimistic toggle: flip the card state first; on failure the request layer
// shows the error and we reload to restore the original server state.
async function addToList(b: BeanListItem) {
  pendingId.value = b.id
  const previous = b.in_list
  b.in_list = true
  try {
    await addBeanToList(b.id)
    ElMessage.success('已加入待喝名单')
  } catch {
    b.in_list = previous
  } finally {
    pendingId.value = 0
  }
}
async function moveOutOfList(b: BeanListItem) {
  pendingId.value = b.id
  const previous = b.in_list
  b.in_list = false
  try {
    await removeBeanFromList(b.id)
    ElMessage.success('已移出待喝名单')
  } catch {
    b.in_list = previous
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
.count-line { display: flex; align-items: center; gap: 8px; color: #999; font-size: 12px; margin-top: 10px; }
.count-line b { color: #7b4b2a; }
.actions { margin-top: 10px; display: flex; gap: 8px; }
</style>
