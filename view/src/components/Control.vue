<template>
    <el-container>
        <el-header class="control-header">
            M3u8Download
        </el-header>
        <el-container>
            <el-aside class="control-aside" width="200px">
                <el-menu class="menu" :default-active="menu.active">
                    <el-menu-item-group title="下载">
                        <el-menu-item @click="select('1')" index="1">正在下载</el-menu-item>
                        <el-menu-item @click="select('2')" index="2">下载完成</el-menu-item>
                        <el-menu-item @click="select('3')" index="3">下载失败</el-menu-item>
                        <el-menu-item @click="select('4')" index="4">停止下载</el-menu-item>
                    </el-menu-item-group>
                    <el-menu-item-group title="设置">
                        <el-menu-item @click="select('0')" index="0">设置</el-menu-item>
                    </el-menu-item-group>
                </el-menu>
            </el-aside>
            <el-container v-if="menu.active != '0'">
                <el-header class="control-header-btn" height="40px">
                    <el-button size="small" @click="handledDelAll" type="primary">
                        清空列表
                        <el-icon class="el-icon--right">
                            <Delete />
                        </el-icon>
                    </el-button>
                    <el-button v-if="menu.active == '1'" @click="handleStopAll" size="small" type="primary">
                        全部停止
                        <el-icon class="el-icon--right">
                            <el-icon>
                                <Stopwatch />
                            </el-icon>
                        </el-icon>
                    </el-button>
                    <el-button @click="handleNewDownBtn" v-if="menu.active == '1'" size="small" type="primary">
                        新建下载
                        <el-icon class="el-icon--right">
                            <Download />
                        </el-icon>
                    </el-button>
                </el-header>
                <el-main style="padding-top: 0;">
                    <div style="background-color: #ebeef5;height: 1px;width: 100%;"></div>
                    <el-table :data="tableData" style="width: 100%" @row-contextmenu="handleRowRightClick">
                        <el-table-column prop="Filename" label="文件名" width="180" />
                        <el-table-column prop="SavePath" label="存储位置" width="180" />
                        <el-table-column v-if="menu.active == '1'" prop="Speed" label="下载速度" width="180" />
                        <el-table-column v-if="menu.active == '1'" prop="ReadSize" label="已下载大小" width="180" />
                        <el-table-column prop="Status" label="状态">
                            <template #default="scope">
                                <el-tag v-if="scope.row.Status == 0" type="info">等待</el-tag>
                                <el-tag v-if="scope.row.Status == 2" type="success">完成</el-tag>
                                <el-tag v-if="scope.row.Status == 1" type="warning">下载</el-tag>
                                <el-tag v-if="scope.row.Status == 3" type="danger">失败</el-tag>
                                <el-tag v-if="scope.row.Status == 4 && menu.active == '1'" type="info">正在停止</el-tag>
                                <el-tag v-if="scope.row.Status == 4 && menu.active != '1'" type="info">停止</el-tag>
                                <el-tag v-if="scope.row.Status == 5 && menu.active == '1'" type="info">正在删除</el-tag>
                            </template>
                        </el-table-column>
                        <el-table-column default="Msg" label="消息"></el-table-column>
                        <el-table-column prop="Threads" label="最大线程" width="180" />
                        <el-table-column v-if="menu.active == '1'" prop="Active" label="活跃线程" width="180" />
                        <el-table-column v-if="menu.active == '1'" prop="Total" label="文件总数" width="180" />
                        <el-table-column v-if="menu.active == '1'" prop="Complete" label="已下载文件" width="180" />
                        <el-table-column v-if="menu.active == '1'" default="Total" label="总进度">
                            <template #default="scope">
                                <span v-if="scope.row.Total > 0">{{ ((scope.row.Complete / scope.row.Total) *
                                    100).toFixed(0) }} %</span>
                                <span v-else>0 %</span>
                            </template>
                        </el-table-column>
                    </el-table>
                    <!-- 自定义上下文菜单 -->
                    <div v-show="contextMenuVisible" class="custom-context-menu" :style="{
                        left: contextMenuPosition.x + 'px',
                        top: contextMenuPosition.y + 'px'
                    }" @contextmenu.prevent>
                        <div v-if="menu.active == '1'" class="context-menu-item" @click="handleMenuClick('stop')">停止
                        </div>
                        <div class="context-menu-item" @click="handleMenuClick('delete')">删除</div>
                    </div>
                </el-main>
            </el-container>
            <el-container v-if="menu.active == '0'" class="control-setting-container">
                <div>
                    <div class="control-setting-form">
                        <div>
                            &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;Token
                        </div>
                        <div class="input"><el-input v-model="conf.Token" style="width: 240px" /></div>
                        <div><el-button @click="handleEditConf('Token', conf.Token)" type="primary" plain>修改</el-button>
                        </div>
                    </div>
                    <br />
                    <div class="control-setting-form">
                        <div>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;同时下载数</div>
                        <div class="input"><el-input-number v-model="conf.maxWorkers" style="width: 240px" :min="1"
                                :max="64" />
                        </div>
                        <div><el-button @click="handleEditConf('maxWorkers', conf.maxWorkers)" type="primary"
                                plain>修改</el-button>
                        </div>
                    </div>
                    <br />
                    <div class="control-setting-form">
                        <div>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;默认线程数</div>
                        <div class="input"><el-input-number v-model="conf.Threads" style="width: 240px" :min="1"
                                :max="64" /></div>
                        <div><el-button @click="handleEditConf('Threads', conf.Threads)" type="primary"
                                plain>修改</el-button></div>
                    </div>
                    <br />
                    <div class="control-setting-form">
                        <div>&nbsp;&nbsp;默认保存位置</div>
                        <div class="input"><el-input v-model="conf.SavePath" style="width: 240px" /></div>
                        <div><el-button @click="handleEditConf('SavePath', conf.SavePath)" type="primary"
                                plain>修改</el-button></div>
                    </div>
                    <br />
                    <div class="control-setting-form">
                        <div>默认UserAgent</div>
                        <div class="input"><el-input v-model="conf.UserAgent" style="width: 240px" /></div>
                        <div><el-button @click="handleEditConf('UserAgent', conf.UserAgent)" type="primary"
                                plain>修改</el-button>
                        </div>
                    </div>
                    <br />
                    <div class="control-setting-form">
                        <div>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;默认Cookie</div>
                        <div class="input"><el-input v-model="conf.Cookie" style="width: 240px" /></div>
                        <div><el-button @click="handleEditConf('Cookie', conf.Cookie)" type="primary"
                                plain>修改</el-button></div>
                    </div>
                    <br />
                    <div class="control-setting-form">
                        <div>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;默认Referer</div>
                        <div class="input"><el-input v-model="conf.Referer" style="width: 240px" /></div>
                        <div><el-button @click="handleEditConf('Referer', conf.Referer)" type="primary"
                                plain>修改</el-button></div>
                    </div>
                    <br />
                    <div class="control-setting-form">
                        <div>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;默认Origin</div>
                        <div class="input"><el-input v-model="conf.Origin" style="width: 240px" /></div>
                        <div><el-button @click="handleEditConf('Origin', conf.Origin)" type="primary"
                                plain>修改</el-button></div>
                    </div>
                    <br />
                    <div class="control-setting-form">
                        <div>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;默认代理</div>
                        <div class="input"><el-input v-model="conf.Proxy" style="width: 240px" /></div>
                        <div><el-button @click="handleEditConf('Proxy', conf.Proxy)" type="primary" plain>修改</el-button>
                        </div>
                    </div>
                </div>
            </el-container>
        </el-container>
    </el-container>
    <el-dialog v-model="dialogVisible" title="新建下载" width="650" :close-on-click-modal="false">
        <el-form :model="form" label-width="auto" style="max-width: 600px">
            <el-form-item label="下载链接">
                <el-input v-model="form.Url" />
            </el-form-item>
            <el-form-item label="保存名称">
                <el-input v-model="form.Filename" />
            </el-form-item>
            <el-form-item label="保存位置">
                <el-input v-model="form.SavePath" />
            </el-form-item>
            <el-row>
                <el-col :span="12">
                    <el-form-item label="最大线程">
                        <el-input-number v-model="form.Threads" :min="1" :max="64" />
                    </el-form-item>
                </el-col>
                <el-col :span="12">
                    <el-form-item label="Referer">
                        <el-input v-model="form.Referer" placeholder="Referer(可选)" />
                    </el-form-item>
                </el-col>
            </el-row>
            <el-row>
                <el-col :span="12">
                    <el-form-item label="UserAgent">
                        <el-input v-model="form.UserAgent" placeholder="UserAgent(可选)" />
                    </el-form-item>
                </el-col>
                <el-col :span="12">
                    <el-form-item label="Cookie">
                        <el-input v-model="form.Cookie" placeholder="Cookie(可选)" />
                    </el-form-item>
                </el-col>
            </el-row>
            <el-row>
                <el-col :span="12">
                    <el-form-item label="Origin">
                        <el-input v-model="form.Origin" placeholder="Origin(可选)" />
                    </el-form-item>
                </el-col>
                <el-col :span="12">
                    <el-form-item label="代理">
                        <el-input v-model="form.Proxy" placeholder="可选(例： http://127.0.0.1:1080)" />
                    </el-form-item>
                </el-col>
            </el-row>
        </el-form>
        <template #footer>
            <div class="dialog-footer">
                <el-button @click="dialogVisible = false">关闭</el-button>
                <el-button type="primary" @click="handleAdd">
                    新建并下载
                </el-button>
            </div>
        </template>
    </el-dialog>
</template>
<script setup lang="ts">
import { Download, Delete, Stopwatch } from '@element-plus/icons-vue'
import { ref, reactive } from 'vue'
import { setToken } from '../utils/auth'
import { list, stop, stopAll, del, delAll, cleanUp, add, getConf, setConf } from "../api/api";
import type { Params } from "../api/api";
import { ElMessageBox, ElLoading, ElMessage } from 'element-plus'
// 组件挂载时添加事件监听
import { onMounted, onUnmounted } from 'vue'
// 右键菜单相关状态
const contextMenuVisible = ref(false)
const contextMenuPosition = ref({ x: 0, y: 0 })
const selectedRow = ref<any>(null)
const dialogVisible = ref(false)
const form = reactive({
    Url: '',
    Filename: '',
    SavePath: '',
    Threads: 64 as any | number | string,
    Referer: "",
    UserAgent: '',
    Cookie: '',
    Origin: '',
    Proxy: '',
})
const menu = ref({
    active: "1"
})
const conf = reactive({
    Cookie: "",
    Origin: "",
    Proxy: "",
    Referer: "",
    SavePath: "",
    Threads: 64 as any | number | string,
    Token: "",
    UserAgent: "",
    maxWorkers: 4 as any | number | string,
})
var tableData = ref([])
var setIntervalControl: number
var setIntervalControlStart: boolean = false
const select = (v: any) => {
    tableData.value = []
    menu.value.active = v
    if (menu.value.active != '0' && setIntervalControlStart == false) {
        setIntervalControlStart = true
        setIntervalControl = setInterval(() => {
            getList()
        }, 1000)
    }
    if (menu.value.active == '0') {
        clearInterval(setIntervalControl)
        setIntervalControlStart = false
    }
}
const handleRowRightClick = (row: any, column: any, event: MouseEvent) => {
    event.preventDefault()
    column = column
    // 记录选中行和位置
    selectedRow.value = row
    contextMenuPosition.value = { x: event.clientX, y: event.clientY }
    // 显示菜单
    contextMenuVisible.value = true
}
const handleMenuClick = (command: string) => {
    // 执行完命令后隐藏菜单
    contextMenuVisible.value = false
    const loading = ElLoading.service({
        lock: true,
        text: 'Loading...',
        background: 'rgba(0, 0, 0, 0.7)',
    })
    switch (command) {
        case 'stop':
            var params: Params = {
                "uuid": selectedRow.value.Uuid
            }
            stop(params).then(res => {
                if (res.code == 200) {
                    ElMessage({
                        message: 'success',
                        type: 'success',
                    })
                } else {
                    ElMessageBox.alert(res.msg, '发生错误', {
                        confirmButtonText: '确认'
                    })
                }
            }).catch(err => {
                ElMessageBox.alert(err.response != undefined && err.response.data != undefined ? err.response.data : err.message, '发生错误', {
                    confirmButtonText: '确认'
                })
            }).finally(() => {
                loading.close()
            })
            break
        case 'delete':
            var params: Params = {
                "uuid": selectedRow.value.Uuid
            }
            del(params).then(res => {
                if (res.code == 200) {
                    ElMessage({
                        message: 'success',
                        type: 'success',
                    })
                } else {
                    ElMessageBox.alert(res.msg, '发生错误', {
                        confirmButtonText: '确认'
                    })
                }
            }).catch(err => {
                ElMessageBox.alert(err.response != undefined && err.response.data != undefined ? err.response.data : err.message, '发生错误', {
                    confirmButtonText: '确认'
                })
            }).finally(() => {
                loading.close()
            })
            break
    }
}
// 点击其他地方隐藏菜单
const handleClickOutside = (event: MouseEvent) => {
    if (contextMenuVisible.value) {
        const contextMenu = document.querySelector('.custom-context-menu')
        if (contextMenu && !contextMenu.contains(event.target as Node)) {
            contextMenuVisible.value = false
        }
    }
}
onMounted(() => {
    document.addEventListener('click', handleClickOutside)
})
onUnmounted(() => {
    document.removeEventListener('click', handleClickOutside)
})
onMounted(() => {
    getList()
    handleGetConf()
    setIntervalControlStart = true
    setIntervalControl = setInterval(() => {
        getList()
    }, 1000)
})
const getList = () => {
    var params: Params = {
        "status": menu.value.active
    }
    list(params).then(res => {
        if (res.code == 200) {
            tableData.value = res.data
        }
    }).catch(() => {
    })
}
const handleStopAll = () => {
    const loading = ElLoading.service({
        lock: true,
        text: 'Loading...',
        background: 'rgba(0, 0, 0, 0.7)',
    })
    var params: Params = {}
    stopAll(params).then(res => {
        if (res.code == 200) {
            tableData.value = res.data
        } else {
            ElMessageBox.alert(res.msg, '发生错误', {
                confirmButtonText: '确认'
            })
        }
    }).catch(err => {
        ElMessageBox.alert(err.response != undefined && err.response.data != undefined ? err.response.data : err.message, '发生错误', {
            confirmButtonText: '确认'
        })
    }).finally(() => {
        loading.close()
    })
}
const handledDelAll = () => {
    const loading = ElLoading.service({
        lock: true,
        text: 'Loading...',
        background: 'rgba(0, 0, 0, 0.7)',
    })
    if (menu.value.active == "1") {
        var params: Params = {}
        delAll(params).then(res => {
            if (res.code == 200) {
                tableData.value = res.data
            } else {
                ElMessageBox.alert(res.msg, '发生错误', {
                    confirmButtonText: '确认'
                })
            }
        }).catch(err => {
            ElMessageBox.alert(err.response != undefined && err.response.data != undefined ? err.response.data : err.message, '发生错误', {
                confirmButtonText: '确认'
            })
        }).finally(() => {
            loading.close()
        })
    } else {
        var params: Params = {
            "status": menu.value.active
        }
        cleanUp(params).then(res => {
            if (res.code == 200) {
                tableData.value = res.data
            } else {
                ElMessageBox.alert(res.msg, '发生错误', {
                    confirmButtonText: '确认'
                })
            }
        }).catch(err => {
            ElMessageBox.alert(err.response != undefined && err.response.data != undefined ? err.response.data : err.message, '发生错误', {
                confirmButtonText: '确认'
            })
        }).finally(() => {
            loading.close()
        })
    }
}
const handleNewDownBtn = () => {
    form.Url = "";
    form.Filename = "";
    form.SavePath = conf.SavePath;
    form.Threads = conf.Threads;
    form.Referer = conf.Referer;
    form.UserAgent = conf.UserAgent;
    form.Cookie = conf.Cookie;
    form.Origin = conf.Origin;
    form.Proxy = conf.Proxy;
    dialogVisible.value = true;
}
const handleAdd = () => {
    if (form.Filename.trim() == "") {
        ElMessageBox.alert('请填写保存名称', '错误', {
            confirmButtonText: 'OK'
        })
        return;
    }
    if (form.SavePath.trim() == "") {
        ElMessageBox.alert('请填写保存位置', '错误', {
            confirmButtonText: 'OK'
        })
        return;
    }
    if (form.Url.trim() == "") {
        ElMessageBox.alert('请填写文件链接', '错误', {
            confirmButtonText: 'OK'
        })
        return;
    }
    const loading = ElLoading.service({
        lock: true,
        text: 'Loading...',
        background: 'rgba(0, 0, 0, 0.7)',
    })
    var params: Params = {
        'Url': form.Url,
        'Filename': form.Filename,
        'SavePath': form.SavePath,
        'Threads': form.Threads,
        'Referer': form.Referer,
        'UserAgent': form.UserAgent,
        'Cookie': form.Cookie,
        'Origin': form.Origin,
        'Proxy': form.Proxy,
    }
    add(params).then(res => {
        if (res.code == 200) {
            dialogVisible.value = false
            tableData.value = res.data
        } else {
            ElMessageBox.alert(res.msg, '发生错误', {
                confirmButtonText: '确认'
            })
        }
    }).catch(err => {
        ElMessageBox.alert(err.response != undefined && err.response.data != undefined ? err.response.data : err.message, '发生错误', {
            confirmButtonText: '确认'
        })
    }).finally(() => {
        loading.close()
    })
}
const handleGetConf = () => {
    var params: Params = {}
    getConf(params).then(res => {
        if (res.code == 200) {
            conf.Cookie = res.data.Cookie
            conf.SavePath = res.data.SavePath
            conf.Threads = res.data.Threads
            conf.Referer = res.data.Referer
            conf.UserAgent = res.data.UserAgent
            conf.Origin = res.data.Origin
            conf.Proxy = res.data.Proxy
            conf.Token = res.data.Token
            conf.maxWorkers = res.data.maxWorkers
        }
    }).catch(() => {
    })
}
const handleEditConf = (key: string, value: string) => {
    const loading = ElLoading.service({
        lock: true,
        text: 'Loading...',
        background: 'rgba(0, 0, 0, 0.7)',
    })
    var params: Params = {
        'key': key,
        'value': value,
    }
    setConf(params).then(res => {
        if (res.code == 200) {
            if (key == "Token") {
                setToken(value, true)
            }
            ElMessage({
                message: 'success',
                type: 'success',
            })
        } else {
            ElMessageBox.alert(res.msg, '发生错误', {
                confirmButtonText: '确认'
            })
        }
    }).catch(err => {
        ElMessageBox.alert(err.response != undefined && err.response.data != undefined ? err.response.data : err.message, '发生错误', {
            confirmButtonText: '确认'
        })
    }).finally(() => {
        loading.close()
        handleGetConf()
    })
}
</script>
<style scoped>
.custom-context-menu {
    position: fixed;
    background: white;
    border: 1px solid #e4e7ed;
    border-radius: 4px;
    box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.1);
    z-index: 3000;
    min-width: 150px;
}

.context-menu-item {
    padding: 8px 16px;
    cursor: pointer;
    font-size: 14px;
}

.context-menu-item:hover {
    background-color: #f5f7fa;
}
</style>