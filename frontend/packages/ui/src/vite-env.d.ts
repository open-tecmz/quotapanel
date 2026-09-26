/// <reference types="vite/client" />
/// <reference types="vue-i18n" />

interface Window {
  __debug?: {
    registerAction(name: string, fn: (params: unknown) => Promise<unknown>): void
    unregisterAction(nameOrNames: string | string[]): void
    hasAction(name: string): boolean
    triggerAction(name: string, params?: unknown): Promise<unknown>
  }
  __test?: {
    setAction(name: string, fn: (arg?: unknown) => Promise<unknown> | unknown): void
    unsetAction(nameOrNames: string | string[]): void
    callAction(name: string, arg?: unknown): Promise<unknown>
    listActions(): string[]
    navigateTo(path: string): Promise<void>
  }
}

interface ImportMetaEnv {
  readonly VITE_APPSTORE_BUILD?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}

declare module '*.vue' {
  import type { DefineComponent } from 'vue'
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const component: DefineComponent<Record<string, unknown>, Record<string, unknown>, any>
  export default component
}
