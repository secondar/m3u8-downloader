import axios from "axios";
import type { AxiosRequestConfig, AxiosInstance, AxiosResponse, InternalAxiosRequestConfig } from 'axios'
import { getToken, setToken } from "./auth";
interface RequestHeaders extends Record<string, any> {
    isToken?: boolean;
    "X-Token"?: string;
}

interface RequestConfig extends AxiosRequestConfig {
    headers?: Partial<Record<string, string>> & RequestHeaders;
    isBlob?: boolean; // 是否返回二进制数据
    isDownload?: boolean; // 是否处理为文件下载
}
interface ErrorResponse {
    message: string;
    code?: number;
    config?: AxiosRequestConfig;
}

// 创建axios实例
const service: AxiosInstance = axios.create({
    baseURL: import.meta.env.VITE_API_BASE_URL,
    timeout: 10000,
});

// 请求拦截器
service.interceptors.request.use(
    (config: InternalAxiosRequestConfig) => {
        const token = getToken();
        if (token) {
            (config.headers as RequestHeaders) = {
                ...config.headers,
                "X-Token": token,
                "Content-Type": "application/x-www-form-urlencoded",
            };
        } else {
            (config.headers as RequestHeaders) = {
                ...config.headers,
                "Content-Type": "application/x-www-form-urlencoded",
            };
        }
        return config;
    },
    (error: any) => {
        return Promise.reject(error);
    }
);

// 响应拦截器
service.interceptors.response.use(
    (response: AxiosResponse) => {
        const config = response.config as RequestConfig;
        // 处理二进制响应
        if (config.isBlob || config.isDownload) {
            return response;
        }
        return Promise.resolve(response);
    },
    (error: ErrorResponse) => {
        const err = error as any;
        if (err.response !== undefined && err.response.data !== undefined && err.response.data.trim() === "Invalid X-Token") {
            if (getToken() != "") {
                setToken("", true);
                window.location.reload();
            }
        }
        return Promise.reject(error);
    }
);

export function get<T = any, P = Record<string, any>>(
    url: string,
    params?: P,
    config?: RequestConfig
): Promise<T> {
    return service({
        method: "get",
        url,
        params,
        ...config,
    }).then((response) => response.data);
}

export function post<T = any, D = Record<string, any>>(
    url: string,
    data?: D,
    config?: RequestConfig
): Promise<T> {
    return service({
        method: "post",
        url,
        data,
        ...config,
    }).then((response) => response.data)
}

export function put<T = any, D = Record<string, any>>(
    url: string,
    data?: D,
    config?: RequestConfig
): Promise<T> {
    return service({
        method: "put",
        url,
        data,
        ...config,
    }).then((response) => response.data)
}

export function del<T = any, D = Record<string, any>>(
    url: string,
    data?: D,
    config?: RequestConfig
): Promise<T> {
    return service({
        method: "delete",
        url,
        data,
        ...config,
    }).then((response) => response.data)
}
export default service;
