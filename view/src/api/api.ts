import { post, get } from '../utils/request'
export interface Params {
  [key: string]: any
}
export function checkToken(data: Params): Promise<any> {
  return post("/checkToken", data)
}
export function list(data: Params): Promise<any> {
  return get("/list", data)
}
export function del(data: Params): Promise<any> {
  return post("/del", data)
}
export function delAll(data: Params): Promise<any> {
  return get("/delAll", data)
}
export function stop(data: Params): Promise<any> {
  return post("/stop", data)
}
export function stopAll(data: Params): Promise<any> {
  return get("/stopAll", data)
}
export function add(data: Params): Promise<any> {
  return post("/add", data)
}
export function cleanUp(data: Params): Promise<any> {
  return get("/cleanUp", data)
}
export function editToken(data: Params): Promise<any> {
  return post("/editToken", data)
}
export function getConf(data: Params): Promise<any> {
  return get("/getConf", data)
}
export function setConf(data: Params): Promise<any> {
  return post("/setConf", data)
}
export function checkForUpdates(data: Params): Promise<any> {
  return get("/checkForUpdates", data)
}