export type ProcessMethod = 'washed' | 'natural' | 'honey' | 'anaerobic'

export const ProcessMethodMap: Record<ProcessMethod, string> = {
  washed: '水洗',
  natural: '日晒',
  honey: '蜜处理',
  anaerobic: '厌氧发酵',
}

export const PROCESS_METHODS = Object.keys(ProcessMethodMap) as ProcessMethod[]

// Bean tracking state for the current viewer.
export type BeanTrackStatus = 'want' | 'tasted'

export const BeanTrackStatusMap: Record<BeanTrackStatus, string> = {
  want: '待喝',
  tasted: '喝过',
}

export interface CoffeeBean {
  id: number
  name: string
  origin: string
  process_method: ProcessMethod
  flavor_tags: string
  description: string
  created_at: string
  // Decorated by GET /beans; present even for anonymous viewers.
  note_total?: number
  // Present for logged-in viewers.
  tracked?: boolean
  track_status?: BeanTrackStatus | ''
  user_note_count?: number
}

// One bean entry inside a profile 待喝/喝过 group.
export interface UserBeanGroupItem extends CoffeeBean {
  note_count: number
  notes: import('@/constants/note').TastingNote[]
}

export interface UserBeanGroups {
  want: UserBeanGroupItem[]
  tasted: UserBeanGroupItem[]
}
