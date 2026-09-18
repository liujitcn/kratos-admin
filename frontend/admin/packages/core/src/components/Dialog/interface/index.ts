/** 弹窗异步打开配置。 */
export interface DialogOpenOptions<T> {
  /** 加载弹窗需要展示的数据，不应在异步过程中修改页面状态。 */
  load: () => Promise<T> | T;
  /** 最新一次加载完成后提交数据并更新页面状态。 */
  commit: (data: T) => void;
}

/** 弹窗异步打开控制器。 */
export interface DialogRequestController {
  /** 执行一次异步打开请求，并丢弃过期请求的结果。 */
  open<T>(options: DialogOpenOptions<T>): Promise<boolean>;
  /** 使当前未完成的打开请求失效。 */
  invalidate(): void;
}

/** 创建弹窗异步打开控制器。 */
export function createDialogRequestController(): DialogRequestController {
  let requestSerial = 0;

  return {
    async open<T>({ load, commit }: DialogOpenOptions<T>) {
      const currentSerial = ++requestSerial;
      let data: T;
      try {
        data = await load();
      } catch (error) {
        if (currentSerial !== requestSerial) return false;
        throw error;
      }
      if (currentSerial !== requestSerial) return false;
      commit(data);
      return currentSerial === requestSerial;
    },
    invalidate() {
      requestSerial += 1;
    }
  };
}
