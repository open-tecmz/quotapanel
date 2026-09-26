/// <reference types="vite/client" />

interface Window {
  __test?: {
    setAction(name: string, fn: (arg?: unknown) => Promise<unknown> | unknown): void
    unsetAction(nameOrNames: string | string[]): void
    callAction(name: string, arg?: unknown): Promise<unknown>
    listActions(): string[]
    navigateTo(path: string): Promise<void>
  }
}

declare module '*.vue' {
  import type { DefineComponent } from 'vue'
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const component: DefineComponent<Record<string, unknown>, Record<string, unknown>, any>
  export default component
}
