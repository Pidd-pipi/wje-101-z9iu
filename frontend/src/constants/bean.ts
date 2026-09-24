export type ProcessMethod = 'washed' | 'natural' | 'honey' | 'anaerobic'

export const ProcessMethodMap: Record<ProcessMethod, string> = {
  washed: '水洗',
  natural: '日晒',
  honey: '蜜处理',
  anaerobic: '厌氧发酵',
}

export const PROCESS_METHODS = Object.keys(ProcessMethodMap) as ProcessMethod[]

export interface CoffeeBean {
  id: number
  name: string
  origin: string
  process_method: ProcessMethod
  flavor_tags: string
  description: string
  created_at: string
}

// Bean card enriched by the backend with tasting-note tallies and, when
// logged in, the viewer's personal want-to-drink state.
export interface BeanListItem extends CoffeeBean {
  note_count: number
  my_note_count: number
  in_list: boolean
}

// Personal profile groups: want = waiting list, drunk = beans with my notes.
export interface UserBeanGroups {
  want: BeanListItem[]
  drunk: BeanListItem[]
}
