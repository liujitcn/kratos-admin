import type { ConfigEnv, Plugin, UserConfig } from 'vite'

export declare const defineConfig: typeof import('vite').defineConfig
export declare const loadConfigFromFile: typeof import('vite').loadConfigFromFile
export declare const loadEnv: typeof import('vite').loadEnv
export declare function createKratosUniPlugin(): Plugin
export type { ConfigEnv, Plugin, UserConfig }
