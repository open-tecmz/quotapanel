import coreZh from './core/zh'
import coreEn from './core/en'
import processesZh from './modules/processes/zh'
import processesEn from './modules/processes/en'
import settingsZh from './modules/settings/zh'
import settingsEn from './modules/settings/en'

export const messages = {
  zh: {
    ...coreZh,
    ...processesZh,
    ...settingsZh,
  },
  en: {
    ...coreEn,
    ...processesEn,
    ...settingsEn,
  },
}
