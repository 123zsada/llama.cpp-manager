import { createRouter, createWebHashHistory } from 'vue-router'

import Runtimes from './views/Runtimes.vue'
import Instances from './views/Instances.vue'
import InstanceEdit from './views/InstanceEdit.vue'
import Logs from './views/Logs.vue'

const routes = [
  { path: '/', redirect: '/instances' },
  { path: '/runtimes', name: 'runtimes', component: Runtimes },
  { path: '/instances', name: 'instances', component: Instances },
  { path: '/instances/new', name: 'instance-new', component: InstanceEdit },
  { path: '/instances/:id/edit', name: 'instance-edit', component: InstanceEdit, props: true },
  { path: '/logs', name: 'logs', component: Logs },
  { path: '/logs/:id', name: 'logs-instance', component: Logs, props: true },
]

export default createRouter({
  history: createWebHashHistory(),
  routes,
})
