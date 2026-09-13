import * as App from '../../wailsjs/go/main/App'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'

export const api = {
  // 版本
  listRuntimes: () => App.ListRuntimes(),
  getRuntime: (id) => App.GetRuntime(id),
  fetchAvailableReleases: () => App.FetchAvailableReleases(),
  downloadRuntime: (tag, backend) => App.DownloadRuntime(tag, backend),
  addLocalRuntime: (dir) => App.AddLocalRuntime(dir),
  removeRuntime: (id) => App.RemoveRuntime(id),
  getRuntimeParams: (id) => App.GetRuntimeParams(id),
  repairRuntime: (id) => App.RepairRuntime(id),
  checkRuntime: (id) => App.CheckRuntime(id),
  listOrphanRuntimes: () => App.ListOrphanRuntimes(),
  deleteOrphanRuntime: (name) => App.DeleteOrphanRuntime(name),

  // 实例
  listInstances: () => App.ListInstances(),
  createInstance: (inst) => App.CreateInstance(inst),
  updateInstance: (inst) => App.UpdateInstance(inst),
  deleteInstance: (id) => App.DeleteInstance(id),
  startInstance: (id) => App.StartInstance(id),
  stopInstance: (id) => App.StopInstance(id),
  restartInstance: (id) => App.RestartInstance(id),
  getInstanceState: (id) => App.GetInstanceState(id),
  getInstanceStates: () => App.GetInstanceStates(),

  // 命令
  previewCommand: (inst) => App.PreviewCommand(inst),
  explainCommand: (inst) => App.ExplainCommand(inst),
  parseCommand: (command, runtimeId) => App.ParseCommand(command, runtimeId),

  // 路径
  selectDirectory: () => App.SelectDirectory(),
  selectFile: (filters) => App.SelectFile(filters),

  // 日志
  getLogs: (id) => App.GetLogs(id),
  clearLogs: (id) => App.ClearLogs(id),

  baseDir: () => App.BaseDir(),
}

export const events = {
  on: EventsOn,
  off: EventsOff,
}

export default api
