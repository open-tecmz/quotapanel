const BATCH_DELAY = 5000

export type AnalyticsEvent = 'Open' | 'Visit' | 'Error'

export interface AnalyticsData {
  page?: string
  error?: string
  [key: string]: unknown
}

interface CollectPayload {
  name: AnalyticsEvent
  data?: AnalyticsData
}

const eventQueue: CollectPayload[] = []
let batchTimer: ReturnType<typeof setTimeout> | null = null

function sendAnalyticsViaBackend(events: CollectPayload[]): void {
  const goCall = (window as any)?.go?.main?.App?.Call
  if (!goCall) {
    console.warn('[Analytics] Go backend not available')
    return
  }
  goCall('setting.sendAnalytics', { events })
}

async function flushEvents(): Promise<void> {
  if (eventQueue.length === 0) return
  const eventsToSend = [...eventQueue]
  eventQueue.length = 0
  batchTimer = null
  sendAnalyticsViaBackend(eventsToSend)
}

function queueEvent(name: AnalyticsEvent, data?: AnalyticsData): void {
  const payload: CollectPayload = { name }
  if (data) payload.data = data
  eventQueue.push(payload)
  if (batchTimer) clearTimeout(batchTimer)
  batchTimer = setTimeout(() => {
    flushEvents()
  }, BATCH_DELAY)
}

export function trackOpen(): void {
  queueEvent('Open')
}

export function trackVisit(page: string): void {
  queueEvent('Visit', { page })
}

export function trackError(error: string): void {
  queueEvent('Error', { error })
}

export default { trackOpen, trackVisit, trackError }
