import { createRouter, createWebHashHistory } from 'vue-router'
import { getModuleRoutes } from '../config/module'

export const router = createRouter({
  history: createWebHashHistory(),
  routes: getModuleRoutes(),
})
