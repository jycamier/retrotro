import { create } from 'zustand'
import type { Item, LCTopicHistory } from '../types'

interface LeanCoffeeState {
  currentTopicId: string | null
  queue: Item[]
  done: Item[]
  topicHistory: LCTopicHistory[]
  allTopicsDone: boolean

  // Actions
  setDiscussionState: (state: {
    currentTopicId: string | null
    queue: Item[]
    done: Item[]
    topicHistory: LCTopicHistory[]
  }) => void
  setCurrentTopicId: (topicId: string | null) => void
  setAllTopicsDone: (done: boolean) => void
  reset: () => void
}

const initialState = {
  currentTopicId: null as string | null,
  queue: [] as Item[],
  done: [] as Item[],
  topicHistory: [] as LCTopicHistory[],
  allTopicsDone: false,
}

export const useLeanCoffeeStore = create<LeanCoffeeState>((set) => ({
  ...initialState,

  setDiscussionState: (state) => set({
    currentTopicId: state.currentTopicId,
    queue: state.queue || [],
    done: state.done || [],
    topicHistory: state.topicHistory || [],
    allTopicsDone: false,
  }),

  setCurrentTopicId: (topicId) => set({ currentTopicId: topicId }),

  setAllTopicsDone: (done) => set({ allTopicsDone: done }),

  reset: () => set({ ...initialState }),
}))
